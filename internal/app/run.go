package app

import (
	"commentservice/internal/api"
	"commentservice/internal/infrastructure/config"
	"commentservice/internal/service"
	transport "commentservice/internal/transport/http"
	"commentservice/storage"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	kfk "github.com/Fau1con/kafkawrapper"
	"github.com/gorilla/mux"
)

// Run запускает Commentservice приложение
func Run() error {
	ctxMain, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig("configs/dev.yaml")
	if err != nil {
		log.Println("failed to load config from config file")
		return fmt.Errorf("failed to load config from config file: %w", err)
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // cfg.Logging.Level
	}))

	commentStorage, err := storage.NewCommentStorage(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to connect to comments database: %w", err)
	}
	defer commentStorage.Close()

	newsStorage, err := storage.NewNewsStorage(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to connect to news database: %w", err)
	}
	defer newsStorage.Close()

	commentService := service.NewCommentService(commentStorage, newsStorage, log)

	apiInstance := api.NewApi(mux.NewRouter(), commentService)

	// port := fmt.Sprintf("%d", cfg.GetPort())
	// addr := "localhost:" + port

	kafkaBrokers := cfg.Kafka.Brokers
	if len(kafkaBrokers) == 0 {
		kafkaBrokers[0] = "kafka:9093"
	}
	consumer, err := kfk.NewConsumer(kafkaBrokers, "news_input")
	if err != nil {
		log.Error("failed to create Kafka consumer: ",
			slog.Any("%v\n", kafkaBrokers))
		return err
	}
	producer, err := kfk.NewProducer(kafkaBrokers)
	log.Info("producer created! Broker: ",
		slog.Any("%v\n", kafkaBrokers))
	if err != nil {
		log.Error("failed to create Kafka poducer",
			slog.Any("%v\n", err))
		return err
	}

	// Горутина для обработки Kafka сообщений

	go func() {
		for {
			log.Info("start getting message and redirecting")
			msg, err := consumer.GetMessages(ctxMain)
			if err != nil {
				log.Error("failed to read message from Kafka",
					slog.Any("%v\n", err))
				continue
			}

			path := string(msg.Value)
			if strings.TrimSpace(path) == "" {
				log.Error("failed to read data from Kafka message",
					slog.Any("error", fmt.Errorf("path is empty")))
				continue
			}

			data, err := sendRequestToLocalhost(string(msg.Value))
			if err != nil {
				log.Error("failed to read data from Kafka message",
					slog.Any("%v\n", err))
				continue
			}

			if strings.Contains(string(msg.Value), "/comments/?newsID=") {
				err := producer.SendMessage(ctxMain, cfg.Kafka.Topics.Comments, data)
				if err != nil {
					log.Error("failed to write messageto Kafka",
						slog.Any("%v\n", err))
					return
				}
			}
			if strings.Contains(string(msg.Value), "/addcomment/?newsID=&comment=") {
				err := producer.SendMessage(ctxMain, cfg.Kafka.Topics.AddComment, data)
				if err != nil {
					log.Error("failed to write message to Kafka",
						slog.Any("%v\n", err))
					return
				}
			}
		}
	}()

	var handler http.Handler = apiInstance.Router()
	handler = transport.CORSMiddleware()(handler)
	handler = transport.RequestIDMiddleware(handler)
	handler = transport.LoggingMiddleware(log)(handler)

	log.Info("server commentservice APP start working at port",
		slog.Any("%v\n", cfg.HTTP.Port))
	return http.ListenAndServe(":"+strconv.Itoa(cfg.HTTP.Port), handler)

	// quit := make(chan os.Signal, 1)
	// signal.Notify(quit, os.Interrupt.syscall.SIGTERM)
	// <-quit
	// log.Info("Shutdown server...")
	// ctxShutdown, cancelSutdown := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancelSutdown()

	// return server.Shutdown(ctxShutdown)
}

// sendRequestToLocalhost выполняет HTTP запрос к локальному сервису
func sendRequestToLocalhost(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	url := fmt.Sprintf("http://localhost:8081%s", path)
	req, err := http.NewRequest(http.MethodGet, url, nil) // TODO: Impl Method POST
	if err != nil {
		log.Printf("failed to create request: %v\n", err)
		return nil, err
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response: %v\n", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected response code: %d\n", resp.StatusCode)
	}
	return body, nil
}

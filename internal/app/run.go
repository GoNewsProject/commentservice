package app

import (
	"commentservice/internal/api"
	"commentservice/internal/infrastructure/config"
	transport "commentservice/internal/transport/http"
	"commentservice/storage"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	kfk "github.com/Fau1con/kafkawrapper"
)

// Run запускает Commentservice приложение
func Run(configPath string) error {
	ctxMain, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Println("Failed to load config from config file")
		return fmt.Errorf("Failed to load config from config file: %w", err)
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // или cfg.Logging.Level?
	}))

	newsBaseURL := os.Getenv("NEWS_SERVICE_URL")
	if newsBaseURL = "" {
		newsBaseURL = "http://localhost:6000"
	}
	newsClient := storage.NewNewsAPIClient(newsBaseURL, log)

	pool, err := storage.NewCommentStorage(ctxMain, "comment_db", log, newsClient)
	if err != nil {
		return fmt.Errorf("failed to create comment storage: %w", err)
	}
	defer pool.Close()

	port := fmt.Sprintf("%d", cfg.GetPort())
	addr := "localhost:" + port

	commentProducer, err := kfk.NewProducer([]string{"localhost:9093"})
	if err != nil {
		log.Error("failed to create comment producer",
			slog.Any("%v\n", err))
		return err
	}

	commentConsumer, err := kfk.NewConsumer([]string{"localhost:9093"}, "comments")
	if err != nil {
		log.Error("failed to create comment consumer",
			slog.Any("%v\n", err))
		return err
	}

	addCommentConsumer, err := kfk.NewConsumer([]string{"localhost:9093"}, "add_comments")
	if err != nil {
		log.Error("failed to create addComment consumer",
			slog.Any("%v\n", err))
		return err
	}

	topics := api.Topics{
		Comments:     cfg.Kafka.Topics.Comment,
		AddComment:   cfg.Kafka.Topics.AddComment,
		CommentInput: cfg.Kafka.Topics.CommentInput,
	}
	apiInstance := api.New(
		ctxMain,
		commentProducer,
		commentConsumer,
		addCommentConsumer,
		log,
		topics,
	)

	var handler http.Handler = apiInstance.Router()
	handler = transport.RequestIDMiddleware(handler)
	handler = transport.CORSMiddleware()(handler)
	handler = transport.LoggingMiddleware(log)(handler)

	log.Info(
		"Starting API commentservice server at:",
		slog.Any("address", addr))

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(
				"Server error",
				"error", err,
			)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info("Shutdown server...")
	ctxShutdown, cancelSutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelSutdown()

	return server.Shutdown(ctxShutdown)
}

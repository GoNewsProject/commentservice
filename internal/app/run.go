package app

import (
	conf "commentservice/internal/infrastructure/config"
	"fmt"
	"log"
	"os"
)

func Run(configPath string) error {
	cfg, err := conf.LoadConfig(configPath)
	if err != nil {
		log.Println("Failed to load config from config file")
		return nil, fmt.Errorf("Failed to load config from config file: %w", err)
	}

	port := os.Getenv("PORT")
	addr := "localhost: + port"

	commentProducer, err := kfk.NewProducer([]string{"localhost:9093"})
	if err != nil {
		log.Printf("Failed to create comment producer: %v\n", err)
		returnt err
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.Logging.Level, // Верна ли передача из cfg?
	}))

	topics := 
}

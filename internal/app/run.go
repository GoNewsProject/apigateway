package app

import (
	"apigateway/internal/api"
	conf "apigateway/internal/infrastructure/config"
	"apigateway/internal/models"
	transport "apigateway/internal/transport/http"
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

// Run запускает API Gateway приложение
func Run(configPath string) error {
	ctxMain, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := conf.LoadConfig(configPath)
	if err != nil {
		log.Printf("Failed to load config from config file")
		return fmt.Errorf("failed to load config from config file: %w", err)
	}

	port := os.Getenv("PORT")
	addr := "localhost:" + port

	responseChan := make(chan models.DetailedResponse, 2)

	// Инициализация Kafka клиентов
	newsProducer, err := kfk.NewProducer([]string{"localhost:9093"})
	if err != nil {
		log.Printf("failed to create news producer: %v\n", err)
		return err
	}

	commentsProducer, err := kfk.NewProducer([]string{"localhost:9093"})
	if err != nil {
		log.Printf("failed to create comment producer: %v\n", err)
		return err
	}

	detailConsumer, err := kfk.NewConsumer([]string{"localhost:9093"}, "newsdetail")
	if err != nil {
		log.Printf("failet to create detail consumer: %v\n", err)
		return err
	}

	listConsumer, err := kfk.NewConsumer([]string{"localhost:9093"}, "newslist")
	if err != nil {
		log.Printf("failed to create list consumer: %v\n", err)
		return err
	}

	filterContentConsumer, err := kfk.NewConsumer([]string{"localhost:9093"}, "filtered_content")
	if err != nil {
		log.Printf("failed to create filter content consumer: %v\n", err)
		return err
	}

	filterPublishedConsumer, err := kfk.NewConsumer([]string{"localhost:9093"}, "filter_published")
	if err != nil {
		log.Printf("failed to create filter published consumer: %v\n", err)
		return err
	}

	commentsConsumer, err := kfk.NewConsumer([]string{"localhost:9093"}, "comments")
	if err != nil {
		log.Printf("failed to create comment consumer: %v\n", err)
		return err
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	topics := api.Topics{
		NewsInput:     cfg.Kafka.Topics.NewsInput,
		CommentsInput: cfg.Kafka.Topics.CommentsInput,
		AddComments:   cfg.Kafka.Topics.AddComments,
	}
	// Создание API и настройка middleware
	apiInstance := api.New(
		ctxMain,
		responseChan,
		newsProducer,
		commentsProducer,
		detailConsumer,
		listConsumer,
		filterContentConsumer,
		filterPublishedConsumer,
		commentsConsumer,
		log,
		topics,
		cfg.App.DefaultNewsLimit,
	)

	var handler http.Handler = apiInstance.Router()
	handler = transport.RequestIDMiddleware(handler)
	handler = transport.CORSMiddleware()(handler)
	handler = transport.LoggingMiddleware(log)(handler)

	log.Info(
		"Starting API gateway server at:",
		slog.Any("address", addr),
	)
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

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info("Shutdown server...")
	ctxShutDown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	return server.Shutdown(ctxShutDown)
}

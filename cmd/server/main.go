package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hyuko21/pubsub-golang/internal/pubsub"
	"github.com/hyuko21/pubsub-golang/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	log.Println("Starting Peril server...")
	url := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Error connecting to Rabbimq server: %s\n", err)
	}
	defer conn.Close()
	log.Println("Connected to Rabbitmq server")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error opening connection channel: %s", err)
	}
	err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	})
	if err != nil {
		log.Fatalf("Error publishing message with channel: %s", err)
	}
	log.Printf("Message published to exchange: '%s' and key: '%s'", routing.ExchangePerilDirect, routing.PauseKey)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server gracefully stopped")
}

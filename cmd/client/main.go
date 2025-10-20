package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hyuko21/pubsub-golang/internal/gamelogic"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	log.Println("Starting Peril client...")
	url := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Error connecting to Rabbitmq server: %s", err)
	}
	defer conn.Close()
	log.Println("Connected to Rabbitmq server")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error welcoming new user: %s", err)
	}
	log.Printf("New user connected: %s", username)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Client gracefully disconnected")
}

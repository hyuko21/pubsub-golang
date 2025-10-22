package main

import (
	"fmt"
	"log"

	"github.com/hyuko21/pubsub-golang/internal/gamelogic"
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

	subscribeToGameLogs(conn)

	gamelogic.PrintServerHelp()
gameloop:
	for {
		ch, err := conn.Channel()
		if err != nil {
			log.Fatalf("Error opening connection channel: %s", err)
		}
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "pause":
			log.Println("Sending pause message...")
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
		case "resume":
			log.Println("Sending resume message...")
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
		case "quit":
			log.Println("Ending game...")
			break gameloop
		case "help":
			gamelogic.PrintServerHelp()
		default:
			log.Printf("unkown command: '%s'\n", input[0])
			continue
		}
		if err != nil {
			log.Fatalf("Error publishing message with channel: %s", err)
		}
	}
	log.Println("Game is done.")
}

func subscribeToGameLogs(conn *amqp.Connection) {
	routingKey := fmt.Sprintf("%s.*", routing.GameLogSlug)
	if err := pubsub.SubscribeGob(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routingKey, pubsub.DurableQueue, handlerGameLogs); err != nil {
		log.Fatalf("Error subscribing to queue: %s", err)
	}
}

func handlerGameLogs(gamelog routing.GameLog) pubsub.AckType {
	defer fmt.Print("> ")
	if err := gamelogic.WriteLog(gamelog); err != nil {
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}

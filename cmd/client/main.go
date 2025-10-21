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
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	_, q, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.TransientQueue)
	if err != nil {
		log.Fatalf("Error declaring queue: %s", err)
	}
	log.Printf("Created new queue for user: '%s'\n", q.Name)

	state := gamelogic.NewGameState(username)
gameloop:
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "spawn":
			err := state.CommandSpawn(input)
			if err != nil {
				log.Println(err)
				continue
			}
		case "move":
			_, err := state.CommandMove(input)
			if err != nil {
				log.Println(err)
			}
			log.Println("Army in motion...")
		case "spam":
			log.Println("Spamming not allowed yet!")
		case "status":
			state.CommandStatus()
		case "quit":
			gamelogic.PrintQuit()
		case "help":
			gamelogic.PrintClientHelp()
			break gameloop
		default:
			log.Printf("unknown command: '%s'\n", input[0])
			continue
		}
	}
}

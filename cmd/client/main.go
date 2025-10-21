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

	state := gamelogic.NewGameState(username)
	subscribeToPause(conn, state, username)
	subscribeToArmyMoves(conn, state, username)
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
			move, err := state.CommandMove(input)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("Army in motion...")
			publishMove(conn, username, move)
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

func getChannel(conn *amqp.Connection) *amqp.Channel {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error opening connection channel: %s", err)
	}
	return ch
}

func subscribeToPause(conn *amqp.Connection, gs *gamelogic.GameState, username string) {
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	err := pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.TransientQueue, handlerPause(gs))
	if err != nil {
		log.Fatalf("Error subscribing to queue: %s", err)
	}
}

func subscribeToArmyMoves(conn *amqp.Connection, gs *gamelogic.GameState, username string) {
	queueName := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
	routingKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
	err := pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueName, routingKey, pubsub.TransientQueue, handlerMove(gs))
	if err != nil {
		log.Fatalf("Error subscribing to queue: %s", err)
	}
}

func publishMove(conn *amqp.Connection, username string, move gamelogic.ArmyMove) {
	routingKey := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
	err := pubsub.PublishJSON(getChannel(conn), routing.ExchangePerilTopic, routingKey, move)
	if err != nil {
		log.Fatalf("Error publishing to queue: %s", err)
	}
	log.Println("Published move event to queue")
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")

		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")

		gs.HandleMove(move)
	}
}

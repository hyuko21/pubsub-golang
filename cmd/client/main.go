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
	subscribeToWarRecognitions(conn, state)
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
	err := pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueName, routingKey, pubsub.TransientQueue, handlerMove(gs, conn))
	if err != nil {
		log.Fatalf("Error subscribing to queue: %s", err)
	}
}

func subscribeToWarRecognitions(conn *amqp.Connection, gs *gamelogic.GameState) {
	routingKey := fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix)
	err := pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, routingKey, pubsub.DurableQueue, handlerWarRecognitions(gs, conn))
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

func publishGameLog(conn *amqp.Connection, username, gLog string) error {
	routingKey := fmt.Sprintf("%s.%s", routing.GameLogSlug, username)
	if err := pubsub.PublishGob(getChannel(conn), routing.ExchangePerilTopic, routingKey, gLog); err != nil {
		return err
	}
	return nil
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, conn *amqp.Connection) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		switch gs.HandleMove(move) {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			routingKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, move.Player.Username)
			err := pubsub.PublishJSON(getChannel(conn), routing.ExchangePerilTopic, routingKey, gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: gs.GetPlayerSnap(),
			})
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		}
		return pubsub.NackDiscard
	}
}

func handlerWarRecognitions(gs *gamelogic.GameState, conn *amqp.Connection) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon:
			gLog := fmt.Sprintf("%s won a war against %s", winner, loser)
			if err := publishGameLog(conn, rw.Attacker.Username, gLog); err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			gLog := fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			if err := publishGameLog(conn, rw.Attacker.Username, gLog); err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			log.Println("Unknown war outcome")
		}
		return pubsub.NackDiscard
	}
}

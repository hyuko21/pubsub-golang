package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType uint8

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, func(data []byte) (T, error) {
		var t T
		buf := bytes.NewBuffer(data)
		decoder := json.NewDecoder(buf)
		if err := decoder.Decode(&t); err != nil {
			return t, err
		}
		return t, nil
	})
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, func(data []byte) (T, error) {
		var t T
		buf := bytes.NewBuffer(data)
		decoder := gob.NewDecoder(buf)
		if err := decoder.Decode(&t); err != nil {
			return t, err
		}
		return t, nil
	})
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, q, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("error declaring queue: %s", err)
	}
	deliveryCh, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error consuming queue: %s", err)
	}
	go func() {
		for msg := range deliveryCh {
			msgData, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("Error processing message data from queue %s: %s", q.Name, err)
				continue
			}
			switch handler(msgData) {
			case Ack:
				log.Println("Acknowledging message...")
				msg.Ack(false)
			case NackRequeue:
				log.Println("Requeuing message...")
				msg.Nack(false, true)
			case NackDiscard:
				log.Println("Discarding message...")
				msg.Nack(false, false)
			}
		}
	}()
	return nil
}

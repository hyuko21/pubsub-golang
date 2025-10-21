package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T),
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
			var msgData T
			err := json.Unmarshal(msg.Body, &msgData)
			if err != nil {
				fmt.Printf("Error processing message data from queue %s: %s", q.Name, err)
				continue
			}
			handler(msgData)
			msg.Ack(false)
		}
	}()
	return nil
}

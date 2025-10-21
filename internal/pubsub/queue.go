package pubsub

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType uint8

const (
	DurableQueue SimpleQueueType = iota
	TransientQueue
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to create channel: %s", err)
	}
	isDurable, isAutoDelete, isExclusive := getQueueOptionsForType(queueType)
	q, err := ch.QueueDeclare(queueName, isDurable, isAutoDelete, isExclusive, false, amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	})
	if err != nil {
		return nil, q, fmt.Errorf("failed to create queue: %s", err)
	}
	err = ch.QueueBind(q.Name, key, exchange, false, nil)
	if err != nil {
		return nil, q, fmt.Errorf("failed to bind queue: %s", err)
	}
	return ch, q, nil
}

func getQueueOptionsForType(queueType SimpleQueueType) (isDurable, isAutoDelete, isExclusive bool) {
	switch queueType {
	case DurableQueue:
		isDurable = true
	case TransientQueue:
		isAutoDelete = true
		isExclusive = true
	}
	return
}

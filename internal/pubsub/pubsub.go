package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
	"context"
)

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	msg := amqp.Publishing{ContentType: "application/json", Body: data}
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)

	return err
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	
	connChan, err := conn.Channel()
	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	durable := (queueType == Durable)
	autoDelete := (queueType == Transient)
	exclusive := (queueType == Transient)

	declaredQueue, err := connChan.QueueDeclare(queueName, durable, autoDelete, exclusive, false, nil)

	err = connChan.QueueBind(queueName, key, exchange, false, nil)

	return connChan, declaredQueue, err
}

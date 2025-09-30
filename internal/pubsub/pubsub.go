package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
	"encoding/gob"
	"context"
	//"log"
	"errors"
	"fmt"
	"bytes"
)

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

type AckType int
const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var encodeBuffer bytes.Buffer
	encoder := gob.NewEncoder(&encodeBuffer)

	err := encoder.Encode(val)
	if err != nil {
		return err
	}

	msg := amqp.Publishing{ContentType: "application/gob", Body: encodeBuffer.Bytes(), DeliveryMode: amqp.Persistent}
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)

	return err
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	msg := amqp.Publishing{ContentType: "application/json", Body: data, DeliveryMode: amqp.Persistent}
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)

	return err
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliverChan, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for body := range deliverChan {
			var data T
			err := json.Unmarshal(body.Body, &data)
			if err != nil {
				//log.Fatal(err)
				continue
			}
			outcome := handler(data)
			switch outcome {
				case Ack:
					err = body.Ack(false)
					fmt.Println("Client ACK")
				case NackDiscard:
					err = body.Nack(false, false)
					fmt.Println("Client DISCARD")
				case NackRequeue:
					err = body.Nack(false, true)
					fmt.Println("Client REQUEUE")
				default:
					err = errors.New(fmt.Sprintf("Unknown outcome detected: %s", outcome))
				
				if err != nil {
					//log.Fatal(err)
					continue
				}
			}
		}
	}()

	return nil
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

	dlxKey := "x-dead-letter-exchange"

	table := amqp.Table{}
	table[dlxKey] = "peril_dlx"

	declaredQueue, err := connChan.QueueDeclare(queueName, durable, autoDelete, exclusive, false, table)

	if err != nil {
		connChan.Close()
		return nil, amqp.Queue{}, err
	}

	err = connChan.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		connChan.Close()
		return nil, amqp.Queue{}, err
	}

	return connChan, declaredQueue, nil
}

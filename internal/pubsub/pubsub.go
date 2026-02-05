package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	QueueDurable   SimpleQueueType = iota //0
	QueueTransient                        // 1
)

var queueType = map[SimpleQueueType]string{
	QueueDurable:   "durable",
	QueueTransient: "transient",
}

func (qt SimpleQueueType) String() string {
	return queueType[qt]
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}
	pub := amqp.Publishing{
		ContentType: "application/json",
		Body:        bytes,
	}
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, pub)
	if err != nil {
		log.Fatalf("Error publishing to channel: %v", err)
	}
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error creating channel: %v", err)
	}

	var durable bool
	var autoDelete bool
	var exclusive bool

	switch queueType {
	case QueueDurable:
		durable = true
		autoDelete = false
		exclusive = false
	case QueueTransient:
		durable = false
		autoDelete = true
		exclusive = true
	}

	q, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, nil)
	if err != nil {
		log.Fatalf("Error declaring queue: %v", err)
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		log.Fatalf("Error binding queue: %v", err)
	}

	return ch, q, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {

	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		log.Fatalf("Error during declare and bind process: %v", err)
	}

	deliveryCh, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error with consumption: %v", err)
	}

	go func() {
		for msg := range deliveryCh {
			var data T
			err := json.Unmarshal(msg.Body, &data)
			if err != nil {
				log.Fatalf("Error unmarshalling data: %v", err)
			}
			handler(data)
			err = msg.Ack(false)
			if err != nil {
				log.Fatalf("Ack error: %v", err)
			}
		}
	}()

	return nil
}

package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	QueueDurable   SimpleQueueType = iota //0
	QueueTransient                        // 1
)

type AckType int

const (
	Ack         AckType = iota //0
	NackRequeue                //1
	NackDiscard                //2
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	pub := amqp.Publishing{
		ContentType: "application/json",
		Body:        bytes,
	}
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, pub)
	if err != nil {
		log.Fatalf("Error publishing to channel: %v\n", err)
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
		return nil, amqp.Queue{}, err
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
	default:
		log.Fatal("Unknown queue type!\n")
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}

	q, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, queueArgs)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, q, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {

	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliveryCh, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range deliveryCh {
			var data T
			err := json.Unmarshal(msg.Body, &data)
			if err != nil {
				log.Fatalf("Error unmarshalling data: %v\n", err)
			}
			switch handler(data) {
			case Ack:
				msg.Ack(false)
				fmt.Println("\nAck")
			case NackDiscard:
				msg.Nack(false, false)
				fmt.Println("\nNackDiscard")
			case NackRequeue:
				msg.Nack(false, true)
				fmt.Println("\nNackRequeue")
			default:
				log.Fatalf("Something went wrong: %v\n", err)
			}
		}
	}()

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var net bytes.Buffer
	enc := gob.NewEncoder(&net)
	err := enc.Encode(val)
	if err != nil {
		return err
	}
	pub := amqp.Publishing{
		ContentType: "application/gob",
		Body:        net.Bytes(),
	}
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, pub)
	if err != nil {
		return err
	}
	return nil
}

func PublishGameLog(ch *amqp.Channel, username, message string) error {
	gl := routing.GameLog{
		Username:    username,
		Message:     message,
		CurrentTime: time.Now(),
	}
	err := PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+username, gl)
	if err != nil {
		return err
	}
	return nil
}

package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

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

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		func(data []byte) (T, error) {
			var target T
			err := json.Unmarshal(data, &target)
			return target, err
		},
	)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		func(data []byte) (T, error) {
			out := bytes.NewBuffer(data)
			dec := gob.NewDecoder(out)
			var target T
			err := dec.Decode(&target)
			return target, err
		},
	)

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
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("Couldn't declare and bind queue: %v", err)
	}

	err = ch.Qos(10, 0, false)
	if err != nil {
		return fmt.Errorf("Couldn't apply prefetch: %v", err)
	}

	deliveryCh, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Couldn't consume messsages: %v", err)
	}

	go func() {
		defer ch.Close()
		for msg := range deliveryCh {
			data, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("Error unmarshalling data: %v\n", err)
				continue
			}
			switch handler(data) {
			case Ack:
				msg.Ack(false)
			case NackDiscard:
				msg.Nack(false, false)
			case NackRequeue:
				msg.Nack(false, true)
			default:
				log.Fatalf("Something went wrong: %v\n", err)
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

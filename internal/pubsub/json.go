package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

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

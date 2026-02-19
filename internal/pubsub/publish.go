package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
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

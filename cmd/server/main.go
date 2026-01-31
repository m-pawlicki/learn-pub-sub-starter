package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	connString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	connCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Channel error: %v", err)
	}
	pubsub.PublishJSON(connCh, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
	defer conn.Close()
	fmt.Println("Connection successful.")
	//waiting for ctrl+c
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	<-signalCh
	fmt.Println("Interrupt detected, shutting down...")
}

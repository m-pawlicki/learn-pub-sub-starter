package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	connString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Something went wrong logging in: %v", err)
	}
	pubsub.DeclareAndBind(conn, "peril_direct", "pause."+username, "pause", pubsub.QueueTransient)
	//waiting for ctrl+c
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	<-signalCh
	fmt.Println("Interrupt detected, shutting down...")
}

package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	connString := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatalf("Could not connect: %v\n", err)
	}
	defer conn.Close()

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Could not create channel: %v\n", err)
	}
	defer publishCh.Close()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Something went wrong logging in: %v\n", err)
	}

	gs := gamelogic.NewGameState(username)

	var (
		userPause = "pause." + username
		userMoves = "army_moves." + username
	)

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, userPause, routing.PauseKey, pubsub.QueueTransient, HandlerPause(gs))
	if err != nil {
		log.Fatalf("Error subscribing to pause exchange: %v\n", err)
	}

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, userMoves, routing.ArmyMovesPrefix+".*", pubsub.QueueTransient, HandlerMove(publishCh, gs))
	if err != nil {
		log.Fatalf("Error subscribing to moves exchange: %v\n", err)
	}

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "war", routing.WarRecognitionsPrefix+".*", pubsub.QueueDurable, HandlerWarOutcome(publishCh, gs))
	if err != nil {
		log.Fatalf("Error subscribing to war exchange: %v\n", err)
	}

client_loop:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			gs.CommandSpawn(words)
		case "move":
			mv, err := gs.CommandMove(words)
			if err != nil {
				fmt.Printf("Move failed: %v\n", mv)
			}
			err = pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, userMoves, mv)
			if err != nil {
				fmt.Printf("Error publishing move: %v\n", err)
				continue
			}
			fmt.Println("Move published successfully.")
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			break client_loop
		default:
			fmt.Println("Sorry, I don't understand that command.")
		}
	}
}

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(mv gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(mv)
	}
}

func main() {
	fmt.Println("Starting Peril client...")

	connString := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Could not create channel: %v", err)
	}
	defer publishCh.Close()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Something went wrong logging in: %v", err)
	}

	gs := gamelogic.NewGameState(username)

	var (
		userPause = "pause." + username
		userMoves = "army_moves." + username
	)

	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, userPause, routing.PauseKey, pubsub.QueueTransient, handlerPause(gs))
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, userMoves, "army_moves.*", pubsub.QueueTransient, handlerMove(gs))

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
				fmt.Println("Move failed.")
			}
			err = pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, userMoves, mv)
			if err != nil {
				fmt.Printf("Error publishing move: %v", err)
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
	//waiting for ctrl+c
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	<-signalCh
	fmt.Println("Interrupt detected, shutting down...")
}

package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"os/signal"
	"log"
	
	"github.com/adammatthes/learn-pub-sub-starter/internal/pubsub"
	"github.com/adammatthes/learn-pub-sub-starter/internal/routing"
	"github.com/adammatthes/learn-pub-sub-starter/internal/gamelogic"


)
func main() {
	fmt.Println("Starting Peril server...")

	connectionStr := "amqp://guest:guest@localhost:5672/"

	myConnection, err := amqp.Dial(connectionStr)
	if err != nil {
		fmt.Println("MQ connection failed:", err)
		return
	}

	defer myConnection.Close()

	fmt.Println("MQ Connection successful")

	
	connChan, err := myConnection.Channel()
	if err != nil {
		fmt.Println("Could not create connection channel")
		return
	}

	pubsub.DeclareAndBind(myConnection, routing.ExchangePerilTopic, "game_logs", "game_logs", pubsub.Durable)

	gamelogic.PrintServerHelp()

	for ;; {
		received := gamelogic.GetInput()
		if len(received) == 0 {
			continue
		}

		switch received[0] {
			case "pause":
				fmt.Println("Sending pause message")
				state := routing.PlayingState{IsPaused: true}

				err = pubsub.PublishJSON(connChan, routing.ExchangePerilDirect, routing.PauseKey, state)
				if err != nil {
					log.Fatal(err)
				}
			case "resume":
				fmt.Println("Sending resume message")
				state := routing.PlayingState{IsPaused: false}
				err = pubsub.PublishJSON(connChan, routing.ExchangePerilDirect, routing.PauseKey, state)
				if err != nil {
					log.Fatal(err)
				}
			case "quit":
				fmt.Println("Exiting loop")
				return
			default:
				fmt.Println("Invalid Command")
		}
	}

		
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("\nProgram shutting down...")
}

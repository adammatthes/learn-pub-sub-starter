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

func handlerWrite(gl routing.GameLog) pubsub.AckType {
	err := gamelogic.WriteLog(gl)
	if err != nil {
		return pubsub.NackDiscard
	}

	return pubsub.Ack
	
}

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

	pubsub.DeclareAndBind(myConnection, routing.ExchangePerilTopic, "game_logs", "game_logs.*", pubsub.Durable)

	err = pubsub.SubscribeGob(myConnection, routing.ExchangePerilTopic, "game_logs", "game_logs.*", pubsub.Durable, handlerWrite)
	if err != nil {
		fmt.Println("Server could not subscribe to game log:", err)
	}

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

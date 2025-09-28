package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"

	"os"
	"os/signal"

	"github.com/adammatthes/learn-pub-sub-starter/internal/gamelogic"
	"github.com/adammatthes/learn-pub-sub-starter/internal/pubsub"
	//"github.com/adammatthes/learn-pub-sub-starter/internal/routing"

)

func main() {
	fmt.Println("Starting Peril client...")

	connectionStr := "amqp://guest:guest@localhost:5672"

	myConnection, err := amqp.Dial(connectionStr)
	if err != nil {
		fmt.Println("MQ Connection Failed:", err)
		return
	}
	
	defer myConnection.Close()
	fmt.Println("Client connection succeeded")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println("Error getting username:", err)
		return
	}

	pubsub.DeclareAndBind(myConnection, "peril_direct", fmt.Sprintf("pause.%s", username), "pause", pubsub.Transient)

	gameState := gamelogic.NewGameState(username)

	selection := ""
	for ; selection != "quit"; {
		input := gamelogic.GetInput()
		selection = input[0]
		switch selection {
			case "spawn":
				err := gameState.CommandSpawn(input)
				if err == nil {
					fmt.Println("Spawn successful:")
				} else {
					fmt.Println("Spawn failed:", err)
				}
			case "move":
				m, err := gameState.CommandMove(input)
				if err == nil {
					fmt.Println("Move successful:", m)
				} else {
					fmt.Println("Move failed:", err, m)
				}
			case "status":
				gameState.CommandStatus()
			case "help":
				gamelogic.PrintClientHelp()
			case "spam":
				fmt.Println("Spamming not allowed yet")
			case "quit":
				gamelogic.PrintQuit()
			default:
				fmt.Println("Invalid Client command")
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("\nExiting Client...")
}

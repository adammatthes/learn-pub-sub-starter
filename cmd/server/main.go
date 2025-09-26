package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"os/signal"

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

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("\nProgram shutting down...")
}

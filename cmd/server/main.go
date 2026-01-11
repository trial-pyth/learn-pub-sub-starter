package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connString := "amqp://guest:guest@localhost:5672/"
	amqpConn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatal(err)
	}
	defer amqpConn.Close()

	fmt.Println("Connected to RabbitMQ successfully")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<- sigChan

	fmt.Println("Shutting down Peril Server...")
}

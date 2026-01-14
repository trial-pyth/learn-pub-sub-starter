package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerGameLog(log routing.GameLog) pubsub.AckType {
	defer fmt.Print("> ")
	err := gamelogic.WriteLog(log)
	if err != nil {
		return pubsub.NackDiscard
	}
	return pubsub.Ack
}

func main() {
	connString := "amqp://guest:guest@localhost:5672/"
	amqpConn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatal(err)
	}
	defer amqpConn.Close()

	fmt.Println("Server connected to RabbitMQ successfully")

	ch, err := amqpConn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	err = pubsub.SubscribeGob(amqpConn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", pubsub.Durable, handlerGameLog)
	if err != nil {
		log.Fatal(err)
	}

	gamelogic.PrintServerHelp()

	// REPL loop
	go func() {
		for {
			words := gamelogic.GetInput()
			if len(words) == 0 {
				continue
			}

			switch words[0] {
			case "pause":
				fmt.Println("Sending pause message...")
				_ = pubsub.PublishJSON(
					ch,
					routing.ExchangePerilTopic,
					routing.PauseKey,
					routing.PlayingState{IsPaused: true},
				)

			case "resume":
				fmt.Println("Sending resume message...")
				_ = pubsub.PublishJSON(
					ch,
					routing.ExchangePerilTopic,
					routing.PauseKey,
					routing.PlayingState{IsPaused: false},
				)

			case "quit":
				fmt.Println("Exiting server...")
				os.Exit(0)

			default:
				fmt.Println("Unknown command")
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Println("Shutting down Peril Server...")
}

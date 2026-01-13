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

func main() {
	connString := "amqp://guest:guest@localhost:5672/"
	amqpConn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatal(err)
	}
	defer amqpConn.Close()

	ch, err := amqpConn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	fmt.Println("Client connected to RabbitMQ successfully")

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	subCh, _, err := pubsub.DeclareAndBind(amqpConn, routing.ExchangePerilDirect, fmt.Sprintf("%s.%s", routing.PauseKey, userName), routing.PauseKey, pubsub.Transient)
	if err != nil {
		log.Fatal(err)
	}

	defer subCh.Close()

	gameState := gamelogic.NewGameState(userName)

	go func() {
		for {
			words := gamelogic.GetInput()
			if len(words) == 0 {
				continue
			}

			switch words[0] {
			case "spawn":
				err := gameState.CommandSpawn(words)
				if err != nil {
					log.Fatal(err)
				}
			case "move":
				mv, err := gameState.CommandMove(words)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("Successfully moved!", mv)
			case "status":
				gameState.CommandStatus()
			case "help":
				gamelogic.PrintClientHelp()
			case "spam":
				fmt.Println("Spamming not allowed yet")
			case "quit":
				gamelogic.PrintQuit()
			default:
				fmt.Println("Unknown command")
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Println("Shutting down Peril Client...")
}

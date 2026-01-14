package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

	gameState := gamelogic.NewGameState(userName)

	err = pubsub.SubscribeJSON(amqpConn, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.PauseKey, userName), routing.PauseKey, pubsub.Transient, handlerPause(gameState))
	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(amqpConn, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, userName), routing.ArmyMovesPrefix+".*", pubsub.Transient, handlerMove(gameState, ch))
	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(amqpConn, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, routing.WarRecognitionsPrefix+".*", pubsub.Durable, handlerWar(gameState, ch))
	if err != nil {
		log.Fatal(err)
	}

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
					fmt.Println(err)
					continue
				}
			case "move":
				mv, err := gameState.CommandMove(words)
				if err != nil {
					fmt.Println(err)
					continue
				}
				err = pubsub.PublishJSON(ch, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, userName), mv)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println("Move published successfully")
			case "status":
				gameState.CommandStatus()
			case "help":
				gamelogic.PrintClientHelp()
			case "spam":
				if len(words) < 2 {
					fmt.Println("Usage: spam <n>")
					continue
				}
				n, err := strconv.Atoi(words[1])
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					continue
				}
				for i := 0; i < n; i++ {
					maliciousMsg := gamelogic.GetMaliciousLog()
					gameLog := routing.GameLog{
						CurrentTime: time.Now(),
						Message:     maliciousMsg,
						Username:    userName,
					}
					routingKey := fmt.Sprintf("%s.%s", routing.GameLogSlug, userName)
					err := pubsub.PublishGob(ch, routing.ExchangePerilTopic, routingKey, gameLog)
					if err != nil {
						fmt.Printf("Error publishing log: %v\n", err)
						continue
					}
				}
			case "quit":
				gamelogic.PrintQuit()
				os.Exit(0)
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

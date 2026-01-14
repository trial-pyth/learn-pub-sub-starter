package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func publishGameLog(ch *amqp.Channel, username, message string) pubsub.AckType {
	gameLog := routing.GameLog{
		CurrentTime: time.Now(),
		Message:     message,
		Username:    username,
	}
	routingKey := fmt.Sprintf("%s.%s", routing.GameLogSlug, username)
	err := pubsub.PublishGob(ch, routing.ExchangePerilTopic, routingKey, gameLog)
	if err != nil {
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)
		if outcome == gamelogic.MoveOutComeSafe {
			return pubsub.Ack
		}
		if outcome == gamelogic.MoveOutcomeMakeWar {
			defender := gs.GetPlayerSnap()
			warMsg := gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: defender,
			}
			routingKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, defender.Username)
			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routingKey, warMsg)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}
		// NackDiscard for MoveOutcomeSamePlayer or anything else
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			logMsg := fmt.Sprintf("%s won a war against %s", winner, loser)
			return publishGameLog(ch, rw.Attacker.Username, logMsg)
		case gamelogic.WarOutcomeYouWon:
			logMsg := fmt.Sprintf("%s won a war against %s", winner, loser)
			return publishGameLog(ch, rw.Attacker.Username, logMsg)
		case gamelogic.WarOutcomeDraw:
			logMsg := fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			return publishGameLog(ch, rw.Attacker.Username, logMsg)
		default:
			fmt.Printf("Error: unknown war outcome: %v\n", outcome)
			return pubsub.NackDiscard
		}
	}
}

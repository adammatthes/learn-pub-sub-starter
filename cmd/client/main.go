package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"

	//"os"
	//"os/signal"

	"github.com/adammatthes/learn-pub-sub-starter/internal/gamelogic"
	"github.com/adammatthes/learn-pub-sub-starter/internal/pubsub"
	"github.com/adammatthes/learn-pub-sub-starter/internal/routing"

)

func publishGameLog(gs *gamelogic.GameState, c *amqp.Channel, outcomeStatement string) error {
	gl := routing.GameLog{CurrentTime: time.Now(),
				Message: outcomeStatement,
				Username: gs.Player.Username}
	
	err := pubsub.PublishGob(c,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.GameLogSlug, gs.Player.Username),
				gl)
	if err != nil {
		return err
	}
				
	return nil
}

func handlerWar(gs *gamelogic.GameState, c *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	defer fmt.Print("> ")
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		outcome, a, d := gs.HandleWar(rw)

		fmt.Println(outcome, a, d)

		not_involved := gamelogic.WarOutcomeNotInvolved
		no_units := gamelogic.WarOutcomeNoUnits
		opponent_won := gamelogic.WarOutcomeOpponentWon
		you_won := gamelogic.WarOutcomeYouWon
		draw := gamelogic.WarOutcomeDraw

		var outcomeStatement string

		if outcome == you_won {
			outcomeStatement = fmt.Sprintf("%s won a war against %s", a, d)
		} else if outcome == opponent_won {
			outcomeStatement = fmt.Sprintf("%s won a war against %s", d, a)
		} else if outcome == draw {
			outcomeStatement = fmt.Sprintf("A war between %s and %s resulted in a draw", a, d)
		} else {
			outcomeStatement = ""
		}

		if len(outcomeStatement) > 0 {
			err := publishGameLog(gs, c, outcomeStatement)
			if err != nil {
				return pubsub.NackRequeue
			}
		}

		switch outcome {
			case not_involved:
				return pubsub.NackRequeue
			case no_units:
				return pubsub.NackDiscard
			case opponent_won:
				return pubsub.Ack
			case you_won:
				return pubsub.Ack
			case draw:
				return pubsub.Ack
			default:
				fmt.Println("Unrecognized war outcome:", outcome)
		}
		return pubsub.NackDiscard
	}
}

func handlerMove(gs *gamelogic.GameState, c *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	defer fmt.Print("> ")
	return func(m gamelogic.ArmyMove) pubsub.AckType {
		outcome := gs.HandleMove(m)
		
		safe := gamelogic.MoveOutComeSafe
		make_war := gamelogic.MoveOutcomeMakeWar
		//same_player := gamelogic.MoveOutcomeSamePlayer

		if outcome == make_war {	
			attacker := m.Player

			row := gamelogic.RecognitionOfWar{Attacker: attacker, Defender: gs.Player}

			err := pubsub.PublishJSON(c, routing.ExchangePerilTopic, "war.#", row)
			if err != nil {
				fmt.Println("Error publishing in handlerMove:", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}

		if (outcome == safe || outcome == make_war) {
			return pubsub.Ack
		}

		return pubsub.NackDiscard
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	defer fmt.Print("> ")

	//s := routing.PlayingState{IsPaused: gs.Paused}

	return func (s routing.PlayingState) pubsub.AckType {
		gs.HandlePause(s)
		return pubsub.Ack
	}
}

func main() {
	fmt.Println("Starting Peril client...")

	connectionStr := "amqp://guest:guest@localhost:5672"

	myConnection, err := amqp.Dial(connectionStr)
	if err != nil {
		fmt.Println("MQ Connection Failed:", err)
		return
	}
	
	fmt.Println("Client connection succeeded")

	moveChan, err := myConnection.Channel()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println("Error getting username:", err)
		return
	}

	queuePause := fmt.Sprintf("pause.%s", username)



	_, _, err = pubsub.DeclareAndBind(myConnection, "peril_direct", "pause", queuePause, pubsub.Transient)
	if err != nil {
		fmt.Println("Error binding to pause queue:", err)
	}
	
	_, _, err = pubsub.DeclareAndBind(myConnection, "peril_topic", "war", "war.#", pubsub.Durable)
	if err != nil {
		fmt.Println("Error binding to war queue:", err)
	}

	_, _, err = pubsub.DeclareAndBind(myConnection, "peril_topic", "game_logs", fmt.Sprintf("%s.*", routing.GameLogSlug), pubsub.Durable)
	if err != nil {
		fmt.Println("Error binding to log queue:", err)
	}

	gameState := gamelogic.NewGameState(username)

	err1 := pubsub.SubscribeJSON(myConnection,
				routing.ExchangePerilDirect,
				"pause",
				queuePause,
				pubsub.Transient,
				handlerPause(gameState),
		)
	
	movesKey := fmt.Sprintf("army_moves.%s", username)
	err2 := pubsub.SubscribeJSON(myConnection,
					routing.ExchangePerilTopic,
					movesKey,
					"army_moves.*",
					pubsub.Transient,
					handlerMove(gameState, moveChan),
				)

	warKey := "war.#"
	err3 := pubsub.SubscribeJSON(myConnection,
					routing.ExchangePerilTopic,
					"war",
					warKey,
					pubsub.Durable,
					handlerWar(gameState, moveChan))


	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Println("Error SubscribeJSON:", err1, err2, err3)
		//return
	}

	selection := ""
	defer myConnection.Close()
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
				
				pubsub.PublishJSON(moveChan, routing.ExchangePerilTopic, movesKey, m)
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

	//signalChan := make(chan os.Signal, 1)
	//signal.Notify(signalChan, os.Interrupt)
	//<-signalChan

	fmt.Println("\nExiting Client...")
}

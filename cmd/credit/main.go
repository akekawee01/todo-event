package main

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	userdomain "todoe/domain/user/domain"
	"todoe/internal/event"
	"todoe/internal/messaging"
)

// fakeCreditAPI returns a deterministic score (300–850) based on email.
// Scores >= 600 are approved.
func fakeCreditAPI(email string) (score int, approved bool) {
	h := fnv.New32a()
	h.Write([]byte(email))
	score = 500 + int(h.Sum32()%551)
	return score, score >= 600
}

func main() {
	amqpURL := os.Getenv("AMQP_URL")
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	conn, ch, err := messaging.ConnectWithRetry(amqpURL, 10, 2)
	if err != nil {
		log.Fatal("rabbit: connection failed after retries:", err)
	}
	defer conn.Close()

	if err := messaging.DeclareExchange(ch, messaging.CreditResultExchange); err != nil {
		log.Fatal("rabbit declare:", err)
	}
	resultPublisher := messaging.NewPublisher(ch, messaging.CreditResultExchange)

	if err := messaging.Subscribe(ch, messaging.UserExchange, messaging.QueueCreditUserEvents, func(msg messaging.Message) {
		if msg.Type != userdomain.EventEmailVerified {
			return
		}

		var user userdomain.User
		if err := json.Unmarshal(msg.Payload, &user); err != nil {
			slog.Error("credit: unmarshal user", "err", err)
			return
		}

		score, approved := fakeCreditAPI(user.Email)
		slog.Info("credit: scored", "user_id", user.ID, "email", user.Email, "score", score, "approved", approved)

		resultPublisher.Publish(context.Background(), event.Event{
			Type: userdomain.EventCreditScored,
			Payload: userdomain.CreditScoredPayload{
				UserID:   user.ID,
				Score:    score,
				Approved: approved,
			},
		})
	}); err != nil {
		log.Fatal("rabbit subscribe:", err)
	}

	slog.Info("credit service listening", "exchange", messaging.UserExchange)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("credit service stopping")
}

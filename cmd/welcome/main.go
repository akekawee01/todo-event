package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	userdomain "todoe/domain/user/domain"
	"todoe/internal/messaging"
)

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

	if err := messaging.Subscribe(ch, messaging.UserExchange, messaging.QueueWelcomeUserEvents, func(msg messaging.Message) {
		switch msg.Type {
		case userdomain.EventRegistered:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			slog.Info("step 1/4: verification email sent",
				"user_id", user.ID, "to", user.Email, "name", user.Name, "token", user.VerificationToken)

		case userdomain.EventEmailVerified:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			slog.Info("step 2/4: email confirmed — running credit check...",
				"user_id", user.ID)

		case userdomain.EventCreditScored:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			if user.CreditApproved {
				slog.Info("step 3/4: credit approved — complete your profile",
					"user_id", user.ID, "score", user.CreditScore)
			} else {
				slog.Info("step 3/4: credit denied — onboarding blocked",
					"user_id", user.ID, "score", user.CreditScore)
			}

		case userdomain.EventProfileCompleted:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			slog.Info("step 4/4: onboarding complete — welcome!",
				"user_id", user.ID, "name", user.Name, "bio", user.Bio)

		case userdomain.EventContactUpdated:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			slog.Info("contact updated",
				"user_id", user.ID, "name", user.Name, "email", user.Email)
		}
	}); err != nil {
		log.Fatal("rabbit subscribe:", err)
	}

	slog.Info("welcome service listening", "exchange", messaging.UserExchange)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("welcome service stopping")
}

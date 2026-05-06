package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"todoe/internal/event"
)

func Connect(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}

func ConnectWithRetry(url string, maxRetries int, delaySeconds int) (*amqp.Connection, *amqp.Channel, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		conn, ch, err := Connect(url)
		if err == nil {
			return conn, ch, nil
		}
		lastErr = err
		slog.Info("rabbit: connection failed, retrying", "attempt", i+1, "max", maxRetries, "err", err)
		time.Sleep(time.Duration(delaySeconds) * time.Second)
	}
	return nil, nil, lastErr
}

func DeclareExchange(ch *amqp.Channel, name string) error {
	return ch.ExchangeDeclare(name, "fanout", true, false, false, false, nil)
}

const (
	QueueAuditTaskEvents         = "audit.task.events"
	QueueAuditUserEvents         = "audit.user.events"
	QueueWelcomeUserEvents       = "welcome.user.events"
	QueueCreditUserEvents        = "credit.user.events"
	QueueAuthenUserEvents        = "authen.user.events"
	QueueOnboardingCreditResults = "onboarding.credit.results"
	QueueProjectionCaptchaEvents = "projection.captcha.events"
	QueueAuditCaptchaEvents      = "audit.captcha.events"
)

type Binding struct {
	Exchange string
	Queue    string
}

func DeclareTopology(ch *amqp.Channel, bindings []Binding) error {
	for _, b := range bindings {
		if err := DeclareExchange(ch, b.Exchange); err != nil {
			return err
		}
		if _, err := ch.QueueDeclare(b.Queue, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(b.Queue, "", b.Exchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}

type Publisher struct {
	ch       *amqp.Channel
	exchange string
}

func NewPublisher(ch *amqp.Channel, exchange string) *Publisher {
	return &Publisher{ch: ch, exchange: exchange}
}

func (p *Publisher) Publish(ctx context.Context, e event.Event) {
	payload, _ := json.Marshal(e.Payload)
	data, _ := json.Marshal(Message{Type: e.Type, Payload: payload})

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.ch.PublishWithContext(ctx, p.exchange, "", false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         data,
		})
		if err == nil {
			return
		}

		slog.Error("rabbit: publish error", "exchange", p.exchange, "attempt", attempt+1, "max", maxRetries, "err", err)

		if attempt < maxRetries-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	slog.Error("rabbit: publish failed after retries", "exchange", p.exchange, "event_type", e.Type)
}

var _ event.Publisher = (*Publisher)(nil)

func Subscribe(ch *amqp.Channel, exchange, queue string, handler func(Message)) error {
	if err := DeclareExchange(ch, exchange); err != nil {
		return err
	}
	q, err := ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	if err := ch.QueueBind(q.Name, "", exchange, false, nil); err != nil {
		return err
	}
	deliveries, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			var msg Message
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				slog.Error("rabbit: unmarshal", "queue", queue, "err", err)
				d.Nack(false, false)
				continue
			}
			handler(msg)
			d.Ack(false)
		}
	}()
	return nil
}

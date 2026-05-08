package messaging

import "encoding/json"

const (
	TaskExchange       = "task.events"
	UserExchange       = "user.events"
	AuthExchange       = "auth.events"
	CaptchaExchange    = "captcha.events"
	OnboardingExchange = "onboarding.events"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

package adapter

import (
	"context"
	"fmt"

	"todoe/domain/user/domain"
	"todoe/internal/event"
)

func NewProjectionHandler(repo *MySQLRepository) func(context.Context, event.Event) error {
	return func(ctx context.Context, e event.Event) error {
		// Skip profile updated events - users_view is updated synchronously in UpdateContact
		if e.Type == domain.EventProfileUpdated {
			return nil
		}

		// Default: expect full User object (for other events)
		user, ok := e.Payload.(domain.User)
		if !ok {
			return fmt.Errorf("unexpected payload type %T", e.Payload)
		}
		result := repo.Upsert(ctx, user)
		if result.IsError() {
			return result.Error()
		}
		return nil
	}
}

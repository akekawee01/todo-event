package adapter

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/samber/mo"

	"todoe/domain/user/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs all pending up-migrations against db. Idempotent — applied
// versions are tracked in the schema_migrations table by golang-migrate.
func Migrate(db *sqlx.DB) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}
	driver, err := migratemysql.WithInstance(db.DB, &migratemysql.Config{})
	if err != nil {
		return fmt.Errorf("migrations driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		return fmt.Errorf("migrations init: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations up: %w", err)
	}
	return nil
}

type MySQLRepository struct {
	db *sqlx.DB
}

func NewMySQLRepository(db *sqlx.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Append(ctx context.Context, aggregateID, eventType string, payload any) mo.Result[struct{}] {
	raw, err := json.Marshal(payload)
	if err != nil {
		return mo.Err[struct{}](err)
	}
	// MySQL JSON columns reject []byte (sent as CHARACTER SET binary by the
	// driver). Cast to string so it lands as utf8mb4.
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO users_events (id, aggregate_id, type, payload, created_at) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), aggregateID, eventType, string(raw), time.Now().UTC(),
	)
	if err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}

func (r *MySQLRepository) Upsert(ctx context.Context, u domain.User) mo.Result[struct{}] {
	_, err := r.db.NamedExecContext(ctx, `
INSERT INTO users_view
  (id, name, email, bio, status, verification_token, credit_score, credit_approved, created_at)
VALUES
  (:id, :name, :email, :bio, :status, :verification_token, :credit_score, :credit_approved, :created_at)
ON DUPLICATE KEY UPDATE
  name=VALUES(name),
  email=VALUES(email),
  bio=VALUES(bio),
  status=VALUES(status),
  verification_token=VALUES(verification_token),
  credit_score=VALUES(credit_score),
  credit_approved=VALUES(credit_approved)`, u)
	if err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}

func (r *MySQLRepository) UpdateContact(ctx context.Context, id, name, email, bio string) mo.Result[struct{}] {
	_, err := r.db.ExecContext(ctx, `
UPDATE users_view
SET name = ?, email = ?, bio = ?
WHERE id = ?`, name, email, bio, id)
	if err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}

func (r *MySQLRepository) FindByID(ctx context.Context, id string) mo.Result[domain.User] {
	var u domain.User
	err := r.db.GetContext(ctx, &u, `
SELECT id, name, email, bio, status, verification_token, credit_score, credit_approved, created_at
FROM users_view
WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return mo.Err[domain.User](err)
	}
	if err != nil {
		return mo.Err[domain.User](err)
	}
	return mo.Ok(u)
}

func (r *MySQLRepository) FindByEmail(ctx context.Context, email string) mo.Result[domain.User] {
	var u domain.User
	err := r.db.GetContext(ctx, &u, `
SELECT id, name, email, bio, status, verification_token, credit_score, credit_approved, created_at
FROM users_view
WHERE email = ?`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return mo.Err[domain.User](err)
	}
	if err != nil {
		return mo.Err[domain.User](err)
	}
	return mo.Ok(u)
}

func (r *MySQLRepository) FindActivated(ctx context.Context) mo.Result[[]domain.User] {
	users := []domain.User{}
	err := r.db.SelectContext(ctx, &users, `
SELECT id, name, email, bio, status, verification_token, credit_score, credit_approved, created_at
FROM users_view
WHERE status = ?
ORDER BY created_at DESC`, string(domain.StatusOnboardingComplete))
	if err != nil {
		return mo.Err[[]domain.User](err)
	}
	return mo.Ok(users)
}

func (r *MySQLRepository) FindEvents(ctx context.Context, aggregateID string) mo.Result[[]domain.UserEvent] {
	events := []domain.UserEvent{}
	err := r.db.SelectContext(ctx, &events, `
SELECT id, aggregate_id, type, payload, created_at
FROM users_events
WHERE aggregate_id = ?
ORDER BY created_at ASC, id ASC`, aggregateID)
	if err != nil {
		return mo.Err[[]domain.UserEvent](err)
	}
	return mo.Ok(events)
}

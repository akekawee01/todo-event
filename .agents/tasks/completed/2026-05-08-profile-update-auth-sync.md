# Task: Sync Profile Updates To Auth

Status: completed
Owner: Codex
Started: 2026-05-08
Last Updated: 2026-05-08

## Goal

When onboarding profile data changes, publish a RabbitMQ event and update API-side auth credentials so old email/session credentials stop working.

## Context

User requested profile/email/name updates to produce an event and have the task/API side consume the payload. Old credentials must not continue to work after email changes.

## Progress

- [x] Added `user.profile_updated` domain event alias for profile/contact updates.
- [x] Updated onboarding update flow to append and publish `user.profile_updated`.
- [x] Added auth repository methods to update credential email by `user_id` and deactivate active sessions by `user_id`.
- [x] Added auth application method `UpdateUserProfile`.
- [x] Updated API RabbitMQ consumer to handle both `user.activated` and `user.profile_updated`.
- [x] Ran Go formatting.
- [x] Ran `go test ./...`.

## Resume Notes

No remaining code action. Rebuild/restart `api` and `onboarding` after code changes if not already done.

## Verification

- `gofmt -w ...`
- `go test ./...`

## Notes

The updated email keeps the existing password hash but invalidates active sessions. The old email no longer matches the updated credential.

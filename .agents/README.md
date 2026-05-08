# Agent Workspace

Read this directory before changing code.

## Load Order

1. Read `.agents/project.md` for architecture, services, routes, and ports.
2. Read `.agents/tasks/README.md` for the task-state protocol.
3. Check `.agents/tasks/active/` for resumable work.
4. Check `git status --short` and treat existing changes as user-owned unless the active task file says otherwise.
5. Use `.agents/skills.md` to decide which installed skills are relevant.

## Operating Rules

- Keep one task file per meaningful unit of work.
- Update the task file before risky changes, after verification, and before stopping.
- Move completed task files from `.agents/tasks/active/` to `.agents/tasks/completed/`.
- Move blocked task files to `.agents/tasks/blocked/` only when local recovery is not reasonable.
- For frontend work, verify with a production build and, when UI behavior matters, browser checks.
- For backend work, run `go test ./...` and relevant Docker Compose checks.

# Task State Protocol

Task files make work resumable after interruption or failure.

## Directory Meaning

- `active/`: work currently in progress or ready to resume.
- `completed/`: finished work with verification notes.
- `blocked/`: work that cannot continue without outside input or broken infrastructure.

## Task File Format

Use `.agents/templates/task.md` for new work. Keep updates concise and factual.

Required fields:
- `Status`: `active`, `blocked`, or `completed`.
- `Owner`: current agent or `unassigned`.
- `Started`: date.
- `Last Updated`: date/time if possible.
- `Goal`: one sentence.
- `Progress`: checklist.
- `Resume Notes`: exact next step.
- `Verification`: commands run and result.

Before ending a turn, update the task file. If work is complete, move it to `completed/`.

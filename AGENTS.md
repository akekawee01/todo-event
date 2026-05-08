# Agent Guide

Before doing any work in this repository, load `.agents/README.md` first.

That file points to the project map, active task state, resume workflow, and repo-specific verification commands. If the current task is interrupted or fails, update the active task file under `.agents/tasks/active/` before stopping so the next agent can resume without rediscovery.

Keep edits scoped to the requested task. Do not overwrite user changes. Prefer the existing domain/event patterns already used in the repo.

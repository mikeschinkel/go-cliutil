# Lessons Learned

A capture log of corrections, newest last. Entries are appended by
`endless lesson write`, which commits this file on the main checkout as it
writes — so recording a lesson never waits on a task landing, and never
dirties a worktree.

## Format

```
### [YYYY-MM-DD] One-line summary
<the lesson — what went wrong, why, and the rule that replaces it>
- **Project**: <project>
```

---

## Entries

### [2026-10-17] Write each task for a reader who knows only its own project: no user names, no driving app, no other projects' task IDs
While filing follow-up tasks across repos for E-2289, I wrote contexts such as 'This repo is a dependency of Endless. Mike's goal (during go-cliutil E-2289) is...'. A reader of the dependency repo doesn't know what Endless is, who Mike is, or what E-2289 means. Mike pointed out that a dependency's tasks must not be driven by, or explained in terms of, the app that wanted them. Rule: describe an upstream dependency well enough to stand alone (e.g. 'go-cliutil is changing RegisterCommand to return nothing'), express cross-project relations only as structured links (blocked-by, relates-to), and never name the user or the session that drove the work.
- **Project**: go-cliutil

### [2026-10-17] Record corrections with endless lesson write, never in Claude Code memory files
I saved two corrections as Claude Code memory files in Claude Code's per-project memory directory, following the global CLAUDE.md. Mike wants no memory writes: memory makes Endless behave better for him than for a product user on first install, and it hides the pain he needs to feel to fix the product. Corrections go to endless lesson write; facts about work go into Endless tasks or decisions.
- **Project**: go-cliutil

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

### [2026-10-27] Explain what findings mean before listing details
During E-2374 (go-cliutil), I reported main's working tree drifting from the wip snapshot and staged .endless gitlinks as raw facts: diffs, file modes, index entries. My options were phrased in terms Mike hadn't seen. He said it 'gave me a lot of minute details but did not explain to me what it actually all meant, and your options all presented details I do not understand.' When surfacing a finding, first say in plain words what it means for him and whether anything is at risk, then give options in those same plain terms; keep low-level evidence out unless he asks for it.
- **Project**: go-cliutil

### [2026-10-27] Fetch remote tags before claiming a version is unreleased
During E-2374 (go-cliutil), I said go-cliutil's latest tag was v0.2.1 and that v0.3.0 did not exist, from `git tag` in a local clone that hadn't fetched tags. v0.3.0 had been released on GitHub on 2025-12-05. That wrong fact shaped the release analysis and the 'ready to tag v0.3.0' handoff; Mike caught it from the GitHub releases page. Before stating which versions exist or proposing a next version, read the remote's tags (`git ls-remote --tags origin`, or fetch tags first).
- **Project**: go-cliutil

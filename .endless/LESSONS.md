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

### [2026-10-29] Search accepted decisions across all projects before raising a conflict as open
While planning go-doterr's E-2383, Mike said his Q1-Q3 answers apply to all of go-pkgs. I raised as an open conflict that dropping replace directives contradicts go-pkgs/CLAUDE.md, and asked him to settle it. He had already settled it in at least three sessions, and it was recorded as accepted decision ED-1639 in the gomion project. I also proposed putting the rules in go-pkgs/CLAUDE.md as if it were my idea, when that was what he had asked for. Rule: before presenting a conflict or open question, run endless decision list --all and search for it; a decision filed in a sibling project (gomion for Go module policy) still governs. When the user asks whether to add something somewhere, the answer is to do it, not to recommend it back.
- **Project**: go-cliutil

### [2026-10-29] Read the user's request as written, and don't cite a plan as a rule binding the user
Planning go-doterr's E-2383, Mike asked 'Can we not duplicate the effort and have E-2383 also update release.yml?', meaning duplicated effort was acceptable. I read it as him wanting to avoid duplication, argued against the edit, and cited E-2465's plan item forbidding hand-edits to generated workflows as if it bound him. A plan is guidance for the agent working it, not a rule for the user; the user can change it at any time. When a request's phrasing is ambiguous, take the reading that does what was asked, or ask, rather than attributing a goal the user didn't state.
- **Project**: go-cliutil

### [2026-10-31] Link a Go package's tasks with blockers so their changes ship in one tagged release
When filing or reviewing tasks for a Go package project, add blockers whenever leaving them out would make the user publish and tag the package more than once in a short span. Several versions in quick succession tell users the project is disorganized, so changes meant for one version should wait on each other and ship together. Changes that never reach a tagged module, such as edits confined to an unversioned examples/ or test/ module, need no such grouping.
- **Project**: go-cliutil

### [2026-10-31] Propose a task and wait for approval before filing it
File a task only when the user asked for that task or approved it. When something seems worth a task, describe the proposed task and ask; the user may want a different fix (for an init/register help gap they wanted the duplicate command dropped, not documented).
- **Project**: go-cliutil

### [2026-10-31] Report only actionable items and questions the user must answer
In replies, give only what the user must act on and the questions only they can answer. Leave out background, explanations of what was found or done, and offers of further summaries; the user will ask when something is unclear. Said while preparing the go-pkgs dependencies for release, after long replies made the process far slower than doing it by hand.
- **Project**: go-cliutil

### [2026-10-31] Report a blocker instead of working around it
While doing E-2469 (adopt gomion's generated workflows in go-cliutil), gomion's main checkout would not build: gommod imports go-tealeaves/teautils/teacolor, but the go-tealeaves checkout has teacolor as its own module and teautils v0.1.0 requires an unpublished teacolor v0.0.0. I started building gomion through a scratch go.work with an extra replace. Mike stopped me: a blockage is a blocker, and his #1 focus right now is cleaning up all the blockers, so do not work around one. When a task's prerequisite is broken, stop, report what is broken and why, and let him decide, rather than inventing a local workaround that hides the blocker.
- **Project**: go-cliutil

### [2026-10-31] Add missing task relations without asking; do not pose a question the facts already answer
In go-cliutil E-2469 I reported missing blocked-by links across twelve tasks and asked Mike whether to add them, and I presented moving E-2511 ahead of E-2510 as a decision even though I had already found that go-testutil's go.mod requires only published versions, which answers it. Mike: a missing blocked-by link is trivial and reversible, so add it and report that you did; asking first on matters that small wastes his attention. And when the facts you gathered settle a question, state the conclusion and act on it rather than handing the question back to him.
- **Project**: go-cliutil

### [2026-11-01] Release handoff: notify Mike, propose version and description; never release
While reviewing go-cliutil E-2430 and the release-prep tasks (E-2510 and siblings), Mike said a session must not do a release itself. When work is ready to tag, notify him that it is ready and give him the proposed version plus a release description he can paste, written in active voice with backticks around code identifiers. Mike tags through the Release workflow.
- **Project**: go-cliutil

### [2026-11-01] Go 1.27 is settled everywhere; do not raise it again
During go-cliutil E-2469 I mentioned that gomion's local go.work still said go 1.25.7 and asked about raising it. Mike: everything bumps to Go 1.27, go.work files included, and he does not want to be asked about that again. Treat the go-pkgs baseline's Go 1.27 rule as applying to every Go file that carries a go line, raise it as part of whatever task touches it, and just report that you did.
- **Project**: go-cliutil

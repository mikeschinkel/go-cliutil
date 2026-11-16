# Open questions: what a go-cliutil handler receives as input

Questions for Mike to answer here. Findings first, so each question can be read on its own.

## Terms used below
- **Raw vector**: the command line after the program name, as go-cliutil receives it (`repo workflows --check .` for `gomion repo workflows --check .`).
- **Command words**: the leading words that select the command (`repo workflows`).
- **Parameters**: everything after the command words, flags and positionals together (`--check .`).
- **Positionals**: the parameters that are neither a flag nor a flag's value (`.`).
- **Target**: the pointer a FlagDef or ArgDef writes its value into (`FlagDef.Bool`, `ArgDef.String`).
- **Variadic tail**: a last positional that collects every remaining positional into a list.

## Findings (go-cliutil main, 2026-11-15)
1. `CmdRunnerArgs.Args`, the one slice a handler sees, is the raw vector minus global flags: command words and flags included, no doc comment. gomion's `repo setup`/`repo workflows` read it as their positionals and treated `repo`, `workflows` and `--check` as directories. gomion's `help` command reads it on purpose (`c.Args[0] == "help"`).
2. `AssignArgs` copies positional *i* into `ArgDefs[i].String` and discards the slice. A positional beyond the declared ArgDefs is silently dropped; no handler can see a variadic tail.
3. `ArgDef.Default` (type `any`) is never applied; `FlagDef.Default` is, in `FlagSet.Build`. gomion declares `Default: "."` on four live ArgDefs (commit, repo setup/workflows, verify) that have never taken effect.
4. **A positional right after a boolean flag is lost.** `classifyFlagArgs` takes the word after any known flag as its value; for a bool flag Go's `flag` stops at it, and the leftover is discarded. Measured with `FlagSet.Parse`: `["--check" "."]` gives positionals `[]`; `["." "--check"]` gives `["."]`. So `gomion repo workflows --check .` cannot reach `.` through the ArgDef either.
5. Flags are found anywhere among the parameters, but `findBestCmdMatch` stops at the first word starting with `-`, so a flag placed before or between command words (`gomion repo --check workflows`) prevents the command from being found.
6. Targets are package-level variables in consumers (gomion's `repoWorkflowsOpts`, `planOpts`), and `FlagSet.Build` resets a target only when its FlagDef has a Default. Two command lines run in one process (a test) share them, and the second run sees the first run's value for a flag it did not pass. Commands register into one package-level registry from `init()`.
7. Consumers of go-cliutil: gomion, endless, doctool, gmover, macmail-ext, gearboxworks homelab, go-cfgstore, go-logutil, go-testutil, go-cli tools. ArgDefs appear in gomion (12 files), doctool (11), macmail-ext (5), gmover (3), homelab (2); raw `Args` reads in gomion (4), homelab (3), gmover (1), endless (1).

## Q1. What does a handler receive?
| Option | Pros | Cons |
|---|---|---|
| A. Three fields with narrow names: the raw vector, the parameters, and the positionals | Each consumer reads exactly what it needs; help keeps the raw vector; a passthrough command can read the parameters | Three overlapping slices; a reader must learn which one to use |
| B. The command words and the positionals only; the raw vector is not exposed | Nothing exposes flags as if they were arguments, so finding 1 cannot recur | homelab's and gmover's raw reads must be rewritten; a command that forwards its parameters to another program has no way to get them |
| C. No slices: every input arrives through a typed target (FlagDefs, ArgDefs, a variadic ArgDef); the command words are given to help separately | One way to receive input, the same for flags and positionals; usage and validation always know every input | No passthrough of undeclared parameters at all; every raw reader must be rewritten |

Recommendation: A, with positionals also reaching targets (Q2): the typed path stays primary, and the slices cover help and passthrough.

## Q2. How is a variadic tail declared?
| Option | Pros | Cons |
|---|---|---|
| A. `ArgDef` gains a `Strings *[]string` target, allowed only on the last ArgDef; `Required` means at least one | Declared, so usage and help show it and go-cliutil validates it | Registration must reject a variadic ArgDef that is not last |
| B. No new target; a handler slices the positionals itself | No change to ArgDef | Usage and help cannot show it; each handler repeats the slicing |

Recommendation: A.

## Q3. What happens to a positional beyond the declared ArgDefs?
| Option | Pros | Cons |
|---|---|---|
| A. Always an error, shown with the command's usage | A typo or extra word never runs silently on the first value only | A command that relied on extras fails until it declares a variadic ArgDef |
| B. An error unless the command opts in (for example `CmdArgs.AllowExtraArgs`) | Forwarding commands keep working without declaring a variadic | Two ways to accept many positionals (opt-in and variadic) |
| C. Dropped silently, as today | No consumer changes | `gomion repo workflows a b` runs on `a` and ignores `b` |

Recommendation: A, since Q2's variadic covers the legitimate case.

## Q4. How does a positional get its default?
| Option | Pros | Cons |
|---|---|---|
| A. Apply `ArgDef.Default` when the positional is absent, as `FlagDef.Default` is; registration rejects a Default whose type does not match the target | gomion's four `"."` defaults start working; usage can show the default; flags and positionals behave alike | `Default any` is checked only at registration, not at compile time |
| B. Typed default fields per target (`DefaultString`, `DefaultStrings`) | The compiler checks the default's type | One more field per target type; differs from FlagDef's `Default any` |
| C. Remove `ArgDef.Default`; handlers default empty values themselves | No hidden behaviour | Usage cannot show the default; each handler repeats it (what gomion does today) |

Recommendation: A.

## Q5. Where may a command's flags appear?
| Option | Pros | Cons |
|---|---|---|
| A. Anywhere after the command words; a flag before a command word belongs to the command named so far (global flags before any word) | Matches Python's click, which Mike cited; command lookup never has to guess whether a word is a flag value | `gomion repo --check workflows` is an error (`--check` is not a `repo` flag) |
| B. Anywhere, including between command words | `gomion repo --check workflows` works | Command lookup must know every command's flag types before it knows the command, or a string flag's value can be mistaken for a command word |
| C. Only before the positionals (Go's `flag` rule) | No ambiguity | `gomion repo workflows . --check` fails |

Recommendation: A.

## Q6. Where do targets live, and is the registry global?
| Option | Pros | Cons |
|---|---|---|
| A. As today: package-level targets, one global registry filled from `init()`; go-cliutil resets every target before each parse | Consumers' command files change least | Tests that run command lines cannot run in parallel; every target must have a known zero to reset to |
| B. Each run builds fresh command values from a constructor the command registers; targets are fields of the command value | No state crosses runs; parallel in-process tests are safe | Every command in every consumer changes how it registers and where its options live |

Recommendation: B. Finding 6 means an in-process test today can pass or fail depending on which test ran first.

## Q7. Where does a test helper for running command lines live?
| Option | Pros | Cons |
|---|---|---|
| A. An exported package in go-cliutil's root module (for example `cliutil/cliutiltest`) that runs a command line in-process and returns output and exit code | One import for every consumer's test module; versioned with the code it drives | Ships in every consumer's module graph, though only test modules import it |
| B. A separately versioned nested module (`cliutiltest/go.mod`) | Consumers that never test commands never download it | One more module to tag in step with go-cliutil |
| C. No helper; consumers call `ParseCmd` and `RunCmd` | Nothing to maintain in go-cliutil | Each consumer writes the same harness, and gets finding 6 wrong in its own way |

Recommendation: A.

## Q8. How do consumers move to the new input design?
| Option | Pros | Cons |
|---|---|---|
| A. One go-cliutil implementation task, then one task per consumer project, each blocked by it | Each project changes in its own repo and review; projects not in active work can wait | Consumers on the old version cannot take go-cliutil fixes until they move |
| B. The go-cliutil task also updates every consumer | One coordinated change | One task spans eight repositories and lands in eight places |

Recommendation: A.

## Decided without asking (Mike can object)
- A boolean flag never takes the following word as its value (finding 4 is a bug, whatever Q5 decides); `--flag=false` stays.
- `--` ends flags: every word after it is a positional, as POSIX and Go's `flag` do.
- The chosen fields and targets get doc comments saying exactly which words they hold, since finding 1 came from an undocumented field.

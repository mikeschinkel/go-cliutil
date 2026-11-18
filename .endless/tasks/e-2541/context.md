Mike stated these starting requirements on 2026-11-18 while planning the go-cliutil handler rebuild. They are input to the discussion, not settled decisions.

- go-cliutil includes a fully fleshed-out default help, driven by the information registered about each command.
- An app can replace help entirely with its own custom help.
- Help renders from one or more text/template files embedded in the Go app and written to disk, so a user can learn from them and customize them.
- Helper methods carry complex expressions, so templates hold only simple loops and branches.
- Help supports named themes. A user can choose a preferred theme and write their own.
- Themes might later extend to how other output is displayed, but that is outside help's scope.

Today each app writes its own help command, and go-cliutil supplies only ShowMainHelp, ShowCmdHelp and two embedded templates. ROADMAP item 3 (flag-specific help functions) is also in scope.

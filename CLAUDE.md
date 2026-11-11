# Working on Pyltjies vision

Rules for any person or AI agent changing this repo. Read PLAN.md first. This is a public, open source repo under AGPL-3.0: see CONTRIBUTING.md.

- **Never commit or push straight to `main`.** Every change: `git switch -c <type>/<name>` (`feat/`, `fix/`, `docs/`, `chore/`), commits, push, `gh pr create`.
- **Tests run before the PR** and pass in CI: `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass. Tests never call Gemini or any other outside service.
- **Every reader is judged by `internal/bench`** on the same labelled photos. A new reader or prompt change says in its PR how it scored.
- **Darts are written one way:** `S20`, `D20`, `T20`, `25`, `50`, `M`, as in `internal/darts`. It matches the Pyltjies pad.
- **Tip positions are kept**, as fractions of the photo from the top left. They are the training data.
- **Docs move with the code:** README.md, PLAN.md and PHOTOS.md are updated in the same PR as the change they describe.
- **PR description:** what changed, why, how it was tested, plus PR comments for anything a reviewer should look at.
- **Public repo:** never commit keys, `.env`, photos, home network details, addresses, host names or personal email addresses. `photos/` is git-ignored except the example label file.
- **Licences:** don't copy code or weights from Dart Sense (non-commercial, non-distributable) or DeepDarts (no licence). Anything added must be compatible with AGPL-3.0.
- **Merging and deploying need the maintainer's go-ahead.** Never merge your own PR.
- **Style:** standard library unless a dependency clearly earns its place. Plain words in messages and docs, sentence case.

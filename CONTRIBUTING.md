# Contributing

Thanks for helping. This is a free, non-commercial project, and contributions are welcome: code, prompt tweaks, test results on other boards, and ideas.

- Fork, make a branch, and open a pull request against `main`. The maintainer reviews it, and nothing merges without their approval. CI on a pull request from a fork waits until the maintainer has looked at it and pressed approve, so don't worry if your checks sit waiting.
- Run `gofmt -l .`, `go vet ./...` and `go test ./...` first. CI runs them too.
- If you change the prompt or add a reader, say in the PR how it did on your photos with `go run ./cmd/eval`, and what board and light you used.
- Don't commit photos, keys or `.env`. If you'd like to share photos for training, open an issue first so we can agree how they're licensed.
- By contributing, you agree your work is released under the [AGPL-3.0](LICENSE), like the rest of the repo.

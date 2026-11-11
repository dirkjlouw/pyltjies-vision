# Pyltjies vision

Camera scoring for [Pyltjies](https://github.com/dirkjlouw/pyltjies), the darts app for tossers. Take a photo of the board, get the darts back, fix any it got wrong.

It's a free, non-commercial community project. Forks and pull requests are welcome. Only the maintainer merges.

## Why a separate repo

The model we'll train uses [Ultralytics YOLO](https://www.ultralytics.com/license), which is AGPL-3.0, and that covers trained models too. So this repo is AGPL-3.0 and the Pyltjies app stays MIT. The app only talks to this service over HTTP.

## How it gets there

The full plan is in [PLAN.md](PLAN.md). In short:

| Phase | What happens | Status |
|---|---|---|
| 0. Photo test | About 30 labelled photos of a real, worn board go through Gemini, and we count how many darts it reads right. | Ready to run |
| 1. Photo, suggest, fix | In a game, tap the camera after three darts. The pad fills in, and you drag any wrong dart to where its tip is. | Planned |
| 2. Our own model | Train YOLO on the corrected photos in a free cloud notebook, export to ONNX, run it here in Go. | Planned |
| 3. Camera on a stand | A fixed phone or webcam spots each new dart. About 60% right is the target, with a tap to fix. | Planned |

## Phase 0: the photo test

You need Go 1.25 or newer and a Gemini API key from [Google AI Studio](https://aistudio.google.com/apikey).

1. Copy `.env.example` to `.env` and put your key in it. `.env` is git-ignored.
2. Take photos as [PHOTOS.md](PHOTOS.md) describes and put them in `photos/`, with a `photos/labels.csv` saying which darts are in each.
3. Run it:

   ```bash
   go run ./cmd/eval
   ```

It prints each photo, what Gemini read, and a summary such as `Darts read right: 61 of 90 (68%)`. Every reading, with the tip positions Gemini gave, is saved in `photos/results/`. Try another model with `-model`, or a quick look at the first five with `-n 5`.

On Google's free tier, Google may use what you send to improve its products. These are photos of a dartboard, but don't send anything you'd mind them seeing.

## The service

`cmd/vision` is the service the Pyltjies server will call.

```bash
go run ./cmd/vision
```

- `POST /v1/read` takes a JPEG, PNG, WebP or HEIC photo, as the raw body or a form field called `photo`, up to 12 MB. It returns the darts, with each tip's position as a fraction of the photo from the top left:

  ```json
  {"reader": "gemini-3.6-flash", "darts": [{"score": "T20", "x": 0.48, "y": 0.21, "sure": 0.9}], "notes": ""}
  ```

- `GET /healthz` says `ok`.
- Without `VISION_TOKEN` it listens on this machine only. With it, callers must send `Authorization: Bearer <token>`.

Scores are written `S20`, `D20`, `T20`, `25`, `50`, or `M` for a dart outside the double ring, the same darts the Pyltjies pad uses.

## Working on it

```bash
go test ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md). The tests never call Gemini, so they need no key.

## Licence

Copyright (C) 2026 Dirk J Louw. Released under the [GNU Affero General Public License v3.0](LICENSE). If you run a changed version as a service for others, you must offer them its source.

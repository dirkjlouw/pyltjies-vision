# Plan

Decided by the maintainer on 2026-11-11, unless marked as our own starting point.

## Goal

Score darts from a phone or webcam well enough that fixing the odd wrong dart is quicker than typing them all. About 60% of darts right is good enough when a fix is one tap. The test board is an old, worn bristle board, which is about as hard as it gets.

## Pieces

- **Reader to start with:** Gemini, through its REST API. The model is set by `GEMINI_MODEL`.
- **Service:** Go, standard library only for now. Later it runs our own model with ONNX Runtime ([onnxruntime_go](https://pkg.go.dev/github.com/yalue/onnxruntime_go)).
- **Training:** Python with Ultralytics YOLO, in a free cloud notebook (Google Colab or Kaggle).
- **The app:** Pyltjies stays Node and MIT, and calls this service over HTTP.

## Phase 0: photo test

About 30 photos of the real board, labelled by hand, see [PHOTOS.md](PHOTOS.md). `cmd/eval` reads them with Gemini and reports darts read right and photos fully right. This tells us early how much the worn board costs before anything is built into the app.

Prompt changes are judged on these photos, and every reader from now on is scored with the same `internal/bench` code.

## Phase 1: photo, suggest, fix

- After three darts, the player taps the camera button. The phone's normal camera picker takes the photo, which avoids the camera bugs in iPhone home screen apps.
- The Pyltjies server sends it to this service. The darts appear on the pad and as markers on the photo.
- A wrong dart is fixed by dragging its marker to where the tip really is. The score follows the marker. A score alone, like `D16`, can't train a model. The tip's position on the photo can.
- Once per camera position, the player taps four points on the board: where the 5/20, 13/6, 17/3 and 8/11 wires meet the outside of the double ring (DeepDarts' calibration points). That's what lets a position on the photo become a score.
- Photos and their corrections are kept for training, only with the player's say-so. Whether they're published as a dataset is a later decision.

## Phase 2: our own model

- Once there are a few hundred corrected photos, train a small YOLO pose or detection model to find the four calibration points and every dart tip, as DeepDarts did.
- Export it to ONNX and run it in `cmd/vision` on the server, with no Gemini call.
- It replaces Gemini only when it beats it on the phase 0 photos plus a held-back set of new ones.
- For scale: DeepDarts trained on about 16,000 photos and Dart Sense on about 8,000, so expect it to get better as games are played. Our own starting estimate.
- Memory on the server is tight. Measure the model's use before shipping it.

## Phase 3: camera on a stand

A phone or webcam is fixed in line of sight. The service compares frames to spot a new dart, reads it, and the app suggests it for the player to confirm. Target: about 60% right.

## Not reused, and why

- **Dart Sense** (bnww/dart-sense) is the best ready model, but its licence (CC BY-NC, non-distributable) allows personal experiments only. It can be tried privately, never shipped here.
- **DeepDarts** (wmcnally/deep-darts) has no code licence, so its code isn't copied. Its paper's method is fair to learn from.

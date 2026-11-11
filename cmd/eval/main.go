// Command eval is the photo test: it runs every labelled photo through Gemini
// and reports how many darts it read right.
//
//	go run ./cmd/eval -photos photos
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/dirkjlouw/pyltjies-vision/internal/bench"
	"github.com/dirkjlouw/pyltjies-vision/internal/env"
	"github.com/dirkjlouw/pyltjies-vision/internal/gemini"
)

func main() {
	log.SetFlags(0)
	dir := flag.String("photos", "photos", "folder with the photos and labels.csv")
	labels := flag.String("labels", "", "label file (default <photos>/labels.csv)")
	out := flag.String("out", "", "where to save every reading as JSON lines (default <photos>/results/<model>-<time>.jsonl)")
	model := flag.String("model", "", "Gemini model (default GEMINI_MODEL, else "+gemini.DefaultModel+")")
	pause := flag.Duration("pause", 4*time.Second, "wait between photos, to stay inside the free tier's rate limit")
	limit := flag.Int("n", 0, "only the first n photos (0 for all)")
	flag.Parse()

	if err := env.Load(".env"); err != nil {
		log.Fatal(err)
	}
	if *model == "" {
		*model = os.Getenv("GEMINI_MODEL")
	}
	if *labels == "" {
		*labels = filepath.Join(*dir, "labels.csv")
	}
	photos, err := bench.LoadLabels(*labels)
	if err != nil {
		log.Fatal(err)
	}
	if *limit > 0 && *limit < len(photos) {
		photos = photos[:*limit]
	}
	client := gemini.New(os.Getenv("GEMINI_API_KEY"), *model)
	if client.Key == "" {
		log.Fatal("no Gemini key: put GEMINI_API_KEY=... in .env (it's git-ignored)")
	}
	if *out == "" {
		*out = filepath.Join(*dir, "results", fmt.Sprintf("%s-%s.jsonl", client.Model, time.Now().Format("20060102-150405")))
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	saved := json.NewEncoder(file)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fmt.Printf("Reading %d photos with %s\n\n", len(photos), client.Model)
	var results []bench.Result
	for i, photo := range photos {
		if i > 0 && *pause > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(*pause):
			}
		}
		if ctx.Err() != nil {
			fmt.Println("\nStopped early.")
			break
		}
		result := bench.Run(ctx, client, *dir, photo, gemini.MimeTypes)
		results = append(results, result)
		saved.Encode(result)
		fmt.Println(bench.Line(result))
	}
	fmt.Printf("\n%s\nEvery reading, with tip positions, is in %s\n", bench.Summarize(results), *out)
}

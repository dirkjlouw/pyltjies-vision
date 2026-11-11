// Command vision is the service the Pyltjies server calls with a board photo.
// Today it asks Gemini; later it runs our own model with ONNX Runtime.
//
// Without VISION_TOKEN it listens on this machine only, so nobody else can
// spend the Gemini quota.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dirkjlouw/pyltjies-vision/internal/env"
	"github.com/dirkjlouw/pyltjies-vision/internal/gemini"
	"github.com/dirkjlouw/pyltjies-vision/internal/server"
)

func main() {
	if err := env.Load(".env"); err != nil {
		log.Fatal(err)
	}
	token := os.Getenv("VISION_TOKEN")
	defaultAddr := "127.0.0.1:8090"
	if token != "" {
		defaultAddr = ":8090"
	}
	addr := flag.String("addr", defaultAddr, "listen address")
	flag.Parse()
	if token == "" && *addr != defaultAddr {
		log.Fatal("set VISION_TOKEN before listening anywhere but this machine")
	}

	client := gemini.New(os.Getenv("GEMINI_API_KEY"), os.Getenv("GEMINI_MODEL"))
	if client.Key == "" {
		log.Fatal("no Gemini key: set GEMINI_API_KEY")
	}
	types := map[string]bool{}
	for _, kind := range gemini.MimeTypes {
		types[kind] = true
	}
	service := &server.Service{Reader: client, Name: client.Model, Token: token, Types: types}
	srv := &http.Server{
		Addr:              *addr,
		Handler:           service.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       time.Minute,
		WriteTimeout:      3 * time.Minute,
	}
	log.Printf("pyltjies-vision on %s, reading with %s", *addr, client.Model)
	log.Fatal(srv.ListenAndServe())
}

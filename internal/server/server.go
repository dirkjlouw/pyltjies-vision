// Package server is the vision service the Pyltjies server calls: send a
// photo, get back the darts and where their tips are.
package server

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"

	"github.com/dirkjlouw/pyltjies-vision/internal/bench"
)

// MaxPhoto is the largest photo accepted. A phone photo is 2 to 5 MB.
const MaxPhoto = 12 << 20

// Service holds what the routes need.
type Service struct {
	Reader bench.Reader
	// Name says which reader answered, such as the Gemini model.
	Name string
	// Token, when set, must come as "Authorization: Bearer <token>".
	Token string
	// Types are the accepted photo types.
	Types map[string]bool
}

// Handler gives the routes.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "ok\n") })
	mux.HandleFunc("POST /v1/read", s.read)
	return mux
}

func (s *Service) read(w http.ResponseWriter, r *http.Request) {
	if s.Token != "" {
		given, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(given), []byte(s.Token)) != 1 {
			fail(w, http.StatusUnauthorized, "missing or wrong token")
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxPhoto+1<<20)
	photo, kind, err := s.photo(r)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			fail(w, http.StatusRequestEntityTooLarge, "photo is too big")
			return
		}
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	reading, err := s.Reader.Read(r.Context(), photo, kind)
	if err != nil {
		log.Printf("read failed: %v", err)
		fail(w, http.StatusBadGateway, "the reader couldn't read this photo")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"reader": s.Name, "darts": reading.Darts, "notes": reading.Notes})
}

// photo takes either a raw image body or a multipart form with a "photo" field.
func (s *Service) photo(r *http.Request) ([]byte, string, error) {
	kind, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	var body io.Reader = r.Body
	if kind == "multipart/form-data" {
		file, header, err := r.FormFile("photo")
		if err != nil {
			return nil, "", errors.New("send the photo in a field called photo")
		}
		defer file.Close()
		kind, _, _ = mime.ParseMediaType(header.Header.Get("Content-Type"))
		body = file
	}
	if !s.Types[kind] {
		return nil, "", errors.New("send a JPEG, PNG, WebP or HEIC photo")
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(body, MaxPhoto+1)); err != nil {
		return nil, "", err
	}
	if buf.Len() > MaxPhoto {
		return nil, "", &http.MaxBytesError{Limit: MaxPhoto}
	}
	if buf.Len() == 0 {
		return nil, "", errors.New("the photo is empty")
	}
	return buf.Bytes(), kind, nil
}

func fail(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

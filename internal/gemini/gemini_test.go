package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func answer(text string) string {
	out, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{
		map[string]any{"text": "thinking about wires", "thought": true},
		map[string]any{"text": text},
	}}}}})
	return string(out)
}

func fake(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := New("test-key", "")
	c.BaseURL = server.URL
	c.Wait = time.Millisecond
	return c
}

func TestReadSendsThePhotoAndParsesDarts(t *testing.T) {
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/"+DefaultModel+":generateContent" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "test-key" || r.URL.RawQuery != "" {
			t.Error("the key goes in the header only")
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"mime_type":"image/jpeg"`) || !strings.Contains(string(body), `"data":"cGhvdG8="`) {
			t.Errorf("photo missing from %s", body)
		}
		io.WriteString(w, answer(`{"darts":[{"score":"T20","x":0.4,"y":0.2,"sure":0.9},{"score":"bull","x":0.5,"y":1.4},{"score":"wire","x":0,"y":0}],"notes":"glare"}`))
	})
	reading, err := c.Read(context.Background(), []byte("photo"), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if len(reading.Darts) != 2 || reading.Darts[0].Dart.String() != "T20" || reading.Darts[1].Dart.String() != "50" {
		t.Fatalf("got %+v", reading)
	}
	if reading.Darts[1].Y != 1 {
		t.Error("positions are kept inside the photo")
	}
	if !strings.Contains(reading.Notes, "glare") || !strings.Contains(reading.Notes, `"wire"`) {
		t.Errorf("notes %q", reading.Notes)
	}
}

func TestReadRetriesWhenRateLimited(t *testing.T) {
	calls := 0
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			io.WriteString(w, `{"error":{"message":"slow down"}}`)
			return
		}
		io.WriteString(w, answer(`{"darts":[]}`))
	})
	reading, err := c.Read(context.Background(), []byte("x"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if reading.Darts == nil {
		t.Error("no darts is an empty list, not null, so the app can loop over it")
	}
	if calls != 3 {
		t.Fatalf("calls %d", calls)
	}
}

func TestReadGivesUpOnABadKey(t *testing.T) {
	calls := 0
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":{"message":"API key not valid"}}`)
	})
	_, err := c.Read(context.Background(), []byte("x"), "image/png")
	if err == nil || !strings.Contains(err.Error(), "API key not valid") || calls != 1 {
		t.Fatalf("err %v after %d calls", err, calls)
	}
}

func TestReadNeedsAKey(t *testing.T) {
	if _, err := New("", "").Read(context.Background(), nil, "image/png"); err == nil {
		t.Fatal("expected an error")
	}
}

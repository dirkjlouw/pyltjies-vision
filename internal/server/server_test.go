package server

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/dirkjlouw/pyltjies-vision/internal/darts"
)

type reader struct{ got string }

func (r *reader) Read(_ context.Context, photo []byte, kind string) (darts.Reading, error) {
	r.got = kind + ":" + string(photo)
	if string(photo) == "broken" {
		return darts.Reading{}, errors.New("secret upstream detail")
	}
	return darts.Reading{Darts: []darts.Seen{{Dart: darts.Dart{N: 20, M: 3}, X: 0.5, Y: 0.1}}}, nil
}

func service() (*Service, *reader) {
	r := &reader{}
	return &Service{Reader: r, Name: "fake", Token: "t0ken", Types: map[string]bool{"image/jpeg": true}}, r
}

func send(s *Service, body []byte, kind, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/read", bytes.NewReader(body))
	req.Header.Set("Content-Type", kind)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestReadARawPhoto(t *testing.T) {
	s, r := service()
	rec := send(s, []byte("jpeg"), "image/jpeg", "t0ken")
	if rec.Code != 200 || r.got != "image/jpeg:jpeg" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if want := `{"darts":[{"score":"T20","x":0.5,"y":0.1}],"notes":"","reader":"fake"}`; strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("body %s", rec.Body)
	}
}

func TestReadAFormUpload(t *testing.T) {
	s, r := service()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="photo"; filename="board.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	part.Write([]byte("from a form"))
	form.Close()
	rec := send(s, body.Bytes(), form.FormDataContentType(), "t0ken")
	if rec.Code != 200 || r.got != "image/jpeg:from a form" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestRefusals(t *testing.T) {
	s, _ := service()
	cases := []struct {
		name, kind, token string
		body              []byte
		code              int
	}{
		{"no token", "image/jpeg", "", []byte("x"), 401},
		{"wrong token", "image/jpeg", "nope", []byte("x"), 401},
		{"not a photo", "text/plain", "t0ken", []byte("x"), 400},
		{"empty", "image/jpeg", "t0ken", nil, 400},
		{"too big", "image/jpeg", "t0ken", make([]byte, MaxPhoto+1), 413},
		{"reader fails", "image/jpeg", "t0ken", []byte("broken"), 502},
	}
	for _, c := range cases {
		rec := send(s, c.body, c.kind, c.token)
		if rec.Code != c.code {
			t.Errorf("%s: %d %s", c.name, rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "secret") {
			t.Errorf("%s: upstream errors stay in the log", c.name)
		}
	}
}

func TestHealth(t *testing.T) {
	s, _ := service()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
}

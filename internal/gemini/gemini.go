// Package gemini reads darts off a board photo with Google's Gemini API.
// It's the starting reader: players correct what it gets wrong, and those
// corrections become training data for a model of our own.
package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dirkjlouw/pyltjies-vision/internal/darts"
)

// DefaultModel is a stable Flash model. Set GEMINI_MODEL to try another.
const DefaultModel = "gemini-3.6-flash"

// DefaultBaseURL is the Gemini API. Tests point it at a fake server.
const DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// Client calls Gemini. The key goes in a header, never in the URL, so it
// can't end up in a log line.
type Client struct {
	Key     string
	Model   string
	BaseURL string
	HTTP    *http.Client
	// Retries is how many times a rate-limited or busy call is tried again.
	Retries int
	// Wait is how long to wait before a retry when Gemini doesn't say.
	Wait time.Duration
}

// New makes a client with sensible defaults.
func New(key, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{Key: key, Model: model, BaseURL: DefaultBaseURL, HTTP: &http.Client{Timeout: 2 * time.Minute}, Retries: 3, Wait: 10 * time.Second}
}

// Prompt explains the board to the model. Kept plain so it's easy to tune
// against the photo test.
const Prompt = `This photo shows a steel-tip dartboard with darts in it. The board may be old and worn, with faded numbers, frayed bristle and bent wires.

List every dart stuck in the board. For each dart, find where its TIP enters the board, not where the flight or barrel points. Work out the segment from the wires around the tip.

The numbers run clockwise from the top: 20, 1, 18, 4, 13, 6, 10, 15, 2, 17, 3, 19, 7, 16, 8, 11, 14, 9, 12, 5. The board may be turned or photographed at an angle, so use the printed numbers you can read to orient it.

Write each score as S, D or T followed by the number (S20 single, D20 the thin outer ring, T20 the thin inner ring), 25 for the outer bull, 50 for the inner bull, or M for a dart in the board outside the double ring.

Give the tip's position as x and y between 0 and 1, measured from the top left of the photo. Give sure between 0 and 1 for how certain you are of each score. Use notes for anything that made this photo hard.`

var schema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"darts": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"score": map[string]any{"type": "STRING", "description": "S20, D20, T20, 25, 50 or M"},
					"x":     map[string]any{"type": "NUMBER"},
					"y":     map[string]any{"type": "NUMBER"},
					"sure":  map[string]any{"type": "NUMBER"},
				},
				"required": []string{"score", "x", "y"},
			},
		},
		"notes": map[string]any{"type": "STRING"},
	},
	"required": []string{"darts"},
}

// MimeTypes are the photo types Gemini accepts, by file extension.
var MimeTypes = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
	".webp": "image/webp", ".heic": "image/heic", ".heif": "image/heif",
}

// Read sends one photo and returns the darts Gemini found.
func (c *Client) Read(ctx context.Context, photo []byte, mime string) (darts.Reading, error) {
	if c.Key == "" {
		return darts.Reading{}, errors.New("no Gemini key: set GEMINI_API_KEY")
	}
	body, err := json.Marshal(map[string]any{
		"contents": []any{map[string]any{"parts": []any{
			map[string]any{"text": Prompt},
			map[string]any{"inline_data": map[string]any{"mime_type": mime, "data": base64.StdEncoding.EncodeToString(photo)}},
		}}},
		"generationConfig": map[string]any{"responseMimeType": "application/json", "responseSchema": schema},
	})
	if err != nil {
		return darts.Reading{}, err
	}
	url := strings.TrimRight(c.BaseURL, "/") + "/models/" + c.Model + ":generateContent"
	for attempt := 0; ; attempt++ {
		text, wait, err := c.call(ctx, url, body)
		if err == nil {
			return parse(text)
		}
		if wait == 0 || attempt >= c.Retries {
			return darts.Reading{}, err
		}
		select {
		case <-ctx.Done():
			return darts.Reading{}, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// call makes one request. A non-zero wait means it's worth trying again.
func (c *Client) call(ctx context.Context, url string, body []byte) (string, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.Key)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", 0, err
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)
	if resp.StatusCode != http.StatusOK {
		message := out.Error.Message
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		err := fmt.Errorf("gemini %d: %s", resp.StatusCode, message)
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return "", c.retryAfter(resp.Header.Get("Retry-After")), err
		}
		return "", 0, err
	}
	if len(out.Candidates) == 0 {
		return "", 0, errors.New("gemini gave no answer")
	}
	var text strings.Builder
	for _, part := range out.Candidates[0].Content.Parts {
		if !part.Thought {
			text.WriteString(part.Text)
		}
	}
	if text.Len() == 0 {
		return "", 0, fmt.Errorf("gemini gave an empty answer (%s)", out.Candidates[0].FinishReason)
	}
	return text.String(), 0, nil
}

func (c *Client) retryAfter(header string) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if c.Wait > 0 {
		return c.Wait
	}
	return time.Second
}

// parse turns Gemini's JSON into a reading. A score it can't read is left out
// and mentioned in the notes rather than failing the whole photo.
func parse(text string) (darts.Reading, error) {
	var raw struct {
		Darts []struct {
			Score string  `json:"score"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
			Sure  float64 `json:"sure"`
		} `json:"darts"`
		Notes string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return darts.Reading{}, fmt.Errorf("gemini answer isn't JSON: %w", err)
	}
	reading := darts.Reading{Darts: []darts.Seen{}, Notes: raw.Notes}
	for _, seen := range raw.Darts {
		d, err := darts.Parse(seen.Score)
		if err != nil {
			reading.Notes = strings.TrimSpace(reading.Notes + " Unreadable score " + strconv.Quote(seen.Score) + ".")
			continue
		}
		reading.Darts = append(reading.Darts, darts.Seen{Dart: d, X: clamp(seen.X), Y: clamp(seen.Y), Sure: clamp(seen.Sure)})
	}
	return reading, nil
}

func clamp(v float64) float64 { return min(max(v, 0), 1) }

package mediamtx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPlaybackClient_List(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/list"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("path"), "camera/front"; got != want {
			t.Fatalf("query path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("start"), start.Format(time.RFC3339Nano); got != want {
			t.Fatalf("query start = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("end"), end.Format(time.RFC3339Nano); got != want {
			t.Fatalf("query end = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("jwt"), "token"; got != want {
			t.Fatalf("query jwt = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`[
			{
				"start": "2026-06-09T12:00:00Z",
				"duration": 12.5,
				"url": "http://localhost:9996/get?path=camera%2Ffront"
			}
		]`))
		if err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewPlaybackClient(server.URL + "?jwt=token")
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}

	items, err := client.List(ctx, PlaybackListParams{
		Path:  "camera/front",
		Start: start,
		End:   end,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(items) = %d, want %d", got, want)
	}
	if got, want := items[0].Start, start; !got.Equal(want) {
		t.Fatalf("items[0].Start = %v, want %v", got, want)
	}
	if got, want := items[0].Duration, 12500*time.Millisecond; got != want {
		t.Fatalf("items[0].Duration = %v, want %v", got, want)
	}
	if got, want := items[0].URL, "http://localhost:9996/get?path=camera%2Ffront"; got != want {
		t.Fatalf("items[0].URL = %q, want %q", got, want)
	}
}

func TestPlaybackClient_Get(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/get"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("path"), "camera/front"; got != want {
			t.Fatalf("query path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("start"), start.Format(time.RFC3339Nano); got != want {
			t.Fatalf("query start = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("duration"), "200.5"; got != want {
			t.Fatalf("query duration = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("format"), "mp4"; got != want {
			t.Fatalf("query format = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer playback-token"; got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Length", "11")
		_, err := w.Write([]byte("video-data!"))
		if err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewPlaybackClient(server.URL, WithClient(&http.Client{
		Transport: bearerTransport{
			token: "playback-token",
			next:  http.DefaultTransport,
		},
	}))
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}

	resp, err := client.Get(ctx, PlaybackGetParams{
		Path:     "camera/front",
		Start:    start,
		Duration: 200*time.Second + 500*time.Millisecond,
		Format:   PlaybackFormatMP4,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close body: %v", err)
		}
	}()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	if got, want := resp.ContentType, "video/mp4"; got != want {
		t.Fatalf("ContentType = %q, want %q", got, want)
	}
	if got, want := resp.ContentLength, int64(11); got != want {
		t.Fatalf("ContentLength = %d, want %d", got, want)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got, want := string(body), "video-data!"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestPlaybackClient_GetDefaultFormat(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "" {
			t.Fatalf("query format = %q, want unset", got)
		}
		_, err := w.Write([]byte("data"))
		if err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewPlaybackClient(server.URL)
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}

	resp, err := client.Get(ctx, PlaybackGetParams{
		Path:     "camera/front",
		Start:    start,
		Duration: time.Second,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
}

func TestPlaybackClient_Error(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(`{
			"status": "error",
			"error": "no recording segments found"
		}`))
		if err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewPlaybackClient(server.URL)
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}

	_, err = client.List(ctx, PlaybackListParams{
		Path: "camera/front",
	})

	var playbackErr *PlaybackError
	if !errors.As(err, &playbackErr) {
		t.Fatalf("List() error = %T, want *PlaybackError", err)
	}
	if got, want := playbackErr.StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	if got, want := playbackErr.ErrorString(), "no recording segments found"; got != want {
		t.Fatalf("ErrorString() = %q, want %q", got, want)
	}
}

func TestPlaybackClient_EmptyError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewPlaybackClient(server.URL)
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}

	_, err = client.List(ctx, PlaybackListParams{
		Path: "camera/front",
	})

	var playbackErr *PlaybackError
	if !errors.As(err, &playbackErr) {
		t.Fatalf("List() error = %T, want *PlaybackError", err)
	}
	if got, want := playbackErr.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	if got := playbackErr.ErrorString(); got != "" {
		t.Fatalf("ErrorString() = %q, want empty", got)
	}
}

func TestPlaybackClient_GetValidation(t *testing.T) {
	client, err := NewPlaybackClient("http://localhost:9996")
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}

	_, err = client.Get(context.Background(), PlaybackGetParams{
		Path:     "camera/front",
		Start:    time.Now(),
		Duration: time.Second,
		Format:   PlaybackFormat("mov"),
	})
	if err == nil {
		t.Fatal("Get() error = nil, want validation error")
	}
}

type bearerTransport struct {
	token string
	next  http.RoundTripper
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.next.RoundTrip(req)
}

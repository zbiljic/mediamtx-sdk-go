package mediamtx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PlaybackClient invokes operations served by the MediaMTX playback server.
type PlaybackClient struct {
	serverURL *url.URL
	baseClient
}

// NewPlaybackClient initializes a client for the MediaMTX playback server.
//
// The playback server usually listens on a different address than the control
// API, for example http://localhost:9996.
func NewPlaybackClient(serverURL string, opts ...ClientOption) (*PlaybackClient, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	trimTrailingSlashes(u)

	c, err := newClientConfig(opts...).baseClient()
	if err != nil {
		return nil, err
	}
	return &PlaybackClient{
		serverURL:  u,
		baseClient: c,
	}, nil
}

// PlaybackFormat is a recording download container format.
type PlaybackFormat string

const (
	// PlaybackFormatFMP4 is fragmented MP4. It is the MediaMTX default.
	PlaybackFormatFMP4 PlaybackFormat = "fmp4"
	// PlaybackFormatMP4 is standard MP4.
	PlaybackFormatMP4 PlaybackFormat = "mp4"
)

// PlaybackListParams is parameters of the playback list operation.
type PlaybackListParams struct {
	// Path is the recorded path name.
	Path string
	// Start filters segments by starting time. Zero means unset.
	Start time.Time
	// End filters segments by ending time. Zero means unset.
	End time.Time
}

// PlaybackListItem is a recording segment returned by the playback server.
type PlaybackListItem struct {
	Start    time.Time
	Duration time.Duration
	URL      string
}

// UnmarshalJSON decodes MediaMTX playback durations from seconds.
func (i *PlaybackListItem) UnmarshalJSON(data []byte) error {
	var wire struct {
		Start    time.Time `json:"start"`
		Duration float64   `json:"duration"`
		URL      string    `json:"url"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	i.Start = wire.Start
	i.Duration = secondsToDuration(wire.Duration)
	i.URL = wire.URL
	return nil
}

// PlaybackGetParams is parameters of the playback get operation.
type PlaybackGetParams struct {
	// Path is the recorded path name.
	Path string
	// Start is the recording start time.
	Start time.Time
	// Duration is the maximum duration of the recording.
	Duration time.Duration
	// Format is the output format. Zero means the MediaMTX default fMP4 format.
	Format PlaybackFormat
}

// PlaybackGetResponse is a streaming response returned by the playback server.
//
// Callers must close Body.
type PlaybackGetResponse struct {
	Body          io.ReadCloser
	Header        http.Header
	ContentType   string
	ContentLength int64
	StatusCode    int
}

// PlaybackError is returned when the playback server responds with an error status.
type PlaybackError struct {
	StatusCode int
	Response   Error
}

func (e *PlaybackError) ErrorString() string {
	if e.Response.GetError().IsSet() {
		return e.Response.GetError().Value
	}
	return ""
}

func (e *PlaybackError) Error() string {
	if msg := e.ErrorString(); msg != "" {
		return fmt.Sprintf("playback server returned HTTP %d: %s", e.StatusCode, msg)
	}
	return fmt.Sprintf("playback server returned HTTP %d", e.StatusCode)
}

// List lists recording segments from the MediaMTX playback server.
func (c *PlaybackClient) List(
	ctx context.Context,
	params PlaybackListParams,
) ([]PlaybackListItem, error) {
	if params.Path == "" {
		return nil, fmt.Errorf("playback list path is required")
	}

	u := c.operationURL("/list")
	query := u.Query()
	query.Set("path", params.Path)
	if !params.Start.IsZero() {
		query.Set("start", params.Start.Format(time.RFC3339Nano))
	}
	if !params.End.IsZero() {
		query.Set("end", params.End.Format(time.RFC3339Nano))
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create playback list request: %w", err)
	}

	resp, err := c.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err := decodePlaybackError(resp); err != nil {
		return nil, err
	}

	var result []PlaybackListItem
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode playback list response: %w", err)
	}

	return result, nil
}

// Get downloads a recording segment from the MediaMTX playback server.
func (c *PlaybackClient) Get(
	ctx context.Context,
	params PlaybackGetParams,
) (*PlaybackGetResponse, error) {
	if params.Path == "" {
		return nil, fmt.Errorf("playback get path is required")
	}
	if params.Start.IsZero() {
		return nil, fmt.Errorf("playback get start is required")
	}
	if params.Duration <= 0 {
		return nil, fmt.Errorf("playback get duration must be positive")
	}
	if params.Format != "" && params.Format != PlaybackFormatFMP4 && params.Format != PlaybackFormatMP4 {
		return nil, fmt.Errorf("unsupported playback format %q", params.Format)
	}

	u := c.operationURL("/get")
	query := u.Query()
	query.Set("path", params.Path)
	query.Set("start", params.Start.Format(time.RFC3339Nano))
	query.Set("duration", durationSeconds(params.Duration))
	if params.Format != "" {
		query.Set("format", string(params.Format))
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create playback get request: %w", err)
	}

	resp, err := c.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if err := decodePlaybackError(resp); err != nil {
		_ = resp.Body.Close()
		return nil, err
	}

	return &PlaybackGetResponse{
		Body:          resp.Body,
		Header:        resp.Header,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
		StatusCode:    resp.StatusCode,
	}, nil
}

func (c *PlaybackClient) operationURL(path string) *url.URL {
	u := *c.serverURL
	u.Path = path
	u.RawPath = ""
	return &u
}

func decodePlaybackError(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	playbackErr := &PlaybackError{StatusCode: resp.StatusCode}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read playback error response: %w", err)
	}
	if len(body) == 0 {
		return playbackErr
	}
	if err := json.Unmarshal(body, &playbackErr.Response); err != nil {
		return fmt.Errorf("decode playback error response: %w", err)
	}
	return playbackErr
}

func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

func durationSeconds(duration time.Duration) string {
	return strconv.FormatFloat(duration.Seconds(), 'f', -1, 64)
}

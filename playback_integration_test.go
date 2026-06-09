package mediamtx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPlaybackClientIntegration(t *testing.T) {
	if os.Getenv("MEDIAMTX_INTEGRATION") != "1" {
		t.Skip("set MEDIAMTX_INTEGRATION=1 to run")
	}

	dockerImage := os.Getenv("MEDIAMTX_DOCKER_IMAGE")
	sourceDir := os.Getenv("MEDIAMTX_SOURCE_DIR")
	disableDockerMoQ := false
	if dockerImage == "" && sourceDir == "" {
		dockerImage = defaultMediaMTXDockerImage(t)
		disableDockerMoQ = true
	}
	rtspPort := freeTCPPort(t)
	playbackPort := freeTCPPort(t)
	pathName := "sdkplayback"
	recordDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "mediamtx.yml")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var serverLog bytes.Buffer
	publish := func(t *testing.T, ctx context.Context) {
		t.Helper()
		publishLocalRecording(t, ctx, rtspPort, pathName)
	}
	if dockerImage != "" {
		writeIntegrationConfig(t, configPath, integrationConfig{
			RTSPAddress:     ":8554",
			PlaybackAddress: ":9996",
			RecordPath:      "/recordings/%path/%Y-%m-%d_%H-%M-%S-%f",
			PathName:        pathName,
			DisableMoQ:      disableDockerMoQ,
		})
		containerName := startDockerMediaMTX(
			t,
			ctx,
			dockerImage,
			configPath,
			recordDir,
			rtspPort,
			playbackPort,
			&serverLog,
		)
		publish = func(t *testing.T, ctx context.Context) {
			t.Helper()
			publishDockerRecording(t, ctx, containerName, pathName)
		}
	} else {
		if sourceDir == "" {
			t.Skip("set MEDIAMTX_DOCKER_IMAGE or MEDIAMTX_SOURCE_DIR")
		}
		if _, err := os.Stat(filepath.Join(sourceDir, "main.go")); err != nil {
			t.Fatalf("MEDIAMTX_SOURCE_DIR does not look like a MediaMTX checkout: %v", err)
		}

		recordPath := filepath.Join(recordDir, "%path/%Y-%m-%d_%H-%M-%S-%f")
		writeIntegrationConfig(t, configPath, integrationConfig{
			RTSPAddress:     fmt.Sprintf("127.0.0.1:%d", rtspPort),
			PlaybackAddress: fmt.Sprintf("127.0.0.1:%d", playbackPort),
			RecordPath:      recordPath,
			PathName:        pathName,
			DisableMoQ:      true,
		})
		startSourceMediaMTX(t, ctx, sourceDir, configPath, &serverLog)
	}

	playbackBaseURL := fmt.Sprintf("http://127.0.0.1:%d", playbackPort)
	waitForHTTPServer(t, ctx, playbackBaseURL+"/list?path="+pathName, &serverLog)
	publish(t, ctx)

	basicClient, err := NewPlaybackClient(playbackBaseURL, WithClient(&http.Client{
		Transport: basicAuthTransport{
			user: "playback",
			pass: "secret",
			next: http.DefaultTransport,
		},
	}))
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}

	items := waitForPlaybackItems(t, ctx, basicClient, pathName, &serverLog)
	first := items[0]

	_, err = basicClient.List(ctx, PlaybackListParams{
		Path:  pathName,
		Start: first.Start.Add(24 * time.Hour),
	})
	var notFoundErr *PlaybackError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("not found List() error = %T, want *PlaybackError", err)
	}
	if got, want := notFoundErr.StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("not found status = %d, want %d", got, want)
	}

	unauthorizedClient, err := NewPlaybackClient(playbackBaseURL)
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}
	_, err = unauthorizedClient.List(ctx, PlaybackListParams{Path: pathName})
	var playbackErr *PlaybackError
	if !errors.As(err, &playbackErr) {
		t.Fatalf("unauthorized List() error = %T, want *PlaybackError", err)
	}
	if got, want := playbackErr.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("unauthorized status = %d, want %d", got, want)
	}

	bearerClient, err := NewPlaybackClient(playbackBaseURL, WithClient(&http.Client{
		Transport: bearerTransport{
			token: "playback:secret",
			next:  http.DefaultTransport,
		},
	}))
	if err != nil {
		t.Fatalf("NewPlaybackClient() error = %v", err)
	}
	if _, err := bearerClient.List(ctx, PlaybackListParams{Path: pathName}); err != nil {
		t.Fatalf("bearer List() error = %v", err)
	}

	for _, format := range []PlaybackFormat{PlaybackFormatFMP4, PlaybackFormatMP4} {
		resp, err := basicClient.Get(ctx, PlaybackGetParams{
			Path:     pathName,
			Start:    first.Start,
			Duration: first.Duration,
			Format:   format,
		})
		if err != nil {
			t.Fatalf("Get(%s) error = %v", format, err)
		}

		body, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if err != nil {
			t.Fatalf("ReadAll(%s) error = %v", format, err)
		}
		if closeErr != nil {
			t.Fatalf("Close(%s) error = %v", format, closeErr)
		}
		if len(body) == 0 {
			t.Fatalf("Get(%s) returned an empty body", format)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Get(%s) status = %d, want %d", format, resp.StatusCode, http.StatusOK)
		}
	}
}

func defaultMediaMTXDockerImage(t *testing.T) string {
	t.Helper()

	version := mediaMTXVersionFromMiseConfig(t)
	if version == "" {
		version = os.Getenv("MEDIAMTX_VERSION")
	}
	if version == "" {
		t.Skip("set MEDIAMTX_DOCKER_IMAGE or MEDIAMTX_SOURCE_DIR")
	}

	version = strings.TrimPrefix(version, "v")
	return "bluenviron/mediamtx:" + version + "-ffmpeg"
}

func mediaMTXVersionFromMiseConfig(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(".config", "mise", "conf.d", "mediamtx.toml"))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "MEDIAMTX_VERSION" {
			continue
		}

		return strings.Trim(strings.TrimSpace(value), `"`)
	}

	return ""
}

type integrationConfig struct {
	RTSPAddress     string
	PlaybackAddress string
	RecordPath      string
	PathName        string
	DisableMoQ      bool
}

func writeIntegrationConfig(t *testing.T, path string, cfg integrationConfig) {
	t.Helper()

	data := fmt.Sprintf(
		`logLevel: info
logDestinations: [stdout]
readTimeout: 10s
writeTimeout: 10s
authMethod: internal
authInternalUsers:
- user: any
  pass:
  ips: []
  permissions:
  - action: publish
- user: playback
  pass: secret
  ips: []
  permissions:
  - action: playback
    path:
api: false
metrics: false
pprof: false
playback: true
playbackAddress: %s
rtsp: true
rtspTransports: [tcp]
rtspEncryption: "no"
rtspAddress: %s
rtmp: false
hls: false
webrtc: false
srt: false
%s
paths:
  %s:
    source: publisher
    record: true
    recordPath: %s
    recordFormat: fmp4
    recordPartDuration: 200ms
    recordSegmentDuration: 1s
    recordDeleteAfter: 1h
`,
		strconv.Quote(cfg.PlaybackAddress),
		strconv.Quote(cfg.RTSPAddress),
		moqConfigLine(cfg.DisableMoQ),
		strconv.Quote(cfg.PathName),
		strconv.Quote(cfg.RecordPath),
	)

	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write MediaMTX config: %v", err)
	}
}

func moqConfigLine(disable bool) string {
	if !disable {
		return ""
	}
	return "moq: false"
}

func startSourceMediaMTX(
	t *testing.T,
	ctx context.Context,
	sourceDir string,
	configPath string,
	serverLog *bytes.Buffer,
) {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "mediamtx")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, ".")
	buildCmd.Dir = sourceDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build MediaMTX: %v\n%s", err, string(output))
	}

	cmd := exec.CommandContext(ctx, binPath, configPath)
	cmd.Dir = sourceDir
	cmd.Stdout = serverLog
	cmd.Stderr = serverLog
	if err := cmd.Start(); err != nil {
		t.Fatalf("start MediaMTX: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil && cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
}

func startDockerMediaMTX(
	t *testing.T,
	ctx context.Context,
	image string,
	configPath string,
	recordDir string,
	rtspPort int,
	playbackPort int,
	serverLog *bytes.Buffer,
) string {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is required for MEDIAMTX_DOCKER_IMAGE integration test")
	}

	name := fmt.Sprintf("mediamtx-sdk-go-%d", time.Now().UnixNano())
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"run",
		"--rm",
		"--name", name,
		"-v", configPath+":/mediamtx.yml:ro",
		"-v", recordDir+":/recordings",
		"-p", fmt.Sprintf("127.0.0.1:%d:8554", rtspPort),
		"-p", fmt.Sprintf("127.0.0.1:%d:9996", playbackPort),
		image,
		"/mediamtx.yml",
	)
	cmd.Stdout = serverLog
	cmd.Stderr = serverLog
	if err := cmd.Start(); err != nil {
		t.Fatalf("start MediaMTX Docker container: %v", err)
	}
	t.Cleanup(func() {
		rmCmd := exec.Command("docker", "rm", "-f", name)
		_ = rmCmd.Run()
		_ = cmd.Wait()
	})

	return name
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on a free TCP port: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	return listener.Addr().(*net.TCPAddr).Port
}

func waitForHTTPServer(t *testing.T, ctx context.Context, url string, serverLog *bytes.Buffer) {
	t.Helper()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("create readiness request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("wait for MediaMTX playback server: %v\n%s", ctx.Err(), serverLog.String())
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func publishLocalRecording(t *testing.T, ctx context.Context, rtspPort int, pathName string) {
	t.Helper()

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for MEDIAMTX_SOURCE_DIR integration test")
	}

	url := fmt.Sprintf("rtsp://127.0.0.1:%d/%s", rtspPort, pathName)
	publishRecording(t, ctx, "ffmpeg", []string{
		"-hide_banner",
		"-loglevel", "error",
		"-f", "lavfi",
		"-i", "testsrc=size=128x72:rate=5",
		"-t", "4",
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-g", "5",
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		url,
	})
}

func publishDockerRecording(t *testing.T, ctx context.Context, containerName, pathName string) {
	t.Helper()

	url := fmt.Sprintf("rtsp://127.0.0.1:8554/%s", pathName)
	publishRecording(t, ctx, "docker", []string{
		"exec",
		containerName,
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-re",
		"-f", "lavfi",
		"-i", "testsrc=size=128x72:rate=5",
		"-t", "6",
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-g", "5",
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		url,
	})
}

func publishRecording(t *testing.T, ctx context.Context, name string, args []string) {
	t.Helper()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("publish recording with ffmpeg: %v\n%s", err, string(output))
	}
}

func waitForPlaybackItems(
	t *testing.T,
	ctx context.Context,
	client *PlaybackClient,
	pathName string,
	serverLog *bytes.Buffer,
) []PlaybackListItem {
	t.Helper()

	for {
		items, err := client.List(ctx, PlaybackListParams{Path: pathName})
		if err == nil && len(items) > 0 {
			return items
		}

		select {
		case <-ctx.Done():
			t.Fatalf("wait for playback recordings: %v\n%s", ctx.Err(), serverLog.String())
		case <-time.After(500 * time.Millisecond):
		}
	}
}

type basicAuthTransport struct {
	user string
	pass string
	next http.RoundTripper
}

func (t basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.SetBasicAuth(t.user, t.pass)
	return t.next.RoundTrip(req)
}

# MediaMTX Go SDK

A comprehensive Go client library for the [MediaMTX](https://github.com/bluenviron/mediamtx) streaming server API, generated from the official OpenAPI specification.

## Features

- Complete MediaMTX API v3 coverage
- Type-safe Go client with full context support
- Monitor streams, sessions, and recordings
- Manage configurations and paths
- Built with [ogen](https://github.com/ogen-go/ogen) for optimal Go integration

## How to use

```bash
go get github.com/zbiljic/mediamtx-sdk-go
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zbiljic/mediamtx-sdk-go"
)

func main() {
    client, err := mediamtx.NewClient("http://localhost:9997")
    if err != nil {
        log.Fatal(err)
    }

    // Get server configuration
    config, err := client.ConfigGlobalGet(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    switch config := config.(type) {
    case *mediamtx.GlobalConf:
        fmt.Printf("MediaMTX API enabled: %t\n", config.API.Value)
    }
}
```

See [example/main.go](example/main.go) to get started. Documentation is available at [pkg.go.dev](https://pkg.go.dev/github.com/zbiljic/mediamtx-sdk-go).

### Playback server

MediaMTX exposes recording playback through a separate HTTP server, outside the
official control API OpenAPI specification. Use `NewPlaybackClient` with the
playback server address:

```go
playback, err := mediamtx.NewPlaybackClient("http://localhost:9996")
if err != nil {
    log.Fatal(err)
}

items, err := playback.List(context.Background(), mediamtx.PlaybackListParams{
    Path: "camera/front",
})
if err != nil {
    log.Fatal(err)
}

for _, item := range items {
    fmt.Println(item.Start, item.Duration, item.URL)
}
```

## API Coverage

The SDK provides complete access to MediaMTX API v3:

- **Configuration**: Global settings and path-specific configuration
- **Monitoring**: Active paths, sessions (RTSP, RTMP, WebRTC, SRT, HLS)
- **Recordings**: List, inspect, and manage recorded content
- **Authentication**: JWT JWKS refresh and session management
- **Playback**: List and download recordings from the playback server

## Building and Development

```bash
# Clone the repository
git clone https://github.com/zbiljic/mediamtx-sdk-go.git
cd mediamtx-sdk-go

# Generate code
mise run generate

# Run linter
mise run go:lint
```

## Requirements

- Go 1.25 or later
- MediaMTX server with API enabled

## Contributing

Contributions are welcome! If you find a bug, have a feature request, or want to improve the codebase, please feel free to open an issue or submit a pull request.

### Development

The project uses `mise` to manage development tools. To install them, run:

```sh
make bootstrap
```

Common development tasks are available through `mise`:

- `mise run go:mod:tidy`: Tidy Go modules.
- `mise run go:fmt`: Format Go code with `gofumpt`.
- `mise run go:lint`: Lint the source code.
- `mise run generate`: Regenerate SDK code.
- `mise run pre-commit`: Run formatters and linters.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

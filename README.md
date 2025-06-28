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

## API Coverage

The SDK provides complete access to MediaMTX API v3:

- **Configuration**: Global settings and path-specific configuration
- **Monitoring**: Active paths, sessions (RTSP, RTMP, WebRTC, SRT, HLS)
- **Recordings**: List, inspect, and manage recorded content
- **Authentication**: JWT JWKS refresh and session management

## Building and Development

```bash
# Clone the repository
git clone https://github.com/zbiljic/mediamtx-sdk-go.git
cd mediamtx-sdk-go

# Generate code
make generate

# Run linter
make lint
```

## Requirements

- Go 1.24 or later
- MediaMTX server with API enabled

## Contributing

Contributions are welcome! If you find a bug, have a feature request, or want to improve the codebase, please feel free to open an issue or submit a pull request.

### Development

The project uses `mise` to manage development tools. To install them, run:

```sh
make bootstrap
```

Common development tasks are available in the `Makefile`:

- `make tidy`: Tidy Go modules.
- `make gofmt`: Format Go code with `gofumpt`.
- `make lint`: Lint the source code.
- `make pre-commit`: Run formatters and linters.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

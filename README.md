# me-monitor

A terminal UI application for monitoring NATS JetStream consumers in real-time.

## Features

- Real-time monitoring of multiple NATS JetStream consumers
- **Multiple windows** with configurable layouts and consumers per window
- Visual flash notifications when consumer state changes
- Grid layout displaying consumer metrics
- Throughput measurement
- Reads NATS connection settings from NATS CLI context

## Requirements

- Go 1.21+
- NATS server with JetStream enabled
- NATS CLI configured with a context

## Configuration

### NATS Context

Set the `NATS_CONTEXT` environment variable to the name of your NATS CLI context:

```bash
export NATS_CONTEXT=my-context
```

The context file is expected at `~/.config/nats/context/<context-name>.json`.

### Consumers Configuration

Create a `consumers.json` file (or set `CONSUMERS_CONFIG` to a custom path).

#### Multi-Window Format (Recommended)

Configure multiple windows, each with its own layout and set of consumers:

```json
{
  "windows": [
    {
      "name": "Partitions 0-7",
      "columns": 4,
      "consumers": [
        { "stream": "my-stream", "consumer": "consumer-0" },
        { "stream": "my-stream", "consumer": "consumer-1" }
      ]
    },
    {
      "name": "Partitions 8-15",
      "columns": 4,
      "consumers": [
        { "stream": "my-stream", "consumer": "consumer-8" },
        { "stream": "my-stream", "consumer": "consumer-9" }
      ]
    }
  ]
}
```

#### Legacy Format

A simple flat list of consumers (displayed in a single window with 4 columns):

```json
{
  "consumers": [
    {
      "stream": "my-stream",
      "consumer": "my-consumer"
    }
  ]
}
```

## Building

```bash
devbox run go build -o me-monitor ./cmd/me-monitor
```

## Running

```bash
export NATS_CONTEXT=my-context
./me-monitor
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `t` | Toggle throughput measurement |
| `c` | Clear throughput results |
| `<` / `>` or Arrow keys | Switch between windows |
| `q` or `Ctrl-C` | Quit |
| Double-click | Copy cell content to clipboard |

## Project Structure

```
├── cmd/
│   └── me-monitor/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration loading (consumers + NATS context)
│   ├── monitor/
│   │   ├── poller.go        # NATS consumer polling logic
│   │   ├── snapshot.go      # Consumer state snapshot for change detection
│   │   └── throughput.go    # Throughput measurement
│   └── ui/
│       ├── app.go           # Terminal UI application
│       ├── colors.go        # Theme/color definitions
│       ├── flash.go         # Flash animation controller
│       ├── format.go        # Formatting utilities
│       └── selectable.go    # Selectable text view with copy support
├── consumers.json           # Consumer configuration
├── devbox.json              # Devbox configuration
└── go.mod                   # Go module definition
```

## Architecture

The application follows a clean separation of concerns:

1. **Config** (`internal/config`): Handles loading consumer configuration and NATS connection settings from environment and files.

2. **Monitor** (`internal/monitor`): Contains the polling logic that periodically fetches consumer information from NATS JetStream and detects state changes.

3. **UI** (`internal/ui`): Manages the terminal user interface using tview, including the flash animation system for highlighting changes.

The main goroutine flow:
- `Poller.Run()` polls NATS every 2 seconds and sends state updates through a channel
- `App.handleUpdates()` receives updates and refreshes the UI
- `FlashController` manages flash animations without race conditions

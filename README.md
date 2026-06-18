# A "go, please" server

A simple game server designed for turn-based games, communicating via WebSockets.

## Getting Started

To start the server, run:
```bash
go run cmd/server/main.go
```

## Configuration
There is nothing configurable right now.

The server will start on port `8090`. If you need to change it, simply update the `Port` constant in [`cmd/server/main.go`](cmd/server/main.go).

## Contributing
We don't have a formal set of rules for contributions yet; everyone is welcome! We appreciate everything from critiques and suggestions to bug fixes and new features. Feel free to open an Issue or submit a Pull Request.
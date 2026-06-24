# A "go, please" server

A simple game server designed for turn-based games, communicating via WebSockets.

## Getting Started

To start the server using the default configs, run:
```bash
go run ./cmd/server
```

## Configuration
By default the server runs on localhost and bind to to port `8080`. For configuring the port and server you can 
do this via passing args.
```bash
# to change the port the server binds to,  use --port. Default is 8080
go run ./cmd/server --port 1738

# to change rwtimeouts operations for sockets, use --timeout flag. Default is 10s
# if you wanted to change it to milliseconds, you can use the `ms` unit
go run ./cmd/server --timeout 10s

# to change the addr the server listens on, use --host flag. Default is "127.0.0.1"
go run ./cmd/server --host <some-ip-addr>
```


## Contributing
We don't have a formal set of rules for contributions yet; everyone is welcome! We appreciate everything from critiques and suggestions to bug fixes and new features. Feel free to open an Issue or submit a Pull Request.

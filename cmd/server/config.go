// Package main
package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	// Default timeout for both reads and writes to a socket.
	defaultRWTimeout = 10 * time.Second
	defaultHost      = "127.0.0.1"
	defaultPort      = "8080"
)

type Config struct {
	// socket timeout read and write operations.
	rwTimeout time.Duration

	// address of the machine.
	host string

	// port goplease server will bind to.
	port string
}

// defaultConfig provides a config with [defaultRWTimeout],
// [defaultHost] and [defaultPort].
func defaultConfig() *Config {
	return &Config{
		rwTimeout: defaultRWTimeout,
		host:      defaultHost,
		port:      defaultPort,
	}
}

// NewConfig constructs a new config with the arguments, provided they
// are correct. Otherwise incorrect fields are replaced with defaults.
func NewConfig(host, port string, rwTimeout time.Duration) *Config {
	cfg := defaultConfig()

	if isValidIP := net.ParseIP(host); isValidIP == nil {
		fmt.Println("invalid host provided, using default host: ", defaultHost)
	} else {
		cfg.host = host
	}

	// makes sure that port isn't a negative number
	// bitSize 16 makes sure max port number is 65535
	_, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		fmt.Println("invalid port provided, using default port: ", defaultPort)
	} else {
		cfg.port = port
	}

	cfg.rwTimeout = rwTimeout

	return cfg
}

func parseArgs() *Config {
	var timeout string
	var host string
	var port string

	flag.StringVar(&timeout, "timeout", defaultRWTimeout.String(),
		`read and write timeouts for socket operations. 
		for timeout in secs us <number>s
		milliseconds <number>ms`)

	flag.StringVar(&host, "host", defaultHost, "host for server to listen on")
	flag.StringVar(&port, "port", defaultPort, "port for server to bind to")

	flag.Parse()

	rwTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		fmt.Println("could not parse timeout arg, using default timeout. reason: ", err)
		rwTimeout = defaultRWTimeout
	}

	return NewConfig(host, port, rwTimeout)
}

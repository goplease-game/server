// Package config ...
package config

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	// DefaultRWTimeout timeout for both reads and writes to a socket.
	DefaultRWTimeout = 10 * time.Second

	// DefaultHost is the IP address the server will listen on.
	DefaultHost = "127.0.0.1"

	// DefaultPort is the port the sever binds to.
	DefaultPort = "8090"
)

// Config is a struct.
type Config struct {
	// Host is the address of the machine.
	Host string

	// Port goplease server will bind to.
	Port string

	// Read and write timeouts for socket operations.
	RWTimeout time.Duration
}

// Default provides a config with [DefaultRWTimeout],
// [DefaultHost] and [DefaultPort].
func Default() *Config {
	return &Config{
		RWTimeout: DefaultRWTimeout,
		Host:      DefaultHost,
		Port:      DefaultPort,
	}
}

// NewConfig constructs a new config with the arguments, provided they
// are correct. Otherwise incorrect fields are replaced with defaults.
func NewConfig(host, port string, rwTimeout time.Duration) *Config {
	cfg := Default()

	if isValidIP := net.ParseIP(host); isValidIP == nil {
		fmt.Println("invalid host provided, using default host: ", DefaultHost)
	} else {
		cfg.Host = host
	}

	// makes sure that port isn't a negative number
	// bitSize 16 makes sure max port number is 65535
	_, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		fmt.Println("invalid port provided, using default port: ", DefaultPort)
	} else {
		cfg.Port = port
	}

	cfg.RWTimeout = rwTimeout

	return cfg
}

// serverConfig is accessed at the package scope to other consumers
// It is nil until ParseCLIArgs is called.
var serverConfig *Config

// Get returns the current configuration for the server.
func Get() *Config {
	return serverConfig
}

// ParseCLIArgs parses commands via the cli from [os.Args]
// If no flags are provided the defaults are used. Invalid arguments are ignored
// and the defaults are used instead.
func ParseCLIArgs() *Config {
	var timeout string
	var host string
	var port string

	flag.StringVar(&timeout, "timeout", DefaultRWTimeout.String(),
		`read and write timeouts for socket operations. 
		for timeout in secs us <number>s
		milliseconds <number>ms`)

	flag.StringVar(&host, "host", DefaultHost, "host for server to listen on")
	flag.StringVar(&port, "port", DefaultPort, "port for server to bind to")

	flag.Parse()

	rwTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		fmt.Println("could not parse timeout arg, using default timeout. reason: ", err)
		rwTimeout = DefaultRWTimeout
	}

	cfg := NewConfig(host, port, rwTimeout)

	// copy newConfig into appConfig.
	serverConfig = cfg
	return cfg
}

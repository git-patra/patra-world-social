package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	Addr, DatabaseURL, AuthOrigin, AuthTrustedProxyCIDRs string
	AuthDevLoopback                                      bool
}

func Load() (Config, error) {
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return Config{}, fmt.Errorf("SERVER_ADDR must be host:port: %w", err)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("SERVER_ADDR port must be between 1 and 65535")
	}
	return Config{Addr: addr, DatabaseURL: os.Getenv("DATABASE_URL"), AuthOrigin: os.Getenv("AUTH_ORIGIN"), AuthTrustedProxyCIDRs: os.Getenv("AUTH_TRUSTED_PROXY_CIDRS"), AuthDevLoopback: os.Getenv("AUTH_DEV_LOOPBACK") == "1"}, nil
}

package config

import "testing"

func TestLoad(t *testing.T) {
	for _, tc := range []struct {
		env, want string
		invalid   bool
	}{
		{"", "127.0.0.1:8080", false},
		{"0.0.0.0:9090", "0.0.0.0:9090", false},
		{"[::1]:8080", "[::1]:8080", false},
		{"localhost", "", true}, {"localhost:0", "", true},
		{"localhost:65536", "", true}, {"localhost:abc", "", true},
	} {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("SERVER_ADDR", tc.env)
			got, err := Load()
			if (err != nil) != tc.invalid {
				t.Fatalf("error = %v", err)
			}
			if got.Addr != tc.want {
				t.Fatalf("addr = %q, want %q", got.Addr, tc.want)
			}
		})
	}
}

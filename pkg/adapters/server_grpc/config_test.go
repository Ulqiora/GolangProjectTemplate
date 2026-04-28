package server_grpc

import "testing"

func TestServerConfigDialAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  ServerConfig
		want string
	}{
		{name: "empty host", cfg: ServerConfig{Port: 50051}, want: "127.0.0.1:50051"},
		{name: "wildcard ipv4", cfg: ServerConfig{Host: "0.0.0.0", Port: 50051}, want: "127.0.0.1:50051"},
		{name: "explicit", cfg: ServerConfig{Host: "localhost", Port: 50051}, want: "localhost:50051"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.cfg.DialAddress(); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

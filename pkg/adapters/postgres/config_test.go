package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigConnectionStringUsesExplicitMaster(t *testing.T) {
	cfg := Config{
		Master: EndpointConfig{
			Host:     "master",
			Port:     "6432",
			User:     "writer",
			Password: "secret",
			Database: "app",
			SSLMode:  "require",
		},
	}

	require.Equal(t, "host=master port=6432 user=writer password=secret dbname=app sslmode=require", cfg.ConnectionString())
}

func TestConfigReadOnlyEndpointSupportsReadOnlyAndROAliases(t *testing.T) {
	readOnly := &EndpointConfig{Host: "read-only", Port: "5432", User: "reader", Password: "secret", Database: "app", SSLMode: "disable"}
	ro := &EndpointConfig{Host: "ro", Port: "5432", User: "reader", Password: "secret", Database: "app", SSLMode: "disable"}

	cfg := Config{
		Master:   EndpointConfig{Host: "master", Port: "5432", User: "writer", Password: "secret", Database: "app", SSLMode: "disable"},
		ReadOnly: readOnly,
		RO:       ro,
	}
	endpoint, configured := cfg.readOnlyEndpoint()
	require.True(t, configured)
	require.Equal(t, "read-only", endpoint.Host)

	cfg.ReadOnly = nil
	endpoint, configured = cfg.readOnlyEndpoint()
	require.True(t, configured)
	require.Equal(t, "ro", endpoint.Host)
}

func TestConfigReadOnlyEndpointFallsBackToMaster(t *testing.T) {
	cfg := Config{
		Master: EndpointConfig{Host: "master", Port: "5432", User: "writer", Password: "secret", Database: "app", SSLMode: "disable"},
	}

	endpoint, configured := cfg.readOnlyEndpoint()
	require.False(t, configured)
	require.Equal(t, "master", endpoint.Host)
}

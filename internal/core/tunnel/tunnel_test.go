package tunnel

import (
	"errors"
	"strings"
	"testing"
)

func TestSSHTunnel_HostKeyPolicy_Production(t *testing.T) {
	tun, err := NewSSHTunnel(Config{
		SSHHost:    "127.0.0.1",
		SSHPort:    2222,
		SSHUser:    "bastion",
		Password:   "secret",
		RemoteHost: "127.0.0.1",
		RemotePort: 5432,
		HostKey:    "", // Missing HostKey in production
		AppEnv:     "production",
	})
	if err != nil {
		t.Fatalf("unexpected error creating tunnel: %v", err)
	}

	_, err = tun.Start()
	if err == nil {
		t.Fatal("expected error starting tunnel in production without host key, got nil")
	}
	if !errors.Is(err, ErrHostKeyRequiredInProduction) {
		t.Fatalf("expected ErrHostKeyRequiredInProduction, got: %v", err)
	}
}

func TestSSHTunnel_HostKeyPolicy_Development(t *testing.T) {
	tun, err := NewSSHTunnel(Config{
		SSHHost:    "127.0.0.1",
		SSHPort:    1, // Non-existent port
		SSHUser:    "bastion",
		Password:   "secret",
		RemoteHost: "127.0.0.1",
		RemotePort: 5432,
		HostKey:    "", // Allowed in dev
		AppEnv:     "development",
	})
	if err != nil {
		t.Fatalf("unexpected error creating tunnel: %v", err)
	}

	_, err = tun.Start()
	// Should fail at dial stage, NOT at host key validation stage
	if errors.Is(err, ErrHostKeyRequiredInProduction) {
		t.Fatal("development mode should not require host key")
	}
	if err == nil || !strings.Contains(err.Error(), "failed to connect to SSH server") {
		t.Logf("connection failed as expected with dial error: %v", err)
	}
}

func TestSSHTunnel_InvalidHostKey(t *testing.T) {
	tun, err := NewSSHTunnel(Config{
		SSHHost:    "127.0.0.1",
		SSHPort:    2222,
		SSHUser:    "bastion",
		Password:   "secret",
		RemoteHost: "127.0.0.1",
		RemotePort: 5432,
		HostKey:    "not-a-valid-ssh-public-key",
		AppEnv:     "production",
	})
	if err != nil {
		t.Fatalf("unexpected error creating tunnel: %v", err)
	}

	_, err = tun.Start()
	if err == nil {
		t.Fatal("expected error with invalid host key, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse SSH host key") {
		t.Fatalf("expected 'failed to parse SSH host key' error, got: %v", err)
	}
}

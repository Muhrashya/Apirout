package egress

import (
	"errors"
	"testing"
)

func TestValidateSafeHost(t *testing.T) {
	tests := []struct {
		host    string
		policy  Policy
		wantErr error
	}{
		{"127.0.0.1", Policy{}, ErrBlockedLoopback},
		{"localhost", Policy{}, ErrBlockedLoopback},
		{"localhost:8080", Policy{}, ErrBlockedLoopback},
		{"127.0.0.1:5432", Policy{}, ErrBlockedLoopback},
		{"169.254.169.254", Policy{}, ErrBlockedMetadata},
		{"169.254.169.254:80", Policy{}, ErrBlockedMetadata},
		{"metadata.google.internal", Policy{}, ErrBlockedMetadata},
		{"10.0.0.1", Policy{}, ErrBlockedPrivate},
		{"172.16.0.1", Policy{}, ErrBlockedPrivate},
		{"192.168.1.1", Policy{}, ErrBlockedPrivate},
		{"10.0.0.1", Policy{AllowPrivate: true}, nil},
		{"127.0.0.1", Policy{AllowLoopback: true}, nil},
		{"8.8.8.8", Policy{}, nil},
		{"1.1.1.1:443", Policy{}, nil},
	}

	for _, tt := range tests {
		err := ValidateSafeHost(tt.host, tt.policy)
		if tt.wantErr != nil {
			if err == nil {
				t.Errorf("ValidateSafeHost(%q, %+v) expected error %v, got nil", tt.host, tt.policy, tt.wantErr)
			} else if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
				t.Errorf("ValidateSafeHost(%q, %+v) expected error %v, got %v", tt.host, tt.policy, tt.wantErr, err)
			}
		} else if err != nil {
			t.Errorf("ValidateSafeHost(%q, %+v) expected nil, got %v", tt.host, tt.policy, err)
		}
	}
}

func TestValidateSafeURL(t *testing.T) {
	tests := []struct {
		url     string
		policy  Policy
		wantErr bool
	}{
		{"http://127.0.0.1/oauth/token", Policy{}, true},
		{"http://localhost:8000/token", Policy{}, true},
		{"http://169.254.169.254/latest/meta-data/", Policy{}, true},
		{"ftp://example.com/file", Policy{}, true},
		{"https://auth.company.com/oauth/token", Policy{}, false},
		{"http://127.0.0.1/oauth/token", Policy{AllowLoopback: true}, false},
	}

	for _, tt := range tests {
		err := ValidateSafeURL(tt.url, tt.policy)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateSafeURL(%q) expected error, got nil", tt.url)
		} else if !tt.wantErr && err != nil {
			t.Errorf("ValidateSafeURL(%q) expected nil, got %v", tt.url, err)
		}
	}
}

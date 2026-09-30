package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHTunnel represents an SSH tunnel connection for bastion/jump hosts
type SSHTunnel struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
	SSHHost    string
	SSHPort    int
	SSHUser    string
	HostKey    string
	AuthMethod ssh.AuthMethod

	listener net.Listener
	client   *ssh.Client
	done     chan struct{}
	mu       sync.Mutex
}

// Config for creating an SSH tunnel
type Config struct {
	SSHHost    string
	SSHPort    int
	SSHUser    string
	Password   string
	PrivateKey string
	HostKey    string // optional base64 or authorized_keys format host public key
	RemoteHost string
	RemotePort int
}

// NewSSHTunnel creates a new SSH tunnel
func NewSSHTunnel(cfg Config) (*SSHTunnel, error) {
	var authMethod ssh.AuthMethod

	if cfg.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else if cfg.Password != "" {
		authMethod = ssh.Password(cfg.Password)
	} else {
		return nil, fmt.Errorf("no authentication method provided")
	}

	return &SSHTunnel{
		RemoteHost: cfg.RemoteHost,
		RemotePort: cfg.RemotePort,
		SSHHost:    cfg.SSHHost,
		SSHPort:    cfg.SSHPort,
		SSHUser:    cfg.SSHUser,
		HostKey:    cfg.HostKey,
		AuthMethod: authMethod,
		done:       make(chan struct{}),
	}, nil
}

// Start starts the SSH tunnel and returns the allocated local port
func (t *SSHTunnel) Start() (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var hostKeyCallback ssh.HostKeyCallback
	if t.HostKey != "" {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(t.HostKey))
		if err != nil {
			return 0, fmt.Errorf("failed to parse SSH host key: %w", err)
		}
		hostKeyCallback = ssh.FixedHostKey(pubKey)
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	sshConfig := &ssh.ClientConfig{
		User:            t.SSHUser,
		Auth:            []ssh.AuthMethod{t.AuthMethod},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	sshAddr := fmt.Sprintf("%s:%d", t.SSHHost, t.SSHPort)
	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	t.client = client

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return 0, fmt.Errorf("failed to start local listener: %w", err)
	}
	t.listener = listener
	t.LocalPort = listener.Addr().(*net.TCPAddr).Port

	go t.handleConnections()

	return t.LocalPort, nil
}

func (t *SSHTunnel) handleConnections() {
	for {
		select {
		case <-t.done:
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				select {
				case <-t.done:
					return
				default:
					continue
				}
			}
			go t.forward(conn)
		}
	}
}

func (t *SSHTunnel) forward(localConn net.Conn) {
	defer localConn.Close()

	remoteAddr := fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort)
	remoteConn, err := t.client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	<-done
}

// Stop closes the tunnel listener and ssh client connection
func (t *SSHTunnel) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	close(t.done)

	if t.listener != nil {
		t.listener.Close()
	}

	if t.client != nil {
		t.client.Close()
	}

	return nil
}

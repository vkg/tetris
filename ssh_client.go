package tetris

import (
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

// SSHClient is a ssh client
type SSHClient struct {
	client *ssh.Client
}

// NewSSHClient returns a new SSHClient
func NewSSHClient(user, addr string, key ssh.Signer) (*SSHClient, error) {
	var auth []ssh.AuthMethod
	auth = append(auth, ssh.PublicKeys(key))

	// set ssh config.
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, xerrors.Errorf("failed to ssh.Dial: %w", err)
	}

	return &SSHClient{
		client: client,
	}, nil
}

// SendCommand sends a command as an exec request
func (c *SSHClient) SendCommand(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", xerrors.Errorf("failed to NewSession: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	switch err.(type) {
	case *ssh.ExitMissingError:
		// todo: how can i fix?
	case *ssh.ExitError:
		// todo: how can i fix?
	default:
		return "", xerrors.Errorf("failed to send SSH command: %w", err)
	}

	return string(out), nil
}

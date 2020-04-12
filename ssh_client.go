package tetris

import (
	"context"
	"io"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type clientSession interface {
	Close() error
}

// SSHClient is a ssh client
type SSHClient struct {
	client   *ssh.Client
	sessions map[string]clientSession
	mux      sync.RWMutex
	logger   *zap.Logger
}

// NewSSHClient returns a new SSHClient
func NewSSHClient(user, addr string, key ssh.Signer, logger *zap.Logger) (*SSHClient, error) {
	var auth []ssh.AuthMethod
	auth = append(auth, ssh.PublicKeys(key))

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
		client:   client,
		sessions: make(map[string]clientSession),
		mux:      sync.RWMutex{},
		logger:   logger,
	}, nil
}

func (c *SSHClient) Close() {
	c.mux.Lock()
	defer c.mux.Unlock()
	for _, s := range c.sessions {
		s.Close()
	}
	c.sessions = make(map[string]clientSession)
	c.client.Close()
}

// NewStream returns a new SSH stream session
func (c *SSHClient) NewStreamSession(ctx context.Context, name string, sendQueSize, recvQueSize int) (*ClientStream, error) {
	logger := c.logger.With(zap.String("session", name))

	c.mux.Lock()
	defer c.mux.Unlock()

	session, in, out, err := c.newSession(name)
	if err != nil {
		return nil, err
	}

	sess := newClientStream(session, in, out, sendQueSize, recvQueSize)

	go func() {
		if sess.StartStream(ctx, logger); err != nil {
			logger.Error("client stream results in fail", zap.Error(err))
		}
	}()

	c.sessions[name] = sess

	return sess, nil
}

// NewUnary returns a new SSH unary client session
func (c *SSHClient) NewUnarySession(name string) (*ClientUnary, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	session, in, out, err := c.newSession(name)
	if err != nil {
		return nil, err
	}

	sess := newClientUnary(session, in, out)
	c.sessions[name] = sess
	return sess, nil
}

func (c *SSHClient) newSession(name string) (*ssh.Session, io.WriteCloser, io.Reader, error) {
	if _, ok := c.sessions[name]; ok {
		return nil, nil, nil, xerrors.Errorf("session %s has already existed", name)
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("failed to NewSession: %w", err)
	}

	in, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, xerrors.Errorf("failed to new pipe: %w", err)
	}
	out, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, xerrors.Errorf("failed to new pipe: %w", err)
	}

	if err := session.Start(name); err != nil {
		session.Close()
		return nil, nil, nil, xerrors.Errorf("failed to start session: %w", err)
	}

	return session, in, out, nil
}

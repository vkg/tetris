package tetris

import (
	"context"
	"net"
	"strings"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type SSHUser struct {
	UserName string
}

type KeyRegister interface {
	Find(conn ssh.ConnMetadata, key ssh.PublicKey) (SSHUser, error)
}

type ServerHandler func(ctx context.Context, stream *ServerStream)

// SSHServer is a ssh server
type SSHServer struct {
	keyRegister  KeyRegister
	mux          sync.RWMutex
	handlers     map[string]ServerHandler // session name -> handler
	userSessions map[string]SSHUser       // pubkey -> user
	streams      []*ServerStream
	logger       *zap.Logger
	listener     net.Listener
	config       *ssh.ServerConfig
	cancelFunc   context.CancelFunc
}

// NewSSHServer returns a ssh server
func NewSSHServer(logger *zap.Logger, addr string, hostKey []byte, keyRegister KeyRegister) (*SSHServer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	hostSigner, err := ssh.ParsePrivateKey(hostKey)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse host key: %w", err)
	}

	server := &SSHServer{
		keyRegister:  keyRegister,
		mux:          sync.RWMutex{},
		userSessions: make(map[string]SSHUser),
		logger:       logger,
		listener:     l,
		config:       &ssh.ServerConfig{},
		handlers:     make(map[string]ServerHandler),
	}

	server.config.PublicKeyCallback = server.publicKeyCallback
	server.config.AddHostKey(hostSigner)

	return server, nil
}

func (s *SSHServer) RegisterHandler(name string, h ServerHandler) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.handlers[name] = h
}

// https://github.com/golang/net/blob/46282727080fcf56da5781d0a9ef2fda184be5e6/http2/server.go#L674
func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}

// Listen starts serving SSH server
func (s *SSHServer) Listen(ctx context.Context) error {
	if s.cancelFunc != nil {
		return xerrors.New("already started")
	}
	ctx, s.cancelFunc = context.WithCancel(ctx)

	for {
		conn, err := s.listener.Accept()
		switch {
		case isClosedConnError(err):
			s.logger.Info("failed to accept due to closed connection", zap.Error(err))
			return nil
		case err != nil:
			s.logger.Error("failed to accept", zap.Error(err))
			return err
		}

		sshConn, chans, _, err := ssh.NewServerConn(conn, s.config)
		if err != nil {
			s.logger.Error("failed to new server conn", zap.Error(err))
			continue
		}

		go func(sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel) {
			s.acceptConnection(ctx, sshConn, chans)
		}(sshConn, chans)
	}
}

func (s *SSHServer) acceptConnection(ctx context.Context, sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel) {
	defer sshConn.Close()

	user, ok := s.userSessions[string(sshConn.SessionID())]
	if !ok {
		user = SSHUser{
			UserName: "UNKNOWN",
		}
	}

	logger := s.logger.With(zap.String("user", user.UserName), zap.Binary("session_id", sshConn.SessionID()),
		zap.String("remote_addr", sshConn.RemoteAddr().String()))

	logger.Info("accept new connection")

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			logger.Warn("unknown channel type", zap.String("type", newChannel.ChannelType()))
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		ch, requests, err := newChannel.Accept()
		if err != nil {
			logger.Error("failed to accept new channel", zap.Error(err))
			return
		}

		go func(ch ssh.Channel, requests <-chan *ssh.Request) {
			defer ch.Close()

			req := <-requests

			if req.Type != "exec" {
				logger.Warn("unknown request type", zap.String("type", req.Type))
				req.Reply(false, nil)
				return
			}

			// exec payload: SSH_MSG_CHANNEL_REQUEST
			// uint32    packet_length
			// byte      padding_length
			// byte[n1]  payload; n1 = packet_length - padding_length - 1
			// byte[n2]  random padding; n2 = padding_length
			cmd := string(req.Payload[4:])
			s.mux.RLock()
			handler, ok := s.handlers[cmd]
			s.mux.RUnlock()

			if !ok {
				logger.Warn("unknown command", zap.String("cmd", cmd))
				req.Reply(false, nil)
				return
			}

			req.Reply(true, nil)

			ss := newServerStream(ch, &user, 0, 0) // TODO: allow to configure que size
			defer ss.Close()

			go func() {
				if err := ss.startStream(ctx, logger); err != nil {
					logger.Error("failed to start server stream", zap.Error(err))
					return
				}
			}()

			handler(ctx, ss)

		}(ch, requests)
	}
}

func (s *SSHServer) publicKeyCallback(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	user, err := s.keyRegister.Find(conn, key)
	if err != nil {
		s.logger.Info("unknown user", zap.Error(err))
		return nil, xerrors.New("unauthorized")
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.userSessions[string(conn.SessionID())] = user
	return nil, nil
}

func (s *SSHServer) Close() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	if s.listener != nil {
		s.listener.Close()
	}
}

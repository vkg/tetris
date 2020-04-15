package tetris

import (
	"context"
	"io"

	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

// Server is a tetris server
type Server struct {
	logger    *zap.Logger
	sshServer *SSHServer
}

func NewServer(logger *zap.Logger, addr string, hostkey []byte, register KeyRegister) (*Server, error) {
	s, err := NewSSHServer(logger, addr, hostkey, register)
	if err != nil {
		logger.Error("failed to new server", zap.Error(err), zap.String("addr", addr))
		return nil, err
	}

	return &Server{
		logger:    logger,
		sshServer: s,
	}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	// register all handlers here
	s.sshServer.RegisterHandler(CowsayName, s.cowsayHandler)

	return s.sshServer.Listen(ctx)
}

func (s *Server) cowsayHandler(ctx context.Context, stream *ServerStream) {
	for {
		var req CowsayRequest
		err := stream.RecvMsgPack(&req)
		switch {
		case xerrors.Is(err, io.EOF):
			return
		case err != nil:
			s.logger.Error("failed to recv, stop to handle", zap.Error(err))
			return
		}

		res := &CowsayResponse{
			Say: "mow", // TODO: fix me
		}

		if err := stream.SendMsgPack(&res); err != nil {
			s.logger.Error("failed to send")
		}
	}
}

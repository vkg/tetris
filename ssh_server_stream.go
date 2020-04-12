package tetris

import (
	"context"
	"io"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

type ServerStream struct {
	channel  ssh.Channel
	user     *SSHUser
	request  chan *Packet
	response chan *Packet
}

func newServerStream(ch ssh.Channel, user *SSHUser, sendQueSize, recvQueSize int) *ServerStream {
	return &ServerStream{
		channel:  ch,
		user:     user,
		request:  make(chan *Packet, sendQueSize),
		response: make(chan *Packet, recvQueSize),
	}
}

func (ss *ServerStream) Close() {
	close(ss.request)
	close(ss.response)
}

func (ss *ServerStream) startStream(ctx context.Context, logger *zap.Logger) error {
	eg, ctx := errgroup.WithContext(ctx)

	// start watching request
	eg.Go(func() error {
		for {
			p := <-ss.request
			if p == nil {
				return nil
			}
			if err := p.Write(ss.channel); err != nil {
				logger.Error("failed to write to server stream", zap.Error(err), zap.Any("request", p))
				return xerrors.Errorf("failed to write to stream: %w", err)
			}
		}
	})

	// start receiving
	eg.Go(func() error {
		for {
			p, err := ReadPacket(ss.channel)
			if xerrors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				logger.Error("failed to read from server stream", zap.Error(err))
				return err
			}
			ss.response <- p
		}
	})

	return eg.Wait()
}

func (ss *ServerStream) Send(p *Packet) error {
	ss.request <- p
	return nil
}

func (ss *ServerStream) Recv() (*Packet, error) {
	p := <-ss.response
	if p == nil {
		return nil, io.EOF
	}
	return p, nil
}

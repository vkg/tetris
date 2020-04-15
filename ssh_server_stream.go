package tetris

import (
	"context"
	"io"

	"github.com/vmihailenco/msgpack/v4"
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

	logger.Debug("start new stream")

	fin := make(chan struct{})
	defer close(fin)

	// start watching request
	eg.Go(func() error {
		for {
			select {
			case p := <-ss.request:
				if p == nil {
					return nil
				}
				logger.Debug("send response", zap.Int("length", len(p.Data)))
				err := p.Write(ss.channel)
				switch {
				case xerrors.Is(err, io.EOF):
					logger.Info("write channel is closed")
					return nil
				case err != nil:
					logger.Error("failed to write to server stream", zap.Error(err), zap.Any("request", p))
					return xerrors.Errorf("failed to write to stream: %w", err)
				}
			case <-fin:
				return nil
			}
		}
	})

	// start receiving
	eg.Go(func() error {
		for {
			p, err := ReadPacket(ss.channel)
			if xerrors.Is(err, io.EOF) {
				fin <- struct{}{}
				return nil
			}
			if err != nil {
				logger.Error("failed to read from server stream", zap.Error(err))
				return err
			}
			logger.Debug("receive request", zap.Int("length", len(p.Data)))
			ss.response <- p
		}
	})

	err := eg.Wait()
	logger.Debug("finish the stream", zap.Error(err))
	return err
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

func (ss *ServerStream) SendMsgPack(v interface{}) error {
	data, err := msgpack.Marshal(v)
	if err != nil {
		return xerrors.Errorf("failed to marshal msgpack: %w", err)
	}
	return ss.Send(&Packet{Data: data})
}

func (ss *ServerStream) RecvMsgPack(out interface{}) error {
	p, err := ss.Recv()
	if err != nil {
		return err
	}

	if err := msgpack.Unmarshal(p.Data, out); err != nil {
		return xerrors.Errorf("failed to unmarshal msgpack: %w", err)
	}

	return nil
}

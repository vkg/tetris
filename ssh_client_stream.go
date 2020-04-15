package tetris

import (
	"context"
	"io"

	"github.com/vmihailenco/msgpack/v4"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

// ClientStream is a ssh session
type ClientStream struct {
	session  *ssh.Session
	writer   io.WriteCloser
	reader   io.Reader
	request  chan *Packet
	response chan *Packet
	closed   *atomic.Bool
}

func newClientStream(session *ssh.Session, writer io.WriteCloser, reader io.Reader, sendQueSize, recvQueSize int) *ClientStream {
	return &ClientStream{
		session:  session,
		writer:   writer,
		reader:   reader,
		request:  make(chan *Packet, sendQueSize),
		response: make(chan *Packet, recvQueSize),
		closed:   atomic.NewBool(false),
	}
}

func (c *ClientStream) Send(p *Packet) error {
	if c.closed.Load() {
		return io.EOF
	}
	c.request <- p
	return nil
}

func (c *ClientStream) Recv() (*Packet, error) {
	if c.closed.Load() {
		return nil, io.EOF
	}
	p := <-c.response
	if p == nil {
		return nil, io.EOF
	}
	return p, nil
}

func (c *ClientStream) SendMsgPack(v interface{}) error {
	data, err := msgpack.Marshal(v)
	if err != nil {
		return xerrors.Errorf("failed to marshal msgpack: %w", err)
	}
	return c.Send(&Packet{Data: data})
}

func (c *ClientStream) RecvMsgPack(out interface{}) error {
	p, err := c.Recv()
	if err != nil {
		return err
	}

	if err := msgpack.Unmarshal(p.Data, out); err != nil {
		return xerrors.Errorf("failed to unmarshal msgpack: %w", err)
	}

	return nil
}

func (c *ClientStream) Close() error {
	var err error
	if c.closed.CAS(false, true) {
		close(c.request)
		close(c.response)
		err = c.session.Close()
	}
	return err
}

func (c *ClientStream) StartStream(ctx context.Context, logger *zap.Logger) error {
	eg, ctx := errgroup.WithContext(ctx)

	// start watching request
	eg.Go(func() error {
		for {
			p := <-c.request
			if p == nil {
				return nil
			}
			err := p.Write(c.writer)
			switch {
			case xerrors.Is(err, io.EOF):
				logger.Debug("write channel is closed")
				return nil
			case err != nil:
				logger.Error("failed to write to client stream", zap.Error(err), zap.Any("request", p))
				return xerrors.Errorf("failed to write to stream: %w", err)
			}
		}
	})

	// start receiving
	eg.Go(func() error {
		for {
			p, err := ReadPacket(c.reader)
			if xerrors.Is(err, io.EOF) {
				logger.Debug("recv channel is closed")
				return nil
			}
			if err != nil {
				logger.Error("failed to read from client stream", zap.Error(err))
				return err
			}
			c.response <- p
		}
	})

	err := eg.Wait()
	logger.Debug("client stream closed", zap.Error(err))
	return err
}

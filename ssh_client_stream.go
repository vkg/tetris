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

// ClientStream is a ssh session
type ClientStream struct {
	session  *ssh.Session
	writer   io.WriteCloser
	reader   io.Reader
	request  chan *Packet
	response chan *Packet
}

func newClientStream(session *ssh.Session, writer io.WriteCloser, reader io.Reader, sendQueSize, recvQueSize int) *ClientStream {
	return &ClientStream{
		session:  session,
		writer:   writer,
		reader:   reader,
		request:  make(chan *Packet, sendQueSize),
		response: make(chan *Packet, recvQueSize),
	}
}

func (c *ClientStream) Send(p *Packet) error {
	c.request <- p
	return nil
}

func (c *ClientStream) Recv() (*Packet, error) {
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
	c.request <- &Packet{Data: data}
	return nil
}

func (c *ClientStream) RecvMsgPack(out interface{}) error {
	p := <-c.response
	if p == nil {
		return io.EOF
	}

	if err := msgpack.Unmarshal(p.Data, out); err != nil {
		return xerrors.Errorf("failed to unmarshal msgpack: %w", err)
	}

	return nil
}

func (c *ClientStream) Close() error {
	close(c.request)
	close(c.response)
	return c.session.Close()
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
			if err := p.Write(c.writer); err != nil {
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
				return nil
			}
			if err != nil {
				logger.Error("failed to read from client stream", zap.Error(err))
				return err
			}
			c.response <- p
		}
	})

	return eg.Wait()
}

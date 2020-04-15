package tetris

import (
	"io"

	"github.com/vmihailenco/msgpack/v4"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

// ClientUnary is a ssh session
type ClientUnary struct {
	session *ssh.Session
	writer  io.WriteCloser
	reader  io.Reader
}

func newClientUnary(session *ssh.Session, writer io.WriteCloser, reader io.Reader) *ClientUnary {
	return &ClientUnary{
		session: session,
		writer:  writer,
		reader:  reader,
	}
}

func (c *ClientUnary) SendAndRecv(req *Packet) (*Packet, error) {
	if err := req.Write(c.writer); err != nil {
		return nil, err
	}
	return ReadPacket(c.reader)
}

func (c *ClientUnary) SendAndRecvMsgPack(req, res interface{}) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return xerrors.Errorf("failed to marshal request: %w", err)
	}
	reqp := &Packet{Data: data}

	if err := reqp.Write(c.writer); err != nil {
		return err
	}
	resp, err := ReadPacket(c.reader)
	if err != nil {
		return err
	}
	if err := msgpack.Unmarshal(resp.Data, &res); err != nil {
		return xerrors.Errorf("failed to unmarshal response: %w", err)
	}
	return nil
}

func (c *ClientUnary) Close() error {
	return c.session.Close()
}

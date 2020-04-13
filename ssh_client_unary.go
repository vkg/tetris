package tetris

import (
	"io"

	"golang.org/x/crypto/ssh"
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

func (c *ClientUnary) Close() error {
	return c.session.Close()
}

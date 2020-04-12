package tetris

import (
	"encoding/binary"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

// SSHClient is a ssh client
type SSHClient struct {
	client   *ssh.Client
	sessions map[string]*SSHSession
	sessMux  sync.RWMutex
}

// SSHSession is a ssh session
type SSHSession struct {
	session *ssh.Session
	writer  io.WriteCloser
	reader  io.Reader
}

// TODO: fix it
type Packet struct {
	Data []byte
}

// NewSSHClient returns a new SSHClient
func NewSSHClient(user, addr string, key ssh.Signer) (*SSHClient, error) {
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
		sessions: make(map[string]*SSHSession),
		sessMux:  sync.RWMutex{},
	}, nil
}
func (c *SSHClient) Close() {
	c.sessMux.Lock()
	defer c.sessMux.Unlock()
	for _, s := range c.sessions {
		s.session.Close()
	}
	c.sessions = make(map[string]*SSHSession)
	c.client.Close()
}

// NewSession returns a new SSH session
func (c *SSHClient) NewSession(name string) (*SSHSession, error) {
	c.sessMux.Lock()
	defer c.sessMux.Unlock()

	if _, ok := c.sessions[name]; ok {
		return nil, xerrors.Errorf("session %s has already existed", name)
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, xerrors.Errorf("failed to NewSession: %w", err)
	}

	in, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, xerrors.Errorf("failed to new pipe: %w", err)
	}
	out, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, xerrors.Errorf("failed to new pipe: %w", err)
	}

	if err := session.Start(name); err != nil {
		session.Close()
		return nil, xerrors.Errorf("failed to start session: %w", err)
	}
	sess := &SSHSession{
		session: session,
		writer:  in,
		reader:  out,
	}

	c.sessions[name] = sess

	return sess, nil
}

func (s *SSHSession) Send(p *Packet) error {
	return p.write(s.writer)
}

func (s *SSHSession) Recv() (*Packet, error) {
	return readPacket(s.reader)
}

func (s *SSHSession) SendAndRecv(p *Packet) (*Packet, error) {
	if err := p.write(s.writer); err != nil {
		return nil, err
	}
	return readPacket(s.reader)
}

func (p *Packet) write(w io.Writer) error {
	header := make([]byte, 4) // TODO: define protocol
	binary.BigEndian.PutUint32(header, uint32(len(p.Data)))
	if _, err := w.Write(header); err != nil {
		return xerrors.Errorf("failed to write header: %w", err)
	}
	if _, err := w.Write(p.Data); err != nil {
		return xerrors.Errorf("failed to write data: %w", err)
	}
	return nil
}

func readPacket(r io.Reader) (*Packet, error) {
	header := make([]byte, 4) // TODO: define protocol
	if _, err := r.Read(header); err != nil {
		return nil, xerrors.Errorf("failed to read header: %w", err)
	}

	len := binary.BigEndian.Uint32(header)
	buf := make([]byte, len)
	if _, err := r.Read(buf); err != nil {
		return nil, xerrors.Errorf("failed to read data: %w", err)
	}
	return &Packet{
		Data: buf,
	}, nil
}

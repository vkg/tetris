package tetris

import (
	"bytes"
	"io"
	"testing"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

func TestSSHClient_SendRecv(t *testing.T) {
	pubkey := defaultPublicKey(t).Marshal()
	c := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() != "test" {
				t.Errorf("unexpected user %s", conn.User())
			}
			if 0 != bytes.Compare(pubkey, key.Marshal()) {
				return nil, xerrors.New("unauthorized user")
			}
			return nil, nil
		},
	}

	handlers := map[string]sshHandler{
		"test": func(r *Packet, w io.Writer) bool {
			res := func(s string) {
				p := Packet{Data: []byte(s)}
				if err := p.Write(w); err != nil {
					t.Fatal(err)
				}
			}
			switch string(r.Data) {
			case "ping":
				res("pong")
				return false
			case "fin":
				res("bye")
				return true
			}
			res("unknown")
			return false
		},
	}
	addr := testSSHServer(t, c, handlers)

	cli, err := NewSSHClient("test", addr.String(), defaultPrivateKey(t), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	sess, err := cli.NewUnarySession("test")
	if err != nil {
		t.Fatal(err)
	}

	res, err := sess.SendAndRecv(&Packet{
		Data: []byte("ping"),
	})
	if string(res.Data) != "pong" {
		t.Error("unexpected")
	}
	res, err = sess.SendAndRecv(&Packet{
		Data: []byte("fin"),
	})
	if string(res.Data) != "bye" {
		t.Error("unexpected")
	}
}

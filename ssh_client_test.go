package tetris

import (
	"bytes"
	"io"
	"testing"

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

	addr := testSSHServer(t, c, func(r *Packet, w io.Writer) bool {
		res := func(s string) {
			p := Packet{Data: []byte(s)}
			if err := p.write(w); err != nil {
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
	})

	cli, err := NewSSHClient("test", addr.String(), defaultPrivateKey(t))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	sess, err := cli.NewSession()
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

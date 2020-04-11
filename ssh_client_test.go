package tetris

import (
	"bytes"
	"fmt"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

func TestSSHClient_SendCommand(t *testing.T) {
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

	addr := testSSHServer(t, c, func(cmd string) string {
		switch cmd {
		case "ping":
			return "pong"
		case "fin":
			return "bye"
		default:
			return fmt.Sprintf("unknown command [%s]", cmd)
		}
	})

	cli, err := NewSSHClient("test", addr.String(), defaultPrivateKey(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := cli.SendCommand("ping")
	if err != nil {
		t.Fatal(err)
	}
	if out != "pong" {
		t.Error("unexpected response")
	}
	out, err = cli.SendCommand("fin")
	if err != nil {
		t.Fatal(err)
	}
	if out != "bye" {
		t.Error("unexpected response")
	}
}

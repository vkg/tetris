package tetris

import (
	"context"
	"io"
	"sync"
	"testing"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

func Test_newClientStream(t *testing.T) {
	c := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil
		},
	}

	handlers := map[string]sshHandler{
		"test": func(r *Packet, w io.Writer) bool {
			res := func(s string) {
				p := Packet{Data: []byte(s)}
				// respond twice
				if err := p.Write(w); err != nil {
					t.Fatal(err)
				}
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

	sess, err := cli.NewStreamSession(context.Background(), "test", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	req := []string{"ping", "ping", "ping", "fooooo", "ping", "baaaaa"}
	var responses []string
	mux := sync.Mutex{}
	var wg sync.WaitGroup
	for _, r := range req {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			if err := sess.Send(&Packet{Data: []byte(r)}); err != nil {
				panic(err)
			}
		}(r)

		wg.Add(1) // number of response is double
		go func() {
			defer wg.Done()
			res1, err := sess.Recv()
			if err != nil {
				panic(err)
			}
			res2, err := sess.Recv()
			if err != nil {
				panic(err)
			}
			mux.Lock()
			defer mux.Unlock()
			responses = append(responses, string(res1.Data), string(res2.Data))
		}()
	}

	wg.Wait()

	if len(responses) != 2*len(req) {
		t.Error("unexpected length")
	}
	var pong, unknown int
	for _, s := range responses {
		switch s {
		case "pong":
			pong++
		case "unknown":
			unknown++
		default:
			t.Fatal("unknown commands included")
		}
	}
	if pong != 8 || unknown != 4 {
		t.Error("unexpected count")
	}
}

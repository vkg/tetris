package tetris

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type mockedKeyRegister struct {
	FindMock func(key ssh.PublicKey) (SSHUser, error)
}

func (m *mockedKeyRegister) Find(key ssh.PublicKey) (SSHUser, error) {
	return m.FindMock(key)
}

func Test_newServerStream(t *testing.T) {
	addr := "127.0.0.1:31113"
	testUser := SSHUser{
		UserName: "test",
	}

	keyRegister := &mockedKeyRegister{
		FindMock: func(key ssh.PublicKey) (SSHUser, error) {
			return testUser, nil
		},
	}

	server, err := NewSSHServer(zap.NewNop(), addr, []byte(testHostKey), keyRegister)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	server.RegisterHandler("handler", func(ctx context.Context, stream *ServerStream) {
		if diff := cmp.Diff(stream.user, &testUser); diff != "" {
			t.Errorf("unexpected user, %s", diff)
		}
		for {
			p, err := stream.Recv()
			switch {
			case xerrors.Is(err, io.EOF):
				return
			case err != nil:
				t.Fatal(err)
			}
			var res Packet
			switch string(p.Data) {
			case "ping":
				res.Data = []byte("pong")
			case "fin":
				res.Data = []byte("bye")
			default:
				continue
			}
			// respond twice
			if err := stream.Send(&res); err != nil {
				t.Fatal(err)
			}
			if err := stream.Send(&res); err != nil {
				t.Fatal(err)
			}
		}
	})

	go func() {
		if err := server.Listen(context.Background()); err != nil {
			panic(err)
		}
	}()

	cli, err := NewSSHClient(testUser.UserName, addr, defaultPrivateKey(t), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	sess, err := cli.NewStreamSession(context.Background(), "handler", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	req := []string{"ping", "ping", "ping", "fin", "ping", "fin"}
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
	var pong, bye int
	for _, s := range responses {
		switch s {
		case "pong":
			pong++
		case "bye":
			bye++
		default:
			t.Fatal("unknown commands included")
		}
	}
	if pong != 8 || bye != 4 {
		t.Error("unexpected count")
	}
}

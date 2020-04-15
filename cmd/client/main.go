package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/vkg/tetris"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

func main() {
	// TODO: fix me

	var (
		addr     = flag.String("addr", "127.0.0.1:8022", "server address")
		keyPath  = flag.String("key", "", "path to your key file, it should be registered on Github")
		username = flag.String("user", "", "user name (Github id)")
	)
	flag.Parse()

	if *keyPath == "" {
		if s, err := user.Current(); err == nil {
			*keyPath = filepath.Join(s.HomeDir, "/.ssh/id_rsa")
		}
	}
	absKeyPath, err := filepath.Abs(*keyPath)
	if err != nil {
		log.Fatalf("failed to solve file path %s: %s", *keyPath, err.Error())
	}
	buf, err := ioutil.ReadFile(absKeyPath)
	if err != nil {
		log.Fatalf("failed to read %s: %s", absKeyPath, err.Error())
	}

	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		log.Fatalf("failed to parse key file %s: %s", *keyPath, err.Error())
	}

	if *username == "" {
		if s, err := user.Current(); err == nil {
			*username = s.Name
		}
	}

	logger, _ := zap.NewDevelopment()
	cli, err := tetris.NewSSHClient(*username, *addr, key, logger)
	if err != nil {
		log.Fatalf("failed to start ssh client: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cowsay, err := cli.NewStreamSession(ctx, tetris.CowsayName, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer cowsay.Close()

	ch := make(chan struct{})
	defer close(ch)

	go func() {
		for {
			var in string
			if _, err := fmt.Scan(&in); err != nil {
				log.Printf("scan error: %v\n", err)
				break
			}
			if strings.HasPrefix(in, "exit") {
				log.Println("exit")
				break
			}
			err := cowsay.SendMsgPack(&tetris.CowsayRequest{Key: in})
			if xerrors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				log.Printf("failed to send: %v\n", err)
				continue
			}
		}
		ch <- struct{}{}
	}()

	go func() {
		for {
			var res tetris.CowsayResponse
			err := cowsay.RecvMsgPack(&res)
			if xerrors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				log.Printf("failed to recv: %v\n", err)
				continue
			}

			log.Println(res.Say)
		}
	}()

	<-ch
	cancel()
}

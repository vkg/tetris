package tetris

import (
	"io"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

const (
	testPrivateKeyPass = "pass"
	testPrivateKey     = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAACmFlczI1Ni1jdHIAAAAGYmNyeXB0AAAAGAAAABCDD9U+iK
roANP2QzNCJ4/sAAAAEAAAAAEAAAEXAAAAB3NzaC1yc2EAAAADAQABAAABAQDyYUJihQrm
pJ6+AoMvNFvyfDcFDivlyUXjBLYUbtmF9EG1KBNlFySMCfYQHmWvms4LhYzTTvOCurndMI
l2bc3ebqbsH6OKkYs66OHLrLQnlsO7cWTKdeRV+oP1gyD4ChrjtK9HYRb+s1oI94X/LbFG
aMMiEBOtTDg+ztm27MVRvg9WCBF0sEHkvVd75bERtMZu3Q5fmukbC8MGyEEHhwYMDyPNxe
b9h1LwmSm5q9zwp8gd/EiU9nhFuwgC/ehazHkMtdNdyL4WHhphOg6etcdLtIKo8h7Qisxe
V22OZb/Orskr4UekKu3f+cRRdrTqWSOt/+vHkNpvpTG33HspvtCTAAAD0JpyLcqFfZdNzp
U8xbWtQ53gie6uOd4f6mt+HUQyjDCuJhG7rfcwCBJNg8st6U0LUCazVKnQ56mRFiTdLeb3
KXU2sJ+UF0oqTJF/xku65rybU+NJfeLD0yXNqhlO2B5DNlY0DRst5MHog/xpzc4pbEwzka
GuhqWeANS4p7U6kHtvvP/NacaZoUTEVKFmoW1aVe7TGkfLV/zGS8Efs8PEpfWxF/4vT9Np
v0xHgUmbOr3JtgT7UWAAhbDFUon4sTXIiGWTr0wQnZg7w50l0dzIhXp78Y5XtpEkSJb/Yc
0iFB1lERAjEmCiQFRqTTaE3tdSeaareW6f68hkz6Vgg3En5Zy7SVFH0OtJfpFC4oOF8TCa
HDxF9AbvEsh9MO7EbOfyJAsGLga4aKsLNBRln28VKnL+eqR0K+UIoUCqgN/DiDUrikketr
Nv+jNMerhAVRaicjNt3NNgmVsHcX78Fk7Putqyn4nq2YvLv7g2UoCSYhBxrNJsSJY/Nh9y
Z+ODdrWJFMPQNPcyzLj37+DkcPm06mKcon+OdtgZq/cImLa/rAo8LK8e3oV/7Ow5jtnqco
ovqwows78xcwX9HJ8HWpiXYzSV6BmC8tlXdW2zeCsy7CzWrsPLtqzAtQgnEx/h2/jh4Zln
0UfqEMMYjHnAZSVHcQoSLbtZIszTiPmpolcVnslV0qYwKVUI8caNFpIb9qGL1vjlybfBLH
u9u/TO8t6sNMrgTWcySMrL33fzbux/PZhKtiNWoLP4kVmT2PAYcODr3Sk8ob9kvFoEK9b8
gDxbgYfZS1TrxtSoGIYmZ4lUGQpZqOxA3ZgnqZ0709z86w7iG1lHZg+p6BDv1O/qGPB1Ii
DaR2J+dWo9iecpKmXBNHD2X4hm8/jc4ofZfKlXRqYd05CUXG8Y15L+aNySVjNrnUUfBimg
/NwkCUF6CGSCJ5epL1vGozEe9vXTSP/mM7yljzyEJMOSospnYXpxGNwzo6tu+KLh9JtHAf
xE61OI7CGnSduilCfGvy2uICAarNdnuo4gjw9F408ny0PA+i7rxYHf/qs/luZ3QSjMhpmp
hB3hg/EizQNkN77/8UOSkaPdZDE/qk5cfhXqY5+VJb2s4tvaXEL1jXaRZ/Ng7XYI6BGNP5
WFKD5ATbSiI4fWhfueLNivm2nSWzHrs8wqKyvR6CfcHE2x/8E0iRj+yr6G/vXvTTE4h5LQ
Oo7RrZJSSZGelGjqjFK5rZU5fmpnvctc9ua2MALlTZoTO9s6lXlsPGopSdSyrDAuKnckVq
IDnFX3yUGsil4m8iX8HgQA4jrq2Ic=
-----END OPENSSH PRIVATE KEY-----`
	testPubKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDyYUJihQrmpJ6+AoMvNFvyfDcFDivlyUXjBLYUbtmF9EG1KBNlFySMCfYQHmWvms4LhYzTTvOCurndMIl2bc3ebqbsH6OKkYs66OHLrLQnlsO7cWTKdeRV+oP1gyD4ChrjtK9HYRb+s1oI94X/LbFGaMMiEBOtTDg+ztm27MVRvg9WCBF0sEHkvVd75bERtMZu3Q5fmukbC8MGyEEHhwYMDyPNxeb9h1LwmSm5q9zwp8gd/EiU9nhFuwgC/ehazHkMtdNdyL4WHhphOg6etcdLtIKo8h7QisxeV22OZb/Orskr4UekKu3f+cRRdrTqWSOt/+vHkNpvpTG33HspvtCT rerorero@PC1833`

	testHostKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAQEA0/VqtpjlYlPy3PwXe1KP6OC/Ud6oacj2tYdNSWwZs508Jt+oN8/K
pqNEUtXrLcBQ93N3rDiwE14QeieJXIbn0ko3qroUCtNOVFOvg9/BQZl/bT7HyPFTGNzV2p
ErBXd0elmQ52qqNryxXB0A43e5LeirmOxlwg79oyV2lb3RHpOX7b8ow6ErhmRLKZYAg7sM
Xy5UnbHlTArIael1IQtgCty3wZdBo/zKzFt0hm1TCa1xOSJgH7XivXQGmbCfloqIhCc9iu
o5Va227rdDkjZYg2WrFmZ/PjnhML795j0ZXxWLZSSt063mqLMUq2T5OUKuZtxv4mHtet6A
Cnqiodu79wAAA8i/AMEbvwDBGwAAAAdzc2gtcnNhAAABAQDT9Wq2mOViU/Lc/Bd7Uo/o4L
9R3qhpyPa1h01JbBmznTwm36g3z8qmo0RS1estwFD3c3esOLATXhB6J4lchufSSjequhQK
005UU6+D38FBmX9tPsfI8VMY3NXakSsFd3R6WZDnaqo2vLFcHQDjd7kt6KuY7GXCDv2jJX
aVvdEek5ftvyjDoSuGZEsplgCDuwxfLlSdseVMCshp6XUhC2AK3LfBl0Gj/MrMW3SGbVMJ
rXE5ImAfteK9dAaZsJ+WioiEJz2K6jlVrbbut0OSNliDZasWZn8+OeEwvv3mPRlfFYtlJK
3TreaosxSrZPk5Qq5m3G/iYe163oAKeqKh27v3AAAAAwEAAQAAAQEAwNxd7SfSEFYydcEr
3JqTN2LIssXWl+q0ERi7ykMCX9yCDx0TAzWfP2Dvmi/rfgWvpnj6O0qZbAX7GCtBYV+fME
k3vbDy66a5byF2YpgGUJpKyCyHvN9YrRbDv8y3SJIY+frlTqxPlN68wwPg+xjE9nDvMoZn
UNwzDW/ZJwdAcIBTDrSw1I/fsXt+9ZRpnWHgyA6q4LFiDMEZXKMzabXOo0GX4+S7C92Rlh
DDObPDHTYTwZhjM1XhD4i8OP+ZPrdvkPfammNZDk8TsWOt96mJNZ8XJXKLdF+EelP9tN2i
shPiA6mUJwi1oDKmGFDqpI/cL4UniDHU/EczkB5Oo7TqgQAAAIBkiFHyYSw1e14tWSJvZ7
gkPtgjnEqIcNnhJwnJRkEvGPOx1RP246JSOdRnY2s/nwjdczc+yf9yL+Cr46M8bI4Bq3lH
/Z51T/BA3zLte4SYgbe0T/oQr8XwUsyYVQjS7ItsOOBEl760el49DAgA0Q5QcKj0YFkSqm
ltJYs2QADZhQAAAIEA9X7uSeIzRMNWcdvnlL+HxZo8vDCHJe6wvKKtJ6zFU7Wlg5nT2Mxe
Ga704m7DHklgW5S7eGtM90UJbYDKOghTbTetKWOSvrV5oDkzh/LrOykXGU/kg2Kwc5V2rJ
wx2evKuBpXZwaZE48UIKzI3K3G60ofrjB8YseE2RmJ0jfCfBMAAACBAN0HIeylqvJdR3nk
KYNjKd/Db1m3knRdi1ZzqDYLfCNxGhnTBFL/N0I8pMMXOQWghx8oKXo8B5HyXTjbpfUwdA
pL9tXGgCCjMNoOOk7owd9UxrfySf/P0BHvQGJJK8YC4/tfohtjPyaWHdyGYDPxpWJD0KVW
kfXuO24a1RHL9rUNAAAAD3Jlcm9yZXJvQFBDMTgzMwECAw==
-----END OPENSSH PRIVATE KEY-----`
	testHostPubKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDT9Wq2mOViU/Lc/Bd7Uo/o4L9R3qhpyPa1h01JbBmznTwm36g3z8qmo0RS1estwFD3c3esOLATXhB6J4lchufSSjequhQK005UU6+D38FBmX9tPsfI8VMY3NXakSsFd3R6WZDnaqo2vLFcHQDjd7kt6KuY7GXCDv2jJXaVvdEek5ftvyjDoSuGZEsplgCDuwxfLlSdseVMCshp6XUhC2AK3LfBl0Gj/MrMW3SGbVMJrXE5ImAfteK9dAaZsJ+WioiEJz2K6jlVrbbut0OSNliDZasWZn8+OeEwvv3mPRlfFYtlJK3TreaosxSrZPk5Qq5m3G/iYe163oAKeqKh27v3 rerorero@PC1833`
)

type sshHandler func(request *Packet, w io.Writer) bool

func defaultPrivateKey(t *testing.T) ssh.Signer {
	t.Helper()
	key, err := ssh.ParsePrivateKeyWithPassphrase([]byte(testPrivateKey), []byte(testPrivateKeyPass))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func defaultPublicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testPubKey))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func defaultHostKey(t *testing.T) ssh.Signer {
	t.Helper()
	hostSigner, err := ssh.ParsePrivateKey([]byte(testHostKey))
	if err != nil {
		t.Fatal(err)
	}
	return hostSigner
}

func testSSHServer(t *testing.T, config *ssh.ServerConfig, handlers map[string]sshHandler) net.Addr {
	t.Helper()

	config.AddHostKey(defaultHostKey(t))

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		l.Close()
	})

	// currently only one connection is acceptable
	go func() {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		sshConn, chans, _, err := ssh.NewServerConn(conn, config)
		if err != nil {
			panic(err)
		}
		defer sshConn.Close()

		for newChannel := range chans {
			if newChannel.ChannelType() != "session" {
				newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				continue
			}
			ch, requests, err := newChannel.Accept()
			if err != nil {
				t.Error(err)
				return
			}

			go func(ch ssh.Channel, requests <-chan *ssh.Request) {
				defer ch.Close()

				req := <-requests

				if req.WantReply {
					req.Reply(true, nil)
				}

				if req.Type != "exec" {
					ch.Write([]byte("request type '" + req.Type + "' is not 'exec'\r\n"))
					return
				}

				// exec payload: SSH_MSG_CHANNEL_REQUEST
				// uint32    packet_length
				// byte      padding_length
				// byte[n1]  payload; n1 = packet_length - padding_length - 1
				// byte[n2]  random padding; n2 = padding_length
				cmd := string(req.Payload[4:])
				handler, ok := handlers[cmd]
				if !ok {
					panic("handler not found: " + cmd)
				}

				for {
					p, err := ReadPacket(ch)
					if xerrors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						panic(err)
					}
					if handler(p, ch) {
						break
					}
				}
			}(ch, requests)
		}
	}()

	return l.Addr()
}

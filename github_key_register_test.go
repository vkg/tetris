package tetris

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

const (
	reroreroKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDS0ldNxZpTDRF0HKRdA5PRnBOIh55CiV+l4aRf1Aqb8OrtxgQ6siU1HXbsBypJN+0ijhlld5ZHJ0ot1LtUWc+lK/kGJFEUBRBodSTBetzMBHIohJZt7e/wN9cs6Qf5NK3tzNoh2d1wrUTzp+c6EDnPsXsxS5b5bpB2RDTIGKAgdkFMHWTyO90BGrvcHUi4oTBlyt66OG1xZHkLhRzDjt0MCogfc1r5NRxNFEP4xG8hWKRRztDjEuwnQZk92dbjSbQLiRwPLPOfG3iaMY9eInndjmTjXZey9qxdfijXUceOWTKJHqwgzLXCzricd2940tAM1kB6uszixN/G6Kg5/tUV"
	codehexKey  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDVBi01tynpQyXwckW36WDW0tS9RJyQfMpqWHUWnM16VZO0OGIPnDs/lHv1Jpw9dT+apE9aPY04iZz0BqK4BIubrrXPLmDyNcrfgqC4KjjuGnkaKyI0fDJiLl3nJ+Xojxyufk1Drf7udXGoqZuGEpBr26y7NuIJlakTyDzDghsP+Dtj3lGe22cmNV4bBocbSfeW6VTy4UBFlVP0MKVYFy+pBMKuWarEfM/vlTkH9kqnogcQqsMrsDGuJnQAsWDHvd1AvJ0XctJmjO64XBHzebOuUyWgFakj7sgk2RhWlbNx2HxZbQNu6Z20ty6HObWi6iDKis0JyTUwYeaEIY7MlNBr"
)

type mockedConnMetadata struct {
	UserMock          func() string
	SessionIDMock     func() []byte
	ClientVersionMock func() []byte
	ServerVersionMock func() []byte
	RemoteAddrMock    func() net.Addr
	LocalAddrMock     func() net.Addr
}

func (m *mockedConnMetadata) User() string {
	return m.UserMock()
}
func (m *mockedConnMetadata) SessionID() []byte {
	return m.SessionIDMock()
}
func (m *mockedConnMetadata) ClientVersion() []byte {
	return m.ClientVersionMock()
}
func (m *mockedConnMetadata) ServerVersion() []byte {
	return m.ServerVersionMock()
}
func (m *mockedConnMetadata) RemoteAddr() net.Addr {
	return m.RemoteAddrMock()
}
func (m *mockedConnMetadata) LocalAddr() net.Addr {
	return m.LocalAddrMock()
}

func TestGithubKeyRegister_Find(t *testing.T) {
	type args struct {
		conn ssh.ConnMetadata
		key  ssh.PublicKey
	}
	tests := []struct {
		name    string
		args    args
		want    SSHUser
		wantErr bool
	}{
		{
			name: "find with rerorero",
			args: args{
				conn: &mockedConnMetadata{
					UserMock: func() string {
						return "rerorero"
					},
				},
				key: parsePubKey(t, reroreroKey),
			},
			want: SSHUser{
				UserName: "rerorero",
			},
			wantErr: false,
		},
		{
			name: "find rerorero with cache",
			args: args{
				conn: &mockedConnMetadata{
					UserMock: func() string {
						return "rerorero"
					},
				},
				key: parsePubKey(t, reroreroKey),
			},
			want: SSHUser{
				UserName: "rerorero",
			},
			wantErr: false,
		},
		{
			name: "find rerorero with invalid user",
			args: args{
				conn: &mockedConnMetadata{
					UserMock: func() string {
						return "EaUdjAamKTjUm91023EEaUdjAamKTjUm91023EaUdjAamKTjUm91023EE"
					},
				},
				key: parsePubKey(t, reroreroKey),
			},
			wantErr: true,
		},
		{
			name: "find codehex with rerorero's key",
			args: args{
				conn: &mockedConnMetadata{
					UserMock: func() string {
						return "Code-Hex"
					},
				},
				key: parsePubKey(t, reroreroKey),
			},
			wantErr: true,
		},
	}

	r := NewGithubKeyRegister(zap.NewNop())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Find(tt.args.conn, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Find() got = %v", diff)
			}
		})
	}
}

func parsePubKey(t *testing.T, key string) ssh.PublicKey {
	t.Helper()
	k, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

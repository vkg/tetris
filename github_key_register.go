package tetris

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

// GithubKeyRegister is public key register that retrieves from Github
type GithubKeyRegister struct {
	logger     *zap.Logger
	httpClient *http.Client
	mux        sync.RWMutex
	cache      map[string]string // pubkey -> github user name
}

// NewGithubKeyRegister returns a new GithubKeyRegister
func NewGithubKeyRegister(logger *zap.Logger) *GithubKeyRegister {
	return &GithubKeyRegister{
		logger:     logger,
		httpClient: new(http.Client),
		mux:        sync.RWMutex{},
		cache:      make(map[string]string),
	}
}

func (r *GithubKeyRegister) Find(conn ssh.ConnMetadata, key ssh.PublicKey) (SSHUser, error) {
	user := conn.User()
	accessKey := string(key.Marshal())
	logger := r.logger.With(zap.String("user", user))

	sshUser, err := r.getUserFromCache(user, accessKey)
	if err == nil {
		return sshUser, nil
	}

	keys, err := r.getKeysFromGithub(user)
	if err != nil {
		logger.Error("failed to get keys from github", zap.Error(err))
		return SSHUser{}, err
	}

	r.mux.Lock()
	for _, k := range keys {
		r.cache[string(k.Marshal())] = user
	}
	r.mux.Unlock()

	return r.getUserFromCache(user, accessKey)
}

func (r *GithubKeyRegister) getUserFromCache(userName, accessKey string) (SSHUser, error) {
	r.mux.RLock()
	githubUser, ok := r.cache[accessKey]
	r.mux.RUnlock()
	if ok {
		if githubUser != userName {
			return SSHUser{}, xerrors.New("key's username is not matched, please use github user name")
		}
		return SSHUser{UserName: githubUser}, nil
	}
	return SSHUser{}, xerrors.New("key not found, please register key on github")
}

func (r *GithubKeyRegister) getKeysFromGithub(userName string) ([]ssh.PublicKey, error) {
	res, err := r.httpClient.Get(fmt.Sprintf("https://github.com/%s.keys", userName))
	switch {
	case err != nil:
		return nil, xerrors.Errorf("failed to GET from github: %w")
	case res.StatusCode == 404:
		return nil, xerrors.New("username is not found, please use github user name")
	case res.StatusCode != 200:
		return nil, xerrors.Errorf("github is unavailable status=%s", res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, xerrors.Errorf("failed to read body: %w", err)
	}
	defer res.Body.Close()

	var keys []ssh.PublicKey
	keyStrings := strings.Split(string(body), "\n")
	for _, k := range keyStrings {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k))
		if err != nil {
			r.logger.Warn("failed to parse pubkey", zap.String("user", userName), zap.String("key", k))
			continue
		}
		keys = append(keys, pubKey)
	}

	return keys, nil
}

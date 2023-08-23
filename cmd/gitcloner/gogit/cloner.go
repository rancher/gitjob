package gogit

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	httpgit "github.com/go-git/go-git/v5/plumbing/transport/http"
	gossh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/rancher/gitjob/cmd/gitcloner/cmd"
	giturls "github.com/whilp/git-urls"
	"golang.org/x/crypto/ssh"
)

const defaultBranch = "master"

var (
	plainClone = git.PlainClone
	readFile   = os.ReadFile
)

type Cloner struct{}

type Options struct {
	Repo            string
	Branch          string
	Auth            transport.AuthMethod
	InsecureSkipTLS bool
	CABundle        []byte
}

func NewCloner() *Cloner {
	return &Cloner{}
}

func (c *Cloner) CloneRepo(opts *cmd.Options) error {
	branch := defaultBranch
	if opts.Branch != "" {
		branch = opts.Branch
	}

	auth, err := createAuthFromOpts(opts)
	if err != nil {
		return err
	}
	caBundle, err := getCABundleFromFile(opts.CABundleFile)
	if err != nil {
		return err
	}

	_, err = plainClone(opts.Path, false, &git.CloneOptions{
		URL:             opts.Repo,
		Auth:            auth,
		InsecureSkipTLS: opts.InsecureSkipTLS,
		CABundle:        caBundle,
		SingleBranch:    true,
		ReferenceName:   plumbing.ReferenceName(branch),
	})
	if err != nil {
		return err
	}

	return nil
}

func getCABundleFromFile(path string) ([]byte, error) {
	if path != "" {
		caBundle, err := readFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		return caBundle, nil
	}

	return nil, nil
}

// addGitAuthToOpts adds auth for cloning git repos based on the parameters provided in opts.
func createAuthFromOpts(opts *cmd.Options) (transport.AuthMethod, error) {
	if opts.SSHPrivateKeyFile != "" {
		privateKey, err := readFile(opts.SSHPrivateKeyFile)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		gitURL, err := giturls.Parse(opts.Repo)
		if err != nil {
			return nil, err
		}
		auth, err := gossh.NewPublicKeys(gitURL.User.Username(), privateKey, "")
		if err != nil {
			return nil, err
		}
		if opts.KnownHostsFile != "" {
			knownHosts, err := readFile(opts.KnownHostsFile)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			knownHostsCallBack, err := createKnownHostsCallBack(knownHosts)
			if err != nil {
				return nil, err
			}
			auth.HostKeyCallback = knownHostsCallBack
		} else {
			//nolint G106: Use of ssh InsecureIgnoreHostKey should be audited
			//this will run in an init-container, so there is no persistence
			auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		}
		return auth, nil
	}

	if opts.Username != "" && opts.PasswordFile != "" {
		password, err := readFile(opts.PasswordFile)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		return &httpgit.BasicAuth{
			Username: opts.Username,
			Password: string(password),
		}, nil
	}

	return nil, nil
}

func createKnownHostsCallBack(knownHosts []byte) (ssh.HostKeyCallback, error) {
	f, err := os.CreateTemp("", "known_hosts")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(f.Name())
	defer f.Close()

	if _, err := f.Write(knownHosts); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("closing knownHosts file %s: %w", f.Name(), err)
	}

	return gossh.NewKnownHostsCallback(f.Name())
}

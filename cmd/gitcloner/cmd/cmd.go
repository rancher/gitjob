package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GitCloner interface {
	CloneRepo(opts *Options) error
}

type Options struct {
	Repo              string
	Path              string
	Branch            string
	Revison           string
	CABundleFile      string
	Username          string
	PasswordFile      string
	SSHPrivateKeyFile string
	InsecureSkipTLS   bool
	KnownHostsFile    string
}

var opts *Options

func New(gitCloner GitCloner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitcloner [REPO] [PATH]",
		Short: "Clones a git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cloneRepoWithArgs(args, gitCloner)
		},
	}
	opts = &Options{}
	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "git branch")
	cmd.Flags().StringVar(&opts.Revison, "revision", "", "git revision")
	cmd.Flags().StringVar(&opts.CABundleFile, "ca-bundle-file", "", "ca bundle file")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "user name for basic auth")
	cmd.Flags().StringVar(&opts.PasswordFile, "password-file", "", "password file for basic auth")
	cmd.Flags().StringVar(&opts.SSHPrivateKeyFile, "ssh-private-key-file", "", "ssh private key file path")
	cmd.Flags().BoolVar(&opts.InsecureSkipTLS, "insecure-skip-tls", false, "do not verify tls certificates")
	cmd.Flags().StringVar(&opts.KnownHostsFile, "known-hosts-file", "", "known hosts file")

	return cmd
}

func cloneRepoWithArgs(args []string, gitCloner GitCloner) error {
	if len(args) < 2 {
		logrus.Errorf("Usage: gitcloner [REPO] [PATH] [flags]")
		os.Exit(1)
	}
	opts.Repo = args[0]
	opts.Path = args[1]

	err := gitCloner.CloneRepo(opts)
	if err != nil {
		return err
	}

	return nil
}

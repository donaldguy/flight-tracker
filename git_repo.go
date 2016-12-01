package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
	pb "gopkg.in/cheggaaa/pb.v1"
	git "gopkg.in/libgit2/git2go.v24"
)

type GitRepo struct {
	*git.Repository
	FetchOptions *git.FetchOptions
}

func initGitRepo(opts *options) (*GitRepo, error) {
	repo := &GitRepo{}
	var err error

	if opts.RepoPath != "" {
		repo.Repository, err = git.OpenRepository(opts.RepoPath)
	} else if opts.RepoURL != "" {
		var tmpdir string
		tmpdir, err = ioutil.TempDir("", "flight-tracker-repo")
		if err != nil {
			return nil, err
		}
		expandRepoURL(opts)
		repo.FetchOptions = fetchOptions(opts)

		if terminal.IsTerminal(int(os.Stdout.Fd())) {
			repo.FetchOptions.RemoteCallbacks.TransferProgressCallback = gitProgressBarCallback()
		}

		cloneOptions := &git.CloneOptions{}
		cloneOptions.FetchOptions = repo.FetchOptions

		fmt.Printf("Cloning %s:\n", opts.RepoURL)
		repo.Repository, err = git.Clone(opts.RepoURL, tmpdir, cloneOptions)
	} else {
		return nil, fmt.Errorf("You must specify either --git-repo-url OR --git-repo-path")
	}

	return repo, err
}

func gitProgressBarCallback() func(stats git.TransferProgress) git.ErrorCode {
	var bar *pb.ProgressBar

	return func(stats git.TransferProgress) git.ErrorCode {
		if stats.ReceivedObjects >= stats.TotalObjects {
			if bar != nil {
				bar.Finish()
			}
			bar = nil
			return git.ErrOk
		}

		if bar == nil {
			bar = pb.New(int(stats.TotalObjects)).Prefix("Objects: ").SetUnits(pb.U_NO)
			bar.Start()
		}
		bar.Set(int(stats.ReceivedObjects))
		return git.ErrOk
	}
}

func fetchOptions(opts *options) *git.FetchOptions {
	fO := &git.FetchOptions{}

	if opts.SSHKey != "" {
		username := strings.Split(opts.RepoURL, "@")[0]
		fO.RemoteCallbacks = git.RemoteCallbacks{
			CredentialsCallback: func(url string, user string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
				ret, cred := git.NewCredSshKey(
					username,
					fmt.Sprintf("%s.pub", opts.SSHKey),
					opts.SSHKey,
					//TODO: support passphrases?
					"",
				)
				return git.ErrorCode(ret), &cred
			},
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				// TODO: sane host key checking?
				return git.ErrOk
			},
		}
	}
	return fO
}

var githubRepoString = regexp.MustCompile(`^\w+/\w+$`)

func expandRepoURL(opts *options) {
	if githubRepoString.MatchString(opts.RepoURL) {
		if opts.SSHKey == "" {
			opts.RepoURL = fmt.Sprintf("https://github.com/%s.git", opts.RepoURL)
		} else {
			opts.RepoURL = fmt.Sprintf("git@github.com:%s.git", opts.RepoURL)
		}
	}
	return
}

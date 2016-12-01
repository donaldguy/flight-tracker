package main

import (
	"fmt"
	"os"
	"time"

	git "gopkg.in/libgit2/git2go.v24"

	"github.com/donaldguy/flightplan"
)

func repoWatcher(repo *GitRepo, pc *PipelineClient, builds chan *Build, opts *options) {
	fmt.Println("Watching repo for changes")
	origin, err := repo.Remotes.Lookup("origin")
	if err != nil {
		panic(err)
	}

	var prevBranchHead *git.Commit
	if !opts.DoFirst {
		prevBranchHead = repo.resolveBranch(opts.Branch)
		fmt.Printf("Branch origin/%s begins at %s\n", opts.Branch, prevBranchHead.Id())
	}

	for {
		origin.Fetch([]string{}, repo.FetchOptions, "")
		newBranchHead := repo.resolveBranch(opts.Branch)
		if (opts.DoFirst && prevBranchHead == nil) || !newBranchHead.Id().Equal(prevBranchHead.Id()) {
			fmt.Printf("New Commit! %s\n", newBranchHead.Id())

			b, err := NewBuild(
				pc,
				flightplan.GitCommit{
					Repo:   repo.Repository,
					Commit: newBranchHead,
				},
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error constructing build in repo watcher: %s", err.Error())
			} else {
				builds <- b
			}

			prevBranchHead = newBranchHead
		}

		time.Sleep(time.Duration(opts.GitPollFreq))
	}
}

func (repo *GitRepo) resolveBranch(b string) *git.Commit {
	obj, err := repo.RevparseSingle("origin/" + b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repo watcher error: %s\n", err.Error())
	}
	gCommit, err := obj.AsCommit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repo watcher error: %s\n", err.Error())
	}
	return gCommit
}

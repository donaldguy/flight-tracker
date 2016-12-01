package main

import (
	"fmt"
	"os"
	"time"

	"github.com/concourse/fly/rc"
	"github.com/donaldguy/flightplan"
	flags "github.com/jessevdk/go-flags"
	git "gopkg.in/libgit2/git2go.v24"
)

type duration time.Duration

func (freq *duration) UnmarshalFlag(value string) (err error) {
	f, err := time.ParseDuration(value)
	*freq = duration(f)
	return
}

type options struct {
	Version func() `short:"v" long:"version" description:"Print the version and exit"`

	Target   rc.TargetName `short:"t" long:"target" description:"Fly target to monitor" env:"FLIGHT_TRACKER_SERVER" required:"true"`
	Pipeline string        `short:"p" long:"pipeline" description:"Name of pipeline you are tracking. Defaults to name of branch"`

	RepoPath string `short:"r" long:"git-repo-path" description:"Path to a local checkout of the git repo you are tracking" required:"true"`
	Branch   string `short:"b" long:"branch" description:"The name of the branch you are tracking" default:"master"`

	SlackToken   string `long:"slack-token" env:"FLIGHT_TRACKER_SLACK_TOKEN" required:"true" description:"Slack token"`
	SlackChannel string `short:"c" long:"slack-channel" required:"true" description:"Slack channel or username to send to"`

	PollFreq duration `short:"f" long:"polling-frequency" default:"5s"`
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	var opts options
	opts.Version = func() {
		fmt.Println("0.0.1")
		os.Exit(0)
	}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)

	args, err := parser.Parse()
	dieIf(err)

	target, err := rc.LoadTarget(opts.Target)
	dieIf(err)

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "You must provide a commit to describe")
		os.Exit(1)
	}
	if opts.Pipeline == "" {
		opts.Pipeline = opts.Branch
	}
	pipelineClient := &PipelineClient{
		Client:       target.Client(),
		Team:         target.Team(),
		PipelineName: opts.Pipeline,
	}

	repo, err := git.OpenRepository(opts.RepoPath)
	dieIf(err)
	obj, err := repo.RevparseSingle(args[0])
	dieIf(err)
	gCommit, err := obj.AsCommit()
	dieIf(err)
	commit := flightplan.GitCommit{Repo: repo, Commit: gCommit}
	dieIf(err)

	build, err := NewBuild(pipelineClient, commit)
	dieIf(err)
	err = build.Observe()
	dieIf(err)

	s := slackInit(opts.SlackToken)
	if opts.SlackChannel[0] == '#' {
		opts.SlackChannel = opts.SlackChannel[1:]
	}
	err = s.WriteBuildToChannel(build, opts.SlackChannel)
	dieIf(err)
	fmt.Printf("%v\n", build.IsDone())
}

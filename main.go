package main

import (
	"fmt"
	"os"
	"time"

	"github.com/concourse/fly/rc"
	flags "github.com/jessevdk/go-flags"
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

	RepoURL  string `short:"r" long:"git-repo-url" description:"git url or github username/repo to clone and watch"`
	RepoPath string `short:"R" long:"git-repo-path" description:"Path to a local checkout of a git repo to use"`
	SSHKey   string `short:"i" long:"ssh-private-key" description:"Path to an ssh key to use for querying git changes"`
	Branch   string `short:"b" long:"branch" description:"The name of the branch you are tracking" default:"master"`

	SlackToken   string `long:"slack-token" env:"FLIGHT_TRACKER_SLACK_TOKEN" required:"true" description:"Slack token"`
	SlackChannel string `short:"c" long:"slack-channel" required:"true" description:"Slack channel or username to send to"`

	ConcoursePollFreq duration `short:"f" long:"concourse-poll-frequency" default:"5s" description:"How frequently to poll concourse for changes"`
	GitPollFreq       duration `short:"g" long:"git-polling-frequency" default:"5s" description:"How frequently to attempt to fetch the git repo"`
	DoFirst           bool     `short:"y" description:"Trigger on the current head of the branch"`
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handleFlags() *options {
	var opts options
	opts.Version = func() {
		fmt.Println("0.0.1")
		os.Exit(0)
	}
	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err := parser.Parse()
	dieIf(err)
	return &opts
}

func initPipelineClient(opts *options) (*PipelineClient, error) {
	target, err := rc.LoadTarget(opts.Target)
	if err != nil {
		return nil, err
	}

	if opts.Pipeline == "" {
		opts.Pipeline = opts.Branch
	}
	pipelineClient := &PipelineClient{
		Client:       target.Client(),
		Team:         target.Team(),
		PipelineName: opts.Pipeline,
	}
	return pipelineClient, nil
}

func main() {
	opts := handleFlags()

	pc, err := initPipelineClient(opts)
	dieIf(err)
	_, err = pc.AuthToken()
	if err != nil {
		fmt.Printf("Not authorized with Concourse Server.\nPlease run:\n\tfly -t %s login\nand try again!\n\n",
			opts.Target)
		os.Exit(1)
	}

	repo, err := initGitRepo(opts)
	dieIf(err)

	builds := make(chan *Build)
	go repoWatcher(repo, pc, builds, opts)

	slack := slackInit(opts.SlackToken)
	if opts.SlackChannel[0] == '#' {
		opts.SlackChannel = opts.SlackChannel[1:]
	}

	for build := range builds {
		go buildWatcher(build, opts, func(b *Build) {
			err = slack.WriteBuildToChannel(b, opts.SlackChannel)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to slack: %s", err.Error())
			}
		})
	}
}

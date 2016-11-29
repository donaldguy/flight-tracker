package main

import (
	"fmt"
	"os"

	"github.com/concourse/fly/rc"
	"github.com/donaldguy/flightplan"
	"github.com/fatih/color"
	flags "github.com/jessevdk/go-flags"
	git "gopkg.in/libgit2/git2go.v24"
)

type Options struct {
	Version func() `short:"v" long:"version" description:"Print the version and exit"`

	Target   rc.TargetName `short:"t" long:"target" description:"Fly target to monitor" env:"FLIGHT_TRACKER_SERVER" required:"true"`
	RepoPath string        `short:"r" long:"git-repo-path" description:"Path to a local checkout of the git repo you are tracking" required:"true"`
	Branch   string        `short:"b" long:"branch" description:"The name of the branch you are tracking" default:"master"`
	Pipeline string        `short:"p" long:"pipeline" description:"Name of pipeline you are tracking. Defaults to name of branch"`
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	var opts Options
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

	teamClient := target.Team()
	pipelineClient := PipelineClient{
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

	bold := color.New(color.Bold).PrintfFunc()
	bold("The story of %s:\n\n", commit.Id())
	shaShort := commit.Id().String()[0:7]
	fmt.Printf("%s was written   on %s by %s\n", shaShort, commit.Author().When, commit.Author().Name)
	fmt.Printf("        and committed on %s by %s\n\n", commit.Committer().When, commit.Committer().Name)

	pipeline, err := flightplan.NewPipeline(teamClient, opts.Pipeline)
	dieIf(err)
	resources, err := commit.ResourcesTriggeredIn(pipeline)
	fmt.Printf("Thus it would come to trigger the following resources: %v\n", color.CyanString("%v", resources))

	for _, rName := range resources {
		resourceName := string(rName)
		fmt.Printf("\nOf %s:\n", color.CyanString("%s", resourceName))
		rv, err := pipelineClient.gitSha2ResourceVersion(resourceName, commit.Id().String())
		dieIf(err)

		fmt.Printf("%s", pipelineClient.describeResourceJourney(rv))
	}
}

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/concourse/atc"
	"github.com/concourse/fly/rc"
	flags "github.com/jessevdk/go-flags"
)

type duration time.Duration

func (freq *duration) UnmarshalFlag(value string) (err error) {
	f, err := time.ParseDuration(value)
	*freq = duration(f)
	return
}

type Options struct {
	Version func() `short:"v" long:"version" description:"Print the version and exit"`

	Target rc.TargetName `short:"t" long:"target" description:"Fly target to monitor" env:"FLIGHT_TRACKER_SERVER"`

	PollFreq duration `short:"f" long:"polling-frequency" default:"5s"`
}

func main() {
	var opts Options
	opts.Version = func() {
		fmt.Println("0.0.1")
		os.Exit(0)
	}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	parser.NamespaceDelimiter = "-"

	//twentythousandtonnesofcrudeoil.TheEnvironmentIsPerfectlySafe(parser, "FLIGHT_TRACKER_")

	_, err := parser.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	target, err := rc.LoadTarget(opts.Target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	apiClient := target.Client()
	newBuilds := make(chan atc.Build)
	go buildPoller(&apiClient, &newBuilds)
}

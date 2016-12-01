package main

import (
	"fmt"
	"time"
)

func buildWatcher(build *Build, opts *options, doneCallback func(*Build)) {
	fmt.Printf("Observing build for %s", build.Commit.Id())
	for !build.IsDone() {
		time.Sleep(time.Duration(opts.ConcoursePollFreq))
		build.Observe()
	}
	doneCallback(build)
}

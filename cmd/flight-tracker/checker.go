package main

import (
	"fmt"
	"time"

	"github.com/concourse/atc"
	"github.com/concourse/go-concourse/concourse"
)

func buildPoller(client *concourse.Client, newBuilds *chan atc.Build) {
	for {
		fmt.Println("poll")
		time.Sleep(5 * time.Second)
	}
}

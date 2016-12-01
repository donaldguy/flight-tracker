package main

import (
	"fmt"
	"os"

	"github.com/donaldguy/flightplan"
)

type Build struct {
	Commit            flightplan.GitCommit
	Expected          map[string]*flightplan.Graph
	Actual            *ObservedBuild
	StartingResources []string
	pc                *PipelineClient
}

func NewBuild(pc *PipelineClient, commit flightplan.GitCommit) (*Build, error) {
	pipeline, err := flightplan.NewPipeline(pc.Team, pc.PipelineName)
	if err != nil {
		return nil, err
	}
	resources, err := commit.ResourcesTriggeredIn(pipeline)
	if err != nil {
		return nil, err
	}

	expected := make(map[string]*flightplan.Graph, len(resources))
	for _, name := range resources {
		expected[name] = pipeline.GraphStartingFrom(name)
	}

	return &Build{
		Commit:            commit,
		StartingResources: resources,
		Expected:          expected,
		Actual:            nil,
		pc:                pc,
	}, nil
}

func (b *Build) Observe() (err error) {
	b.Actual, err = NewObservedBuild(b.pc, b.Commit, b.StartingResources)
	return
}

func (b *Build) IsDone() bool {
	if b.Actual == nil {
		err := b.Observe()
		if err != nil {
			panic(err)
		}
	}

	for _, resourceName := range b.StartingResources {
		for jobName := range b.Expected[resourceName].JobIndex {
			if realBuild, exists := b.Actual.BuildIndex[string(jobName)]; exists {
				if realBuild.Build.IsRunning() {
					return false
				}
			} else {
				j, _, err := b.pc.Job(b.pc.PipelineName, string(jobName))
				if err == nil && j.Paused {
					continue
				}
				if err != nil {
					fmt.Fprintf(
						os.Stderr, "Error querying %s/%s state: %s\n",
						b.pc.PipelineName,
						jobName,
						err,
					)
				}
				return false
			}
		}
	}
	return true
}

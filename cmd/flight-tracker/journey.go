package main

import (
	"fmt"
	"os"

	"github.com/bradfitz/slice"
	"github.com/concourse/atc"
	"github.com/donaldguy/flightplan"
)

type Journey struct {
	BaseURL           string
	StartingCommit    flightplan.GitCommit
	StartingResources []*ResourceSection
	BuildIndex        map[string]*BuildSection
}

type ResourceSection struct {
	Name            string
	Resource        *atc.VersionedResource
	TriggeredBuilds []*BuildSection
}

type BuildSection struct {
	Name    string
	Build   *atc.Build
	Outputs []*ResourceSection
}

func NewJourney(pc *PipelineClient, commit flightplan.GitCommit) (*Journey, error) {
	pipeline, err := flightplan.NewPipeline(pc.Team, pc.PipelineName)
	if err != nil {
		return nil, err
	}
	resources, err := commit.ResourcesTriggeredIn(pipeline)
	if err != nil {
		return nil, err
	}

	versionedResources := make([]*atc.VersionedResource, len(resources))
	for i, resourceName := range resources {
		versionedResources[i], err = pc.gitSha2ResourceVersion(resourceName, commit.Id().String())
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}

	slice.Sort(versionedResources, func(i, j int) bool {
		return versionedResources[i].Resource < versionedResources[j].Resource
	})

	j := &Journey{
		BaseURL:           pc.Client.URL(),
		StartingCommit:    commit,
		StartingResources: []*ResourceSection{},
		BuildIndex:        make(map[string]*BuildSection),
	}

	for _, rv := range versionedResources {
		if rv == nil {
			continue
		}
		j.StartingResources = append(j.StartingResources, newResourceJourney(pc, rv, j.BuildIndex))
	}

	return j, nil
}

func newResourceJourney(pc *PipelineClient, rv *atc.VersionedResource, buildMap map[string]*BuildSection) *ResourceSection {
	builds, _, err := pc.Team.BuildsWithVersionAsInput(pc.PipelineName, rv.Resource, rv.ID)
	dieIf(err)

	r := &ResourceSection{
		Name:            rv.Resource,
		Resource:        rv,
		TriggeredBuilds: []*BuildSection{},
	}

	slice.Sort(builds, func(i, j int) bool {
		return builds[i].JobName < builds[j].JobName
	})
	for i, build := range builds {
		if _, exists := buildMap[build.JobName]; exists {
			continue
		}

		b := &BuildSection{
			Name:    fmt.Sprintf("%s #%s", build.JobName, build.Name),
			Build:   &builds[i],
			Outputs: []*ResourceSection{},
		}
		r.TriggeredBuilds = append(r.TriggeredBuilds, b)
		buildMap[b.Build.JobName] = b

		rio, _, err := pc.Client.BuildResources(build.ID)
		dieIf(err)

		for _, output := range rio.Outputs {
			realOutput, err := pc.findVersion(output.Resource, output.Version)
			if err == nil {
				if realOutput.ID == rv.ID {
					continue // avoid infinite recursion
				}
			}
			b.Outputs = append(b.Outputs, newResourceJourney(pc, realOutput, buildMap))
		}
	}
	return r
}

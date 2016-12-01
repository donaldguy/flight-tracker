package main

import (
	"fmt"
	"os"

	"github.com/bradfitz/slice"
	"github.com/concourse/atc"
	"github.com/donaldguy/flightplan"
)

type ObservedBuild struct {
	BaseURL           string
	StartingResources map[string]*ResourceSection
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

func NewObservedBuild(pc *PipelineClient, commit flightplan.GitCommit, resources []string) (ob *ObservedBuild, err error) {
	versionedResources := make([]*atc.VersionedResource, len(resources))
	for i, resourceName := range resources {
		versionedResources[i], err = pc.gitSha2ResourceVersion(resourceName, commit.Id().String())
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			err = nil
		}
	}

	ob = &ObservedBuild{
		BaseURL:           pc.Client.URL(),
		StartingResources: map[string]*ResourceSection{},
		BuildIndex:        make(map[string]*BuildSection),
	}

	for _, rv := range versionedResources {
		if rv == nil {
			continue
		}
		ob.StartingResources[rv.Resource], err = newResourceObservedBuild(pc, rv, ob.BuildIndex)
		if err != nil {
			return nil, err
		}
	}

	return
}

func newResourceObservedBuild(pc *PipelineClient, rv *atc.VersionedResource, buildMap map[string]*BuildSection) (*ResourceSection, error) {
	builds, _, err := pc.Team.BuildsWithVersionAsInput(pc.PipelineName, rv.Resource, rv.ID)
	if err != nil {
		return nil, err
	}

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
		if err != nil {
			return nil, err
		}

		for _, output := range rio.Outputs {
			realOutput, err := pc.findVersion(output.Resource, output.Version)
			if err == nil {
				if realOutput.ID == rv.ID {
					continue // avoid infinite recursion
				}
			} else {
				return nil, err
			}
			rob, err := newResourceObservedBuild(pc, realOutput, buildMap)
			if err != nil {
				return nil, err
			}
			b.Outputs = append(b.Outputs, rob)
		}
	}
	return r, nil
}

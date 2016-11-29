package main

import (
	"fmt"

	"github.com/bradfitz/slice"
	"github.com/concourse/atc"
)

type ResourceSection struct {
	Name            string
	BaseURL         string
	Resource        *atc.VersionedResource
	TriggeredBuilds []*BuildSection
}

type BuildSection struct {
	Name    string
	Build   *atc.Build
	Outputs []*ResourceSection
}

func NewResourceJourney(pc *PipelineClient, rv *atc.VersionedResource) *ResourceSection {
	return newResourceJourney(pc, rv, make(map[string]*BuildSection))
}

func newResourceJourney(pc *PipelineClient, rv *atc.VersionedResource, buildMap map[string]*BuildSection) *ResourceSection {
	builds, _, err := pc.Team.BuildsWithVersionAsInput(pc.PipelineName, rv.Resource, rv.ID)
	dieIf(err)

	r := &ResourceSection{
		Name:            rv.Resource,
		BaseURL:         pc.Client.URL(),
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

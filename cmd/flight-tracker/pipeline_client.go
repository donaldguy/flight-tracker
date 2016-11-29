package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/concourse/atc"
	"github.com/concourse/go-concourse/concourse"
	"github.com/fatih/color"
)

type PipelineClient struct {
	concourse.Team
	concourse.Client
	PipelineName string
}

func (pc *PipelineClient) gitSha2ResourceVersion(resourceName, sha string) (*atc.VersionedResource, error) {
	rvs, _, any, err := pc.Team.ResourceVersions(pc.PipelineName, resourceName, concourse.Page{Limit: 10})
	if err != nil {
		return nil, err
	}

	if any {
		if rvs[0].Type != "git" {
			return nil, fmt.Errorf("Resource named %s doesn't appear to be a git repo!", resourceName)
		}
	}

	for _, rv := range rvs {
		for _, rvm := range rv.Metadata {
			if rvm.Name == "commit" && rvm.Value == sha {
				return &rv, nil
			}
		}
	}

	return nil, fmt.Errorf("No version of %s in %s found with sha %s", resourceName, pc.PipelineName, sha)
}

func (pc *PipelineClient) describeResourceJourney(rv *atc.VersionedResource) []byte {
	var b bytes.Buffer

	builds, _, err := pc.Team.BuildsWithVersionAsInput(pc.PipelineName, rv.Resource, rv.ID)
	dieIf(err)
	for _, build := range builds {
		if build.IsRunning() {
			fmt.Fprintf(&b, "it triggered %s #%s, which is still running...",
				color.YellowString("%s", build.JobName),
				build.Name,
			)
		} else {
			fmt.Fprintf(&b, "it triggered %s #%s, which %s after %s",
				color.YellowString("%s", build.JobName),
				build.Name,
				build.Status,
				(time.Duration(build.EndTime-build.StartTime) * time.Second).String(),
			)
		}

		rio, _, err := pc.Client.BuildResources(build.ID)
		dieIf(err)
		for _, input := range rio.Inputs {
			if input.Resource != rv.Resource {
				fmt.Fprintf(&b,
					"  (it also consumed a %s)\n",
					color.CyanString("%s", input.Resource),
				)
			}
		}
		for _, output := range rio.Outputs {
			fmt.Fprintf(&b,
				"  It output %s - %s:\n",
				color.CyanString("%s", output.Resource),
				output.Version,
			)
			b2 := pc.describeResourceJourney(&output)
			for _, line := range bytes.Split(b2, []byte("\n")) {
				fmt.Fprintf(&b, "  %s", line)
			}
		}

	}

	return b.Bytes()
}

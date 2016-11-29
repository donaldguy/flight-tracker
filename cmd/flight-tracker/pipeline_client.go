package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bradfitz/slice"
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

func (pc *PipelineClient) findVersion(resourceName string, ver atc.Version) (*atc.VersionedResource, error) {
	rvs, _, any, err := pc.Team.ResourceVersions(pc.PipelineName, resourceName, concourse.Page{Limit: 10})
	if err != nil {
		return nil, err
	}

	if any {
		for _, rv := range rvs {
		OUTER:
			for candidateK, candidateV := range rv.Version {
				for targetK, targetV := range ver {
					if targetK == candidateK {
						if targetV == candidateV {
							return &rv, nil
						}
						continue OUTER
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("Could not find the version with ver: %v", ver)
}

func (pc *PipelineClient) DescribeResourceJourney(rv *atc.VersionedResource) []byte {
	return pc.describeResourceJourney(rv, make(map[string]string))
}

func (pc *PipelineClient) describeResourceJourney(rv *atc.VersionedResource, alreadySaid map[string]string) []byte {
	var b bytes.Buffer

	builds, _, err := pc.Team.BuildsWithVersionAsInput(pc.PipelineName, rv.Resource, rv.ID)
	dieIf(err)

	slice.Sort(builds, func(i, j int) bool {
		return builds[i].JobName < builds[j].JobName
	})
	for _, build := range builds {
		if id, ok := alreadySaid[build.JobName]; ok && id == build.Name {
			continue
		}
		alreadySaid[build.JobName] = build.Name

		if build.IsRunning() {
			fmt.Fprintf(&b, "it triggered %s #%s, which is still running...\n",
				color.YellowString("%s", build.JobName),
				build.Name,
			)
		} else {
			fmt.Fprintf(&b, "it triggered %s, which %s after %s\n",
				fmtBuild(&build),
				fmtBuildStatus(build.Status),
				(time.Duration(build.EndTime-build.StartTime) * time.Second).String(),
			)
		}

		rio, _, err := pc.Client.BuildResources(build.ID)
		dieIf(err)
		// for _, input := range rio.Inputs {
		// 	if input.Resource != rv.Resource {
		// 		fmt.Fprintf(&b,
		// 			"  (it also consumed a %s)\n",
		// 			fmtResourceName(input.Resource),
		// 		)
		// 	}
		// }
		for _, output := range rio.Outputs {
			realOutput, err := pc.findVersion(output.Resource, output.Version)
			if err == nil {
				if realOutput.ID == rv.ID {
					continue // avoid infinite recursion
				}
			}
			fmt.Fprintf(&b,
				"  it output %s - %s",
				fmtResourceName(output.Resource),
				fmtVersion(output.Version),
			)

			b2 := pc.describeResourceJourney(realOutput, alreadySaid)
			if len(b2) > 0 {
				fmt.Fprintf(&b, ":\n")
				for _, line := range bytes.Split(b2, []byte("\n")) {
					if len(line) > 0 {
						fmt.Fprintf(&b, "    %s\n", line)
					}
				}
			} else {
				fmt.Fprint(&b, "\n")
			}
		}
	}

	return b.Bytes()
}

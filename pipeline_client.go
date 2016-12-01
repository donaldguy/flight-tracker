package main

import (
	"fmt"

	"github.com/concourse/atc"
	"github.com/concourse/go-concourse/concourse"
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

	if any && len(rvs) > 0 {
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

package main

import "github.com/donaldguy/flightplan"

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

func (b *Build) EvalStatus(j *flightplan.JobNode) (triggeredBuild *BuildSection, statusText string, doneRunning bool, shouldEvalChildren bool) {
	var seen bool
	triggeredBuild, seen = b.Actual.BuildIndex[string(j.Name)]
	if seen {
		if triggeredBuild.Build.IsRunning() {
			return triggeredBuild, "running", false, true
		} else {
			statusText = triggeredBuild.Build.Status
			switch statusText {
			case "succeeded":
				return triggeredBuild, statusText, true, true
			case "failed":
				fallthrough
			case "errored":
				fallthrough
			case "aborted":
				return triggeredBuild, statusText, true, false
			}
			return triggeredBuild, statusText, false, true
		}
	} else {
		job, _, err := b.pc.Job(b.pc.PipelineName, string(j.Name))
		if err != nil {
			return nil, "error querying", false, true
		}
		if job.Paused {
			return nil, "paused", true, false
		}
	}
	return nil, "pending", false, true
}

func (b *Build) isSubtreeDone(r *flightplan.ResourceNode) bool {
	for _, tj := range r.TriggeredJobs {
		_, _, done, shouldEvalChildren := b.EvalStatus(tj)
		if !shouldEvalChildren || len(tj.Outputs) == 0 {
			return done
		} else {
			childrenDone := done
			for _, child := range tj.Outputs {
				if !done {
					return false
				}
				childrenDone = childrenDone && b.isSubtreeDone(child)
			}
		}
	}
	return true
}

func (b *Build) IsDone() bool {
	if b.Actual == nil {
		err := b.Observe()
		if err != nil {
			panic(err)
		}
	}

	for _, resourceName := range b.StartingResources {
		graph := b.Expected[resourceName]
		if !b.isSubtreeDone(graph.Start) {
			return false
		}
	}

	return true
}

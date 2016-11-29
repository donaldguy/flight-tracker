package main

import (
	"fmt"
	"strings"

	"github.com/bluele/slack"
	"github.com/donaldguy/flightplan"
)

func phabMessageFields(commit flightplan.GitCommit) []*slack.AttachmentField {
	m := commit.Message()

	fields := make([]*slack.AttachmentField, 2)

	lines := strings.Split(m, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Reviewed By:") {
			fields[0] = &slack.AttachmentField{
				Title: "Reviewed By",
				Value: strings.Split(line, ": ")[1],
				Short: true,
			}
		}
		if strings.HasPrefix(line, "Differential Revision:") {
			url := strings.Split(line, ": ")[1]
			paths := strings.Split(url, "/")
			d := paths[len(paths)-1]
			fields[1] = &slack.AttachmentField{
				Title: "Diff",
				Value: fmt.Sprintf("<%s|%s>", url, d),
				Short: true,
			}
		}
	}
	return fields
}

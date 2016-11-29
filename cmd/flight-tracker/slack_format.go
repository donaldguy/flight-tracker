package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bluele/slack"
	"github.com/donaldguy/flightplan"
)

var githubRegex = regexp.MustCompile(`github.com(?:/|:)([^/]+/[^\.]+)`)

func slackGithubCommitLink(commit flightplan.GitCommit) string {
	origin, err := commit.Repo.Remotes.Lookup("origin")
	dieIf(err)

	matches := githubRegex.FindStringSubmatch(origin.Url())
	if len(matches) == 2 {
		return fmt.Sprintf("<https://github.com/%s/commit/%s|%s>", matches[1], commit.Id().String(), commit.Id().String())
	}
	return commit.Id().String()
}

func slackAttachmentAuthorComitter(commit flightplan.GitCommit) *slack.Attachment {
	at := &slack.Attachment{
		Color:   "#47AAB0",
		Pretext: fmt.Sprintf(":octocat: %s", slackGithubCommitLink(commit)),
		Fields: []*slack.AttachmentField{
			&slack.AttachmentField{
				Title: "Message",
				Value: strings.Split(commit.Message(), "\n")[0],
			},
			&slack.AttachmentField{
				Title: "Author",
				Value: commit.Author().Name,
				Short: true,
			},
			&slack.AttachmentField{
				Title: "Written",
				Value: commit.Author().When.Format("3:04 PM on Mon 01/02/2006"),
				Short: true,
			},
		},
	}

	if commit.Author().Name != commit.Committer().Name {
		at.Fields = append(at.Fields, []*slack.AttachmentField{
			&slack.AttachmentField{
				Title: "Comitted",
				Value: commit.Committer().When.Format("3:04 PM on Mon 01/02/2006"),
				Short: true,
			},
			&slack.AttachmentField{
				Title: "Committer",
				Value: commit.Committer().Name,
				Short: true,
			},
		}...)
	}

	return at
}

func slackStatusColor(status string) (color string) {
	switch status {
	case "succeeded":
		color = "#2CC55E"
	case "failed":
		color = "#DF342E"
	case "canceled":
		color = "#7B3922"
	case "started":
		color = "#EDBA10"
	default:
		color = "#DE6A1B"
	}
	return
}

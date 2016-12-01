package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bluele/slack"
	"github.com/donaldguy/flightplan"
)

type Slack struct {
	api *slack.Slack
}

func slackInit(token string) *Slack {
	return &Slack{
		api: slack.New(token),
	}
}

func (s *Slack) WriteBuildToChannel(build *Build, channelName string) error {
	api := s.api
	//u, err := api.FindUserByName(name)
	var channelId string
	if strings.HasPrefix(channelName, "@") {
		channelId = channelName
	} else {
		channel, err := api.FindChannelByName(channelName)
		if err != nil {
			return err
		}
		channelId = channel.Id
	}

	ats := []*slack.Attachment{
		slackAttachmentAuthorComitter(build.Commit),
	}
	ats[0].Fields = append(ats[0].Fields, phabMessageFields(build.Commit)...)

	for _, res := range build.StartingResources {
		addAttatchemntsForBuildResource(&ats, build, build.Expected[res].Start)
	}

	opts := &slack.ChatPostMessageOpt{
		AsUser:      false,
		IconEmoji:   ":airplane:",
		Username:    "flight-tracker",
		Attachments: ats,
	}

	return api.ChatPostMessage(channelId, "", opts)
}

func addAttatchemntsForBuildResource(ats *[]*slack.Attachment, build *Build, resource *flightplan.ResourceNode) {
	var title string
	var status string
	var cause string

	if resource.OutputBy == nil {
		cause = fmt.Sprintf(":package: %s", resource.Name)
	} else {
		cause = fmt.Sprintf(":outbox_tray: %s -> %s", resource.OutputBy.Name, resource.Name)
	}

	for _, tj := range resource.TriggeredJobs {
		tb, statusText, done, evalChildren := build.EvalStatus(tj)
		if tb == nil {
			title = string(tj.Name)
			status = statusText
		} else {
			title = fmt.Sprintf("<%s%s|%s>",
				build.Actual.BaseURL,
				tb.Build.URL,
				tb.Name,
			)
			if done {
				status = fmt.Sprintf("%s after %s\n",
					tb.Build.Status,
					(time.Duration(tb.Build.EndTime-tb.Build.StartTime) * time.Second).String(),
				)
			}
		}

		*ats = append(*ats, &slack.Attachment{
			Color: slackStatusColor(status),
			Title: title,
			Fields: []*slack.AttachmentField{
				&slack.AttachmentField{
					Title: "Status",
					Value: status,
					Short: true,
				},
				&slack.AttachmentField{
					Title: "Cause",
					Value: cause,
					Short: true,
				},
			},
		})
		if evalChildren {
			for _, o := range tj.Outputs {
				addAttatchemntsForBuildResource(ats, build, o)
			}
		}
	}
}

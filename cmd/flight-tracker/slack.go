package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bluele/slack"
)

type Slack struct {
	api *slack.Slack
}

func slackInit(token string) *Slack {
	return &Slack{
		api: slack.New(token),
	}
}

func (s *Slack) WriteJourneyToChannel(journey *Journey, channelName string) error {
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
		slackAttachmentAuthorComitter(journey.StartingCommit),
	}
	ats[0].Fields = append(ats[0].Fields, phabMessageFields(journey.StartingCommit)...)

	for _, res := range journey.StartingResources {
		addAttatchemntsForResourceBuild(&ats, res, journey.BaseURL)
	}

	opts := &slack.ChatPostMessageOpt{
		AsUser:      false,
		IconEmoji:   ":airplane:",
		Username:    "flight-tracker",
		Attachments: ats,
	}

	return api.ChatPostMessage(channelId, "", opts)
}

func addAttatchemntsForResourceBuild(ats *[]*slack.Attachment, build *ResourceSection, baseURL string) {
	*ats = append(*ats, &slack.Attachment{
		Title: fmt.Sprintf(":package: %s", build.Name),
	})
	var postLink string
	for _, tb := range build.TriggeredBuilds {
		if tb.Build.IsRunning() {
			postLink = ": running"
		} else {
			postLink = fmt.Sprintf(" - %s after %s\n",
				tb.Build.Status,
				(time.Duration(tb.Build.EndTime-tb.Build.StartTime) * time.Second).String(),
			)
		}

		*ats = append(*ats, &slack.Attachment{
			Color: slackStatusColor(tb.Build.Status),
			Title: fmt.Sprintf("<%s%s|%s> %s",
				baseURL,
				tb.Build.URL,
				tb.Name,
				postLink,
			),
		})
	}
}

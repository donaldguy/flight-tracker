package main

import (
	"time"

	"github.com/concourse/atc"
	"github.com/fatih/color"
)

func fmtTime(t time.Time) string {
	return t.Format("3:04 PM on Mon 01/02/2006")
}

func fmtResourceName(n string) string {
	return color.CyanString("%s", n)
}

func fmtVersion(v atc.Version) string {
	v2 := map[string]string(v)
	for _, val := range v2 {
		return val
	}
	return ""
}

func fmtBuild(b *atc.Build) string {
	return color.YellowString("%s #%s", b.JobName, b.Name)
}

func fmtBuildStatus(status string) string {
	if status == "succeeded" {
		return color.GreenString("%s", status)
	} else if status == "failed" {
		return color.RedString("%s", status)
	} else {
		return color.MagentaString("%s", status)
	}
}

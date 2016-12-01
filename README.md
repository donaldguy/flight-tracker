# flight-tracker

an intelligent, independent (polling-based) git-commit centric slack notifier for [Concourse](https://concourse.ci)

![Example output](/docs/img/example_message.png)

(Not pictured only because `jest` failed:
<br/>inclusion of jobs triggered by outputs of other jobs)

## Goals
1. Cleanly answer the question: "Did my commit break the build?"
2. Cleanly answer the question: "Have all the jobs from my commit been run?"
3. Intelligently handle a monolithic repo ("monorepo") split up into modules using the `paths` attribute of [git-resource](https://github.com/concourse/git-resource)

## Installing / Building

1. Install libgit2 0.24 (e.g. `brew install libgit2`, `apt-get install libgit2-24`)
2. `go get github.com/donaldguy/flight-tracker`

## Usage

1. `flight-tracker` borrows its auth from the local `~/.flyrc`, so before you do anything
`fly login` if needed
2. You'll also need a Slack Token which you can get from `https://<yourslackteam>.slack.com/apps/manage/custom-integrations`

after that, a minimal invocation, imaging we are tracking changes on the master branch of a public github repo named `donaldguy/flightplan`, and a pipeline named `flightplan` on target `local`, outputing to a channel called `#concourse` looks like

```
FLIGHT_TRACKER_SLACK_TOKEN=xoxb-1234567891011-1am4valid5lackt0ken \
flight-tracker -t local -r donaldguy/flightplan -p flightplan -c concourse
```
or in is long-form
````
FLIGHT_TRACKER_SLACK_TOKEN=xoxb-1234567891011-1am4valid5lackt0ken \
flight-tracker --target=local --git-repo-url donaldguy/flightplan --pipeline flightplan --slack-channel concourse
```

An example using a private repo should also include e.g. `-i ~/.ssh/id_rsa`

### concourse args

#### `-t, --target=` **(required)**
The same things you would pass to `fly`. You can use `fly login` (or copy the resulting `~/.flyrc`) to make sure you are authed

#### `-p, --pipeline` *(optional)*
The name of the pipeline to look at. If none is specified, we assume it matches the name of the branch

#### -f, --concourse-poll-frequency *(optional, default: `5s`)*
How frequently to poll for job completions, etc. Specified as expected by Go's [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration)

### git args

#### You must specify one of `-r, --git-repo-url` or `-R, --git-repo-path`.

#### `-r, --git-repo-url`
Given this `flight-tracker` will clone the repo under `$TMPDIR` with the prefix `flight-tracker-repo`.

For github repos, can just specify the `user/repo`. If you specify `-i, --ssh-private-key` will use SSH, otherwise HTTPS

For other repos, specify the whole url as e.g. `git@mycorporateserver.example.com:repo.git`

#### `-R, --git-repo-path`
Use an existing repo checkout out at the given path. Will detect changes to the `origin` remote only.

#### `-i, --ssh-private-key`  *(optional)*
Can be used with either of the above. Use an ssh key to fetch a private repo. For some reason [git2go](https://github.com/libgit2/git2go), requires you pass both a private and public key. `flight-tracker` assumes it can find the public key by adding `.pub` to the value given for private key. If this bothers you, feel free to PR.

#### `-b, --branch`  *(optional, default: `master`)*
The branch to track. Defaults to `master`. (You shouldn't write it, but there's an implicit `origin/`; you are actually watching the tracking branch)

#### `-g, --git-poll-frequency` *(optional, default: `5s`)*
How frequently to `fetch` and check for new commits on the branch specified by `-b`. Specified as expected by Go's [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration)

### slack args

#### `--slack-token=, $FLIGHT_TRACKER_SLACK_TOKEN` **(required)**
The slack token to use to auth for slack interaction. Can be generated or fetched from `https://<yourslackteam>.slack.com/apps/manage/custom-integrations`

Ideally should be specified as an env var.

#### `-c, --slack-channel` **(required)**
The slack channel to post results to. Can be a bare-name (implies a `#`), or can use a `@user` to send to a specific user (in their slackbot IM session)

### misc args

#### `-y`
Normally flight-tracker will not report the commit that is the HEAD of master when it starts. Pass this flag to have it treat that first commit as new.

This is useful if you have to quickly restart `flight-tracker` when a build is in progress

## Roadmap
Things I would like to add:

1. Have the bot edit the existing message on the fly to show pending jobs (as are already anticipated by [flightplan](https://github.com/donaldguy/flightplan) )
2. With the above, send an additional message on failure
3. With (1), optionally send a message on full success
4. Plug-ins to do some initial failure parsing / tracking

## License
Copyright (C) 2016 Donald Guy, Tulip Interfaces

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this project except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

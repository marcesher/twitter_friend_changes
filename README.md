## Skee-Lo
[I Wish](https://www.youtube.com/watch?v=ryDOy3AosBw) I didn't have to write this.

I still get bitten by twitter's "unfollow bug", where people I'm following disappear from my "friends" list.

## Intended use

This tiny app is intended to run on a schedule, and simply notifies you when your twitter "friends" list changes.

Me, well, I run this as a nightly cron job on a server that I use for other things.

## How it works

It uses the twitter API to fetch your friends. It stores them in a file.

The next time it runs, it does the same thing, and then checks to see if there were any differences.

This is lame but I like it anyways: If a user changes their name, location, or description, they'll show up in here as well. The lame part is that it will show them as being both added and deleted. I wrote this in an afternoon after mowing.

So if you follow someone new, unfollow someone intentionally or otherwise, or if one of your twitter homies changes some of their stuff, it'll tell you.

## Use it yourself

You really have to be annoyed by this if you want to read further and maybe even try to use this yourself.

Seriously, this is dependency shitshow. It's a LEWP: Least Effing Work Possible

### Get the code

#### Source
This is written in [Google Go](https://golang.org/).

If you're familiar with Go and want to build from source:

`go get github.com/marcesher/twitter_friend_changes`

In a bit, after configuration, you can `go build`, or `go run main.go`

#### Binary

Check out the Releases section of this repo for some pre-build binaries for Linux, Mac, and Windows.
You do not need the Go toolchain when using binaries.

### Initialize the config file

`cp config_sample.json config.json`

Open `config.json` in an editor. You're going to put stuff in here.

### Set up a twitter access token

Go to https://dev.twitter.com/oauth/overview/application-owner-access-tokens and set up a new app.

It'll create 4 things you'll need:

- Consumer Key
- Consumer Secret
- Access Token
- Access Token Secret

Populate the appropriate fields in `config.json` with those values

### If you want to be emailed

If you want this thing to email you, hooboy, you're in for it. 
I can't walk you through creating an AWS account, configuring it, getting keys, etc. 
Well, I could, but I won't.

So, go do that, and get yourself an AWS_ACCESS_KEY_ID and AWS_SECRET_KEY, and configure SES to be able to send email from an email address.

>>> Side note: If you can do ^^^ that, put "DevOps" on your resume and hunker down for the LinkedIn Recruiter Horde.

1. In `config.json` set the from and to email addresses
1. When you run this application, set those AWS... values as environment variables, like:

`$ AWS_ACCESS_KEY_ID="..." AWS_SECRET_KEY="..." twitter_friend_changes`


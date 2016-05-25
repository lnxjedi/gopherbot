# Gopherbot
A chatops-oriented chatbot written in Go, inspired by Hubot. Currently only has a connector for Slack.

## Development Status
Gopherbot now has a thread-safe brain, and ability to use OTP if desired for privileged commands. At some point in the coming months I will work on packaging it for easy deployment.

## Running Your Own
If you 'go get github.com/parsley42/gopherbot', you should be able to run './build.sh' to build the binary (requires go 1.6 or later for vendoring). You'll need to copy the 'example.gopherbot' to ~/.gopherbot, and edit the appropriate files to set the credentials.

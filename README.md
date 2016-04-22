# Gopherbot
A chatops-oriented chatbot written in Go, inspired by Hubot

## Quick Start
1. Download and unzip the appropriate .zip file for your platform into an install directory
2. Create a local configuration directory, and copy the conf/ directory there; optionally create plugins/external there for creating your own external plugins
3. Export GOPHER_LOCALDIR=\<your local config directory\>
4. Edit $GOPHER_LOCALDIR/conf/gopherbot.conf and replace \<your_token_here\> with your own Slack token
5. Run gopherbot with ./gopherbot.sh
6. To stop gopherbot, just pkill gopherbot

## Development Status
First scripting plugins are ready, but the codebase still needs to be cleaned up. First slack-only release expected in the next month.

## Notes on Writing Documentation for Developers

Gopherbot documentation is written in Markdown and processed by [`mdbook`](https://github.com/rust-lang/mdBook) to be published with [Github Pages](https://lnxjedi.github.io/gopherbot).

These instructions are primarily for me (David Parsley) to remind me of the few steps in setting up a dev environment for writing Gopherbot documentation, which I primarily do on a Chromebook with Linux (Crostini) installed.

1. Check the VM IP with `ip addr show dev eth0` and make sure `/etc/hosts` lists the IP ad `penguin.linux.test`
1. Download the `mdbook` binary to `$HOME/bin/mdbook`; make sure `$HOME/bin` is added to `$PATH` in `~/.bashrc`
1. Run `mdbook serve -n penguin.linux.test` from the `doc/` directory
1. View the work in progress in the browser at `http://penguin.linux.test:3000`

## CI/CD Notes

Whenever the `master` branch is updated, the pipeline automatically updates the `gh-pages` branch (if documentation changed).
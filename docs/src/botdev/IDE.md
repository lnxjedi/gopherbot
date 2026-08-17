# The Gopherbot IDE

Starting with version 2.6, the **Gopherbot** CI/CD pipelines are configured to create a pre-built development container with a daily snapshot of the most current code. This container is very large, >2G, but the base is built weekly to pick up the latest security updates, and includes the following bundled software:
* A Debian base image with Ruby and Python3 (and other requirements)
* The most recent release of [Go](https://go.dev/dl)
* The most recent release of [OpenVSCode Server](https://github.com/gitpod-io/openvscode-server)
* Pre-installed OpenVSCode language extensions for python, ruby and Go, including required modules

> The **Gopherbot** IDE was developed mainly for me - the Gopherbot developer - and for lazy[^lazy] DevOps engineers and programmers that just want to quickly get up and running. All new development and documentation after Dec '22 will be created with the IDE, but the **Gopherbot** software will remain flexible enough to support a variety of workflows and development environments. This is just my effort at providing something that definitely works.

[^lazy]: [Laziness is a Virtue](https://thethreevirtues.com/)

## Requirements

To work with the **Gopherbot IDE**, you'll need one of:
* A Linux host with docker installed (note that Podman may work but is untested)
* A MacOS host with [Docker Desktop](https://www.docker.com/products/docker-desktop/) installled
* A Windows host with the Windows Subsystem for Linux (WSL/WSL2) installed, and docker installed there

> Note that the common theme here is `docker` and `bash` - the primary script for managing the Gopherbot IDE (`cbot.sh`) is written in **bash**.

The Gopherbot IDE is designed for use with **ssh** git credentials, though it also includes the `gh` utility. It would be helpful in following the tutorials to have an ssh keypair to use for development; if you're only using the IDE for managing a single robot, this could be a special-purpose keypair that you configure as a read-write deployment key for your robot git repository. Generating and configuring read-write deployment keys is outside the scope of this manual.

## Previewing Gopherbot

If you're just previewing the **Gopherbot** software, you can just download the `cbot.sh` wrapper script and run the preview, otherwise you should skip down to [Getting Started](#getting-started).

1. Download the latest `cbot.sh` wrapper script for docker:
```
curl -o cbot.sh https://raw.githubusercontent.com/lnxjedi/gopherbot/main/cbot.sh
```
2. Make the script executable:
```
chmod +x cbot.sh
```

3. Run the preview container:
```
./cbot.sh preview
```

4. Connect to the URL, then:
   * open a terminal window in the home (`bot`) directory (or type `cd` to change to the home directory)
   * run `gopherbot` to start **Floyd**, the default robot

5. To clean up:
```
./cbot.sh preview -r
docker image rm ghcr.io/lnxjedi/gopherbot-dev:latest
```

## Getting Started

> This section will describe setting up a simple directory called `botwork` for working with you robot(s); feel free to adjust specific paths and commands according to your preferences and comfort with the command line.

For this walk-through, the commands you'll use, along with sample generated output, are shown in text boxes. You'll need to copy/paste (or type) the commands shown, modifying for your particular setup/robot. Also note that some of the commands shown may be wider than the text box, so you'll need to slide the scrollbar to see the full command if you're typing them yourself.

1. In a terminal window create a new `botwork` directory, then change to that directory:
```shell
~$ mkdir botwork
~$ cd botwork/
~/botwork$
```

2. Download the latest `cbot.sh` script, mark it executable, then run it without any arguments for help/usage:
```shell
$ curl -o cbot.sh https://raw.githubusercontent.com/lnxjedi/gopherbot/main/cbot.sh
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  4787  100  4787    0     0  13599      0 --:--:-- --:--:-- --:--:-- 13560
~/botwork$ chmod +x cbot.sh
~/botwork$ ./cbot.sh
Usage: ./botc.sh profile|dev|start|stop|remove (options...) (arguments...)
...
```
3. Generate a new profile for working on your robot; in the example command shown, you'll need to change the robot name, full name and email address (used for attributing git pushes):
```shell
$ ./cbot.sh profile -k ~/.ssh/id_rsa dolores "David Parsley" parsley@linuxjedi.org | tee dolores.env
## Lines starting with #| are used by the cbot.sh script
GIT_AUTHOR_NAME="David Parsley"
GIT_AUTHOR_EMAIL=parsley@linuxjedi.org
GIT_COMMITTER_NAME="David Parsley"
GIT_COMMITTER_EMAIL=parsley@linuxjedi.org
#|CONTAINERNAME=dolores
#|SSH_KEY_PATH=/home/david/.ssh/id_rsa
```
> Note that the ssh-key argument is optional. OpenVSCode includes a GitHub extension that can be used for publishing to GitHub, but that workflow isn't documented here.

4. Start a development container:
```shell
$ ./cbot.sh start dolores.env
Starting 'dolores-dev':
dolores
Copying /home/david/.ssh/id_rsa to dolores:/home/bot/.ssh/id_ssh ...
Access your dev environment at: http://localhost:7777/?workspace=/home/bot/gopherbot.code-workspace&tkn=XXXX
```

5. Connect to the IDE
Copy the URL provided and paste it in to your browser; note that VSCode server operates best in a separate browser tab running full-screen.

That's it! You're ready to start setting up a new robot.

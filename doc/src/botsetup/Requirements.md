# Robot Requirements

To set up your robot you'll need:
* Access to a Linux host with the **Gopherbot** software installed, or an environment where you can run and interact with the [official Gopherbot container image](https://hub.docker.com/repository/docker/lnxjedi/gopherbot)
   * If you've signed up for an account on [Gitpod](https://gitpod.io) (free accounts available), you can set up a new robot with the online [Gopherbot Gitpod Demo](https://gitpod.io/#https://github.com/lnxjedi/gopherbot)
* Credentials your robot can use to log in to your team chat; you can obtain a **Gopherbot**-compatible [Slack](https://slack.com) token here: https://\<team-name\>.slack.com/services/new/bot
* The name of a channel where your robot will run jobs by default, e.g. `clu-jobs` or `floyd-jobs`
* An empty *git* repository (public or private) to store your robot, normally *botname*-gopherbot; for example you can find **Clu** at [https://github.com/parsley42/clu-gopherbot](https://github.com/parsley42/clu-gopherbot)
* If you're using a container and/or the **setup plugin**, you'll need to be able to configure a read/write deploy key for the robot's repository - this is widely supported with almost all of the major *git* hosting services and applications, check your repository settings or consult the documentation for your particular service

The other requirements listed here are mainly items for consideration before setting up your **Gopherbot** robot.

## Git Access
**Gopherbot** version 2 integrates heavily with *git*, using *ssh* as the authentication mechanism. This guide and the setup plugin require a *git* repository that your robot can push to it with it's encrypted management ssh key (`manage_rsa`), which will be set up as a read-write deployment key. In addition to saving it's initial configuration to this repository, the standard robot configured with this guide will back up it's long-term memories to a separate `robot-state` branch.

> Note: The standard robot configured with this guide will have THREE DIFFERENT SSH KEYS, with the following uses:
> * A dedicated encrypted `manage_rsa` key, configured as a read-write deploy key for the robot's git repository; the robot will use this for saving it's initial configuration and backing up it's long-term memories from the `state/` directory
> * An unencrypted `deploy_rsa` read-only deploy key that can be used for deploying your robot to e.g. a container or new VM
> * A default encrypted `robot_rsa` key which the robot will use for all other CI/CD and remote ssh jobs

Additionally, you may want to take advantage of **Gopherbot**'s CI/CD funcationality or ability to run git-driven jobs, which can be scheduled and/or on-demand; for example, **Floyd** updates the `gh-pages` branch of [lnxjedi/gopherbot](https://github.com/lnxjedi/gopherbot) and also the [gopherbot-docker](https://github.com/lnxjedi) repository after a successful build - see [gopherbot/.gopherci/pipeline.sh](https://github.com/lnxjedi/gopherbot/blob/master/.gopherci/pipeline.sh). It's worth considering how you'll set up your robot to access *git* repositories.

### Machine Users
It's a good idea to create a machine account for your robot with the *git* service of your choice. Both [Floyd](https://github.com/floyd42) and [Clu](https://github.com/clu49) have machine accounts and belong to the [lnxjedi](https://github.com/lnxjedi) organization on [Github](https://github.com). Having an organization and adding robots to teams makes it easy to provide flexible read/write access to repositories without having to jump through repository collaborator hoops.

### Deploy Keys
*Github*, at least, allows you to associate unique ssh deploy keys with a single repository, and even grant read-write access. The limitation of one repository per key pair increases administration overhead, and makes your robot's life more difficult. Though not fully documented here, it's possible to do this with **Gopherbot** by carefully managing the `KEYNAME` and `BOT_SSH_PHRASE` parameters (environment variables). See the section on [task environment variables](../pipelines/TaskEnvironment.md) for more information on parameter precedence.

The standard setup uses a read-write deploy key because it is the easiest means of configuring your robot initially, compatible with private repositories.

### User SSH Keys
Git services also allow you to add multiple ssh keys to an individual user. It's also possible to add your robot's `robot_rsa.pub` key, allowing your robot read-write access to all the repositories you have access to. This is the least recommended means of providing *git* repository write access for your robot, but may be the most expedient.

## Brain Storage
**Gopherbot** supports the notion of long-term memories, which are technically just key-blob stores. The included `lists` and `links` plugins both use long-term memory storage.

### File backed brains
The standard configuration for a new robot uses the file-backed brain, with memories stored in `$GOPHER_HOME/state/brain`. You can schedule the included `backup` job to run periodically, and by default your robot will commit new memories as they're updated to a separate `robot-state` branch of it's configuration repository. If you want to store state in a separate repository, you can configure the `GOPHER_STATE_REPOSITORY` and optionally `GOPHER_STATE_BRANCH` for the included `backup` and `restore` jobs.

### DynamoDB brains
As of this writing, the [AWS](https://aws.amazon.com/) free tier provides a very generous 25GB of free DynamoDB storage - far more than any reasonable robot should use. See the section on [configuring the DynamoDB brain](TODO).

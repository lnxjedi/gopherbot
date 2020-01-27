# Robot Requirements

The *requirements* listed here are mainly items for consideration before setting up your **Gopherbot** robot.

## Git User
**Gopherbot** version 2 integrates heavily with *git*. A **Gopherbot** robot may frequently clone repositories to it's workspace; either public repositories via `https`, or authenticated with the robot's encrypted SSH key. There are also a number of cases where it's helpful for the robot to commit and push to repositories using SSH authentication:
* Saving the custom configuration created by the `setup` plugin in a container
* Pushing updates to dependent repositories; for instance, **Floyd** updates the `gh-pages` branch of [lnxjedi/gopherbot](https://github.com/lnxjedi/gopherbot) and updates the [gopherbot-docker](https://github.com/lnxjedi) repository after a successful build, to update documentation and container builds

### Machine Users
It's a good idea to create a machine account for your robot with the *git* service of your choice. Both [Floyd](https://github.com/floyd42) and [Clu](https://github.com/clu49) have machine accounts and belong to the [lnxjedi](https://github.com/lnxjedi) organization on [Github](https://github.com). Having an organization and adding robots to teams makes it easy to provide flexible write access to repositories without having to jump through repository collaborator hoops.

### Deploy Keys
*Github*, at least, allows you to associate unique ssh deploy keys with a single repository, and even grant read-write access. The limitation of one repository per key pair increases administration overhead, and makes your robot's life more difficult. Though not fully documented here, it's possible to do this with **Gopherbot** by carefully managing the `KEYNAME` and `BOT_SSH_PHRASE` parameters (environment variables). See the section on [task environment variables](../pipelines/TaskEnvironment.md) for more information on parameter precedence.

It's possible to fully set up a robot with two deployment keys, though it hinders the ability to push to other repositories without complex setup:
* A read-only unencrypted deploy key, normally `custom/ssh/deploy_rsa.pub`, for deploying your robot
* a read-write encypted deploy key, normally `custom/ssh/robot_rsa.pub`, that allows the robot to save it's configuration and back up state to `$GOPHER_CUSTOM_REPOSITORY`

This is the easiest means of configuring your robot initially, and can be smoothly transitioned to a machine user later on by removing the read-write deploy key and adding it as a machine user ssh key.

### User SSH Keys
Git services also allow you to add multiple ssh keys to an individual user. It's also possible to add your robot's public key, allowing your robot read-write access to all the repositories you have access to. This is the least recommended means of providing *git* repository write access for your robot, but may be the most expedient.

## Brain Storage
**Gopherbot** supports the notion of long-term memories, which are technically just key-blob stores. The included `lists` and `links` plugins both use long-term memory storage.

### File backed brains
The default configuration for a new robot uses the file-backed brain, with memories stored in `$GOPHER_HOME/state/brain`. If you create a *state* repository for your robot, you can schedule the included `backup` job to run periodically, and your robot will commit new memories as they're updated. Otherwise, you'll need to provide some means of backing up these files if you wish to preserve your file-backed long-term memories.

### DynamoDB brains
As of this writing, the [AWS](https://aws.amazon.com/) free tier provides a very generous 25GB of free DynamoDB storage - far more than any reasonable robot should use. See the section on [configuring the DynamoDB brain](TODO).

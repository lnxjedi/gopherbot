# Saving Your Robot to Git
At this point the contents of `custom/` should be committed to a git repository, and your robot should be run with `GOPHER_CUSTOM_REPOSITORY=<clone-url>`, where the `clone-url` is for use with ssh credentials. You can perform this step manually, or set up your robot's deploy keys and let the robot save it's own configuration.

## Configuring Deploy Keys
As noted in [requirements](Requirements.md#git-access), the standard robot has three ssh keypairs by default, two of which are configured as deploy keys for the repository.

Find the **deploy keys** section for the robot's git repository, then:
* Configure a **read-write** deploy key with the contents of `custom/ssh/manage_key.pub`; this corresponds to the encrypted `manage_key` the robot can use to save it's own configuration, or back up it's state / brain
* Configure a **read-only** deploy key with the contents of `custom/ssh/deploy_key.pub`; this corresponds to the unencrypted `GOPHER_DEPLOY_KEY` in the `.env` file, and the gopherbot **bootstrap** plugin can use this to deploy your robot to e.g. a generic container or new VM

Once you've set up the deploy keys, open a private chat with your robot and tell it to `save`; if all has gone well, your robot will push the contents of `custom/` to it's git repository, and should be ready to go.

This setup allows for a common workflow where the robot is running in a container, VM, or server in your infrastructure, and updates to robot configuration are made by updating the robot's configuration repository, followed by an administrator `update` command, (e.g. `clu, update`). The `update` command will cause the robot to `git pull` it's custom configuration, then perform a reload and report any errors.

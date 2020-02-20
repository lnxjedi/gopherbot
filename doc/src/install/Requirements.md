# Requirements

Since **Gopherbot** is primarily a *Go* daemon that utilizes external tools and scripts to perform most of the real work, you'll probably want to have most of the (common) listed dependencies. Note that if you deploy your robot in a container, most of these are included in the base container.

* `git` - with version 2, **Gopherbot** is tightly integrated with git for updating configuration and keeping state; **Gopherbot** requires fairly recent versions of *git* supporting `git remote get-url ...`
   * Note that the version of *git* in CentOS 7 is not supported, see the [Inline with Upstream Stable](https://ius.io/) site for a newer version
* `ssh` - most robots should have an encrypted private key / public key pair for performing *git* operations and running remote jobs
* `bash` - the majority of the *batteries included* scripts included with **Gopherbot** are written in good 'ol **Bash**; this is nearly universal but listed here for containerized deployments
* `jq` - required by the `gopherbot/lib/gopherbot_v1.sh` **bash** library for parsing the **JSON** responses from the robot
   * Note this is available from the [EPEL](https://fedoraproject.org/wiki/EPEL) repositories for CentOS 7
* `python` - (version 3) next to `bash`, the second most common language for extensions is **python** version 3, which includes several management jobs

**Optional**
* `ruby` - ruby isn't heavily used with the default extensions, but is a supported language and included the pre-built containers
* `go` - the pre-built containers all include a recent version of *Go*, which can be used for e.g. building loadable modules for *Go* plugins, jobs and tasks

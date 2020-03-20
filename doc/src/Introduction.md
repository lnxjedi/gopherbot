# Introduction

**Gopherbot** DevOps Chatbot is a tool for teams of developers, operators, infrastructure engineers and support personnel - primarily for those that are already using Slack or another team chat platform for day-to-day communication. It belongs to and integrates well with a larger family of tools including *Ansible*, *git* and *ssh*, and is able to perform many tasks similar to *Jenkins* or *TravisCI*; all of this functionality is made available to your team via the chat platform you're already using.

To help give you an idea of the kinds of tasks you can accomplish, here are a few of the things my teams have done with **Gopherbot** over the years:

* Generating and destroying [AWS](https://aws.amazon.com) instances on demand
* Running software build, test and deploy pipelines, triggered by git service integration with team chat
* Updating service status on the department website
* Allowing support personnel to search and query user attributes
* Running scheduled backups to gather artifacts over `ssh` and publish them to an artifact service
* Occasionally - generating silly memes

The primary strengths of **Gopherbot** stem from it's simplicity and flexibility. It installs and bootstraps readily on a VM or in a container with just a few environment variables, and can be run behind a firewall where it can perform tasks like rebooting server hardware over IPMI. Simple command plugins can be written in `bash` or `python`, with easy to use encrypted secrets for accomplishing privileged tasks. Like any user, the robot can also have it's own (encrypted, naturally) ssh key for performing remote work and interfacing with *git* services.

The philosophy underlying **Gopherbot** is the idea of solving the most problems with the smallest set of general purpose tools, accomplishing a wide variety of tasks reasonably well. It's true that **Gopherbot** doesn't do CI/CD to the level of, say, *Jenkins* - but it's amazing how productive I am with it. Perhaps the best example of it's flexibility can be seen in the CI/CD pipeline for **Gopherbot** itself; start reading with `pipeline.sh` in [the `.gopherci` directory](https://github.com/lnxjedi/gopherbot/tree/master/.gopherci).

The major design goals for **Gopherbot** are reliability and portability, leaning heavily on "configuration as code". Ideally, custom add-on plugins and jobs that work for a robot instance in [Slack](https://slack.com) should work just as well if your team moves, say, to [Rocket.Chat](https://rocket.chat). This goal ends up being a trade-off with supporting specialized features of a given platform, though the **Gopherbot** API enables platform-specific customizations if desired.

Secondary but important design goals are configurability and security. Individual commands can be constrained to a subset of channels and/or users, require external authorization or elevation plugins, and administrators can customize help and command matching patterns for stock plugins. **Gopherbot** has been built with security considerations in mind from the start; employing strong encryption, privilege separation, and a host of other measures to make your robot a difficult target.

Version 2 for the most part assumes that your robot will employ encryption and get it's configuration from a *git* repository. Other deployments are possible, but not well documented. This manual will focus on working with **Gopherbot** instances whose configuration is stored on [Github](https://github.com), but other *git* services are easy to use, as well.

That's it for the "marketing" portion of this manual - by now you should have an idea whether **Gopherbot** would be a good addition to your DevOps tool set.

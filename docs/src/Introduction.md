# Introduction

**Gopherbot** DevOps Chatbot is a tool for teams of developers, operators, infrastructure engineers and support personnel - primarily for those that are already using Slack or another team chat platform for day-to-day communication. It belongs to and integrates well with a larger family of tools including *Ansible*, *git* and *ssh*, and is able to perform many tasks similar to *Jenkins* or *TravisCI*; all of this functionality is made available to your team via the chat platform you're already using.

To help give you an idea of the kinds of tasks you can accomplish, here are a few of the things my teams have done with **Gopherbot** over the years:

- Generating and destroying [AWS](https://aws.amazon.com) instances on demand
- Running software build, test and deploy pipelines, triggered by git service integration with team chat
- Updating service status on the department website
- Allowing support personnel to search and query user attributes
- Running scheduled backups to gather artifacts over `ssh` and publish them to an artifact service
- Occasionally generating silly memes

The primary strengths of **Gopherbot** stem from its simplicity and flexibility. In v3, the preferred workflow is to run the engine directly on a Linux workstation while you build and test your robot locally, then deploy that same robot where it belongs later. Gopherbot still bootstraps cleanly onto servers and containers, but local development is now the default story rather than a special case.

Simple command plugins can still be written in `bash`, `python` or `ruby`, but v3 leans much harder on built-in interpreters and Go-based extensions so robots depend on fewer external tools. Like any user, the robot can also have its own encrypted `ssh` key for performing remote work and interfacing with *git* services.

The philosophy underlying **Gopherbot** is the idea of solving the most problems with the smallest set of general purpose tools, accomplishing a wide variety of tasks reasonably well. The interface is much closer to a CLI than a web GUI, but it's remarkable what can be accomplished with a shared CLI for your team's infrastructure.

The major design goals for **Gopherbot** are reliability and portability, leaning heavily on "configuration as code". Ideally, custom add-on plugins and jobs that work for a robot instance in [Slack](https://slack.com) should work just as well if your team moves, say, to [Rocket.Chat](https://rocket.chat). This goal ends up being a trade-off with supporting specialized features of a given platform, though the **Gopherbot** API enables platform-specific customizations if desired.

Secondary but important design goals are configurability and security. Individual commands can be constrained to a subset of channels and/or users, require external authorization or elevation plugins, and administrators can customize help and command matching patterns for stock plugins. **Gopherbot** has been built with security considerations in mind from the start, employing strong encryption, privilege separation, and a host of other measures to make your robot a difficult target for potential attackers.

Modern Gopherbot still assumes that serious robots keep configuration in git, and this manual continues to focus on that model. The big difference is that the day-to-day authoring loop now starts on your workstation: run the engine locally, edit the robot repo locally, reload quickly, and only then hand that robot off to a server, VM, or container.

That's it for the "marketing" portion of this manual - by now you should have an idea whether **Gopherbot** would be a good addition to your DevOps tool set.

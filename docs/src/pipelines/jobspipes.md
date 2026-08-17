# Pipelines, Jobs, and Tasks

Pipelines are the execution model behind most non-trivial Gopherbot automation.

Whenever a plugin or job starts work, the engine manages up to three related stages:

- the **primary** stage for the main work
- the **final** stage for cleanup that should always run
- the **fail** stage for failure-specific handling

## Mental model

- plugins are the interactive entry points
- jobs are scheduled or triggered entry points
- tasks are the reusable worker units inside those flows

The Robot API lets plugins and jobs build pipelines dynamically, which is why Gopherbot can express CI/CD-like behavior without forcing you into a giant static YAML pipeline language.

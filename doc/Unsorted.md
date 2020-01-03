## Job Arguments

A **Gopherbot** job can be started with an arbitrary number of space-separated arguments, either using e.g. `AddJob myjob foo bar baz` (with `bash`) in a a pipeline, or interactively in chat with `run job myjob foo bar baz`. If there are configured `Arguments:` in the `<jobname>.yaml` file, the robot will prompt for these if not given, and check supplied arguments against configured regular expressions.

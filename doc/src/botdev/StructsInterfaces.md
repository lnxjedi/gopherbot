# Important Structs and Interfaces

## The `robot` package

The `robot` package defines structs and interfaces for **Go** tasks, plugins and jobs.

### The `Robot` interface
`Robot` defines the methods available to a **Go** task, plugin or job. Whenever the engine calls a handler for one of these, the first argument to the handler is always an object that implements the `Robot` interface. Internally (in the `bot` package), this is a `bot.Robot` struct, with methods that implement the `robot.Robot` interface.

### The `Message` struct
The `GetMessage()` method on the `Robot` will return a `robot.Message`, which contains information about the user and channel, as well as a pointer to a copy of the original incoming data structure from the external connector. The complete definition is available from [godoc.org](https://godoc.org/github.com/lnxjedi/gopherbot/robot#Message).

## The `bot` package

The `bot` package contains all the logic for a running robot.

### Pipelines
Any time **Gopherbot** runs a task, plugin or job, it is part of a pipeline (which may consist only of a single job or plugin executing). The data structures for representing a pipeline are the `worker` and the `pipeContext`.

### Workers
The `worker` struct represents a thread of execution. The worker contains all the invariant information about how it was created, whether from an incoming message, a scheduled job, or a job spawn from another worker. Workers are created for every incoming message, regardless of whether a pipeline is ever started. The worker includes a `context` pointer that is populated when `startPipeline(...)` is called.

### Pipe Context
Whenever a pipeline is started, the `context` member of the `worker` struct is populated with a `pipeContext` (by `bot.registerActive`), to keep state for the pipeline. At this time four bytes of entropy are consumed for the `GOPHER_CALLER_ID` environment variable, to be passed to external scripts that run in the pipeline. This allows `bot/http.go` to look up the correct robot on each external script method call. The `pipeContext` contains a `sync.Mutex` member for locking, as it can be changed in different threads of execution. Note, however, that a well written task, plugin or job is not multi-threaded, and under normal circumstances all access to the `pipeContext` *should* be serialized.

### Robots
The `bot.Robot` struct is created any time a task, plugin or job is run, and contains a pointer to the `robot.Message`, a pointer to the `bot.worker`, and a snapshot of the current `pipeContext` when `w.makeRobot` was called.

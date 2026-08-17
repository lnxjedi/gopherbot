___

**restart-robot** - *privileged*

Usage: `AddTask restart-robot`

Allow a privileged pipeline to queue a restart of the robot after the pipeline completes. Heavily used by bootstrap and setup plugins. **NOTE:** This behavior means that tasks added after **restart-robot** are actually completed *before* the restart. In practice, the current pipeline will change the state of the robot, and after restarting a plugin **init** function will check for the state change (e.g. presence of `.restore` file) and start a new pipeline.
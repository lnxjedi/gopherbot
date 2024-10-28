// File: bot/registration.go
package bot

import (
	"log"

	"github.com/lnxjedi/gopherbot/robot"
)

// ProcessRegistrations processes the plugin, job, and task registrations collected in the robot package.
func ProcessRegistrations() {
	registrations := robot.GetRegistrations()
	if registrations == nil {
		log.Fatalf("ProcessRegistrations called multiple times or no registrations available")
	}

	// Process Plugins
	for name, handler := range registrations.Plugins {
		// Create a new Task for the plugin
		task := &Task{
			name:     name,
			taskType: taskGo,
		}

		// Create a new Plugin with the Task
		plugin := &Plugin{
			Task: task,
		}

		// Add the plugin to the current configuration
		currentCfg.addTask(plugin)

		// Add the handler to the pluginHandlers map
		pluginHandlers[name] = handler
	}

	// Process Jobs
	for name, handler := range registrations.Jobs {
		// Create a new Task for the job
		task := &Task{
			name:     name,
			taskType: taskGo,
		}

		// Create a new Job with the Task
		job := &Job{
			Task: task,
		}

		// Add the job to the current configuration
		currentCfg.addTask(job)

		// Add the handler to the jobHandlers map
		jobHandlers[name] = handler
	}

	// Process Tasks
	for name, reg := range registrations.Tasks {
		// Create a new Task for the task
		task := &Task{
			name:       name,
			taskType:   taskGo,
			Privileged: reg.Privileged,
		}

		// Add the task to the current configuration
		currentCfg.addTask(task)

		// Add the handler to the taskHandlers map
		taskHandlers[name] = reg.Handler
	}
}

// Package timetrack implements a time tracking plugin for Gopherbot
package timetrack

type trackConfig struct {
	AllowDoubleCharge bool   // whether time can be allocated to multiple tasks
	ReportingPeriod   string // "week" or "month"
	WeekStart         string // e.g. "Sunday", "Monday"
	CheckinMinutes    int    // how often the robot polls the user
	CheckinContext    string // "channel" or "direct", how the robot polls the User
	AutoClose         int    // how long to wait before automatically closing the task
	AutoCloseCharge   int    // how many minutes to charge an autoclosed task
}

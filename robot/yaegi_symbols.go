package robot

import (
	"reflect"
)

// Symbols exports the symbols from the robot package for Yaegi.
// This map allows the Yaegi interpreter to recognize and use the
// types, constants, and interfaces defined in the robot package.
var Symbols = map[string]map[string]reflect.Value{
	"github.com/lnxjedi/gopherbot/robot": {
		// ----- Types -----
		// Expose the Robot interface
		"Robot": reflect.ValueOf((*Robot)(nil)),

		// Expose structs
		"AttrRet": reflect.ValueOf(AttrRet{}),
		"Message": reflect.ValueOf(Message{}),

		// Expose type definitions
		"LogLevel":      reflect.ValueOf((*LogLevel)(nil)),
		"TaskRetVal":    reflect.ValueOf((*TaskRetVal)(nil)),
		"Protocol":      reflect.ValueOf((*Protocol)(nil)),
		"MessageFormat": reflect.ValueOf((*MessageFormat)(nil)),

		// ----- Constants -----
		// LogLevel constants
		"Trace": reflect.ValueOf(Trace),
		"Debug": reflect.ValueOf(Debug),
		"Info":  reflect.ValueOf(Info),
		"Audit": reflect.ValueOf(Audit),
		"Warn":  reflect.ValueOf(Warn),
		"Error": reflect.ValueOf(Error),
		"Fatal": reflect.ValueOf(Fatal),

		// TaskRetVal constants
		"Normal":             reflect.ValueOf(Normal),
		"Fail":               reflect.ValueOf(Fail),
		"MechanismFail":      reflect.ValueOf(MechanismFail),
		"ConfigurationError": reflect.ValueOf(ConfigurationError),
		"PipelineAborted":    reflect.ValueOf(PipelineAborted),
		"RobotStopping":      reflect.ValueOf(RobotStopping),
		"NotFound":           reflect.ValueOf(NotFound),
		"Success":            reflect.ValueOf(Success),

		// Protocol constants
		"Slack":    reflect.ValueOf(Slack),
		"Rocket":   reflect.ValueOf(Rocket),
		"Terminal": reflect.ValueOf(Terminal),
		"Test":     reflect.ValueOf(Test),
		"Null":     reflect.ValueOf(Null),

		// MessageFormat constants
		"Raw":      reflect.ValueOf(Raw),
		"Fixed":    reflect.ValueOf(Fixed),
		"Variable": reflect.ValueOf(Variable),

		// ----- Additional Symbols (If Any) -----
		// If your robot package includes functions or additional types
		// that plugins need to access, add them here similarly.
	},
}

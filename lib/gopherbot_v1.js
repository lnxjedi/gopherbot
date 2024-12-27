// gopherbot_constants.js
// This module defines constants from the Go 'robot' package for use in JavaScript scripts.

const ret = {
    // Connector Issues
    Ok: 0, // robot.Ok
    UserNotFound: 1, // robot.UserNotFound
    ChannelNotFound: 2, // robot.ChannelNotFound
    AttributeNotFound: 3, // robot.AttributeNotFound
    FailedMessageSend: 4, // robot.FailedMessageSend
    FailedChannelJoin: 5, // robot.FailedChannelJoin
  
    // Brain Maladies
    DatumNotFound: 6, // robot.DatumNotFound
    DatumLockExpired: 7, // robot.DatumLockExpired
    DataFormatError: 8, // robot.DataFormatError
    BrainFailed: 9, // robot.BrainFailed
    InvalidDatumKey: 10, // robot.InvalidDatumKey
  
    // GetTaskConfig
    InvalidConfigPointer: 11, // robot.InvalidConfigPointer
    ConfigUnmarshalError: 12, // robot.ConfigUnmarshalError
    NoConfigFound: 13, // robot.NoConfigFound
  
    // PromptForReply
    RetryPrompt: 14, // robot.RetryPrompt
    ReplyNotMatched: 15, // robot.ReplyNotMatched
    UseDefaultValue: 16, // robot.UseDefaultValue
    TimeoutExpired: 17, // robot.TimeoutExpired
    Interrupted: 18, // robot.Interrupted
    MatcherNotFound: 19, // robot.MatcherNotFound
  
    // Email
    NoUserEmail: 20, // robot.NoUserEmail
    NoBotEmail: 21, // robot.NoBotEmail
    MailError: 22, // robot.MailError
  
    // Pipeline Errors
    TaskNotFound: 23, // robot.TaskNotFound
    MissingArguments: 24, // robot.MissingArguments
    InvalidStage: 25, // robot.InvalidStage
    InvalidTaskType: 26, // robot.InvalidTaskType
    CommandNotMatched: 27, // robot.CommandNotMatched
    TaskDisabled: 28, // robot.TaskDisabled
    PrivilegeViolation: 29, // robot.PrivilegeViolation
  
    // General Failure
    Failed: 63, // robot.Failed
  };
  
  ret.string = function (val) {
    for (const k in this) {
      if (this[k] === val) {
        return k;
      }
    }
    return "UnknownRetVal";
  };
  
  const task = {
    Normal: 0, // robot.Normal
    Fail: 1, // robot.Fail
    MechanismFail: 2, // robot.MechanismFail
    ConfigurationError: 3, // robot.ConfigurationError
    PipelineAborted: 4, // robot.PipelineAborted
    RobotStopping: 5, // robot.RobotStopping
    NotFound: 6, // robot.NotFound
    Success: 7, // robot.Success
  };
  
  task.string = function (val) {
    for (const k in this) {
      if (this[k] === val) {
        return k;
      }
    }
    return "UnknownTaskRetVal";
  };
  
  const log = {
    Trace: 0, // robot.Trace
    Debug: 1, // robot.Debug
    Info: 2, // robot.Info
    Audit: 3, // robot.Audit
    Warn: 4, // robot.Warn
    Error: 5, // robot.Error
    Fatal: 6, // robot.Fatal
  };
  
  log.string = function (val) {
    for (const k in this) {
      if (this[k] === val) {
        return k;
      }
    }
    return "UnknownLogLevel";
  };
  
  const fmt = {
    Raw: 0, // robot.Raw
    Fixed: 1, // robot.Fixed
    Variable: 2, // robot.Variable
  };
  
  fmt.string = function (val) {
    for (const k in this) {
      if (this[k] === val) {
        return k;
      }
    }
    return "UnknownMessageFormat";
  };
  
  const proto = {
    Slack: 0, // robot.Slack
    Rocket: 1, // robot.Rocket
    Terminal: 2, // robot.Terminal
    Test: 3, // robot.Test
    Null: 4, // robot.Null
  };
  
  proto.string = function (val) {
    for (const k in this) {
      if (this[k] === val) {
        return k;
      }
    }
    return "UnknownProtocol";
  };
  
  module.exports = function () {
    return { ret, task, log, fmt, proto };
  };

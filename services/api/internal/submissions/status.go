package submissions

type Status string

const (
  StatusQueued             Status = "queued"
  StatusRunning            Status = "running"
  StatusAccepted           Status = "accepted"
  StatusWrongAnswer        Status = "wrong_answer"
  StatusCompileError       Status = "compile_error"
  StatusRuntimeError       Status = "runtime_error"
  StatusTimeLimitExceeded  Status = "time_limit_exceeded"
)

func IsFinal(status Status) bool {
  switch status {
  case StatusAccepted, StatusWrongAnswer, StatusCompileError, StatusRuntimeError, StatusTimeLimitExceeded:
    return true
  default:
    return false
  }
}

package submissions

import "testing"

func TestIsFinal(t *testing.T) {
  cases := []struct {
    name   string
    status Status
    want   bool
  }{
    {name: "queued is not final", status: StatusQueued, want: false},
    {name: "running is not final", status: StatusRunning, want: false},
    {name: "accepted is final", status: StatusAccepted, want: true},
    {name: "wrong answer is final", status: StatusWrongAnswer, want: true},
    {name: "compile error is final", status: StatusCompileError, want: true},
    {name: "runtime error is final", status: StatusRuntimeError, want: true},
    {name: "time limit exceeded is final", status: StatusTimeLimitExceeded, want: true},
  }

  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      if got := IsFinal(tc.status); got != tc.want {
        t.Fatalf("IsFinal(%q) = %t, want %t", tc.status, got, tc.want)
      }
    })
  }
}

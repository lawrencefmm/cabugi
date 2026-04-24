package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

type stubProcessor struct {
	results []bool
	errs    []error
	calls   int
	hook    func(int)
}

func (processor *stubProcessor) ProcessOne(context.Context) (bool, error) {
	callIndex := processor.calls
	processor.calls++
	if processor.hook != nil {
		processor.hook(callIndex)
	}

	var processed bool
	if callIndex < len(processor.results) {
		processed = processor.results[callIndex]
	}

	var err error
	if callIndex < len(processor.errs) {
		err = processor.errs[callIndex]
	}

	return processed, err
}

func TestRunOnceCommandPrintsNoQueuedSubmissions(t *testing.T) {
	processor := &stubProcessor{results: []bool{false}}
	var output bytes.Buffer

	if err := runOnceCommand(context.Background(), processor, &output); err != nil {
		t.Fatalf("runOnceCommand() error = %v", err)
	}
	if got := output.String(); got != "no queued submissions\n" {
		t.Fatalf("runOnceCommand() output = %q, want %q", got, "no queued submissions\n")
	}
}

func TestRunLoopSleepsOnEmptyQueueAndStopsCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	processor := &stubProcessor{results: []bool{false, false}, hook: func(call int) {
		if call == 1 {
			cancel()
		}
	}}
	var output bytes.Buffer

	if err := runLoop(ctx, processor, time.Millisecond, &output); err != nil {
		t.Fatalf("runLoop() error = %v", err)
	}
	if processor.calls < 2 {
		t.Fatalf("runLoop() called ProcessOne() %d times, want at least 2", processor.calls)
	}
	if output.Len() != 0 {
		t.Fatalf("runLoop() output = %q, want empty output for idle loop", output.String())
	}
}

func TestRunLoopReturnsProcessorErrors(t *testing.T) {
	processor := &stubProcessor{errs: []error{errors.New("db unavailable")}}

	err := runLoop(context.Background(), processor, time.Millisecond, &bytes.Buffer{})
	if err == nil {
		t.Fatal("runLoop() should return processor errors")
	}
}

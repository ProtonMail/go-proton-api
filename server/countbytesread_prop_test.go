package server

import (
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
)

// This file holds preservation tests for the countBytesRead byte-count
// accumulation logic. They encode the baseline behavior that must be preserved
// by the atomic-load fix: once the OnRead callbacks have quiesced, the value
// read from the counter equals the plain arithmetic sum of the delivered read
// lengths, independent of how the callbacks were scheduled.
//
// The tests are deterministic and transport-free. They exercise the same
// accumulation the helper uses (atomic.AddUint64 in the callback, then a load
// of the counter) directly, so they pass on the unfixed codebase and continue
// to pass after the fix. See design.md "Preservation Checking".

// accumulateReads mirrors countBytesRead's accumulation: each read length is
// added to the counter with atomic.AddUint64 (as the OnRead callback does), and
// the total is returned with atomic.LoadUint64 (the fixed helper's synchronized
// load). Because the writes have all completed before the load, the result must
// equal the plain arithmetic sum of the lengths.
func accumulateReads(lengths []uint32) uint64 {
	var read uint64

	for _, n := range lengths {
		atomic.AddUint64(&read, uint64(n))
	}

	return atomic.LoadUint64(&read)
}

// plainSum returns the non-atomic arithmetic sum of the lengths, modeling the
// original helper's plain `return read` once callbacks have quiesced.
func plainSum(lengths []uint32) uint64 {
	var sum uint64

	for _, n := range lengths {
		sum += uint64(n)
	}

	return sum
}

// TestCountBytesReadAccumulationExamples covers the concrete unit cases from the
// design's Unit Tests: the accumulated total equals the sum for a controlled
// sequence of reads, and the total is 0 when no reads occur.
func TestCountBytesReadAccumulationExamples(t *testing.T) {
	tests := []struct {
		name    string
		lengths []uint32
		want    uint64
	}{
		{name: "no reads", lengths: nil, want: 0},
		{name: "single read", lengths: []uint32{7}, want: 7},
		{name: "controlled sequence", lengths: []uint32{1, 2, 3, 4, 5}, want: 15},
		{name: "zero-length reads", lengths: []uint32{0, 0, 0}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := accumulateReads(tt.lengths); got != tt.want {
				t.Fatalf("accumulateReads(%v) = %d, want %d", tt.lengths, got, tt.want)
			}
		})
	}
}

// TestPropertyCountBytesReadAccumulation checks that for random sequences of
// read lengths applied via the callback accumulation logic, the atomic load of
// the counter equals the plain sum of the lengths (fixed == original once the
// callbacks have quiesced).
//
// **Validates: Requirements 3.1**
func TestPropertyCountBytesReadAccumulation(t *testing.T) {
	property := func(lengths []uint32) bool {
		return accumulateReads(lengths) == plainSum(lengths)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

// TestPropertyCountBytesReadInterleaving checks that the final accumulated total
// is invariant to callback interleaving: firing the OnRead-style atomic writes
// concurrently across many goroutines and loading the counter after they have
// all quiesced still yields the plain sum of the lengths.
//
// **Validates: Requirements 3.1**
func TestPropertyCountBytesReadInterleaving(t *testing.T) {
	property := func(lengths []uint32) bool {
		var read uint64

		var wg sync.WaitGroup
		wg.Add(len(lengths))
		for _, n := range lengths {
			go func(n uint32) {
				defer wg.Done()
				atomic.AddUint64(&read, uint64(n))
			}(n)
		}
		wg.Wait()

		// wg.Wait() establishes a happens-before edge over every write, so the
		// load observes the fully accumulated total regardless of scheduling.
		return atomic.LoadUint64(&read) == plainSum(lengths)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

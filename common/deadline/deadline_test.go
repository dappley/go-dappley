package deadline

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDeadline_IsPassed(t *testing.T) {
	deadlineInMs := time.Now().UnixNano() / NanoSecsInMilliSec
	deadline := NewDeadline(deadlineInMs)
	time.Sleep(time.Millisecond * 10)
	assert.True(t, deadline.IsPassed())

	tests := []struct {
		name          string
		deadlineInMs  int64
		sleepTimeInMs int
		expected      bool
	}{
		{
			name:          "DeadlineIsPassed",
			deadlineInMs:  time.Now().UnixNano() / NanoSecsInMilliSec,
			sleepTimeInMs: 10,
			expected:      true,
		},
		{
			name:          "DeadlineIsNotPassed",
			deadlineInMs:  time.Now().UnixNano()/NanoSecsInMilliSec + 1000,
			sleepTimeInMs: 10,
			expected:      false,
		},
		{
			name:          "NoDeadline",
			deadlineInMs:  0,
			sleepTimeInMs: 10,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deadline := NewDeadline(tt.deadlineInMs)
			time.Sleep(time.Millisecond * time.Duration(tt.sleepTimeInMs))
			assert.Equal(t, tt.expected, deadline.IsPassed())
		})
	}
}

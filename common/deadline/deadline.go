package deadline

import "time"

const NanoSecsInMilliSec = 1000000

type Deadline struct {
	deadlineInMs int64
}

//NewDeadline creates a new deadline instance
func NewDeadline(deadlineInMs int64) Deadline {
	return Deadline{
		deadlineInMs,
	}
}

//NewUnlimitedDeadline creates an unlimited deadline
func NewUnlimitedDeadline() Deadline {
	return Deadline{
		0,
	}
}

//IsPassed returns if the deadline is passed
func (d Deadline) IsPassed() bool {
	return d.deadlineInMs > 0 && time.Now().UnixNano()/NanoSecsInMilliSec >= d.deadlineInMs
}

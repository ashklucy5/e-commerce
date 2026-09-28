package metrics

import "time"

type Trace struct {
	startedAt time.Time
}

func StartTrace() Trace {
	return Trace{
		startedAt: time.Now(),
	}
}

func (t Trace) Duration() time.Duration {
	if t.startedAt.IsZero() {
		return 0
	}

	duration :=
		time.Since(
			t.startedAt,
		)

	if duration < 0 {
		return 0
	}

	return duration
}

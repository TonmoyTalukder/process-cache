package processcache

import "time"

// RealClock uses time.Now for expiration checks.
type RealClock struct{}

// Now returns the current wall-clock time.
func (RealClock) Now() time.Time {
	return time.Now()
}

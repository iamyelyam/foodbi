// Package timezone provides a single source of truth for the restaurant's
// local timezone. Uses a fixed UTC+5 offset (Kazakhstan) which is correct
// permanently — Kazakhstan does NOT observe daylight saving time.
//
// Rationale: time.LoadLocation("Asia/Almaty") requires IANA tzdata which
// isn't available on Alpine Linux containers by default. LoadLocation returns
// (nil, err) there, and any subsequent Time.In(nil) panics. Using FixedZone
// removes that fragility — works on any OS without any dependencies.
package timezone

import "time"

// offsetSeconds is UTC+5 for Kazakhstan (no DST).
const offsetSeconds = 5 * 60 * 60

// almaty is a fixed zone constructed once at package init. Safe to share
// across goroutines — *time.Location is immutable.
var almaty = time.FixedZone("Almaty", offsetSeconds)

// Almaty returns the restaurant's local timezone (UTC+5, never nil).
func Almaty() *time.Location {
	return almaty
}

// Now returns time.Now() already in Almaty local time.
func Now() time.Time {
	return time.Now().In(almaty)
}

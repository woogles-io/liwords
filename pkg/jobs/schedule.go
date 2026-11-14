package jobs

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// A Schedule says when a scheduled job runs next.
type Schedule interface {
	// Next is the first run strictly after the given time.
	Next(after time.Time) time.Time
	// String is stored in jobs.schedule. A schedule whose string changes
	// starts over from its new first run.
	String() string
}

// cronParser reads standard five-field cron expressions, with an optional
// leading seconds field.
var cronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// Cron is a schedule from a cron expression, in UTC: five fields (minute,
// hour, day of month, month, day of week), or six with seconds first.
//
//	Cron("*/10 * * * * *")  every 10 seconds, at :00, :10, :20...
//	Cron("*/30 * * * * *")  every 30 seconds
//	Cron("1 * * * *")       hourly at :01
//	Cron("0 5 * * *")       daily at 05:00 UTC
//
// Runs happen at these clock times, whichever server runs them, so a job's
// runs line up with its schedule in the logs. A run missed while no server
// was running happens once, as soon as one is, and the next is at the next
// time the expression gives. Panics on an invalid expression: schedules are
// fixed in code.
func Cron(expr string) Schedule {
	s, err := cronParser.Parse(expr)
	if err != nil {
		panic(fmt.Sprintf("jobs: bad cron expression %q: %v", expr, err))
	}
	return cronSchedule{expr: expr, s: s}
}

type cronSchedule struct {
	expr string
	s    cron.Schedule
}

func (c cronSchedule) Next(after time.Time) time.Time { return c.s.Next(after.UTC()) }
func (c cronSchedule) String() string                 { return c.expr }

// CronEvery is the cron expression for running every d, at times lined up
// with the clock. d must be a whole number of seconds dividing a minute, or
// a whole number of minutes dividing an hour.
func CronEvery(d time.Duration) (string, error) {
	switch {
	case d >= time.Second && d < time.Minute && d%time.Second == 0 && time.Minute%d == 0:
		return fmt.Sprintf("*/%d * * * * *", int(d/time.Second)), nil
	case d >= time.Minute && d <= time.Hour && d%time.Minute == 0 && time.Hour%d == 0:
		return fmt.Sprintf("0 */%d * * * *", int(d/time.Minute)), nil
	}
	return "", fmt.Errorf("jobs: %v can't be a cron interval: use whole seconds dividing a minute, or whole minutes dividing an hour", d)
}

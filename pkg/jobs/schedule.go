package jobs

import (
	"fmt"
	"time"
)

// A Schedule says when a scheduled job runs next. Times are in UTC.
type Schedule interface {
	// Next is the first run strictly after the given time.
	Next(after time.Time) time.Time
	// String is stored in jobs.schedule. A schedule whose string changes
	// starts over from its new first run.
	String() string
}

// Every runs a job d after its previous run ended.
func Every(d time.Duration) Schedule {
	if d <= 0 {
		panic("jobs: Every needs a positive duration")
	}
	return every(d)
}

type every time.Duration

func (e every) Next(after time.Time) time.Time { return after.Add(time.Duration(e)) }
func (e every) String() string                 { return "@every " + time.Duration(e).String() }

// Hourly runs a job each hour at the given minute.
func Hourly(minute int) Schedule {
	if minute < 0 || minute > 59 {
		panic("jobs: Hourly minute out of range")
	}
	return hourly(minute)
}

type hourly int

func (h hourly) Next(after time.Time) time.Time {
	after = after.UTC()
	t := after.Truncate(time.Hour).Add(time.Duration(h) * time.Minute)
	if !t.After(after) {
		t = t.Add(time.Hour)
	}
	return t
}

func (h hourly) String() string { return fmt.Sprintf("@hourly :%02d", int(h)) }

// Daily runs a job each day at the given UTC time.
func Daily(hour, minute int) Schedule {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		panic("jobs: Daily time out of range")
	}
	return daily{hour, minute}
}

type daily struct{ hour, minute int }

func (d daily) Next(after time.Time) time.Time {
	after = after.UTC()
	t := time.Date(after.Year(), after.Month(), after.Day(), d.hour, d.minute, 0, 0, time.UTC)
	if !t.After(after) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

func (d daily) String() string { return fmt.Sprintf("@daily %02d:%02d", d.hour, d.minute) }

package main

import (
	"testing"

	"github.com/woogles-io/liwords/pkg/jobs"
	"github.com/woogles-io/liwords/pkg/vdowebhook"
)

// The API's schedules parse (jobs.Cron panics otherwise), with the VDO
// interval production uses.
func TestRegisterJobs(t *testing.T) {
	r := jobs.NewRunner(nil, jobs.Options{})
	if err := registerJobs(r, jobDeps{vdo: vdowebhook.NewVDOWebhookService(nil, 300)}); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterJobsRejectsUnalignedVDOInterval(t *testing.T) {
	r := jobs.NewRunner(nil, jobs.Options{})
	if err := registerJobs(r, jobDeps{vdo: vdowebhook.NewVDOWebhookService(nil, 90)}); err == nil {
		t.Fatal("a 90s VDO interval was accepted")
	}
}

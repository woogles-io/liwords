package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// fakeTargetHealth answers with states in turn, an empty state being an
// error, and then with the last one.
type fakeTargetHealth struct {
	states []elbtypes.TargetHealthStateEnum
	calls  int
	asked  *elb.DescribeTargetHealthInput
}

func (f *fakeTargetHealth) DescribeTargetHealth(_ context.Context, in *elb.DescribeTargetHealthInput,
	_ ...func(*elb.Options)) (*elb.DescribeTargetHealthOutput, error) {
	f.asked = in
	s := f.states[min(f.calls, len(f.states)-1)]
	f.calls++
	if s == "" {
		return nil, errors.New("AccessDenied")
	}
	return &elb.DescribeTargetHealthOutput{TargetHealthDescriptions: []elbtypes.TargetHealthDescription{
		{Target: &f.asked.Targets[0], TargetHealth: &elbtypes.TargetHealth{State: s}},
	}}, nil
}

func TestWatchForDraining(t *testing.T) {
	api := &fakeTargetHealth{states: []elbtypes.TargetHealthStateEnum{
		elbtypes.TargetHealthStateEnumInitial,
		elbtypes.TargetHealthStateEnumHealthy,
		"", // an error doesn't stop the watch
		elbtypes.TargetHealthStateEnumHealthy,
		elbtypes.TargetHealthStateEnumDraining,
	}}
	target := elbtypes.TargetDescription{Id: aws.String("10.0.1.2"), Port: aws.Int32(8087)}
	drained := 0
	done := make(chan struct{})
	go func() {
		watchForDraining(context.Background(), api, "arn:tg", target, time.Millisecond, func() { drained++ })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("watch didn't end after the target started draining")
	}
	if drained != 1 {
		t.Fatalf("onDraining called %d times, want 1", drained)
	}
	if api.calls != 5 {
		t.Fatalf("polled %d times, want 5", api.calls)
	}
	if aws.ToString(api.asked.TargetGroupArn) != "arn:tg" || aws.ToString(api.asked.Targets[0].Id) != "10.0.1.2" ||
		aws.ToInt32(api.asked.Targets[0].Port) != 8087 {
		t.Fatalf("asked about %+v", api.asked)
	}
}

func TestWatchForDrainingStopsWithContext(t *testing.T) {
	api := &fakeTargetHealth{states: []elbtypes.TargetHealthStateEnum{elbtypes.TargetHealthStateEnumHealthy}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		watchForDraining(ctx, api, "arn:tg", elbtypes.TargetDescription{Id: aws.String("ip")},
			time.Hour, func() { t.Error("drained while healthy") })
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("watch didn't stop")
	}
}

func TestTaskIP(t *testing.T) {
	// Trimmed from a real container metadata response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"DockerId":"abc","Name":"liwords-socket","Networks":[{"NetworkMode":"awsvpc","IPv4Addresses":["10.0.1.2"],"IPv4SubnetCIDRBlock":"10.0.1.0/24"}]}`))
	}))
	defer srv.Close()
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", srv.URL)
	ip, err := taskIP(context.Background())
	if err != nil || ip != "10.0.1.2" {
		t.Fatalf("taskIP = %q, %v", ip, err)
	}

	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", "")
	if _, err := taskIP(context.Background()); err == nil {
		t.Fatal("expected an error off ECS")
	}
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/rs/zerolog/log"
)

// ECS deregisters a task from its load balancer, waits for its connections
// to drain, and only then sends it SIGTERM: by then the load balancer has cut
// every socket. So the server watches its own target instead, and starts
// closing its sockets as soon as the target is draining.

const targetHealthPollPeriod = 5 * time.Second

// targetHealthErrorLogPeriod limits how often a failing poll is logged: one
// that fails for lack of permission fails every time.
const targetHealthErrorLogPeriod = time.Minute

type targetHealthAPI interface {
	DescribeTargetHealth(context.Context, *elb.DescribeTargetHealthInput, ...func(*elb.Options)) (*elb.DescribeTargetHealthOutput, error)
}

// watchForDraining polls the health of target in targetGroupARN and calls
// onDraining once, when the load balancer starts draining it.
func watchForDraining(ctx context.Context, api targetHealthAPI, targetGroupARN string,
	target elbtypes.TargetDescription, poll time.Duration, onDraining func()) {

	t := time.NewTicker(poll)
	defer t.Stop()
	var lastErrorLogged time.Time
	for {
		out, err := api.DescribeTargetHealth(ctx, &elb.DescribeTargetHealthInput{
			TargetGroupArn: aws.String(targetGroupARN),
			Targets:        []elbtypes.TargetDescription{target},
		})
		if err != nil {
			if time.Since(lastErrorLogged) >= targetHealthErrorLogPeriod {
				log.Err(err).Msg("describe-target-health")
				lastErrorLogged = time.Now()
			}
		} else {
			for _, d := range out.TargetHealthDescriptions {
				if d.TargetHealth != nil && d.TargetHealth.State == elbtypes.TargetHealthStateEnumDraining {
					log.Info().Str("target", aws.ToString(target.Id)).Msg("target-draining")
					onDraining()
					return
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// taskIP returns this ECS task's private IP, which is its target ID in an ip
// target group, from the task metadata endpoint.
func taskIP(ctx context.Context) (string, error) {
	uri := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	if uri == "" {
		return "", errors.New("not running on ECS: ECS_CONTAINER_METADATA_URI_V4 is unset")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("task metadata: %s", resp.Status)
	}
	var meta struct {
		Networks []struct {
			IPv4Addresses []string
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}
	for _, n := range meta.Networks {
		for _, ip := range n.IPv4Addresses {
			return ip, nil
		}
	}
	return "", errors.New("task metadata has no IPv4 address")
}

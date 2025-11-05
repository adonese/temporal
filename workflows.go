package iplocate

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type data struct {
	result string
}

func GetAddressFromIP(ctx workflow.Context, name string) (data, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			MaximumInterval:    time.Minute,
			BackoffCoefficient: 2,
		},
	}
	var ipActivities *IPActivities
	ctx = workflow.WithActivityOptions(ctx, ao)

	var ip string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
	if err != nil {
		return data{}, fmt.Errorf("failed to get ip: %s", err)
	}
	var location string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		return data{}, fmt.Errorf("failed to get location: %s", err)
	}
	return data{location}, nil
}

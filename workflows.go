package iplocate

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func GetAddressFromIP(ctx workflow.Context, name string) (string, error) {
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

	workflow.GetLogger(ctx).Info("Version 1: Starting workflow - will fetch IP, wait, then get location")

	var ip string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
	if err != nil {
		return "", fmt.Errorf("failed to get ip: %s", err)
	}
	workflow.GetLogger(ctx).Info("IP fetched", "ip", ip)
	// Sleep for 45 seconds to give us time to modify code while workflow is running
	workflow.GetLogger(ctx).Info("Sleeping for 45 seconds... (this is when you'll modify the code)")
	workflow.Sleep(ctx, 45*time.Second)
	workflow.GetLogger(ctx).Info("Awake! Now fetching location...")

	var location string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		return "", fmt.Errorf("failed to get location: %s", err)
	}

	return location, nil
}

func GetAddressFromIPV2(ctx workflow.Context, name string) (Data, error) {
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

	workflow.GetLogger(ctx).Info("Version 1: Starting workflow - will fetch IP, wait, then get location")

	var ip string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
	if err != nil {
		return Data{}, fmt.Errorf("failed to get ip: %s", err)
	}
	workflow.GetLogger(ctx).Info("IP fetched", "ip", ip)

	var recordedIp string
	err = workflow.ExecuteActivity(ctx, ipActivities.RecordLookup, ip).Get(ctx, &recordedIp)
	if err != nil {
		return Data{}, fmt.Errorf("failed to record lookup: %s", err)
	}

	// Sleep for 45 seconds to give us time to modify code while workflow is running
	workflow.GetLogger(ctx).Info("Sleeping for 5 seconds... (this is when you'll modify the code)")
	workflow.Sleep(ctx, 30*time.Second)
	workflow.GetLogger(ctx).Info("Awake! Now fetching location...")

	var location string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		return Data{}, fmt.Errorf("failed to get location: %s", err)
	}

	var zone string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetTimeZone, ip).Get(ctx, &zone)
	if err != nil {
		return Data{}, fmt.Errorf("failed to get timezone: %s", err)
	}

	return Data{
		Result:   ip,
		Location: location,
		Zone:     zone,
	}, nil

}

type Data struct {
	Result   string
	Location string
	Zone     string
}

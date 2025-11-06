package iplocate

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// VERSIONING STRATEGY:
// We need to support both old and new workflows:
// - Old: returns lowercase 'data' struct with only 'result'
// - New: returns uppercase 'Data' struct with 'result' and 'timezone'

// Old struct (keep for compatibility)
type data struct {
	result string
}

// New struct
type Data struct {
	Result   string // Made public (uppercase) if you want to export it
	Timezone string
}

func GetAddressFromIP(ctx workflow.Context, name string) (interface{}, error) {
	// Note: Return type is now interface{} to support both data and Data
	// This allows old workflows (returning data) and new workflows (returning Data) to coexist

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

	// VERSION CHECK: This determines which version of the workflow we're running
	version := workflow.GetVersion(ctx, "add-timezone-and-change-return-type", workflow.DefaultVersion, 1)

	if version == workflow.DefaultVersion {
		// OLD VERSION: Original workflow logic
		workflow.GetLogger(ctx).Info("DefaultVersion: Running original workflow (no timezone)")

		var ip string
		err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
		if err != nil {
			return data{}, fmt.Errorf("failed to get ip: %s", err)
		}

		workflow.GetLogger(ctx).Info("Sleeping for 45 seconds...")
		workflow.Sleep(ctx, 45*time.Second)

		var location string
		err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
		if err != nil {
			return data{}, fmt.Errorf("failed to get location: %s", err)
		}

		// Return old struct type
		return data{result: location}, nil

	} else {
		// NEW VERSION: With timezone fetch
		workflow.GetLogger(ctx).Info("Version 1: Running new workflow (with timezone)")

		var ip string
		err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
		if err != nil {
			return Data{}, fmt.Errorf("failed to get ip: %s", err)
		}
		workflow.GetLogger(ctx).Info("IP fetched", "ip", ip)

		workflow.GetLogger(ctx).Info("Sleeping for 45 seconds...")
		workflow.Sleep(ctx, 45*time.Second)
		workflow.GetLogger(ctx).Info("Awake! Now fetching location and timezone...")

		var location string
		err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
		if err != nil {
			return Data{}, fmt.Errorf("failed to get location: %s", err)
		}

		var timezone string
		err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
		if err != nil {
			return Data{}, fmt.Errorf("failed to get timezone: %s", err)
		}

		// Return new struct type
		return Data{Result: location, Timezone: timezone}, nil
	}
}

// NEW WORKFLOW: This is safe! It doesn't affect existing workflows
func GetZoneFromIP(ctx workflow.Context, ip string) (Data, error) {
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

	workflow.GetLogger(ctx).Info("GetZoneFromIP: Fetching timezone for IP", "ip", ip)

	var timezone string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
	if err != nil {
		return Data{}, fmt.Errorf("failed to get timezone: %s", err)
	}

	return Data{Timezone: timezone}, nil
}

// ALTERNATIVE APPROACH: Keep return types separate (cleaner but requires two workflow functions)
// This is actually better practice - don't change existing workflows, create new ones

func GetAddressFromIPV1(ctx workflow.Context, name string) (data, error) {
	// Original workflow - frozen, never changes
	// ... original code
	return data{}, nil
}

func GetAddressFromIPV2(ctx workflow.Context, name string) (Data, error) {
	// New workflow with timezone
	// ... new code with timezone
	return Data{}, nil
}

// Then in starter, choose which workflow to use:
// c.ExecuteWorkflow(ctx, options, iplocate.GetAddressFromIPV2, "")

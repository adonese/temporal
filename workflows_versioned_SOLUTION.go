package iplocate

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// This is the SOLUTION file showing proper workflow versioning
// Use this as reference after you've experienced the non-deterministic error

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

	workflow.GetLogger(ctx).Info("Starting workflow with versioning support")

	// Step 1: Fetch IP (same for all versions)
	var ip string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
	if err != nil {
		return data{}, fmt.Errorf("failed to get ip: %s", err)
	}
	workflow.GetLogger(ctx).Info("IP fetched", "ip", ip)

	// Step 2: Sleep (same for all versions)
	workflow.GetLogger(ctx).Info("Sleeping for 45 seconds...")
	workflow.Sleep(ctx, 45*time.Second)
	workflow.GetLogger(ctx).Info("Awake!")

	// Step 3: VERSION CHECK - This is where the magic happens!
	// changeID: "add-timezone-feature" - unique identifier for this change
	// minSupported: workflow.DefaultVersion - we still support workflows without this feature
	// maxSupported: 1 - this is version 1 of this feature
	version := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 1)

	var timezone string
	if version == 1 {
		// VERSION 1: New workflows get timezone info
		workflow.GetLogger(ctx).Info("Version 1: Fetching timezone (new feature)")
		err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
		if err != nil {
			return data{}, fmt.Errorf("failed to get timezone: %s", err)
		}
		workflow.GetLogger(ctx).Info("Timezone fetched", "timezone", timezone)
	} else {
		// DefaultVersion: Old workflows skip timezone (they were started before this feature existed)
		workflow.GetLogger(ctx).Info("DefaultVersion: Skipping timezone (old workflow)")
	}

	// Step 4: Fetch location (same for all versions)
	workflow.GetLogger(ctx).Info("Fetching location...")
	var location string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		return data{}, fmt.Errorf("failed to get location: %s", err)
	}

	// Step 5: Return result (version-aware)
	result := location
	if version == 1 && timezone != "" {
		result = fmt.Sprintf("%s, Timezone: %s", location, timezone)
	}

	workflow.GetLogger(ctx).Info("Workflow completed successfully", "result", result)
	return data{result}, nil
}

// EXAMPLE 2: Adding a second change (version 2)
// After all version 0 workflows complete, you might add another feature:
//
// version := workflow.GetVersion(ctx, "add-timezone-feature", 1, 2)
//
// if version >= 1 {
//     // fetch timezone (now required for versions 1 and 2)
// }
//
// if version == 2 {
//     // NEW: fetch weather info
// }

// EXAMPLE 3: Version lifecycle (months later, after all DefaultVersion workflows finish)
//
// version := workflow.GetVersion(ctx, "add-timezone-feature", 1, 2)
// // minSupported changed from DefaultVersion to 1
// // This means we no longer support workflows without timezone
// // But we MUST keep the GetVersion call forever (for replay)

package iplocate

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowResult is designed for evolution
// - All fields use json tags for stable serialization
// - Optional fields use omitempty
// - Can add new fields without breaking existing workflows
type WorkflowResult struct {
	Location string `json:"location"`
	Timezone string `json:"timezone,omitempty"` // Optional field from day 1
}

// GetAddressFromIPClean is the PROPER way to do versioning
// - Return type never changes
// - GetVersion controls execution path
// - Old and new workflows coexist safely
func GetAddressFromIPClean(ctx workflow.Context, name string) (WorkflowResult, error) {
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

	workflow.GetLogger(ctx).Info("Starting clean versioned workflow")

	// Step 1: Fetch IP (same for all versions)
	var ip string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("failed to get ip: %s", err)
	}
	workflow.GetLogger(ctx).Info("IP fetched", "ip", ip)

	// Step 2: Sleep (gives us time to change code during exercise)
	workflow.GetLogger(ctx).Info("Sleeping for 45 seconds...")
	workflow.Sleep(ctx, 45*time.Second)
	workflow.GetLogger(ctx).Info("Awake!")

	// Step 3: VERSION CHECK - The proper way
	// This determines execution path, NOT return type
	version := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 1)

	// Step 4: Fetch location (same for all versions)
	var location string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("failed to get location: %s", err)
	}

	// Build result with required fields
	result := WorkflowResult{
		Location: location,
	}

	// Step 5: Conditionally fetch timezone (only version 1)
	if version == 1 {
		workflow.GetLogger(ctx).Info("Version 1: Fetching timezone")
		var timezone string
		err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
		if err != nil {
			return WorkflowResult{}, fmt.Errorf("failed to get timezone: %s", err)
		}
		result.Timezone = timezone
	} else {
		workflow.GetLogger(ctx).Info("DefaultVersion: Skipping timezone")
		// result.Timezone stays empty (omitempty in JSON)
	}

	workflow.GetLogger(ctx).Info("Workflow completed", "hasTimezone", result.Timezone != "")
	return result, nil
}

// ALTERNATIVE: Create a completely new workflow (even cleaner)
// This is the approach used in production for major changes

// GetAddressFromIPV1 - Original workflow, frozen forever
func GetAddressFromIPV1(ctx workflow.Context, name string) (WorkflowResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	var ipActivities *IPActivities
	ctx = workflow.WithActivityOptions(ctx, ao)

	var ip string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
	if err != nil {
		return WorkflowResult{}, err
	}

	var location string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		return WorkflowResult{}, err
	}

	return WorkflowResult{Location: location}, nil
}

// GetAddressFromIPV2 - New workflow with timezone
// Separate function = no versioning complexity
func GetAddressFromIPV2(ctx workflow.Context, name string) (WorkflowResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	var ipActivities *IPActivities
	ctx = workflow.WithActivityOptions(ctx, ao)

	var ip string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
	if err != nil {
		return WorkflowResult{}, err
	}

	// V2: Always fetch both location and timezone
	var location string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		return WorkflowResult{}, err
	}

	var timezone string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
	if err != nil {
		return WorkflowResult{}, err
	}

	return WorkflowResult{
		Location: location,
		Timezone: timezone,
	}, nil
}

// COMPARISON:
//
// GetVersion Approach (GetAddressFromIPClean):
// ✅ Single workflow name
// ✅ Automatic version detection
// ❌ More complex code
// ❌ Versioning logic forever
//
// Separate Workflows Approach (V1/V2):
// ✅ Simple, clean code
// ✅ No versioning logic
// ✅ Easy to understand
// ❌ Must manage multiple workflow names
// ❌ Callers must choose version
//
// Production recommendation: Use separate workflows for major changes,
// use GetVersion for minor changes within a workflow.

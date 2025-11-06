package iplocate

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// IPLookupWorkflow - The workflow being observed
// Slowly fetches IP location, exposing its progress via queries
func IPLookupWorkflow(ctx workflow.Context, ip string) (string, error) {
	logger := workflow.GetLogger(ctx)

	// State that can be queried
	status := "starting"
	result := ""

	// QUERY HANDLER: Let others ask "what's your status?"
	workflow.SetQueryHandler(ctx, "status", func() string {
		return status
	})

	workflow.SetQueryHandler(ctx, "result", func() string {
		return result
	})

	// Slow process - plenty of time to query it
	logger.Info("Starting IP lookup", "ip", ip)
	status = "fetching location"
	workflow.Sleep(ctx, 10 * time.Second)

	var ipActivities *IPActivities
	var location string
	err := workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
	if err != nil {
		status = "failed"
		return "", err
	}

	status = "complete"
	result = location
	logger.Info("Lookup complete", "location", location)

	// Stay alive a bit so we can query the result
	workflow.Sleep(ctx, 20 * time.Second)

	return location, nil
}

// StatusCheckerWorkflow - The observer workflow
// THIS IS THE MAGIC: One workflow querying another workflow!
func StatusCheckerWorkflow(ctx workflow.Context, targetWorkflowID string) (string, error) {
	logger := workflow.GetLogger(ctx)

	logger.Info("Querying target workflow", "target", targetWorkflowID)

	// THE AHA MOMENT: Query another workflow from inside a workflow!
	statusQuery := workflow.QueryExternalWorkflow(ctx, targetWorkflowID, "", "status")

	var status string
	err := statusQuery.Get(&status)
	if err != nil {
		return "", fmt.Errorf("failed to query workflow: %w", err)
	}

	logger.Info("Received status from target", "status", status)

	// Try to get the result too
	resultQuery := workflow.QueryExternalWorkflow(ctx, targetWorkflowID, "", "result")
	var result string
	resultQuery.Get(&result) // Might be empty if not done yet

	summary := fmt.Sprintf("Workflow %s is '%s'", targetWorkflowID, status)
	if result != "" {
		summary += fmt.Sprintf(" - Result: %s", result)
	}

	return summary, nil
}

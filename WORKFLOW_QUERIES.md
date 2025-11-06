# The "Aha Moment": One Workflow Querying Another

## The Elegant Insight

**Workflows can query other workflows.**

No database. No shared memory. No message queue.
Pure Temporal orchestration.

## The Setup (3 terminals)

```bash
# Terminal 1: Temporal server
temporal server start-dev

# Terminal 2: Worker
go run worker/main.go

# Terminal 3a: Start the observable workflow
go run starter/start_lookup.go

# Terminal 3b: While it's running, query it from another workflow
go run starter/check_status.go
```

## What Happens

### Workflow A: IPLookupWorkflow
```go
func IPLookupWorkflow(ctx workflow.Context, ip string) (string, error) {
    status := "starting"

    // Expose query handler
    workflow.SetQueryHandler(ctx, "status", func() string {
        return status
    })

    status = "fetching location"
    // ... slow work for 30 seconds ...
    status = "complete"
}
```

### Workflow B: StatusCheckerWorkflow
```go
func StatusCheckerWorkflow(ctx workflow.Context, targetWorkflowID string) (string, error) {
    // THE MAGIC: Query another workflow!
    statusQuery := workflow.QueryExternalWorkflow(ctx, targetWorkflowID, "", "status")

    var status string
    statusQuery.Get(&status)

    return fmt.Sprintf("Workflow %s is '%s'", targetWorkflowID, status)
}
```

## The Output

```
═══════════════════════════════════════════════
  RESULT FROM QUERYING ANOTHER WORKFLOW:
  Workflow ip-lookup-observable is 'fetching location'
═══════════════════════════════════════════════

✓ One workflow just queried another workflow!
```

## Why This Matters

**Before:** To coordinate workflows, you'd need:
- Shared database
- Message queue
- Polling
- Complex state management

**After:** Just query it directly.

```go
// Need to know if Workflow A is done?
result := workflow.QueryExternalWorkflow(ctx, "workflow-a", "", "is-done")

// Need Workflow A's intermediate results?
result := workflow.QueryExternalWorkflow(ctx, "workflow-a", "", "current-count")
```

## Real-World Use Cases

1. **Dashboard Workflow**: Queries multiple data-processing workflows for progress
2. **Circuit Breaker**: Check if dependent workflow is healthy before proceeding
3. **Saga Coordinator**: Query sub-workflows before deciding to compensate
4. **Monitoring**: One workflow checks health of a fleet of worker workflows

## The Code

- **Workflow A**: `simple_workflows.go:11` (IPLookupWorkflow)
- **Workflow B**: `simple_workflows.go:46` (StatusCheckerWorkflow)
- **Starters**: `starter/start_lookup.go` and `starter/check_status.go`

**Total lines:** ~70 (vs the bloated 200+ before)

## Try This

Run Workflow A, then:
- Query from Workflow B ✓
- Query from CLI: `temporal workflow query --workflow-id ip-lookup-observable --name status`
- Query from Go client: `c.QueryWorkflow(ctx, "ip-lookup-observable", "", "status")`

All three ways work. That's the elegance.

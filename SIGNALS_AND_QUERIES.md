# Temporal Signals and Queries - Learning Guide

This document explains the **Signals and Queries** implementation in our IP Monitor workflow - two powerful Temporal features for dynamic workflow control.

## What Are Signals and Queries?

### Signals 📨
**Signals** send data TO a running workflow to change its behavior dynamically.
- **Mutable**: They modify workflow state
- **Asynchronous**: Fire-and-forget, don't wait for response
- **Durable**: Recorded in workflow history (survives worker crashes)
- **Use cases**: Pause/resume, update parameters, cancel operations

### Queries 🔍
**Queries** read data FROM a running workflow without side effects.
- **Read-only**: Cannot modify workflow state
- **Synchronous**: Return immediate results
- **Not recorded**: Don't appear in workflow history
- **Use cases**: Get status, check progress, inspect state

## Our Implementation: IP Monitor Workflow

### The Scenario
A **long-running workflow** that monitors an IP address periodically and reports its geolocation. You can control it dynamically without restarting!

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    IPMonitorWorkflow                        │
│                                                             │
│  State:                                                     │
│  • currentIP, checkInterval, isPaused, shouldStop          │
│  • totalChecks, history[], lastResult                       │
│                                                             │
│  Signals (write):              Queries (read):             │
│  • pause                       • status                     │
│  • resume                      • history                    │
│  • change-ip                   • stats                      │
│  • change-interval                                          │
│  • stop                                                     │
│                                                             │
│  Main Loop:                                                 │
│  1. Check for signals (non-blocking)                        │
│  2. Wait for timer or signal                                │
│  3. If not paused: fetch IP location                        │
│  4. Update history                                          │
│  5. Repeat until stopped                                    │
└─────────────────────────────────────────────────────────────┘
```

## File Structure

### monitor_workflow.go
**Location**: `/home/user/temporal/monitor_workflow.go`

Key components:
1. **State Variables**: Track current IP, interval, pause state, history
2. **Signal Channels**: Receive commands from external clients
3. **Query Handlers**: Return current state on demand
4. **Main Loop**: Uses `workflow.Selector` for non-blocking signal handling

```go
// Signal setup
pauseChan := workflow.GetSignalChannel(ctx, "pause")
resumeChan := workflow.GetSignalChannel(ctx, "resume")
// ... more signals

// Query setup
workflow.SetQueryHandler(ctx, "status", func() (MonitorStatus, error) {
    return MonitorStatus{State: state, CurrentIP: currentIP, ...}, nil
})
```

**Critical Pattern: workflow.Selector**
```go
selector := workflow.NewSelector(ctx)

// Add signal handlers (non-blocking)
selector.AddReceive(pauseChan, func(c workflow.ReceiveChannel, more bool) {
    var sig PauseSignal
    c.Receive(ctx, &sig)
    isPaused = true
})

// Add timer
timer := workflow.NewTimer(ctx, checkInterval)
selector.AddFuture(timer, func(f workflow.Future) {
    // Timer fired
})

// Wait for EITHER signal OR timer
selector.Select(ctx)
```

This pattern allows the workflow to:
- Process signals immediately when received
- Continue normal operation (timer-based checks) otherwise
- Never block indefinitely

### starter/monitor_demo.go
**Location**: `/home/user/temporal/starter/monitor_demo.go`

Demonstrates the full lifecycle:
1. **Start workflow** with initial config
2. **Query status** at different points
3. **Send signals** to modify behavior:
   - Pause → verify paused → resume
   - Change IP address mid-flight
   - Change check interval
   - Stop gracefully
4. **Verify changes** via queries

## How to Run

### Terminal 1: Start Temporal Server
```bash
temporal server start-dev
```

### Terminal 2: Start Worker
```bash
go run worker/main.go
```
Keep this running! Worker must be active to process workflows.

### Terminal 3: Run the Demo
```bash
go run starter/monitor_demo.go
```

## What You'll See

```
┌─────────────────────────────────────────────────────────────┐
│      Temporal Signals & Queries Demo - IP Monitor          │
└─────────────────────────────────────────────────────────────┘

✓ Workflow started!
  WorkflowID: ip-monitor-demo-1731234567
  RunID: abc123...
  View in UI: http://localhost:8233

⏳ Waiting 3 seconds for first check...

📊 QUERY: Getting initial status...
   State: running
   Current IP: 8.8.8.8
   Check Interval: 5s
   Total Checks: 1

⚡ SIGNAL: Sending PAUSE signal...
   ✓ Pause signal sent

📊 QUERY: Verifying paused state...
   State: paused
   Current IP: 8.8.8.8

⚡ SIGNAL: Changing monitored IP to 1.1.1.1 (Cloudflare DNS)...
   ✓ Change-IP signal sent

⚡ SIGNAL: Sending RESUME signal...
   ✓ Resume signal sent

📊 QUERY: Checking history (should show new IP)...
   Total entries: 3
   Recent checks:
     ✓ [14:30:05] IP: 8.8.8.8 → City: Mountain View, Region: California, Country: US
     ✓ [14:30:10] IP: 8.8.8.8 → City: Mountain View, Region: California, Country: US
     ✓ [14:30:22] IP: 1.1.1.1 → City: San Francisco, Region: California, Country: US

⚡ SIGNAL: Sending STOP signal...
   ✓ Stop signal sent

✓ Workflow completed gracefully!
```

## Key Learning Points

### 1. Signals Are Durable
If the worker crashes AFTER a signal is sent but BEFORE it's processed, the signal is NOT lost. When the worker restarts, Temporal replays the workflow history including all signals.

**Try it:**
1. Start monitor workflow
2. Send pause signal
3. Kill the worker (`Ctrl+C`)
4. Restart worker
5. Query status → It's still paused! ✓

### 2. Queries Are Not Durable
Queries don't modify state and aren't recorded in history. They're lightweight "peek" operations.

**Try it:**
1. Query workflow status 100 times
2. Check workflow history in UI
3. No query events appear! (Only signals and activities)

### 3. Selector Pattern for Non-Blocking Signals
Without `Selector`, you'd have to choose:
- Block waiting for signals → Can't do timer-based work
- Ignore signals → Can't be controlled dynamically

`Selector` lets you wait for MULTIPLE events simultaneously:
```go
selector.Select(ctx)  // Waits for FIRST of: signal OR timer
```

### 4. Signal Ordering is Guaranteed
If client sends: `pause` → `change-ip` → `resume`
Workflow receives them in EXACT same order.

### 5. Queries Can Fail If Workflow Is Completed
```go
// Workflow completed
val, err := c.QueryWorkflow(ctx, workflowID, "", "status")
// err will indicate workflow is closed
```

## Real-World Use Cases

### Signals
1. **Order Processing**: Cancel order, update shipping address
2. **Data Pipeline**: Pause processing, add more data sources
3. **Long-Running Jobs**: Increase/decrease parallelism, kill gracefully
4. **Approval Workflows**: Approve/reject/request-changes

### Queries
1. **Progress Tracking**: "How many records processed?"
2. **Health Checks**: "Is this workflow stuck?"
3. **Debugging**: "What's the current state?"
4. **Dashboards**: Real-time monitoring without database queries

## Advanced Patterns

### 1. Buffered Signals
Signals sent BEFORE workflow starts are buffered and delivered when it does start.

### 2. Signal-With-Start
Start a workflow OR signal it if already running:
```go
c.SignalWithStartWorkflow(ctx, workflowID, signalName, signalValue, workflowOptions, workflowFunc)
```

### 3. Typed Signals (Type Safety)
```go
// Instead of: c.SignalWorkflow(ctx, id, "", "pause", PauseSignal{})
// Use workflow methods for type safety (advanced pattern)
```

### 4. Conditional Logic Based on Signals
```go
if isPaused {
    workflow.GetLogger(ctx).Info("Skipping work - paused")
    continue
}
// ... do work
```

### 5. Acknowledgment Pattern
```go
// Workflow: Send activity after receiving signal to "ack" it
selector.AddReceive(importantSignalChan, func(...) {
    workflow.ExecuteActivity(ctx, SendAckNotification, signalData)
})
```

## Common Mistakes

### ❌ Blocking Receive
```go
// BAD: Blocks forever if no signal sent
pauseChan.Receive(ctx, &sig)
```

### ✅ Non-Blocking with Selector
```go
// GOOD: Continues if no signal
selector.AddReceive(pauseChan, func(...) { ... })
selector.Select(ctx)
```

### ❌ Modifying State in Query Handler
```go
workflow.SetQueryHandler(ctx, "bad-query", func() (string, error) {
    totalChecks++ // BUG! Queries must be read-only
    return "result", nil
})
```

### ✅ Read-Only Queries
```go
workflow.SetQueryHandler(ctx, "good-query", func() (int, error) {
    return totalChecks, nil // Just reading
})
```

### ❌ Not Using workflow.Now()
```go
// BAD: Non-deterministic!
timestamp := time.Now()
```

### ✅ Use workflow.Now()
```go
// GOOD: Deterministic, uses workflow logical time
timestamp := workflow.Now(ctx)
```

## Testing Signals and Queries

### Manual Testing (Demo Script)
Run `starter/monitor_demo.go` - it exercises all signals and queries automatically.

### Unit Testing
```go
func TestMonitorWorkflow_PauseResume(t *testing.T) {
    env := testenv.NewTestWorkflowEnvironment()

    // Start workflow in background
    env.ExecuteWorkflow(IPMonitorWorkflow, config)

    // Let it run a bit
    env.Sleep(5 * time.Second)

    // Send pause signal
    env.SignalWorkflow("pause", PauseSignal{})

    // Query status
    val, err := env.QueryWorkflow("status")
    var status MonitorStatus
    val.Get(&status)

    assert.Equal(t, "paused", status.State)
}
```

## Next Steps

After mastering Signals and Queries, explore:
1. **Child Workflows**: Compose workflows (break geolocation into sub-workflow)
2. **Continue-As-New**: Handle infinite/long-running workflows
3. **Search Attributes**: Make workflows discoverable by business attributes
4. **Activity Heartbeats**: Long-running activities with progress tracking
5. **Saga Pattern**: Complete the compensation logic (RecordLookup + CompensateLookup)

## Resources

- **Temporal Docs**: https://docs.temporal.io/workflows#signal
- **This Workflow**: `monitor_workflow.go:11` (IPMonitorWorkflow)
- **Demo Starter**: `starter/monitor_demo.go:1`
- **UI**: http://localhost:8233 (view signal/query events)

---

**Questions?** Run the demo, watch the logs, explore the UI, and see signals/queries in action! 🚀

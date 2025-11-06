# Quick Start Guide - Temporal Signals & Queries Demo

## What You'll Learn
How to dynamically control running workflows using **Signals** (send commands) and **Queries** (read state).

## Prerequisites
- Temporal server running
- Worker running
- Go installed

## 3-Step Setup

### Step 1: Start Temporal Server
```bash
temporal server start-dev
```
Leave this running. Access UI at http://localhost:8233

### Step 2: Start Worker
```bash
go run worker/main.go
```
Leave this running. It processes workflows.

### Step 3: Run the Demo
```bash
go run starter/monitor_demo.go
```
Watch it:
1. Start a workflow that monitors IP geolocation
2. Query its status
3. Pause it via signal
4. Change the monitored IP
5. Resume it
6. Stop it gracefully

All without restarting the workflow!

## What Happens

The demo workflow monitors an IP address every 5 seconds. While it's running, you'll see:

**Queries (reading state):**
```
📊 QUERY: Getting initial status...
   State: running
   Current IP: 8.8.8.8
   Total Checks: 2
```

**Signals (modifying behavior):**
```
⚡ SIGNAL: Sending PAUSE signal...
   ✓ Pause signal sent

⚡ SIGNAL: Changing monitored IP to 1.1.1.1...
   ✓ Change-IP signal sent

⚡ SIGNAL: Sending RESUME signal...
   ✓ Resume signal sent
```

## Available Signals

Send these to control the running workflow:

| Signal | Purpose | Example |
|--------|---------|---------|
| `pause` | Stop checking temporarily | Maintenance window |
| `resume` | Resume checking | Maintenance complete |
| `change-ip` | Monitor different IP | Switch targets |
| `change-interval` | Adjust check frequency | Rate limiting |
| `stop` | Gracefully terminate | Cleanup and exit |

## Available Queries

Read these from the running workflow:

| Query | Returns | Use Case |
|-------|---------|----------|
| `status` | Full state snapshot | Dashboard display |
| `history` | All check results | Audit trail |
| `stats` | Summary metrics | Monitoring |

## Sending Signals Manually

Using Temporal CLI:
```bash
# Pause the workflow
temporal workflow signal \
  --workflow-id ip-monitor-demo-1731234567 \
  --name pause

# Change IP
temporal workflow signal \
  --workflow-id ip-monitor-demo-1731234567 \
  --name change-ip \
  --input '{"NewIP": "1.1.1.1"}'
```

Using Go client:
```go
c.SignalWorkflow(ctx, workflowID, "", "pause", iplocate.PauseSignal{})

c.SignalWorkflow(ctx, workflowID, "", "change-ip", iplocate.ChangeIPSignal{
    NewIP: "1.1.1.1",
})
```

## Querying State Manually

Using Temporal CLI:
```bash
# Get status
temporal workflow query \
  --workflow-id ip-monitor-demo-1731234567 \
  --name status

# Get history
temporal workflow query \
  --workflow-id ip-monitor-demo-1731234567 \
  --name history
```

Using Go client:
```go
var status iplocate.MonitorStatus
val, err := c.QueryWorkflow(ctx, workflowID, "", "status")
val.Get(&status)
```

## View in Web UI

1. Open http://localhost:8233
2. Find your workflow: `ip-monitor-demo-*`
3. Click to see details
4. Observe:
   - **History**: Shows signal events (pause, resume, change-ip)
   - **Queries**: Not recorded (lightweight reads)
   - **Activities**: Shows GetLocationInfo calls

## Experiments to Try

### 1. Pause/Resume Pattern
```bash
# Terminal 1: Start demo
go run starter/monitor_demo.go

# While running, in UI:
# - Notice checks happening every 5s
# - After pause signal: checks stop
# - After resume signal: checks restart
```

### 2. Dynamic IP Change
```bash
# Start monitoring 8.8.8.8 (Google DNS)
# Send change-ip signal with 1.1.1.1 (Cloudflare)
# Query history - see both IPs in results!
```

### 3. Workflow Resilience
```bash
# Start workflow
# Send pause signal
# Kill worker (Ctrl+C)
# Restart worker
# Query status → Still paused! (durable signals)
```

### 4. Multiple Clients
```bash
# Terminal 1: Start workflow (starter/monitor_demo.go)
# Terminal 2: Send signals via temporal CLI
# Terminal 3: Query state via temporal CLI
# All interact with same workflow instance!
```

## Troubleshooting

### "Workflow not found"
- Check WorkflowID is correct
- Workflow may have completed already
- Use UI to find actual ID

### "No such query type"
- Query name must match exactly: "status", "history", "stats"
- Query handlers are case-sensitive

### "Worker not registered"
- Make sure `worker/main.go` includes:
  ```go
  w.RegisterWorkflow(iplocate.IPMonitorWorkflow)
  ```
- Restart worker after code changes

### Signals not working
- Check workflow is still running (not completed)
- Verify signal name matches: "pause", "resume", etc.
- Check worker logs for errors

## Next: Deep Dive

See `SIGNALS_AND_QUERIES.md` for:
- Architecture details
- Code walkthrough
- Advanced patterns
- Real-world use cases
- Testing strategies

## Files Reference

- **Workflow**: `monitor_workflow.go` (line 44: IPMonitorWorkflow)
- **Demo Starter**: `starter/monitor_demo.go`
- **Worker**: `worker/main.go` (registers workflow)
- **Documentation**: `SIGNALS_AND_QUERIES.md`

---

**Ready?** Run the 3 steps above and watch dynamic workflow control in action! 🎯

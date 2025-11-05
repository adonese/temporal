# Project Context for Claude

## What This Is
A Temporal workflow demo that fetches your public IP and looks up its geographic location.

## Architecture

### Critical Pattern: Worker vs Starter
- **worker/main.go**: Long-running service, polls Temporal for tasks, executes workflows/activities
- **starter/main.go**: Fire-and-forget client that triggers workflows then exits immediately
- This separation is BY DESIGN in Temporal - workers are services, starters are clients

### File Responsibilities
Treat those as samples, not the ultimate source of truth.
- `activities.go`: HTTP calls to external APIs (GetIP, GetLocationInfo)
- `workflows.go`: Orchestration logic (calls activities in sequence)
- `shared.go`: Constants like TaskQueueName
- `worker/main.go`: Connects to Temporal, registers workflows/activities, runs worker
- `starter/main.go`: Connects to Temporal, starts workflow, exits



### 2. Workflow ID Strategy
Current: `"ip-geolocation-workflow-" + timestamp` (unique per execution)
Alternatives:
- Constant ID: Prevents duplicate executions (Temporal rejects if already running)
- Entity-based: `"order-" + orderID` (one workflow per entity)














## Module Structure
Module name: `temporal-ip-geolocation/iplocate` (defined in go.mod)
Package: `iplocate` (all .go files in root)
Imports use: `temporal-ip-geolocation/iplocate`

## How to Run

1. Start Temporal server: `temporal server start-dev`
2. Start worker: `go run worker/main.go` (keep running)
3. Trigger workflow: `go run starter/main.go` (exits immediately)
4. View results: http://localhost:8233

## Testing
Worker must be running to process workflows. Starter exits immediately after triggering.
Workflow execution happens asynchronously in the worker.

## Important Notes for Future Claude Sessions

1. **Never combine worker and starter** - this is Temporal's design pattern
2. **Worker must stay running** - it's a service, not a one-shot script
3. **Starter is fire-and-forget** - it doesn't wait for results by design
4. **Insecure gRPC is correct** for local dev server - not a bug
5. **HTTP (not HTTPS) for ip-api.com** is correct for free tier - not a typo
6. After changing activities/workflows, **restart the worker** to pick up changes

## Temporal Concepts

- **Workflow**: Orchestration logic (workflows.go)
- **Activity**: Actual work/side effects (activities.go)
- **Worker**: Service that executes workflows/activities
- **TaskQueue**: Named queue that connects workers and starters (must match)
- **WorkflowID**: Unique identifier for workflow execution (affects idempotency)

## Environment
- Platform: WSL2 (affects DNS resolution)
- Temporal Server: localhost:7233 (gRPC), localhost:8233 (Web UI)
- Go version: 1.25.3

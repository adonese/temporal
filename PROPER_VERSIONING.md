# Proper Workflow Versioning - No Hacks

## The Golden Rule

**Never change a workflow's signature (parameters or return types) for running workflows.**

If you need to change signatures → Create a new workflow function.

---

## Exercise: The Right Way

### Scenario
You have a running workflow that returns location info. You want to add timezone.

### ❌ WRONG: Change return type

```go
// Original
func GetAddressFromIP(ctx workflow.Context, name string) (data, error)

// Changed - BREAKS RUNNING WORKFLOWS
func GetAddressFromIP(ctx workflow.Context, name string) (Data, error)
```

### ✅ RIGHT: Option A - New Workflow

```go
// V1: Keep forever, never modify
func GetAddressFromIP(ctx workflow.Context, name string) (data, error) {
    // Original logic
}

// V2: New workflow with new return type
func GetAddressWithTimezone(ctx workflow.Context, name string) (Data, error) {
    // New logic with timezone
}
```

### ✅ RIGHT: Option B - Stable Type, Version Logic

```go
// Keep the same return type, add optional field
type data struct {
    Result   string `json:"result"`
    Timezone string `json:"timezone,omitempty"`  // Optional
}

func GetAddressFromIP(ctx workflow.Context, name string) (data, error) {
    version := workflow.GetVersion(ctx, "add-timezone", workflow.DefaultVersion, 1)

    var ip string
    err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
    if err != nil {
        return data{}, err
    }

    workflow.Sleep(ctx, 45*time.Second)

    var location string
    err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
    if err != nil {
        return data{}, err
    }

    result := data{Result: location}

    // Version check - only affects activity execution
    if version == 1 {
        var timezone string
        err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
        if err != nil {
            return data{}, err
        }
        result.Timezone = timezone
    }

    return result, nil
}
```

---

## The Proper Exercise Steps

### Phase 1: Start with Clean Code

**workflows.go:**
```go
type WorkflowResult struct {
    Location string `json:"location"`
    Timezone string `json:"timezone,omitempty"`  // Optional from day 1
}

func GetAddressFromIP(ctx workflow.Context, name string) (WorkflowResult, error) {
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
        return WorkflowResult{}, fmt.Errorf("failed to get ip: %s", err)
    }

    workflow.Sleep(ctx, 45*time.Second)

    var location string
    err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
    if err != nil {
        return WorkflowResult{}, fmt.Errorf("failed to get location: %s", err)
    }

    // V1: Don't fetch timezone yet
    return WorkflowResult{Location: location}, nil
}
```

### Phase 2: Start a Workflow (Will Run with V1 Logic)

```bash
# Terminal 1
go run worker/main.go

# Terminal 2
go run starter/main.go
```

Workflow sleeps for 45 seconds...

### Phase 3: Add Timezone Feature (Versioned)

While workflow sleeps, modify workflows.go:

```go
func GetAddressFromIP(ctx workflow.Context, name string) (WorkflowResult, error) {
    // ... setup code same as before

    var ip string
    err := workflow.ExecuteActivity(ctx, ipActivities.GetIP).Get(ctx, &ip)
    if err != nil {
        return WorkflowResult{}, fmt.Errorf("failed to get ip: %s", err)
    }

    workflow.Sleep(ctx, 45*time.Second)

    // VERSION CHECK - this is the key
    version := workflow.GetVersion(ctx, "add-timezone", workflow.DefaultVersion, 1)

    var location string
    err = workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, ip).Get(ctx, &location)
    if err != nil {
        return WorkflowResult{}, fmt.Errorf("failed to get location: %s", err)
    }

    result := WorkflowResult{Location: location}

    // New feature: Fetch timezone (only for version 1)
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
    }

    return result, nil
}
```

### Phase 4: Restart Worker

```bash
# Ctrl+C in Terminal 1
go run worker/main.go
```

### Phase 5: Observe Success ✅

Old workflow completes:
- Version = DefaultVersion
- Timezone NOT fetched
- Returns: `WorkflowResult{Location: "...", Timezone: ""}`

Start new workflow:
- Version = 1
- Timezone IS fetched
- Returns: `WorkflowResult{Location: "...", Timezone: "America/New_York"}`

**Both use the same return type!** No `interface{}` hack needed.

---

## Key Principles

### 1. Plan for Evolution from Day 1

```go
// Bad: Hard to extend
type Result struct {
    Location string
}

// Good: Easy to extend
type Result struct {
    Location string              `json:"location"`
    Timezone string              `json:"timezone,omitempty"`
    ISP      string              `json:"isp,omitempty"`
    // Future fields can be added here
}
```

### 2. Return Type Stability

**The return type is part of the contract.** If you need to change it, you need a new workflow.

```go
// V1 - returns string
func ProcessOrder(ctx workflow.Context) (string, error)

// V2 - returns struct (different contract = new workflow)
func ProcessOrderV2(ctx workflow.Context) (OrderResult, error)
```

### 3. Use GetVersion for Logic Changes, Not Type Changes

```go
// ✅ Good: Same return type, different execution path
version := workflow.GetVersion(ctx, "add-step", workflow.DefaultVersion, 1)
if version == 1 {
    doNewActivity()
}
return Result{...}  // Same type for all versions

// ❌ Bad: Trying to use GetVersion to change return type
version := workflow.GetVersion(ctx, "change-type", workflow.DefaultVersion, 1)
if version == 1 {
    return NewType{...}  // Won't compile! Return type is fixed
}
return OldType{...}
```

### 4. Deprecation Strategy

When you create V2 of a workflow:

1. **Month 1-3:** Both V1 and V2 live side-by-side
   - New workflows use V2
   - Old workflows complete with V1

2. **Month 3-6:** Stop creating V1 workflows
   - Update all starters to use V2
   - V1 still runs for old workflows

3. **Month 6+:** V1 only exists for history replay
   - No new V1 workflows
   - Keep V1 code forever (for replay)

4. **Never:** Remove V1 code
   - Temporal needs it to replay old workflows
   - Mark as deprecated in comments

---

## Real-World Pattern: Workflow Factories

Production teams often use workflow factories:

```go
// Factory pattern for versioning
func GetAddressWorkflow(ctx workflow.Context, version int, input WorkflowInput) (WorkflowResult, error) {
    switch version {
    case 1:
        return getAddressV1(ctx, input)
    case 2:
        return getAddressV2(ctx, input)
    default:
        return WorkflowResult{}, fmt.Errorf("unsupported version: %d", version)
    }
}

func getAddressV1(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error) {
    // V1 logic
}

func getAddressV2(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error) {
    // V2 logic with timezone
}
```

Caller explicitly chooses version:
```go
c.ExecuteWorkflow(ctx, opts, GetAddressWorkflow, 2, input)
```

---

## Summary: What to Do

| Scenario | Solution | Use GetVersion? |
|----------|----------|----------------|
| Add activity | Version internally | ✅ Yes |
| Remove activity | Version internally | ✅ Yes |
| Reorder activities | Version internally | ✅ Yes |
| Change parameter types | Create new workflow | ❌ No |
| Change return type | Create new workflow | ❌ No |
| Add optional field to return type | Version internally | ✅ Yes |
| Change workflow name | Create new workflow | ❌ No |

**Rule of thumb:** If the function signature changes, create a new workflow. If the logic changes, use GetVersion.

---

## Next Steps

Now let's implement the clean exercise:
1. Define `WorkflowResult` with optional `Timezone` field
2. Start workflow without timezone
3. Add versioned timezone logic while it sleeps
4. See both versions coexist with same return type

No hacks. No `interface{}`. Just clean, production-ready code.

# Temporal Workflow Versioning Exercise

## The Problem We're Solving

When you deploy new code, you might have workflows that are currently running with the OLD code.
Temporal replays workflow history to restore state. If the new code doesn't match the recorded history,
you get a **non-deterministic error** and the workflow fails.

## Exercise: Experience the Pain, Then Fix It

### Setup (Version 1 - Current Code)

Your workflow now:
1. Fetches IP
2. **Sleeps 45 seconds** ← This gives us time to change code!
3. Fetches location

### Step 1: Start the Original Workflow

```bash
# Terminal 1: Start the worker
go run worker/main.go

# Terminal 2: Start a workflow
go run starter/main.go
```

The workflow will start, fetch the IP, then sleep for 45 seconds.
**Watch the worker logs** - you'll see "Sleeping for 45 seconds..."

---

### Step 2: Make a Breaking Change (While Workflow is Sleeping!)

**THE BREAKING CHANGE**: Let's say we want to add a new feature - fetch timezone info BEFORE location.

While the workflow is still sleeping, modify `workflows.go`:

**Add this BEFORE the location fetch** (around line 40):

```go
// NEW FEATURE: Get timezone info
workflow.GetLogger(ctx).Info("Version 2: Now fetching timezone!")
var timezone string
err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
if err != nil {
    return data{}, fmt.Errorf("failed to get timezone: %s", err)
}
workflow.GetLogger(ctx).Info("Timezone fetched", "timezone", timezone)
```

**Then restart the worker** (Ctrl+C and `go run worker/main.go` again)

---

### Step 3: Watch It Crash! 💥

When the sleeping workflow wakes up, Temporal replays the workflow:
- **Recorded history says**: GetIP → Sleep → GetLocation
- **New code says**: GetIP → Sleep → **GetTimezone** → GetLocation

**Result**: Non-deterministic error! Workflow fails!

You'll see an error like:
```
nondeterministic workflow: history event is ActivityTaskScheduled for
GetLocationInfo but workflow code expected ActivityTaskScheduled for GetTimezone
```

---

### Step 4: Fix It With Versioning (The Right Way)

Now we'll fix this using `workflow.GetVersion()`. This is the pattern used in production.

**The Fix**: Modify `workflows.go` to support BOTH old and new versions:

```go
// After the sleep, BEFORE fetching location:

// Check which version this workflow is running
version := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 1)

if version == 1 {
    // NEW VERSION: Fetch timezone first
    workflow.GetLogger(ctx).Info("Version 2: Fetching timezone (new feature)")
    var timezone string
    err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
    if err != nil {
        return data{}, fmt.Errorf("failed to get timezone: %s", err)
    }
    workflow.GetLogger(ctx).Info("Timezone fetched", "timezone", timezone)
}
// If version == DefaultVersion, skip timezone (old behavior)

// Continue with location fetch (both versions do this)
workflow.GetLogger(ctx).Info("Fetching location...")
```

**What This Does**:
- **Old workflows** (started before the change): `GetVersion` returns `DefaultVersion` → skip timezone
- **New workflows** (started after the change): `GetVersion` returns `1` → fetch timezone
- Both versions can coexist safely!

---

### Step 5: Deploy and Test

1. **Restart the worker** with the versioned code
2. **Old running workflows** complete successfully (skip timezone)
3. **Start a new workflow**: `go run starter/main.go`
4. **New workflows** execute the timezone fetch

---

## Key Concepts

### workflow.GetVersion() Parameters

```go
workflow.GetVersion(ctx, changeID, minSupported, maxSupported)
```

- **changeID**: Unique identifier for this change (e.g., "add-timezone-feature")
- **minSupported**: Lowest version your code still supports (usually DefaultVersion)
- **maxSupported**: Latest version (increment for each change: 1, 2, 3...)

### Version Lifecycle

**Phase 1: Deploy (both versions live)**
```go
version := workflow.GetVersion(ctx, "my-change", workflow.DefaultVersion, 1)
if version == 1 {
    // new code
} else {
    // old code (for running workflows)
}
```

**Phase 2: After all old workflows complete (weeks/months later)**
```go
version := workflow.GetVersion(ctx, "my-change", 1, 1) // min=1, max=1
// Only new code runs, but marker stays for replay
```

**Phase 3: Never remove GetVersion** (it's permanent!)
The GetVersion marker must stay forever - it's part of the workflow's history.

---

## Common Breaking Changes

These require versioning:
- ✅ Adding/removing activities
- ✅ Reordering activities
- ✅ Changing activity parameters
- ✅ Adding/removing workflow.Sleep()
- ✅ Changing conditional logic (if/else branches)

These are safe (no versioning needed):
- ❌ Changing activity implementation (as long as signature stays same)
- ❌ Changing logs
- ❌ Updating retry policies (ActivityOptions)
- ❌ Bug fixes in activities (not workflow logic)

---

## Next Steps

Once you've completed this exercise:
1. Try adding another change (version 2)
2. Practice the cleanup phase (minSupported = 1)
3. Read: [Temporal Versioning Docs](https://docs.temporal.io/workflows#versioning)

---

## Troubleshooting

**Q: My workflow still fails with versioning!**
- Make sure you restart the worker after code changes
- Check that minSupported includes the version old workflows expect
- Verify the changeID is unique for each distinct change

**Q: Can I remove old version code?**
- Only after ALL workflows started before that version have completed
- Check Temporal UI to see if any workflows with old versions are running
- Always increment minSupported, never decrease maxSupported

**Q: What if I have multiple changes?**
- Each change gets its own GetVersion call with a unique changeID
- Versions stack: version1, version2, version3...
- Order matters - GetVersion calls must always execute in the same order

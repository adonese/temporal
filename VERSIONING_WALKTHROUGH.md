# Complete Versioning Walkthrough - What You'll See

This document shows exactly what to expect at each step, including terminal outputs.

## Prerequisites

```bash
# Make sure Temporal server is running
temporal server start-dev
```

---

## 🔴 PHASE 1: Experience the Pain (Non-Deterministic Error)

### Step 1A: Start Worker (Terminal 1)

```bash
cd /home/user/temporal
go run worker/main.go
```

**Expected Output:**
```
2025/11/06 10:00:00 Attempting to connect to Temporal server at: 127.0.0.1:7233
2025/11/06 10:00:00 Successfully connected to Temporal server
2025/11/06 10:00:00 INFO  Worker started Namespace default TaskQueue ip-finder WorkerID ...
```

### Step 1B: Start Workflow (Terminal 2)

```bash
cd /home/user/temporal
go run starter/main.go
```

**Expected Output:**
```
2025/11/06 10:00:05 ✓ Workflow started successfully!
2025/11/06 10:00:05   WorkflowID: ip-geolocation-workflow-1730880005
2025/11/06 10:00:05   RunID: 1a2b3c4d-5e6f-7g8h-9i0j-k1l2m3n4o5p6
2025/11/06 10:00:05   View in UI: http://localhost:8233
2025/11/06 10:00:05
Workflow executing in background. Starter exiting...
```

### Step 1C: Watch Worker Logs (Terminal 1)

**You'll see:**
```
INFO  Version 1: Starting workflow - will fetch IP, wait, then get location
INFO  IP fetched ip=123.45.67.89
INFO  Sleeping for 45 seconds... (this is when you'll modify the code)
```

⏰ **NOW YOU HAVE 45 SECONDS TO MAKE THE BREAKING CHANGE!**

---

### Step 2: Make Breaking Change (While Workflow Sleeps!)

**Edit workflows.go** - Add this code AFTER line 38 (after the Sleep), BEFORE the "Awake!" log:

```go
	// BREAKING CHANGE: Add timezone fetch
	workflow.GetLogger(ctx).Info("Version 2: Now fetching timezone!")
	var timezone string
	err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
	if err != nil {
		return data{}, fmt.Errorf("failed to get timezone: %s", err)
	}
	workflow.GetLogger(ctx).Info("Timezone fetched", "timezone", timezone)
```

### Step 3: Restart Worker (Terminal 1)

```bash
# Press Ctrl+C to stop the worker
^C

# Restart with new code
go run worker/main.go
```

**Expected Output:**
```
2025/11/06 10:00:52 Successfully connected to Temporal server
INFO  Worker started
```

### Step 4: Watch the Crash! 💥

**After ~3 seconds, you'll see the error:**
```
ERROR Workflow panic Namespace default TaskQueue ip-finder WorkerID ... WorkflowType GetAddressFromIP
      Error non-deterministic workflow: history event is ActivityTaskScheduled,
      activityType: GetLocationInfo, but workflow code expected ActivityTaskScheduled,
      activityType: GetTimezone
```

**In the Temporal UI (http://localhost:8233):**
- Workflow status: FAILED
- Error: "nondeterministic workflow"
- History shows: GetIP → Sleep → (expected GetLocation, got GetTimezone)

**🎓 LESSON LEARNED:** You can't change workflow logic for running workflows!

---

## 🟢 PHASE 2: Fix It With Versioning

### Step 5: Implement Versioning

**Replace the breaking change code in workflows.go with versioned code:**

```go
	workflow.GetLogger(ctx).Info("Awake!")

	// VERSION CHECK - This is the fix!
	version := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 1)

	var timezone string
	if version == 1 {
		// NEW workflows: fetch timezone
		workflow.GetLogger(ctx).Info("Version 1: Fetching timezone (new feature)")
		err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
		if err != nil {
			return data{}, fmt.Errorf("failed to get timezone: %s", err)
		}
		workflow.GetLogger(ctx).Info("Timezone fetched", "timezone", timezone)
	} else {
		// OLD workflows: skip timezone
		workflow.GetLogger(ctx).Info("DefaultVersion: Skipping timezone (old workflow)")
	}

	workflow.GetLogger(ctx).Info("Fetching location...")
```

**Or simply copy the solution:**
```bash
cp workflows_versioned_SOLUTION.go workflows.go
```

### Step 6: Restart Worker Again

```bash
# In Terminal 1
# Press Ctrl+C, then restart
go run worker/main.go
```

### Step 7: Test Old Workflow Recovery

The failed workflow will automatically retry!

**Expected Output (Terminal 1):**
```
INFO  Workflow task processing started
INFO  DefaultVersion: Skipping timezone (old workflow)
INFO  Fetching location...
INFO  Workflow completed successfully result="City: ..., Region: ..., Country: ..."
```

**🎓 SUCCESS!** Old workflow recovered and completed without timezone fetch.

---

## 🟦 PHASE 3: Test New Workflows

### Step 8: Start a Fresh Workflow

```bash
# Terminal 2
go run starter/main.go
```

### Step 9: Watch New Behavior (Terminal 1)

**Expected Output:**
```
INFO  Starting workflow with versioning support
INFO  IP fetched ip=123.45.67.89
INFO  Sleeping for 45 seconds...
INFO  Awake!
INFO  Version 1: Fetching timezone (new feature)
DEBUG: Fetching timezone for IP [123.45.67.89] from URL: http://ip-api.com/json/...
INFO  Timezone fetched timezone=America/New_York
INFO  Fetching location...
INFO  Workflow completed successfully result="City: ..., Region: ..., Country: ..., Timezone: America/New_York"
```

**🎓 SUCCESS!** New workflow runs the timezone fetch.

---

## 🟣 PHASE 4: Coexistence Test

Let's prove old and new versions can run simultaneously.

### Step 10: Start Multiple Workflows

```bash
# Start 3 new workflows (they'll all use version 1)
go run starter/main.go
sleep 2
go run starter/main.go
sleep 2
go run starter/main.go
```

**Expected Behavior:**
- All 3 workflows run version 1 (with timezone)
- All complete successfully
- Check Temporal UI: 3 workflows in "Completed" state

---

## 🟡 PHASE 5: Understanding the Lifecycle

### Scenario: Months Later (All Old Workflows Done)

After weeks/months, all workflows started before the timezone feature have completed.

**You can now increment minSupported:**

```go
// Before (supports both old and new):
version := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 1)

// After (only supports new):
version := workflow.GetVersion(ctx, "add-timezone-feature", 1, 1)
```

**What changes:**
- `minSupported: DefaultVersion → 1`
- Now ALL workflows MUST have timezone (version >= 1)
- You can simplify the if/else (but GetVersion stays!)

```go
version := workflow.GetVersion(ctx, "add-timezone-feature", 1, 1)
// No if statement needed - version is always 1
err = workflow.ExecuteActivity(ctx, ipActivities.GetTimezone, ip).Get(ctx, &timezone)
// ... timezone is always fetched
```

**⚠️ CRITICAL:** Never remove the GetVersion call! It's permanent.

---

## 📊 Comparison: Before vs After

### WITHOUT Versioning

```
Timeline:
T=0s:   Start workflow (GetIP → Sleep → GetLocation)
T=30s:  Deploy new code (GetIP → Sleep → GetTimezone → GetLocation)
T=45s:  Workflow wakes up, replays history
        CRASH: "Expected GetLocation, got GetTimezone"
```

### WITH Versioning

```
Timeline:
T=0s:   Start workflow (GetIP → Sleep → GetVersion(returns Default) → GetLocation)
T=30s:  Deploy versioned code
T=45s:  Workflow wakes up, replays history
        GetVersion returns DefaultVersion → Skip timezone
        SUCCESS: Completes with GetLocation

T=60s:  Start new workflow
        GetVersion returns 1 → Fetch timezone
        SUCCESS: Completes with GetTimezone → GetLocation
```

---

## 🎯 Key Takeaways

1. **Temporal replays workflows** - Code must match recorded history
2. **GetVersion = time machine** - Tells replays which path to take
3. **changeID is forever** - Pick good names, never change them
4. **minSupported = cleanup tool** - Increment after old workflows done
5. **maxSupported = version counter** - Always increment for new changes
6. **Coexistence is safe** - Old and new workflows run side-by-side

---

## 🐛 Troubleshooting

### Error: "Cannot change workflow execution path"
**Cause:** GetVersion calls executing in different order
**Fix:** Ensure GetVersion calls always execute in the same sequence

### Error: Workflow still fails after adding GetVersion
**Cause:** Worker not restarted, or GetVersion placed after the divergence point
**Fix:** Restart worker, place GetVersion BEFORE the code change

### Error: "Version too old"
**Cause:** minSupported increased but old workflow still running
**Fix:** Lower minSupported or wait for old workflow to complete

---

## 🚀 Next Exercise: Add a Second Change (Version 2)

Try adding another feature:

1. Add a new activity: `GetISP(ip)` (fetch ISP name)
2. Add it with a NEW GetVersion call
3. Test that it works alongside the timezone feature

**Hint:**
```go
v1 := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 1)
// ... timezone code

v2 := workflow.GetVersion(ctx, "add-isp-feature", workflow.DefaultVersion, 1)
if v2 == 1 {
    // fetch ISP
}
```

---

## 📚 Files Created for This Exercise

- `VERSIONING_EXERCISE.md` - Step-by-step guide
- `VERSIONING_WALKTHROUGH.md` - This file (detailed outputs)
- `VERSIONING_REFERENCE.md` - Quick reference patterns
- `workflows_versioned_SOLUTION.go` - Complete solution code
- `workflows.go` - Modified with sleep timer (starting point)

---

## 🎓 Congratulations!

You've now experienced:
- ❌ The pain of non-deterministic errors
- ✅ How to fix it with workflow.GetVersion()
- 🔄 How old and new workflows coexist
- 📈 The version lifecycle (DefaultVersion → 1 → cleanup)

This is production-critical knowledge. Many Temporal users learn this the hard way in production - you just learned it safely!

**You're now ready for:**
- Signals (interactive workflows)
- Queries (read workflow state)
- Continue-As-New (long-running workflows)
- Child workflows (decomposition)

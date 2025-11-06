# Temporal Workflow Versioning - Quick Reference

## The Golden Rules

1. **NEVER remove workflow.GetVersion() calls** - They're permanent once deployed
2. **ALWAYS increment maxSupported** for new changes (1→2→3...)
3. **ONLY increment minSupported** after old workflows complete
4. **Order matters** - GetVersion calls must execute in the same order every time

---

## Common Patterns

### Pattern 1: Adding a New Activity

**Before:**
```go
activityA()
activityB()
```

**After (versioned):**
```go
activityA()

version := workflow.GetVersion(ctx, "add-activityC", workflow.DefaultVersion, 1)
if version == 1 {
    activityC()  // New activity
}

activityB()
```

---

### Pattern 2: Removing an Activity

**Before:**
```go
activityA()
activityB()  // Want to remove this
activityC()
```

**After (versioned):**
```go
activityA()

version := workflow.GetVersion(ctx, "remove-activityB", workflow.DefaultVersion, 1)
if version == workflow.DefaultVersion {
    activityB()  // Only for old workflows
}

activityC()
```

---

### Pattern 3: Changing Activity Logic (Reordering)

**Before:**
```go
activityA()
activityB()
```

**After (versioned):**
```go
version := workflow.GetVersion(ctx, "reverse-order", workflow.DefaultVersion, 1)

if version == 1 {
    activityB()  // New: B before A
    activityA()
} else {
    activityA()  // Old: A before B
    activityB()
}
```

---

### Pattern 4: Multiple Changes (Stacked Versions)

```go
// First change (added activityB)
v1 := workflow.GetVersion(ctx, "add-B", workflow.DefaultVersion, 1)

activityA()

if v1 == 1 {
    activityB()
}

// Second change (added activityC)
v2 := workflow.GetVersion(ctx, "add-C", workflow.DefaultVersion, 1)

if v2 == 1 {
    activityC()
}
```

---

### Pattern 5: Conditional Logic Changes

**Before:**
```go
if someCondition {
    activityA()
}
```

**After (versioned):**
```go
version := workflow.GetVersion(ctx, "change-condition", workflow.DefaultVersion, 1)

if version == 1 {
    // New condition logic
    if newCondition {
        activityA()
    }
} else {
    // Old condition logic
    if someCondition {
        activityA()
    }
}
```

---

## Version Lifecycle Example

### Phase 1: Initial Deployment (Week 0)
```go
// No versioning yet
activityA()
activityB()
```

### Phase 2: Add Feature (Week 1)
```go
activityA()

version := workflow.GetVersion(ctx, "add-C", workflow.DefaultVersion, 1)
if version == 1 {
    activityC()  // New feature
}

activityB()
```
- Old workflows (started in Week 0): skip activityC
- New workflows (started in Week 1+): run activityC

### Phase 3: Add Another Feature (Week 4)
```go
activityA()

v1 := workflow.GetVersion(ctx, "add-C", workflow.DefaultVersion, 1)
if v1 == 1 {
    activityC()
}

v2 := workflow.GetVersion(ctx, "add-D", workflow.DefaultVersion, 1)
if v2 == 1 {
    activityD()  // Another new feature
}

activityB()
```

### Phase 4: Cleanup Old Versions (Week 20 - after all Week 0 workflows done)
```go
activityA()

// Changed minSupported: DefaultVersion → 1
// All workflows now must have activityC
v1 := workflow.GetVersion(ctx, "add-C", 1, 1)
activityC()  // Always runs now

v2 := workflow.GetVersion(ctx, "add-D", workflow.DefaultVersion, 1)
if v2 == 1 {
    activityD()
}

activityB()
```
⚠️ Note: GetVersion("add-C") still exists! You can simplify the if/else, but the marker must stay.

### Phase 5: Both Features Mandatory (Week 40)
```go
activityA()

v1 := workflow.GetVersion(ctx, "add-C", 1, 1)
activityC()

v2 := workflow.GetVersion(ctx, "add-D", 1, 1)
activityD()

activityB()
```
Both GetVersion calls remain forever.

---

## Decision Tree: Do I Need Versioning?

```
Did you change workflow code?
├─ No → Safe to deploy
└─ Yes
   └─ Does it affect workflow execution path?
      ├─ No (e.g., logs, comments, activity implementation) → Safe to deploy
      └─ Yes (e.g., add/remove/reorder activities)
         └─ Are there running workflows?
            ├─ No (fresh deployment) → Safe to deploy
            └─ Yes → ✅ USE VERSIONING
```

---

## Debugging Non-Deterministic Errors

**Error Message:**
```
non-deterministic workflow error: history event is ActivityTaskScheduled
for 'GetLocation' but workflow code expected 'GetTimezone'
```

**Translation:**
- History recorded: GetLocation was next
- New code expects: GetTimezone was next
- Fix: Add GetVersion to handle both paths

**Steps:**
1. Identify where the code diverged (check the activity names)
2. Add GetVersion before that point
3. Put old code in `version == DefaultVersion` branch
4. Put new code in `version == 1` branch

---

## Testing Versioning

### Test 1: Old Workflow Completes
```bash
# Start old version
go run worker/main.go  # Version without GetVersion
go run starter/main.go

# Deploy new version (with GetVersion)
# Ctrl+C the worker
# Update code
go run worker/main.go  # New version with GetVersion

# Old workflow should complete successfully
```

### Test 2: New Workflow Uses New Code
```bash
# With versioned code running
go run starter/main.go

# Check logs: should see "Version 1: ..." messages
```

### Test 3: Multiple Versions Coexist
```bash
# Start 3 workflows with old code
# Start 3 workflows with new code
# All 6 should complete successfully
```

---

## Common Mistakes

❌ **Changing changeID**
```go
// WRONG: Changed "add-C" to "add-feature-C"
workflow.GetVersion(ctx, "add-feature-C", workflow.DefaultVersion, 1)
```
✅ **Keep changeID stable forever**

---

❌ **Decreasing maxSupported**
```go
// WRONG: Changed 2 → 1
workflow.GetVersion(ctx, "my-change", workflow.DefaultVersion, 1)
```
✅ **Only increase maxSupported**

---

❌ **Removing GetVersion**
```go
// WRONG: Deleted GetVersion after cleanup
activityC()  // Just always run it
```
✅ **Keep GetVersion forever**
```go
workflow.GetVersion(ctx, "add-C", 1, 1)  // Marker stays
activityC()
```

---

❌ **Forgetting to restart worker**
```go
// Code updated, but worker still running old code
```
✅ **Always restart worker after code changes**

---

## Advanced: Patch API (Alternative)

Temporal also offers `workflow.Patch()` - simpler but less flexible:

```go
// Instead of GetVersion
if workflow.Patch(ctx, "my-change", false) {
    newCode()
} else {
    oldCode()
}
```

**GetVersion vs Patch:**
- GetVersion: More explicit, supports multiple versions (0, 1, 2, 3...)
- Patch: Simpler, binary (patched vs not patched)
- Recommendation: Use GetVersion for learning and complex cases

---

## Real-World Scenario

**E-commerce Order Workflow:**

```go
// Week 1: Original
processPayment()
shipOrder()

// Week 3: Add fraud check
v1 := workflow.GetVersion(ctx, "fraud-check", workflow.DefaultVersion, 1)
processPayment()
if v1 == 1 {
    runFraudCheck()  // New
}
shipOrder()

// Week 10: Add gift wrapping
v1 := workflow.GetVersion(ctx, "fraud-check", workflow.DefaultVersion, 1)
processPayment()
if v1 == 1 {
    runFraudCheck()
}

v2 := workflow.GetVersion(ctx, "gift-wrap", workflow.DefaultVersion, 1)
if v2 == 1 {
    applyGiftWrap()  // New
}

shipOrder()

// Week 100: Fraud check is now mandatory (all old orders completed)
v1 := workflow.GetVersion(ctx, "fraud-check", 1, 1)  // min changed
processPayment()
runFraudCheck()  // Always runs

v2 := workflow.GetVersion(ctx, "gift-wrap", workflow.DefaultVersion, 1)
if v2 == 1 {
    applyGiftWrap()
}

shipOrder()
```

---

## Further Reading

- [Official Docs: Versioning](https://docs.temporal.io/workflows#versioning)
- [Official Docs: Workflow.GetVersion](https://pkg.go.dev/go.temporal.io/sdk/workflow#GetVersion)
- [Blog: Safe Deployment Practices](https://temporal.io/blog/workflow-versioning)

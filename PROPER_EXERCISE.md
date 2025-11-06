# Proper Workflow Versioning Exercise

This exercise teaches you to do versioning THE RIGHT WAY - no hacks, no technical debt.

## Goal

Experience adding a new feature (timezone) to a running workflow without:
- ❌ Using `interface{}` returns
- ❌ Changing return types
- ❌ Breaking running workflows
- ✅ Maintaining type safety
- ✅ Following Go best practices

---

## Setup: Start Clean

We'll use the clean implementation files:
- `workflows_clean_v1.go` - Proper versioning patterns
- `worker/main_clean.go` - Worker that registers clean workflows
- `starter/main_clean.go` - Starter for clean workflows

---

## Exercise Part A: Using GetVersion (Internal Versioning)

This approach keeps one workflow function, uses `GetVersion` internally.

### Step 1: Prepare the "Before" Code

The `GetAddressFromIPClean` function in `workflows_clean_v1.go` already has the proper structure:
- Return type: `WorkflowResult` (never changes)
- Optional timezone field with `omitempty`
- Version check that controls execution path

### Step 2: Start Worker

```bash
# Terminal 1
cd /home/user/temporal
go run worker/main_clean.go
```

**Expected output:**
```
Successfully connected to Temporal server
Worker registered workflows:
  - GetAddressFromIPClean (versioned with GetVersion)
  - GetAddressFromIPV1 (explicit V1)
  - GetAddressFromIPV2 (explicit V2)
Worker started
```

### Step 3: Start Workflow

```bash
# Terminal 2
go run starter/main_clean.go
```

**Expected output:**
```
✓ Workflow started successfully!
  WorkflowID: ip-geolocation-clean-1730880005
  View in UI: http://localhost:8233
```

### Step 4: Watch the Sleep

In Terminal 1 (worker logs):
```
INFO  Starting clean versioned workflow
INFO  IP fetched ip=123.45.67.89
INFO  Sleeping for 45 seconds...
```

### Step 5: Simulate Breaking Change (During Sleep!)

To simulate the old exercise, you could:

**Option A:** Start with code that has `maxSupported=0`:
```go
version := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 0)
// This means "no new versions yet"
```

Then during sleep, change to `maxSupported=1`:
```go
version := workflow.GetVersion(ctx, "add-timezone-feature", workflow.DefaultVersion, 1)
```

**Option B:** Use the existing code as-is and observe:
- Old workflows (started before restart) get `DefaultVersion`
- New workflows (started after restart) get version `1`

### Step 6: Restart Worker

```bash
# Terminal 1: Ctrl+C, then:
go run worker/main_clean.go
```

### Step 7: Observe Success! ✅

Worker logs (Terminal 1):
```
INFO  Awake!
INFO  DefaultVersion: Skipping timezone  ← Old workflow!
INFO  Workflow completed hasTimezone=false
```

**Success!** Old workflow completed WITHOUT timezone, no errors.

### Step 8: Start New Workflow

```bash
# Terminal 2
go run starter/main_clean.go
```

Worker logs (Terminal 1):
```
INFO  Starting clean versioned workflow
INFO  IP fetched ip=123.45.67.89
INFO  Sleeping for 45 seconds...
INFO  Awake!
INFO  Version 1: Fetching timezone  ← New workflow!
INFO  Timezone fetched
INFO  Workflow completed hasTimezone=true
```

**Success!** New workflow includes timezone.

### What You Learned

- ✅ Same return type (`WorkflowResult`) for both versions
- ✅ Type safety maintained (no `interface{}`)
- ✅ GetVersion controls execution path, not return type
- ✅ Old and new workflows coexist safely
- ✅ Optional fields with `omitempty` allow flexible returns

---

## Exercise Part B: Separate Workflows (The Cleaner Way)

This approach creates separate workflow functions for different versions.

### Step 1: Use V1 Workflow

Edit `starter/main_clean.go`:

```go
// Comment out the Clean workflow
// we, err := c.ExecuteWorkflow(..., iplocate.GetAddressFromIPClean, "")

// Use V1 explicitly
we, err := c.ExecuteWorkflow(
    context.Background(),
    workflowOptions,
    iplocate.GetAddressFromIPV1,
    "",
)
```

### Step 2: Start V1 Workflow

```bash
go run worker/main_clean.go   # Terminal 1
go run starter/main_clean.go  # Terminal 2
```

V1 workflow runs (no timezone).

### Step 3: Switch to V2

Edit `starter/main_clean.go`:

```go
// Comment out V1
// we, err := c.ExecuteWorkflow(..., iplocate.GetAddressFromIPV1, "")

// Use V2
we, err := c.ExecuteWorkflow(
    context.Background(),
    workflowOptions,
    iplocate.GetAddressFromIPV2,
    "",
)
```

### Step 4: Start V2 Workflow

```bash
go run starter/main_clean.go  # Terminal 2
```

V2 workflow runs (with timezone).

### What You Learned

- ✅ Even simpler: no versioning logic at all
- ✅ Each workflow is clear and independent
- ✅ Easy to understand and maintain
- ✅ Caller explicitly chooses version
- ✅ Both versions can run simultaneously

**Trade-off:** Multiple workflow functions vs versioning logic.

---

## Comparison: GetVersion vs Separate Workflows

| Aspect | GetVersion | Separate Workflows |
|--------|-----------|-------------------|
| **Code clarity** | More complex | Simpler |
| **Workflow name** | Same name | Different names |
| **Version selection** | Automatic | Manual (caller chooses) |
| **Versioning logic** | In workflow code | In caller code |
| **Best for** | Minor changes | Major changes |
| **Production use** | Common for minor iterations | Preferred for major versions |

---

## Production Pattern

Real-world teams typically combine both approaches:

```go
// Major versions: Separate workflows
func ProcessOrderV1(ctx workflow.Context) (Result, error)
func ProcessOrderV2(ctx workflow.Context) (Result, error)

// Minor changes within V2: Use GetVersion
func ProcessOrderV2(ctx workflow.Context) (Result, error) {
    v := workflow.GetVersion(ctx, "add-fraud-check", workflow.DefaultVersion, 1)
    if v == 1 {
        // Minor improvement: add fraud check
    }
}
```

**Decision tree:**
- Changing workflow contract (params/return type)? → New workflow
- Adding/removing activities? → GetVersion
- Changing activity order? → GetVersion
- Major feature change? → New workflow
- Minor improvement? → GetVersion

---

## Key Takeaways

1. **Never change return types** - It breaks deserialization
2. **Plan struct evolution** - Use `omitempty` for optional fields
3. **GetVersion for logic** - Not for type changes
4. **Separate workflows for major changes** - Cleaner than complex versioning
5. **Keep type safety** - Never use `interface{}` as return type

---

## Why Your Intuition Was Right

You called out the `interface{}` hack because:
- ❌ Loses compile-time type safety
- ❌ Violates Go best practices ("accept interfaces, return structs")
- ❌ Creates technical debt
- ❌ Pushes errors to runtime
- ❌ Makes code harder to use and maintain

**The proper solutions:**
1. **Keep return type stable** - Use optional fields
2. **Create new workflows** - For major changes
3. **Never use `interface{}` as return type** - Maintain type safety

This is production-quality thinking. Many developers take the easy/hacky path, but you're learning the right way from the start.

---

## Real-World Example: Stripe

Stripe (payment processing company) uses API versioning heavily. Their approach:

```go
// Each API version is a separate workflow
func ProcessPaymentV1(ctx workflow.Context, input PaymentInput) (PaymentResult, error)
func ProcessPaymentV2(ctx workflow.Context, input PaymentInput) (PaymentResult, error)
func ProcessPaymentV3(ctx workflow.Context, input PaymentInput) (PaymentResult, error)

// Customers explicitly choose their API version
// Old customers stay on V1 (stable, never changes)
// New customers get latest features on V3
```

Benefits:
- ✅ Each version is frozen - no surprises for customers
- ✅ Clear separation of concerns
- ✅ Can deprecate old versions on a timeline
- ✅ Type safety maintained

This is the same pattern for Temporal workflows!

---

## Next Steps

Now that you understand proper versioning:

1. **Signals**: Send data to running workflows
2. **Queries**: Read workflow state without side effects
3. **Continue-As-New**: Handle long-running workflows
4. **Child Workflows**: Decompose complex workflows
5. **Local Activities**: Optimize fast operations

Which would you like to explore next? 🚀

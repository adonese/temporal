package iplocate

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// MonitorConfig is the input for the monitoring workflow
type MonitorConfig struct {
	InitialIP       string
	CheckInterval   time.Duration
	MaxChecks       int // 0 = unlimited
}

// MonitorStatus represents the current state of the monitor
type MonitorStatus struct {
	State          string    // "running", "paused", "stopped"
	CurrentIP      string
	CheckInterval  time.Duration
	TotalChecks    int
	LastCheckTime  time.Time
	LastResult     string
	History        []HistoryEntry
}

// HistoryEntry records a single lookup
type HistoryEntry struct {
	Timestamp time.Time
	IP        string
	Location  string
	Error     string
}

// Signal types for controlling the workflow
type PauseSignal struct{}
type ResumeSignal struct{}
type ChangeIPSignal struct {
	NewIP string
}
type ChangeIntervalSignal struct {
	NewInterval time.Duration
}
type StopSignal struct{}

// IPMonitorWorkflow demonstrates Signals and Queries
//
// This workflow monitors an IP address periodically and can be controlled dynamically:
// - Signals: pause, resume, change-ip, change-interval, stop
// - Queries: status, history, stats
func IPMonitorWorkflow(ctx workflow.Context, config MonitorConfig) error {
	logger := workflow.GetLogger(ctx)

	// State that can be modified via signals
	currentIP := config.InitialIP
	checkInterval := config.CheckInterval
	isPaused := false
	shouldStop := false
	totalChecks := 0
	history := []HistoryEntry{}
	lastCheckTime := time.Time{}
	lastResult := ""

	// Setup activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Setup signal channels
	pauseChan := workflow.GetSignalChannel(ctx, "pause")
	resumeChan := workflow.GetSignalChannel(ctx, "resume")
	changeIPChan := workflow.GetSignalChannel(ctx, "change-ip")
	changeIntervalChan := workflow.GetSignalChannel(ctx, "change-interval")
	stopChan := workflow.GetSignalChannel(ctx, "stop")

	// Setup query handlers (read-only state inspection)
	err := workflow.SetQueryHandler(ctx, "status", func() (MonitorStatus, error) {
		state := "running"
		if isPaused {
			state = "paused"
		} else if shouldStop {
			state = "stopped"
		}

		return MonitorStatus{
			State:         state,
			CurrentIP:     currentIP,
			CheckInterval: checkInterval,
			TotalChecks:   totalChecks,
			LastCheckTime: lastCheckTime,
			LastResult:    lastResult,
			History:       history,
		}, nil
	})
	if err != nil {
		return err
	}

	err = workflow.SetQueryHandler(ctx, "history", func() ([]HistoryEntry, error) {
		return history, nil
	})
	if err != nil {
		return err
	}

	err = workflow.SetQueryHandler(ctx, "stats", func() (map[string]interface{}, error) {
		return map[string]interface{}{
			"total_checks":    totalChecks,
			"current_ip":      currentIP,
			"is_paused":       isPaused,
			"check_interval":  checkInterval.String(),
			"last_check_time": lastCheckTime.Format(time.RFC3339),
		}, nil
	})
	if err != nil {
		return err
	}

	logger.Info("IP Monitor started",
		"ip", currentIP,
		"interval", checkInterval,
		"max_checks", config.MaxChecks)

	// Main monitoring loop
	for {
		// Check if we should stop
		if shouldStop {
			logger.Info("Monitor stopped by signal", "total_checks", totalChecks)
			break
		}

		// Check max checks limit
		if config.MaxChecks > 0 && totalChecks >= config.MaxChecks {
			logger.Info("Max checks reached, stopping", "total_checks", totalChecks)
			break
		}

		// Process signals (non-blocking)
		selector := workflow.NewSelector(ctx)

		// Add signal handlers
		selector.AddReceive(pauseChan, func(c workflow.ReceiveChannel, more bool) {
			var sig PauseSignal
			c.Receive(ctx, &sig)
			isPaused = true
			logger.Info("Monitor paused")
		})

		selector.AddReceive(resumeChan, func(c workflow.ReceiveChannel, more bool) {
			var sig ResumeSignal
			c.Receive(ctx, &sig)
			isPaused = false
			logger.Info("Monitor resumed")
		})

		selector.AddReceive(changeIPChan, func(c workflow.ReceiveChannel, more bool) {
			var sig ChangeIPSignal
			c.Receive(ctx, &sig)
			logger.Info("Changing monitored IP", "old_ip", currentIP, "new_ip", sig.NewIP)
			currentIP = sig.NewIP
		})

		selector.AddReceive(changeIntervalChan, func(c workflow.ReceiveChannel, more bool) {
			var sig ChangeIntervalSignal
			c.Receive(ctx, &sig)
			logger.Info("Changing check interval", "old", checkInterval, "new", sig.NewInterval)
			checkInterval = sig.NewInterval
		})

		selector.AddReceive(stopChan, func(c workflow.ReceiveChannel, more bool) {
			var sig StopSignal
			c.Receive(ctx, &sig)
			logger.Info("Stop signal received")
			shouldStop = true
		})

		// Add timer for next check
		timer := workflow.NewTimer(ctx, checkInterval)
		selector.AddFuture(timer, func(f workflow.Future) {
			// Timer fired, do nothing (will proceed to check)
		})

		// Wait for either a signal or timer (non-blocking)
		selector.Select(ctx)

		// Skip check if paused
		if isPaused {
			logger.Info("Skipping check - monitor is paused")
			continue
		}

		// Perform the IP lookup
		logger.Info("Performing IP check", "ip", currentIP, "check_number", totalChecks+1)

		var ipActivities *IPActivities
		var location string
		checkTime := workflow.Now(ctx)

		err := workflow.ExecuteActivity(ctx, ipActivities.GetLocationInfo, currentIP).Get(ctx, &location)

		entry := HistoryEntry{
			Timestamp: checkTime,
			IP:        currentIP,
		}

		if err != nil {
			logger.Error("Failed to get location", "error", err)
			entry.Error = err.Error()
			lastResult = fmt.Sprintf("ERROR: %v", err)
		} else {
			logger.Info("Location retrieved", "location", location)
			entry.Location = location
			lastResult = location
		}

		// Update state
		history = append(history, entry)
		totalChecks++
		lastCheckTime = checkTime

		// Keep history bounded (last 50 entries)
		if len(history) > 50 {
			history = history[len(history)-50:]
		}
	}

	return nil
}

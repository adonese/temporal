package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"temporal-ip-geolocation/iplocate"
	"time"

	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// This demo showcases Temporal's Signals and Queries
//
// Signals: Send data to a running workflow (pause, resume, change IP, stop)
// Queries: Read workflow state without side effects (status, history, stats)
//
// Run this to see dynamic workflow control in action!

func main() {
	// Connect to Temporal server
	c, err := client.Dial(client.Options{
		HostPort:  "127.0.0.1:7233",
		Namespace: "default",
		ConnectionOptions: client.ConnectionOptions{
			TLS: nil,
			DialOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			},
		},
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Start the monitoring workflow
	workflowID := "ip-monitor-demo-" + fmt.Sprint(time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: iplocate.TaskQueueName,
	}

	config := iplocate.MonitorConfig{
		InitialIP:     "8.8.8.8", // Google DNS
		CheckInterval: 5 * time.Second,
		MaxChecks:     0, // unlimited, will stop via signal
	}

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, iplocate.IPMonitorWorkflow, config)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│      Temporal Signals & Queries Demo - IP Monitor          │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("✓ Workflow started!")
	fmt.Printf("  WorkflowID: %s\n", we.GetID())
	fmt.Printf("  RunID: %s\n", we.GetRunID())
	fmt.Printf("  View in UI: http://localhost:8233\n\n")

	// Give workflow time to do first check
	fmt.Println("⏳ Waiting 3 seconds for first check...")
	time.Sleep(3 * time.Second)

	// QUERY #1: Get initial status
	fmt.Println("\n📊 QUERY: Getting initial status...")
	queryStatus(ctx, c, workflowID)

	// Wait for another check
	fmt.Println("\n⏳ Letting it run for 6 seconds (should do 1-2 more checks)...")
	time.Sleep(6 * time.Second)

	// QUERY #2: Check history
	fmt.Println("\n📊 QUERY: Checking history...")
	queryHistory(ctx, c, workflowID)

	// SIGNAL #1: Pause the monitor
	fmt.Println("\n⚡ SIGNAL: Sending PAUSE signal...")
	err = c.SignalWorkflow(ctx, workflowID, "", "pause", iplocate.PauseSignal{})
	if err != nil {
		log.Printf("Failed to send pause signal: %v\n", err)
	} else {
		fmt.Println("   ✓ Pause signal sent")
	}

	time.Sleep(2 * time.Second)

	// QUERY #3: Verify it's paused
	fmt.Println("\n📊 QUERY: Verifying paused state...")
	queryStatus(ctx, c, workflowID)

	fmt.Println("\n⏳ Waiting 6 seconds (monitor is paused, no new checks should happen)...")
	time.Sleep(6 * time.Second)

	// QUERY #4: Confirm no new checks
	fmt.Println("\n📊 QUERY: Confirming no new checks while paused...")
	queryStats(ctx, c, workflowID)

	// SIGNAL #2: Change the IP address
	fmt.Println("\n⚡ SIGNAL: Changing monitored IP to 1.1.1.1 (Cloudflare DNS)...")
	err = c.SignalWorkflow(ctx, workflowID, "", "change-ip", iplocate.ChangeIPSignal{
		NewIP: "1.1.1.1",
	})
	if err != nil {
		log.Printf("Failed to send change-ip signal: %v\n", err)
	} else {
		fmt.Println("   ✓ Change-IP signal sent")
	}

	time.Sleep(1 * time.Second)

	// SIGNAL #3: Resume
	fmt.Println("\n⚡ SIGNAL: Sending RESUME signal...")
	err = c.SignalWorkflow(ctx, workflowID, "", "resume", iplocate.ResumeSignal{})
	if err != nil {
		log.Printf("Failed to send resume signal: %v\n", err)
	} else {
		fmt.Println("   ✓ Resume signal sent")
	}

	fmt.Println("\n⏳ Waiting 7 seconds for checks with new IP...")
	time.Sleep(7 * time.Second)

	// QUERY #5: See the new IP in action
	fmt.Println("\n📊 QUERY: Checking history (should show new IP)...")
	queryHistory(ctx, c, workflowID)

	// SIGNAL #4: Change interval
	fmt.Println("\n⚡ SIGNAL: Changing interval to 3 seconds (faster checks)...")
	err = c.SignalWorkflow(ctx, workflowID, "", "change-interval", iplocate.ChangeIntervalSignal{
		NewInterval: 3 * time.Second,
	})
	if err != nil {
		log.Printf("Failed to send change-interval signal: %v\n", err)
	} else {
		fmt.Println("   ✓ Change-Interval signal sent")
	}

	fmt.Println("\n⏳ Waiting 10 seconds (should see faster checks)...")
	time.Sleep(10 * time.Second)

	// QUERY #6: Final stats
	fmt.Println("\n📊 QUERY: Final statistics...")
	queryStats(ctx, c, workflowID)

	// SIGNAL #5: Stop the workflow gracefully
	fmt.Println("\n⚡ SIGNAL: Sending STOP signal...")
	err = c.SignalWorkflow(ctx, workflowID, "", "stop", iplocate.StopSignal{})
	if err != nil {
		log.Printf("Failed to send stop signal: %v\n", err)
	} else {
		fmt.Println("   ✓ Stop signal sent")
	}

	fmt.Println("\n⏳ Waiting for workflow to complete...")
	err = we.Get(ctx, nil)
	if err != nil {
		log.Printf("Workflow execution error: %v\n", err)
	} else {
		fmt.Println("✓ Workflow completed gracefully!")
	}

	fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│                     Demo Complete!                          │")
	fmt.Println("│                                                             │")
	fmt.Println("│  You just saw:                                              │")
	fmt.Println("│  ✓ Signals: pause, resume, change-ip, change-interval, stop │")
	fmt.Println("│  ✓ Queries: status, history, stats                         │")
	fmt.Println("│  ✓ Dynamic workflow control without restarts!              │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

func queryStatus(ctx context.Context, c client.Client, workflowID string) {
	var status iplocate.MonitorStatus
	val, err := c.QueryWorkflow(ctx, workflowID, "", "status")
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		return
	}
	err = val.Get(&status)
	if err != nil {
		log.Printf("Failed to decode status: %v\n", err)
		return
	}

	fmt.Printf("   State: %s\n", status.State)
	fmt.Printf("   Current IP: %s\n", status.CurrentIP)
	fmt.Printf("   Check Interval: %s\n", status.CheckInterval)
	fmt.Printf("   Total Checks: %d\n", status.TotalChecks)
	if !status.LastCheckTime.IsZero() {
		fmt.Printf("   Last Check: %s\n", status.LastCheckTime.Format(time.RFC3339))
		fmt.Printf("   Last Result: %s\n", status.LastResult)
	}
}

func queryHistory(ctx context.Context, c client.Client, workflowID string) {
	var history []iplocate.HistoryEntry
	val, err := c.QueryWorkflow(ctx, workflowID, "", "history")
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		return
	}
	err = val.Get(&history)
	if err != nil {
		log.Printf("Failed to decode history: %v\n", err)
		return
	}

	fmt.Printf("   Total entries: %d\n", len(history))
	if len(history) > 0 {
		fmt.Println("   Recent checks:")
		// Show last 3 entries
		start := len(history) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(history); i++ {
			entry := history[i]
			status := "✓"
			result := entry.Location
			if entry.Error != "" {
				status = "✗"
				result = entry.Error
			}
			fmt.Printf("     %s [%s] IP: %s → %s\n",
				status,
				entry.Timestamp.Format("15:04:05"),
				entry.IP,
				result)
		}
	}
}

func queryStats(ctx context.Context, c client.Client, workflowID string) {
	var stats map[string]interface{}
	val, err := c.QueryWorkflow(ctx, workflowID, "", "stats")
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		return
	}
	err = val.Get(&stats)
	if err != nil {
		log.Printf("Failed to decode stats: %v\n", err)
		return
	}

	jsonData, _ := json.MarshalIndent(stats, "   ", "  ")
	fmt.Printf("   %s\n", string(jsonData))
}

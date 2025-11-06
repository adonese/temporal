package main

import (
	"context"
	"fmt"
	"log"
	"temporal-ip-geolocation/iplocate"
	"time"

	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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

	// Workflow ID strategy:
	// - Unique IDs (timestamp/UUID): Each execution is independent
	// - Constant IDs: Ensures idempotency, prevents duplicate executions
	// - Entity-based IDs: One workflow per business entity (e.g., "user-123")
	//
	// For this example, we use timestamp for unique executions.
	// In production, consider: "ip-lookup-" + requestID for idempotency
	workflowOptions := client.StartWorkflowOptions{
		ID:        "ip-geolocation-workflow-" + fmt.Sprint(time.Now().Unix()),
		TaskQueue: iplocate.TaskQueueName,
		// StartDelay: 10 * time.Second,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, iplocate.GetAddressFromIPV2, "")
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Println("✓ Workflow started successfully!")
	log.Println("  WorkflowID:", we.GetID())
	log.Println("  RunID:", we.GetRunID())
	log.Println("  View in UI: http://localhost:8233")
	log.Println("\nWorkflow executing in background. Starter exiting...")
}

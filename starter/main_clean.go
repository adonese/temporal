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

// This starter demonstrates the clean versioning approach
// You can choose which workflow to use

func main() {
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

	workflowOptions := client.StartWorkflowOptions{
		ID:        "ip-geolocation-clean-" + fmt.Sprint(time.Now().Unix()),
		TaskQueue: iplocate.TaskQueueName,
	}

	// OPTION 1: Use the versioned workflow (GetVersion approach)
	// This workflow uses workflow.GetVersion() internally
	we, err := c.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		iplocate.GetAddressFromIPClean,
		"",
	)

	// OPTION 2: Use V1 workflow explicitly (separate workflow approach)
	// Uncomment to use this approach:
	// we, err := c.ExecuteWorkflow(
	// 	context.Background(),
	// 	workflowOptions,
	// 	iplocate.GetAddressFromIPV1,
	// 	"",
	// )

	// OPTION 3: Use V2 workflow explicitly (with timezone)
	// Uncomment to use this approach:
	// we, err := c.ExecuteWorkflow(
	// 	context.Background(),
	// 	workflowOptions,
	// 	iplocate.GetAddressFromIPV2,
	// 	"",
	// )

	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Println("✓ Workflow started successfully!")
	log.Println("  WorkflowID:", we.GetID())
	log.Println("  RunID:", we.GetRunID())
	log.Println("  View in UI: http://localhost:8233")
	log.Println("\nWorkflow executing in background. Starter exiting...")
}

package main

import (
	"context"
	"fmt"
	"log"
	"temporal-ip-geolocation/iplocate"

	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Starts a workflow that QUERIES another workflow
// The "aha moment": workflow-to-workflow communication!

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

	targetWorkflowID := "ip-lookup-observable"

	workflowOptions := client.StartWorkflowOptions{
		ID:        "status-checker",
		TaskQueue: iplocate.TaskQueueName,
	}

	fmt.Println("Starting StatusCheckerWorkflow...")
	fmt.Printf("Target: %s\n\n", targetWorkflowID)

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, iplocate.StatusCheckerWorkflow, targetWorkflowID)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	// Wait for result
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Workflow failed", err)
	}

	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("  RESULT FROM QUERYING ANOTHER WORKFLOW:")
	fmt.Printf("  %s\n", result)
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("\n✓ One workflow just queried another workflow!")
}

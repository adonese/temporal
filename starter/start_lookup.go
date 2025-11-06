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

// Starts the slow IP lookup workflow
// This workflow exposes query handlers so others can check its status

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

	workflowID := "ip-lookup-observable"

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: iplocate.TaskQueueName,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, iplocate.IPLookupWorkflow, "8.8.8.8")
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	fmt.Println("✓ IPLookupWorkflow started!")
	fmt.Printf("  WorkflowID: %s\n", we.GetID())
	fmt.Println("\nThis workflow runs for ~30 seconds.")
	fmt.Println("While it's running, start the status checker:")
	fmt.Println("  go run starter/check_status.go")
}

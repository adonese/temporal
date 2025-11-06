package main

import (
	"log"
	"net/http"
	"temporal-ip-geolocation/iplocate"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Clean worker that registers all workflow versions
// This demonstrates proper workflow registration

func main() {
	log.Println("Attempting to connect to Temporal server at: 127.0.0.1:7233")
	c, err := client.Dial(client.Options{
		HostPort:  "127.0.0.1:7233",
		Namespace: "default",
	})
	if err != nil {
		log.Fatalln("error in dialing: ", err)
	}
	defer c.Close()
	log.Println("Successfully connected to Temporal server")

	w := worker.New(c, iplocate.TaskQueueName, worker.Options{})

	// Register activities (same for all workflow versions)
	activities := &iplocate.IPActivities{
		HTTPClient: http.DefaultClient,
	}
	w.RegisterActivity(activities)

	// Register all workflow versions
	// All versions must be registered to support old running workflows

	// Clean versioned workflow (uses GetVersion internally)
	w.RegisterWorkflow(iplocate.GetAddressFromIPClean)

	// Separate workflow versions (alternative approach)
	w.RegisterWorkflow(iplocate.GetAddressFromIPV1)
	w.RegisterWorkflow(iplocate.GetAddressFromIPV2)

	// Also register the original workflow for backwards compatibility
	w.RegisterWorkflow(iplocate.GetAddressFromIP)

	log.Println("Worker registered workflows:")
	log.Println("  - GetAddressFromIPClean (versioned with GetVersion)")
	log.Println("  - GetAddressFromIPV1 (explicit V1)")
	log.Println("  - GetAddressFromIPV2 (explicit V2)")
	log.Println("  - GetAddressFromIP (original)")

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start temporal worker", err)
	}
}

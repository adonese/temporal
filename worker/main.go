package main

import (
	"log"
	"net/http"
	"temporal-ip-geolocation/iplocate"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	log.Println("Attempting to connect to Temporal server at: 127.0.0.1:7233")
	c, err := client.Dial(client.Options{
		HostPort:  "127.0.0.1:7233",
		Namespace: "default",
		// ConnectionOptions: client.ConnectionOptions{
		// 	TLS: nil, // Use insecure connection for local dev
		// 	DialOptions: []grpc.DialOption{
		// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// 	},
		// },
	})
	if err != nil {
		log.Fatalln("error in dialing: ", err)
	}
	defer c.Close()
	log.Println("Successfully connected to Temporal server")

	w := worker.New(c, iplocate.TaskQueueName, worker.Options{})

	activities := &iplocate.IPActivities{
		HTTPClient: http.DefaultClient,
	}
	w.RegisterWorkflow(iplocate.GetAddressFromIP)
	w.RegisterActivity(activities)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start temporal worker", err)
	}
}

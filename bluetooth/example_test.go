package bluetooth_test

import (
	"context"
	"log"
	"time"

	meshcore_bluetooth "github.com/kellegous/meshcore/bluetooth"
	"tinygo.org/x/bluetooth"
)

// Connect to a specific device by name over bluetooth.
func ExampleClient_LookupDevice() {
	client, err := meshcore_bluetooth.NewClient(bluetooth.DefaultAdapter)
	if err != nil {
		log.Fatal(err)
	}

	device, err := client.LookupDevice(context.Background(), "MeshCore-1234567890")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := client.Connect(context.Background(), device.Address)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Disconnect()
}

// Discover MeshCore devices over bluetooth.
func ExampleClient_DiscoverDevices() {
	client, err := meshcore_bluetooth.NewClient(bluetooth.DefaultAdapter)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var addr *bluetooth.Address
	for device, err := range client.DiscoverDevices(ctx) {
		if err != nil {
			log.Fatal(err)
		}

		addr = &device.Address
	}
	if addr == nil {
		log.Fatal("no device found")
	}

	conn, err := client.Connect(ctx, *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Disconnect()
}

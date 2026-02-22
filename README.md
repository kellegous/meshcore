# Meshcore Companion Radio in Go

This module is the Go analog to libraries like [meshcore.js](https://github.com/meshcore-dev/meshcore.js) and [meshcore_py](https://github.com/meshcore-dev/meshcore_py). This allows you to interact with [MeshCore](https://meshcore.co.uk/) companion radio devices over Bluetooth or USB/Serial.

## Installation

```bash
go get github.com/kellegous/meshcore@latest
```

## Quick Start

### Sending a text message to a contact:

[example]: # "example_test.go:ExampleConn_SendTextMessage"

```go
import (
	"context"
	"fmt"
	"log"
	"github.com/kellegous/meshcore"
	"github.com/kellegous/meshcore/serial"
)

// Send a text message to a contact.
ctx := context.Background()

conn, err := serial.Connect(context.Background(), "/dev/cu.usbserial-0001")
if err != nil {
	log.Fatal(err)
}
defer conn.Disconnect()

contacts, err := conn.GetContacts(ctx, nil)
if err != nil {
	log.Fatal(err)
}
if len(contacts) == 0 {
	log.Fatal("no contacts found")
}
contact := contacts[0]

sr, err := conn.SendTextMessage(
	ctx,
	&contact.PublicKey,
	"Hello, world!",
	meshcore.TextTypePlain,
)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("sent message: %+v\n", sr)
```

### Connecting to a device by name over Bluetooth:

[example]: # "bluetooth/example_test.go:ExampleClient_LookupDevice"

```go
import (
	"context"
	"log"
	meshcore_bluetooth "github.com/kellegous/meshcore/bluetooth"
	"tinygo.org/x/bluetooth"
)

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
```

### Discovering & connecting to a device over Bluetooth:

[example]: # "bluetooth/example_test.go:ExampleClient_DiscoverDevices"

```go
import (
	"context"
	"log"
	"time"
	meshcore_bluetooth "github.com/kellegous/meshcore/bluetooth"
	"tinygo.org/x/bluetooth"
)

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
```

## Authors

- [@kellegous](https://github.com/kellegous)

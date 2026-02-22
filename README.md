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

### Enforcing timeouts

An important note on timeouts. Most commands in the MeshCore companion radio protocol work by issuing a command and then waiting for a response to return from the device. Many common failures result in the device never responding to the command. Thus, **you should always use a context with a timeout when calling any of the methods on `Conn`** (`Notifications` is an exception to this rule since it returns an iterator). A `context.Context` is the proper way to do timeouts in Go, so this library assumes the callers will setup a reasonable timeout.

[example]: # "example_test.go:ExampleConn"

```go
import (
	"context"
	"fmt"
	"log"
	"time"
)

// Use context.WithTimeout for timeouts.
ctx, done := context.WithTimeout(ctx, 10*time.Second)
defer done()

status, err := conn.GetStatus(ctx, contact.PublicKey)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("status: %+v\n", status)
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

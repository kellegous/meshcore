package meshcore_test

import (
	"context"
	"fmt"
	"log"

	"github.com/kellegous/meshcore"
	"github.com/kellegous/meshcore/serial"
)

func ExampleConn_SendTextMessage() {
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
}

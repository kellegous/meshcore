package meshcore

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/kellegous/poop"
)

type Transport interface {
	io.Writer
	Disconnect() error
	Notifier() *Notifier
}

type Conn struct {
	tx Transport
}

func NewConnection(tx Transport) *Conn {
	return &Conn{
		tx: tx,
	}
}

func (c *Conn) Disconnect() error {
	return c.tx.Disconnect()
}

type GetContactsOptions struct {
	Since time.Time
}

// AddOrUpdateContact adds or updates a contact on the device.
func (c *Conn) AddOrUpdateContact(ctx context.Context, contact *Contact) error {
	notifier := c.tx.Notifier()

	var err error

	ch := make(chan struct{})

	unsubOk := notifier.Subscribe(ResponseOk, func(data []byte) {
		close(ch)
	})
	defer unsubOk()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data[1:])
		close(ch)
	})
	defer unsubErr()

	var buf bytes.Buffer
	if _, err := buf.Write([]byte{byte(CommandAddUpdateContact)}); err != nil {
		return poop.Chain(err)
	}

	if err := contact.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}

	if _, err := c.tx.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetContacts returns the list of contacts from the device.
func (c *Conn) GetContacts(ctx context.Context, opts *GetContactsOptions) ([]*Contact, error) {
	notifier := c.tx.Notifier()

	if opts == nil {
		opts = &GetContactsOptions{}
	}

	var buf bytes.Buffer

	if _, err := buf.Write([]byte{byte(CommandGetContacts)}); err != nil {
		return nil, poop.Chain(err)
	}

	if !opts.Since.IsZero() {
		if err := binary.Write(&buf, binary.LittleEndian, uint32(opts.Since.Unix())); err != nil {
			return nil, poop.Chain(err)
		}
	}

	ch := make(chan []byte)
	unsubContact := notifier.Subscribe(ResponseContact, func(data []byte) {
		ch <- data
	})
	defer func() {
		unsubContact()
		// discard any pending frames on the channel
		select {
		case <-ch:
		default:
		}
	}()

	unsubEndOfContacts := notifier.Subscribe(ResponseEndOfContacts, func(data []byte) {
		close(ch)
	})
	defer unsubEndOfContacts()

	if _, err := c.tx.Write(buf.Bytes()); err != nil {
		return nil, poop.Chain(err)
	}

	var contacts []*Contact
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return contacts, nil
			}
			var contact Contact
			if err := contact.readFrom(bytes.NewReader(data[1:])); err != nil {
				return nil, poop.Chain(err)
			}
			contacts = append(contacts, &contact)
		case <-ctx.Done():
			return contacts, ctx.Err()
		}
	}
}

// GetDeviceTime returns the current device time.
func (c *Conn) GetDeviceTime(ctx context.Context) (time.Time, error) {
	notifier := c.tx.Notifier()

	var t time.Time
	var err error

	ch := make(chan struct{})

	unsubTime := notifier.Subscribe(ResponseCurrTime, func(data []byte) {
		t, err = readTime(bytes.NewReader(data[1:]))
		close(ch)
	})
	defer unsubTime()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data[1:])
		close(ch)
	})
	defer unsubErr()

	if _, err := c.tx.Write([]byte{byte(CommandGetDeviceTime)}); err != nil {
		return time.Time{}, poop.Chain(err)
	}

	select {
	case <-ch:
		return t, err
	case <-ctx.Done():
		return time.Time{}, ctx.Err()
	}
}

// GetBatteryVoltage returns the current battery voltage in millivolts.
func (c *Conn) GetBatteryVoltage(ctx context.Context) (uint16, error) {
	notifier := c.tx.Notifier()

	var voltage uint16
	var err error

	ch := make(chan struct{})

	unsubVoltage := notifier.Subscribe(ResponseBatteryVoltage, func(data []byte) {
		err = binary.Read(bytes.NewReader(data[1:]), binary.LittleEndian, &voltage)
		close(ch)
	})
	defer unsubVoltage()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data[1:])
		close(ch)
	})
	defer unsubErr()

	if _, err := c.tx.Write([]byte{byte(CommandGetBatteryVoltage)}); err != nil {
		return 0, poop.Chain(err)
	}

	select {
	case <-ch:
		return voltage, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// SendTextMessage sends a text message to the recipient.
func (c *Conn) SendTextMessage(
	ctx context.Context,
	recipient *PublicKey,
	message string,
	textType TextType,
) (*SentResponse, error) {
	notifier := c.tx.Notifier()

	var sr SentResponse
	var err error

	ch := make(chan struct{})

	unsubSent := notifier.Subscribe(ResponseSent, func(data []byte) {
		err = sr.readFrom(bytes.NewReader(data[1:]))
		close(ch)
	})
	defer unsubSent()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data[1:])
		close(ch)
	})
	defer unsubErr()

	var buf bytes.Buffer
	var attempt byte
	sendTime := uint32(time.Now().Unix())

	if _, err := buf.Write([]byte{byte(CommandSendTxtMsg)}); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Write(&buf, binary.LittleEndian, byte(textType)); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Write(&buf, binary.LittleEndian, attempt); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Write(&buf, binary.LittleEndian, sendTime); err != nil {
		return nil, poop.Chain(err)
	}

	if _, err := buf.Write(recipient.Prefix(6)); err != nil {
		return nil, poop.Chain(err)
	}

	if _, err := buf.Write([]byte(message)); err != nil {
		return nil, poop.Chain(err)
	}

	if _, err := c.tx.Write(buf.Bytes()); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return &sr, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

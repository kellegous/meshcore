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
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeAddOrUpdateContactCommand(c.tx, contact); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RemoveContact removes a contact from the device.
func (c *Conn) RemoveContact(ctx context.Context, key *PublicKey) error {
	notifier := c.tx.Notifier()

	var err error

	ch := make(chan struct{})

	unsubOk := notifier.Subscribe(ResponseOk, func(data []byte) {
		close(ch)
	})
	defer unsubOk()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeRemoveContactCommand(c.tx, key); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type GetContactsOptions struct {
	Since time.Time
}

// GetContacts returns the list of contacts from the device.
func (c *Conn) GetContacts(ctx context.Context, opts *GetContactsOptions) ([]*Contact, error) {
	notifier := c.tx.Notifier()

	if opts == nil {
		opts = &GetContactsOptions{}
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

	if err := writeGetContactsCommand(c.tx, opts.Since); err != nil {
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
			if err := contact.readFrom(bytes.NewReader(data)); err != nil {
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
		t, err = readTime(bytes.NewReader(data))
		close(ch)
	})
	defer unsubTime()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeCommandCode(c.tx, CommandGetDeviceTime); err != nil {
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
		err = binary.Read(bytes.NewReader(data), binary.LittleEndian, &voltage)
		close(ch)
	})
	defer unsubVoltage()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeCommandCode(c.tx, CommandGetBatteryVoltage); err != nil {
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
		err = sr.readFrom(bytes.NewReader(data))
		close(ch)
	})
	defer unsubSent()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeSendTextMessageCommand(
		c.tx,
		recipient,
		message,
		textType,
		0, // attempt
		time.Now(),
	); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return &sr, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

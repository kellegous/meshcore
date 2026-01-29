package meshcore

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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

// SendChannelTextMessage sends a text message to the given channel.
func (c *Conn) SendChannelTextMessage(
	ctx context.Context,
	channelIndex byte,
	message string,
	textType TextType,
) error {
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

	if err := writeSendChannelTextMessageCommand(
		c.tx,
		channelIndex,
		message,
		textType,
		time.Now(),
	); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetTelemetry returns the telemetry data for the given contact key.
func (c *Conn) GetTelemetry(
	ctx context.Context,
	key *PublicKey,
) (*TelemetryResponse, error) {
	notifier := c.tx.Notifier()

	var telemetry TelemetryResponse
	var err error

	ch := make(chan struct{})

	// TODO(kellegous): Why is this a push event?
	unsubTelemetry := notifier.Subscribe(PushTelemetryResponse, func(data []byte) {
		err = telemetry.readFrom(bytes.NewReader(data))
		close(ch)
	})
	defer unsubTelemetry()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeGetTelemetryCommand(c.tx, key); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		if err != nil {
			return nil, poop.Chain(err)
		}

		if !bytes.Equal(key.Prefix(6), telemetry.pubKeyPrefix[:]) {
			return nil, poop.New("telemetry response is not for the given contact key")
		}

		return &telemetry, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetChannel returns the channel information for the given index.
func (c *Conn) GetChannel(
	ctx context.Context,
	idx uint8,
) (*ChannelInfo, error) {
	notifier := c.tx.Notifier()

	var channel ChannelInfo
	var err error

	ch := make(chan struct{})

	unsubChannel := notifier.Subscribe(ResponseChannelInfo, func(data []byte) {
		err = channel.readFrom(bytes.NewReader(data))
		close(ch)
	})
	defer unsubChannel()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeGetChannelCommand(c.tx, idx); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return &channel, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetChannels returns the list of all channels.
func (c *Conn) GetChannels(
	ctx context.Context,
) ([]*ChannelInfo, error) {
	var channels []*ChannelInfo

	for i := uint8(0); ; i++ {
		channel, err := c.GetChannel(ctx, i)
		if hasErrorCode(err, ErrorCodeNotFound) {
			break
		} else if err != nil {
			return nil, poop.Chain(err)
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// SetChannel sets or updates a channel on the device.
func (c *Conn) SetChannel(ctx context.Context, channel *ChannelInfo) error {
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

	if err := writeSetChannelCommand(c.tx, channel); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DeleteChannel deletes a channel from the device.
func (c *Conn) DeleteChannel(ctx context.Context, idx uint8) error {
	var secret [16]byte
	return c.SetChannel(ctx, &ChannelInfo{
		Index:  idx,
		Name:   "",
		Secret: secret[:],
	})
}

// DeviceQuery queries the device information.
func (c *Conn) DeviceQuery(ctx context.Context, appTargetVer byte) (*DeviceInfo, error) {
	notifier := c.tx.Notifier()

	var deviceInfo DeviceInfo
	var err error

	ch := make(chan struct{})

	unsubDeviceInfo := notifier.Subscribe(ResponseDeviceInfo, func(data []byte) {
		err = deviceInfo.readFrom(bytes.NewReader(data))
		close(ch)
	})
	defer unsubDeviceInfo()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeDeviceQueryCommand(c.tx, appTargetVer); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return &deviceInfo, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Reboot reboots the device.
func (c *Conn) Reboot(ctx context.Context) error {
	var rErr *ResponseError
	if err := writeRebootCommand(c.tx); err != nil {
		// Only return an error if we get a response error. In the
		// common case, this will timeout on writing the command and
		// we'll ignore the timeout error coming from the underlying
		// transport.
		if errors.As(err, &rErr) {
			return poop.Chain(rErr)
		}
	}
	return nil
}

// SyncNextMessage synchronizes the next message from the device.
func (c *Conn) SyncNextMessage(ctx context.Context) (Message, error) {
	notifier := c.tx.Notifier()

	var message Message
	var err error

	ch := make(chan struct{})

	unsubContactMessage := notifier.Subscribe(ResponseContactMsgRecv, func(data []byte) {
		var contactMessage ContactMessage
		err = contactMessage.readFrom(bytes.NewReader(data))
		if err == nil {
			message = &contactMessage
		}
		close(ch)
	})
	defer unsubContactMessage()

	unsubChannelMessage := notifier.Subscribe(ResponseChannelMsgRecv, func(data []byte) {
		var channelMessage ChannelMessage
		err = channelMessage.readFrom(bytes.NewReader(data))
		if err == nil {
			message = &channelMessage
		}
		close(ch)
	})
	defer unsubChannelMessage()

	unsubNoMoreMessages := notifier.Subscribe(ResponseNoMoreMessages, func(data []byte) {
		close(ch)
	})
	defer unsubNoMoreMessages()

	if err := writeCommandCode(c.tx, CommandSyncNextMessage); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return message, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SendAdvert sends an advert to the device.
func (c *Conn) SendAdvert(ctx context.Context, advertType SelfAdvertType) error {
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

	if err := writeSendAdvertCommand(c.tx, advertType); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ExportContact exports a contact from the device. if key is nil, the
// device's self contact is exported.
func (c *Conn) ExportContact(ctx context.Context, key *PublicKey) ([]byte, error) {
	notifier := c.tx.Notifier()

	var advertPacket []byte
	var err error

	ch := make(chan struct{})

	unsubExportContact := notifier.Subscribe(ResponseExportContact, func(data []byte) {
		// TODO(kellegous): not sure if a copy is needed here.
		advertPacket = make([]byte, len(data))
		copy(advertPacket, data)
		close(ch)
	})
	defer unsubExportContact()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeExportContactCommand(c.tx, key); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return advertPacket, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ImportContact imports a contact into the device.
func (c *Conn) ImportContact(ctx context.Context, advertPacket []byte) error {
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

	if err := writeImportContactCommand(c.tx, advertPacket); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ExportPrivateKey exports the private key from the device.
func (c *Conn) ExportPrivateKey(ctx context.Context) ([]byte, error) {
	notifier := c.tx.Notifier()

	var privateKey [64]byte
	var err error

	ch := make(chan struct{})

	unsubPrivateKey := notifier.Subscribe(ResponsePrivateKey, func(data []byte) {
		_, err = io.ReadFull(bytes.NewReader(data), privateKey[:])
		close(ch)
	})
	defer unsubPrivateKey()

	unsubDisabled := notifier.Subscribe(ResponseDisabled, func(data []byte) {
		err = poop.New("private key is disabled")
		close(ch)
	})
	defer unsubDisabled()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeCommandCode(c.tx, CommandExportPrivateKey); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return privateKey[:], err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ImportPrivateKey imports a private key into the device.
func (c *Conn) ImportPrivateKey(ctx context.Context, privateKey []byte) error {
	notifier := c.tx.Notifier()

	var err error

	ch := make(chan struct{})

	unsubOk := notifier.Subscribe(ResponseOk, func(data []byte) {
		close(ch)
	})
	defer unsubOk()

	unsubDisabled := notifier.Subscribe(ResponseDisabled, func(data []byte) {
		err = poop.New("private key is disabled")
		close(ch)
	})
	defer unsubDisabled()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeImportPrivateKeyCommand(c.tx, privateKey); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TODO(kellegous): This is not working on real devices currently. We seed the
// SentResponse arrive, but we never get a PushStatusResponse.
func (c *Conn) GetStatus(ctx context.Context, key *PublicKey) (*StatusResponse, error) {
	notifier := c.tx.Notifier()

	var status StatusResponse
	var err error

	ch := make(chan struct{})

	// TODO(kellegous): Why is this a push event?
	// TODO(kellegous): We should reject responses where the key prefix
	// doesn't match the given key.
	unsubStatus := notifier.Subscribe(PushStatusResponse, func(data []byte) {
		err = status.readFrom(bytes.NewReader(data))
		close(ch)
	})
	defer unsubStatus()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeGetStatusCommand(c.tx, key); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return &status, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SetAdvertLatLon sets the advert latitude and longitude.
func (c *Conn) SetAdvertLatLon(ctx context.Context, lat float64, lon float64) error {
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

	if err := writeSetAdvertLatLonCommand(c.tx, lat, lon); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetAdvertName sets the advert name.
func (c *Conn) SetAdvertName(ctx context.Context, name string) error {
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

	if err := writeSetAdvertNameCommand(c.tx, name); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Conn) SetDeviceTime(ctx context.Context, time time.Time) error {
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

	if err := writeSetDeviceTimeCommand(c.tx, time); err != nil {
		return poop.Chain(err)
	}

	select {
	case <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

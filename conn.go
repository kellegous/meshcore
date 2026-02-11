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

type expectation struct {
	ch     chan struct{}
	unsubs []func()
}

func (e *expectation) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-e.ch:
			if !ok {
				return nil
			}
		}
	}
}

func (e *expectation) Unsubscribe() {
	for _, unsub := range e.unsubs {
		unsub()
	}
}

func expect(
	notifier *Notifier,
	fn func(NotificationCode, []byte) bool,
	codes ...NotificationCode,
) *expectation {
	e := &expectation{
		ch: make(chan struct{}),
	}

	for _, code := range codes {
		e.unsubs = append(e.unsubs, notifier.Subscribe(code, func(data []byte) {
			if fn(code, data) {
				e.ch <- struct{}{}
			} else {
				close(e.ch)
			}
		}))
	}

	return e
}

// AddOrUpdateContact adds or updates a contact on the device.
func (c *Conn) AddOrUpdateContact(ctx context.Context, contact *Contact) error {
	var err error

	subs := expect(c.tx.Notifier(), func(code NotificationCode, data []byte) bool {
		switch code {
		case ResponseOk:
		case ResponseErr:
			err = readError(data)
		}
		return false
	}, ResponseOk, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeAddOrUpdateContactCommand(c.tx, contact); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// RemoveContact removes a contact from the device.
func (c *Conn) RemoveContact(ctx context.Context, key *PublicKey) error {
	var err error

	subs := expect(c.tx.Notifier(), func(code NotificationCode, data []byte) bool {
		switch code {
		case ResponseOk:
		case ResponseErr:
			err = readError(data)
		}
		return false
	}, ResponseOk, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeRemoveContactCommand(c.tx, key); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

type GetContactsOptions struct {
	Since time.Time
}

// GetContacts returns the list of contacts from the device.
func (c *Conn) GetContacts(ctx context.Context, opts *GetContactsOptions) ([]*Contact, error) {
	if opts == nil {
		opts = &GetContactsOptions{}
	}

	var contacts []*Contact
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseContactsStart:
				return true
			case ResponseErr:
				err = readError(data)
				return false
			case ResponseContact:
				var contact Contact
				if err := contact.readFrom(bytes.NewReader(data)); err != nil {
					return false
				}
				contacts = append(contacts, &contact)
				return true
			case ResponseEndOfContacts:
				return false
			}
			panic("unreachable")
		},
		ResponseContactsStart,
		ResponseContact,
		ResponseEndOfContacts,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeGetContactsCommand(c.tx, opts.Since); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return contacts, err
}

// GetDeviceTime returns the current device time.
func (c *Conn) GetDeviceTime(ctx context.Context) (time.Time, error) {
	var t time.Time
	var err error

	subs := expect(c.tx.Notifier(), func(code NotificationCode, data []byte) bool {
		switch code {
		case ResponseCurrTime:
			t, err = readTime(bytes.NewReader(data))
		case ResponseErr:
			err = readError(data)
		}
		return false
	}, ResponseCurrTime, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeCommandCode(c.tx, CommandGetDeviceTime); err != nil {
		return time.Time{}, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return time.Time{}, poop.Chain(err)
	}

	return t, err
}

// GetBatteryVoltage returns the current battery voltage in millivolts.
func (c *Conn) GetBatteryVoltage(ctx context.Context) (uint16, error) {
	var voltage uint16
	var err error

	subs := expect(c.tx.Notifier(), func(code NotificationCode, data []byte) bool {
		switch code {
		case ResponseBatteryVoltage:
			err = binary.Read(bytes.NewReader(data), binary.LittleEndian, &voltage)
		case ResponseErr:
			err = readError(data)
		}
		return false
	}, ResponseBatteryVoltage, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeCommandCode(c.tx, CommandGetBatteryVoltage); err != nil {
		return 0, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return 0, poop.Chain(err)
	}

	return voltage, err
}

// SendTextMessage sends a text message to the recipient.
func (c *Conn) SendTextMessage(
	ctx context.Context,
	recipient *PublicKey,
	message string,
	textType TextType,
) (*SentResponse, error) {
	var sr SentResponse
	var err error

	subs := expect(c.tx.Notifier(), func(code NotificationCode, data []byte) bool {
		switch code {
		case ResponseSent:
			sr.readFrom(bytes.NewReader(data))
		case ResponseErr:
			err = readError(data)
		}
		return false
	}, ResponseSent, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeSendTextMessageCommand(c.tx, recipient, message, textType, 0, time.Now()); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return &sr, err
}

// SendChannelTextMessage sends a text message to the given channel.
func (c *Conn) SendChannelTextMessage(
	ctx context.Context,
	channelIndex byte,
	message string,
	textType TextType,
) error {
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeSendChannelTextMessageCommand(
		c.tx,
		channelIndex,
		message,
		textType,
		time.Now(),
	); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// GetTelemetry returns the telemetry data for the given contact key.
func (c *Conn) GetTelemetry(
	ctx context.Context,
	key *PublicKey,
) (*TelemetryResponse, error) {
	var telemetry TelemetryResponse
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case PushTelemetryResponse:
				err = telemetry.readFrom(bytes.NewReader(data))
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		PushTelemetryResponse,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeGetTelemetryCommand(c.tx, key); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return &telemetry, err
}

// GetChannel returns the channel information for the given index.
func (c *Conn) GetChannel(
	ctx context.Context,
	idx uint8,
) (*ChannelInfo, error) {
	var channel ChannelInfo
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseChannelInfo:
				err = channel.readFrom(bytes.NewReader(data))
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseChannelInfo,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeGetChannelCommand(c.tx, idx); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return &channel, err
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
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeSetChannelCommand(c.tx, channel); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
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
	var deviceInfo DeviceInfo
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseDeviceInfo:
				err = deviceInfo.readFrom(bytes.NewReader(data))
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseDeviceInfo,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeDeviceQueryCommand(c.tx, appTargetVer); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return &deviceInfo, err
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
	var message Message
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseContactMsgRecv:
				var contactMessage ContactMessage
				err = contactMessage.readFrom(bytes.NewReader(data))
				if err == nil {
					message = &contactMessage
				}
			case ResponseChannelMsgRecv:
				var channelMessage ChannelMessage
				err = channelMessage.readFrom(bytes.NewReader(data))
				if err == nil {
					message = &channelMessage
				}
			case ResponseErr:
				err = readError(data)
			case ResponseNoMoreMessages:
			}
			return false
		},
		ResponseContactMsgRecv,
		ResponseChannelMsgRecv,
		ResponseErr,
		ResponseNoMoreMessages)
	defer subs.Unsubscribe()

	if err := writeCommandCode(c.tx, CommandSyncNextMessage); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return message, err
}

// SendAdvert sends an advert to the device.
func (c *Conn) SendAdvert(ctx context.Context, advertType SelfAdvertType) error {
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeSendAdvertCommand(c.tx, advertType); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// ExportContact exports a contact from the device. if key is nil, the
// device's self contact is exported.
func (c *Conn) ExportContact(ctx context.Context, key *PublicKey) ([]byte, error) {
	notifier := c.tx.Notifier()

	var advertPacket []byte
	var err error

	subs := expect(notifier, func(code NotificationCode, data []byte) bool {
		switch code {
		case ResponseExportContact:
			advertPacket = make([]byte, len(data))
			copy(advertPacket, data)
		case ResponseErr:
			err = readError(data)
		}
		return false
	}, ResponseExportContact, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeExportContactCommand(c.tx, key); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return advertPacket, err
}

// ImportContact imports a contact into the device.
func (c *Conn) ImportContact(ctx context.Context, advertPacket []byte) error {
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeImportContactCommand(c.tx, advertPacket); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// ShareContact shares a contact with the device.
func (c *Conn) ShareContact(ctx context.Context, key PublicKey) error {
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		}, ResponseOk, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeShareContactCommand(c.tx, &key); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// ExportPrivateKey exports the private key from the device.
func (c *Conn) ExportPrivateKey(ctx context.Context) ([]byte, error) {
	var privateKey [64]byte
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponsePrivateKey:
				_, err = io.ReadFull(bytes.NewReader(data), privateKey[:])
			case ResponseDisabled:
				err = poop.New("private key is disabled")
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponsePrivateKey,
		ResponseDisabled,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeCommandCode(c.tx, CommandExportPrivateKey); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return privateKey[:], err
}

// ImportPrivateKey imports a private key into the device.
func (c *Conn) ImportPrivateKey(ctx context.Context, privateKey []byte) error {
	var err error

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseDisabled:
				err = poop.New("private key is disabled")
			case ResponseErr:
				err = readError(data)
			}
			return false
		}, ResponseOk, ResponseDisabled, ResponseErr)
	defer subs.Unsubscribe()

	if err := writeImportPrivateKeyCommand(c.tx, privateKey); err != nil {
		return poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// TODO(kellegous): This is not working on real devices currently. We seed the
// SentResponse arrive, but we never get a PushStatusResponse.
func (c *Conn) GetStatus(ctx context.Context, key PublicKey) (*StatusResponse, error) {
	var status StatusResponse
	var err error

	expect := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			// TODO(kellegous): Why is this a push event?
			// TODO(kellegous): We should reject responses where the key prefix
			// doesn't match the given key.
			case PushStatusResponse:
				err = status.readFrom(bytes.NewReader(data))
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		PushStatusResponse,
		ResponseErr)
	defer expect.Unsubscribe()

	if err := writeGetStatusCommand(c.tx, &key); err != nil {
		return nil, poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return &status, err
}

// SetAdvertLatLon sets the advert latitude and longitude.
func (c *Conn) SetAdvertLatLon(ctx context.Context, lat float64, lon float64) error {
	var err error

	expect := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer expect.Unsubscribe()

	if err := writeSetAdvertLatLonCommand(c.tx, lat, lon); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// SetAdvertName sets the advert name.
func (c *Conn) SetAdvertName(ctx context.Context, name string) error {
	var err error

	expect := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer expect.Unsubscribe()

	if err := writeSetAdvertNameCommand(c.tx, name); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// SetDeviceTime sets the device time.
func (c *Conn) SetDeviceTime(ctx context.Context, time time.Time) error {
	var err error

	expect := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer expect.Unsubscribe()

	if err := writeSetDeviceTimeCommand(c.tx, time); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// ResetPath resets the path for the given contact key.
func (c *Conn) ResetPath(ctx context.Context, key PublicKey) error {
	var err error

	expect := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer expect.Unsubscribe()

	if err := writeResetPathCommand(c.tx, &key); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// GetSelfInfo returns the self information from the device.
func (c *Conn) GetSelfInfo(ctx context.Context) (*SelfInfoResponse, error) {
	notifier := c.tx.Notifier()

	var selfInfo SelfInfoResponse
	var err error

	ch := make(chan struct{})

	unsubSelfInfo := notifier.Subscribe(ResponseSelfInfo, func(data []byte) {
		err = selfInfo.readFrom(bytes.NewReader(data))
		close(ch)
	})
	defer unsubSelfInfo()

	unsubErr := notifier.Subscribe(ResponseErr, func(data []byte) {
		err = readError(data)
		close(ch)
	})
	defer unsubErr()

	if err := writeCommandAppStartCommand(c.tx); err != nil {
		return nil, poop.Chain(err)
	}

	select {
	case <-ch:
		return &selfInfo, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Sign signs the given data.
func (c *Conn) Sign(ctx context.Context, data []byte) ([]byte, error) {
	const chunkSize = 128
	buf := bytes.NewReader(data)

	var err error
	var signature []byte

	sendNextChunk := func() error {
		var chunk [128]byte
		n, err := io.ReadFull(buf, chunk[:])
		if err != nil && err != io.ErrUnexpectedEOF {
			return poop.Chain(err)
		}

		if err := writeSignDataCommand(c.tx, chunk[:n]); err != nil {
			return poop.Chain(err)
		}

		return nil
	}

	onSignStart := func(data []byte) error {
		var signStartResponse SignStartResponse
		if err := signStartResponse.readFrom(bytes.NewReader(data)); err != nil {
			return poop.Chain(err)
		}
		if buf.Len() > int(signStartResponse.MaxSignDataLen) {
			return poop.New("data is too long")
		}
		if err := sendNextChunk(); err != nil {
			return poop.Chain(err)
		}
		return nil
	}
	onOk := func() error {
		if buf.Len() > 0 {
			return poop.Chain(sendNextChunk())
		}
		if err := writeCommandCode(c.tx, CommandSignFinish); err != nil {
			return poop.Chain(err)
		}
		return nil
	}

	subs := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseSignStart:
				err = onSignStart(data)
				return err == nil
			case ResponseOk:
				err = onOk()
				return err == nil
			case ResponseSignature:
				var res SignatureResponse
				err = res.readFrom(bytes.NewReader(data))
				if err == nil {
					signature = res.Signature[:]
				}
				return false
			case ResponseErr:
				err = readError(data)
				return false
			}
			panic("unreachable")
		},
		ResponseSignStart,
		ResponseOk,
		ResponseSignature,
		ResponseErr)
	defer subs.Unsubscribe()

	if err := writeCommandCode(c.tx, CommandSignStart); err != nil {
		return nil, poop.Chain(err)
	}

	if err := subs.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return signature, err
}

// SetRadioParams sets the radio parameters.
func (c *Conn) SetRadioParams(
	ctx context.Context,
	radioFreq float64, // how is this represented?
	radioBw float64,
	radioSf byte,
	radioCr byte,
) error {
	var err error

	expect := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseOk:
			case ResponseErr:
				err = readError(data)
			}
			return false
		},
		ResponseOk,
		ResponseErr)
	defer expect.Unsubscribe()

	if err := writeSetRadioParamsCommand(
		c.tx,
		uint32(radioFreq*1000),
		uint32(radioBw*1000),
		radioSf,
		radioCr,
	); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// SendBinaryRequest sends a binary request to the given recipient.
func (c *Conn) SendBinaryRequest(
	ctx context.Context,
	recipient PublicKey,
	payload []byte,
) error {
	var err error

	var tag uint32

	expect := expect(
		c.tx.Notifier(),
		func(code NotificationCode, data []byte) bool {
			switch code {
			case ResponseSent:
				var sr SentResponse
				err = sr.readFrom(bytes.NewReader(data))
				if err == nil {
					tag = sr.ExpectedAckCRC
				}
				return err == nil
			case PushBinaryResponse:
				var binaryResponse BinaryResponse
				err = binaryResponse.readFrom(bytes.NewReader(data))
				if err != nil {
					return false
				}
				if binaryResponse.Tag != tag {
					return true
				}
				return false
			case ResponseErr:
				err = readError(data)
				return false
			}
			return false
		},
		PushBinaryResponse,
		ResponseSent,
		ResponseErr)
	defer expect.Unsubscribe()

	if err := writeSendBinaryRequestCommand(c.tx, recipient, payload); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

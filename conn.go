package meshcore

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"iter"
	"time"

	"github.com/kellegous/poop"
)

type Transport interface {
	io.Writer
	Disconnect() error
	Subscribe(code NotificationCode, fn func(data []byte)) func()
	// TODO(kellegous): Rename this to Subscribe.
	Subscribe2(ctx context.Context, codes ...NotificationCode) iter.Seq2[Notification, error]
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
	tx Transport,
	fn func(NotificationCode, []byte) bool,
	codes ...NotificationCode,
) *expectation {
	e := &expectation{
		ch: make(chan struct{}),
	}

	for _, code := range codes {
		e.unsubs = append(e.unsubs, tx.Subscribe(code, func(data []byte) {
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
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeOk, NotificationTypeErr),
	)
	defer done()

	if err := writeAddOrUpdateContactCommand(c.tx, contact); err != nil {
		return poop.Chain(err)
	}

	res, err, ok := next()
	if !ok {
		return poop.Chain(io.ErrUnexpectedEOF)
	} else if err != nil {
		return poop.Chain(err)
	}

	switch t := res.(type) {
	case *OkNotification:
		return nil
	case *ErrNotification:
		return poop.Chain(t.Error())
	}

	panic("unreachable")
}

// RemoveContact removes a contact from the device.
func (c *Conn) RemoveContact(ctx context.Context, key *PublicKey) error {
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeOk, NotificationTypeErr),
	)
	defer done()

	if err := writeRemoveContactCommand(c.tx, key); err != nil {
		return poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return poop.Chain(err)
	}

	switch t := res.(type) {
	case *OkNotification:
		return nil
	case *ErrNotification:
		return poop.Chain(t.Error())
	}

	panic("unreachable")
}

type GetContactsOptions struct {
	Since time.Time
}

// GetContacts returns the list of contacts from the device.
func (c *Conn) GetContacts(ctx context.Context, opts *GetContactsOptions) ([]*Contact, error) {
	if opts == nil {
		opts = &GetContactsOptions{}
	}

	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeContactsStart, NotificationTypeErr, NotificationTypeContact, NotificationTypeEndOfContacts),
	)
	defer done()

	if err := writeGetContactsCommand(c.tx, opts.Since); err != nil {
		return nil, poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return nil, poop.Chain(err)
	}

	switch t := res.(type) {
	case *ContactStartNotification:
	case *ErrNotification:
		return nil, poop.Chain(t.Error())
	}

	var contacts []*Contact
	for {
		res, err, _ := next()
		if err != nil {
			return nil, poop.Chain(err)
		}

		switch t := res.(type) {
		case *ContactNotification:
			contacts = append(contacts, &t.Contact)
		case *EndOfContactsNotification:
			return contacts, nil
		}
	}
}

// GetDeviceTime returns the current device time.
func (c *Conn) GetDeviceTime(ctx context.Context) (time.Time, error) {
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeCurrTime, NotificationTypeErr),
	)
	defer done()

	if err := writeCommandCode(c.tx, CommandGetDeviceTime); err != nil {
		return time.Time{}, poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return time.Time{}, poop.Chain(err)
	}

	switch t := res.(type) {
	case *CurrTimeNotification:
		return t.Time, nil
	case *ErrNotification:
		return time.Time{}, poop.Chain(t.Error())
	}

	panic("unreachable")
}

// GetBatteryVoltage returns the current battery voltage in millivolts.
func (c *Conn) GetBatteryVoltage(ctx context.Context) (uint16, error) {
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeBatteryVoltage, NotificationTypeErr),
	)
	defer done()

	if err := writeCommandCode(c.tx, CommandGetBatteryVoltage); err != nil {
		return 0, poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return 0, poop.Chain(err)
	}

	switch t := res.(type) {
	case *BatteryVoltageNotification:
		return t.Voltage, nil
	case *ErrNotification:
		return 0, poop.Chain(t.Error())
	}

	panic("unreachable")
}

// SendTextMessage sends a text message to the recipient.
func (c *Conn) SendTextMessage(
	ctx context.Context,
	recipient *PublicKey,
	message string,
	textType TextType,
) (*SentNotification, error) {
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeSent, NotificationTypeErr),
	)
	defer done()

	if err := writeSendTextMessageCommand(c.tx, recipient, message, textType, 0, time.Now()); err != nil {
		return nil, poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return nil, poop.Chain(err)
	}

	switch t := res.(type) {
	case *SentNotification:
		return t, nil
	case *ErrNotification:
		return nil, poop.Chain(t.Error())
	}

	panic("unreachable")
}

// SendChannelTextMessage sends a text message to the given channel.
func (c *Conn) SendChannelTextMessage(
	ctx context.Context,
	channelIndex byte,
	message string,
	textType TextType,
) error {
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeOk, NotificationTypeErr),
	)
	defer done()

	if err := writeSendChannelTextMessageCommand(c.tx, channelIndex, message, textType, time.Now()); err != nil {
		return poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return poop.Chain(err)
	}

	switch t := res.(type) {
	case *OkNotification:
		return nil
	case *ErrNotification:
		return poop.Chain(t.Error())
	}

	panic("unreachable")
}

// GetTelemetry returns the telemetry data for the given contact key.
func (c *Conn) GetTelemetry(
	ctx context.Context,
	key *PublicKey,
) (*TelemetryResponseNotification, error) {
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeTelemetryResponse, NotificationTypeErr),
	)
	defer done()

	if err := writeGetTelemetryCommand(c.tx, key); err != nil {
		return nil, poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return nil, poop.Chain(err)
	}

	switch t := res.(type) {
	case *TelemetryResponseNotification:
		return t, nil
	case *ErrNotification:
		return nil, poop.Chain(t.Error())
	}

	panic("unreachable")
}

// GetChannel returns the channel information for the given index.
func (c *Conn) GetChannel(
	ctx context.Context,
	idx uint8,
) (*ChannelInfo, error) {
	next, done := iter.Pull2(
		c.tx.Subscribe2(ctx, NotificationTypeChannelInfo, NotificationTypeErr),
	)
	defer done()

	if err := writeGetChannelCommand(c.tx, idx); err != nil {
		return nil, poop.Chain(err)
	}

	res, err, _ := next()
	if err != nil {
		return nil, poop.Chain(err)
	}

	switch t := res.(type) {
	case *ChannelInfoNotification:
		return &t.ChannelInfo, nil
	case *ErrNotification:
		return nil, poop.Chain(t.Error())
	}

	panic("unreachable")
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeDeviceInfo:
				err = deviceInfo.readFrom(bytes.NewReader(data))
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeDeviceInfo,
		NotificationTypeErr)
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
	var rErr *CommandError
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeContactMsgRecv:
				var contactMessage ContactMessage
				err = contactMessage.readFrom(bytes.NewReader(data))
				if err == nil {
					message = &contactMessage
				}
			case NotificationTypeChannelMsgRecv:
				var channelMessage ChannelMessage
				err = channelMessage.readFrom(bytes.NewReader(data))
				if err == nil {
					message = &channelMessage
				}
			case NotificationTypeErr:
				err = readError(data)
			case NotificationTypeNoMoreMessages:
			}
			return false
		},
		NotificationTypeContactMsgRecv,
		NotificationTypeChannelMsgRecv,
		NotificationTypeErr,
		NotificationTypeNoMoreMessages)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
	var advertPacket []byte
	var err error

	subs := expect(c.tx, func(code NotificationCode, data []byte) bool {
		switch code {
		case NotificationTypeExportContact:
			advertPacket = make([]byte, len(data))
			copy(advertPacket, data)
		case NotificationTypeErr:
			err = readError(data)
		}
		return false
	}, NotificationTypeExportContact, NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		}, NotificationTypeOk, NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypePrivateKey:
				_, err = io.ReadFull(bytes.NewReader(data), privateKey[:])
			case NotificationTypeDisabled:
				err = poop.New("private key is disabled")
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypePrivateKey,
		NotificationTypeDisabled,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeDisabled:
				err = poop.New("private key is disabled")
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		}, NotificationTypeOk, NotificationTypeDisabled, NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			// TODO(kellegous): Why is this a push event?
			// TODO(kellegous): We should reject responses where the key prefix
			// doesn't match the given key.
			case NotificationTypeStatusResponse:
				err = status.readFrom(bytes.NewReader(data))
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeStatusResponse,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
	var selfInfo SelfInfoResponse
	var err error

	ch := make(chan struct{})

	unsubSelfInfo := c.tx.Subscribe(NotificationTypeSelfInfo, func(data []byte) {
		err = selfInfo.readFrom(bytes.NewReader(data))
		close(ch)
	})
	defer unsubSelfInfo()

	unsubErr := c.tx.Subscribe(NotificationTypeErr, func(data []byte) {
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeSignStart:
				err = onSignStart(data)
				return err == nil
			case NotificationTypeOk:
				err = onOk()
				return err == nil
			case NotificationTypeSignature:
				var res SignatureResponse
				err = res.readFrom(bytes.NewReader(data))
				if err == nil {
					signature = res.Signature[:]
				}
				return false
			case NotificationTypeErr:
				err = readError(data)
				return false
			}
			panic("unreachable")
		},
		NotificationTypeSignStart,
		NotificationTypeOk,
		NotificationTypeSignature,
		NotificationTypeErr)
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
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
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
) (*BinaryResponse, error) {
	var binaryResponse BinaryResponse
	var err error

	var tag uint32

	expect := expect(
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeSent:
				var sr *SentNotification
				sr, err = readSentNotification(data)
				if err == nil {
					tag = sr.ExpectedAckCRC
				}
				return err == nil
			case NotificationTypeBinaryResponse:
				err = binaryResponse.readFrom(bytes.NewReader(data))
				if err != nil {
					return false
				}
				if binaryResponse.Tag != tag {
					return true
				}
				return false
			case NotificationTypeErr:
				err = readError(data)
				return false
			}
			return false
		},
		NotificationTypeBinaryResponse,
		NotificationTypeSent,
		NotificationTypeErr)
	defer expect.Unsubscribe()

	if err := writeSendBinaryRequestCommand(c.tx, recipient, payload); err != nil {
		return nil, poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return &binaryResponse, err
}

// SetTXPower sets the TX power.
func (c *Conn) SetTXPower(ctx context.Context, power byte) error {
	var err error

	expect := expect(
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
	defer expect.Unsubscribe()

	if err := writeSetTXPowerCommand(c.tx, power); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// SetOtherParams sets the other parameters.
func (c *Conn) SetOtherParams(ctx context.Context, manualAddContacts bool) error {
	var err error

	expect := expect(
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeOk:
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeOk,
		NotificationTypeErr)
	defer expect.Unsubscribe()

	if err := writeSetOtherParamsCommand(c.tx, manualAddContacts); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

type NeighborsOrder byte

const (
	NeighborsOrderNewestToOldest     NeighborsOrder = 0
	NeighborsOrderOldestToNewest     NeighborsOrder = 1
	NeighborsOrderStrongestToWeakest NeighborsOrder = 2
	NeighborsOrderWeakestToStrongest NeighborsOrder = 3
)

func (c *Conn) GetNeighbours(
	ctx context.Context,
	recipient PublicKey,
	count uint8,
	offset uint16,
	orderBy NeighborsOrder,
	pubKeyPrefixLength byte,
) ([]*Neighbour, error) {
	var payload bytes.Buffer
	if err := binary.Write(&payload, binary.LittleEndian, byte(BinaryRequestTypeGetNeighbours)); err != nil {
		return nil, poop.Chain(err)
	}
	// request_version=0
	if err := binary.Write(&payload, binary.LittleEndian, byte(0)); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Write(&payload, binary.LittleEndian, count); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Write(&payload, binary.LittleEndian, offset); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Write(&payload, binary.LittleEndian, byte(orderBy)); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Write(&payload, binary.LittleEndian, pubKeyPrefixLength); err != nil {
		return nil, poop.Chain(err)
	}
	// random blob (help hash)
	if _, err := io.CopyN(&payload, rand.Reader, 4); err != nil {
		return nil, poop.Chain(err)
	}

	res, err := c.SendBinaryRequest(ctx, recipient, payload.Bytes())
	if err != nil {
		return nil, poop.Chain(err)
	}

	buf := bytes.NewBuffer(res.ResponseData)
	var totalNeighboursCount uint16
	if err := binary.Read(buf, binary.LittleEndian, &totalNeighboursCount); err != nil {
		return nil, poop.Chain(err)
	}
	var resultsCount uint16
	if err := binary.Read(buf, binary.LittleEndian, &resultsCount); err != nil {
		return nil, poop.Chain(err)
	}

	neighbours := make([]*Neighbour, 0, resultsCount)
	for i := 0; i < int(resultsCount); i++ {
		var neighbour Neighbour
		if err := neighbour.readFrom(buf, pubKeyPrefixLength); err != nil {
			return nil, poop.Chain(err)
		}
		neighbours = append(neighbours, &neighbour)
	}

	return neighbours, nil
}

// TracePath traces the given path and returns the trace data.
func (c *Conn) TracePath(ctx context.Context, path []byte) (*TraceData, error) {
	var traceData TraceData
	var err error

	// generate a random tag for this trace, so we can listen for the correct response
	var tag uint32
	if err := binary.Read(rand.Reader, binary.LittleEndian, &tag); err != nil {
		return nil, poop.Chain(err)
	}

	expect := expect(
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeTraceData:
				err = traceData.readFrom(bytes.NewReader(data))
				if err != nil {
					return false
				}
				if traceData.Tag != tag {
					// not the right data, continue
					return true
				}
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeTraceData,
		NotificationTypeErr)
	defer expect.Unsubscribe()

	if err := writeSendTracePathCommand(c.tx, tag, 0 /* auth */, path); err != nil {
		return nil, poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return nil, poop.Chain(err)
	}

	return &traceData, err
}

func (c *Conn) Login(ctx context.Context, key PublicKey, password string) error {
	var loginSuccess LoginSuccessResponse
	var err error

	expect := expect(
		c.tx,
		func(code NotificationCode, data []byte) bool {
			switch code {
			case NotificationTypeLoginSuccess:
				err = loginSuccess.readFrom(bytes.NewReader(data))
				if err != nil {
					return false
				}
				if !bytes.Equal(loginSuccess.PubKeyPrefix[:], key.Prefix(6)) {
					// not the right data, continue waiting
					return true
				}
			case NotificationTypeErr:
				err = readError(data)
			}
			return false
		},
		NotificationTypeLoginSuccess,
		NotificationTypeErr)
	defer expect.Unsubscribe()

	if err := writeLoginCommand(c.tx, key, password); err != nil {
		return poop.Chain(err)
	}

	if err := expect.Wait(ctx); err != nil {
		return poop.Chain(err)
	}

	return err
}

// OnAdvert subscribes to advert events.
func (c *Conn) OnAdvert(fn func(*AdvertEvent)) func() {
	return c.tx.Subscribe(NotificationTypeAdvert, func(data []byte) {
		// TODO(kellegous): Errors should be propagated to the
		// receive error callback in the transport.
		var advertEvent AdvertEvent
		if err := advertEvent.readFrom(bytes.NewReader(data)); err != nil {
			return
		}
		fn(&advertEvent)
	})
}

func (c *Conn) OnNewAdvert(fn func(*NewAdvertEvent)) func() {
	return c.tx.Subscribe(NotificationTypeNewAdvert, func(data []byte) {
		// TODO(kellegous): Errors should be propagated to the
		// receive error callback in the transport.
		var newAdvertEvent NewAdvertEvent
		if err := newAdvertEvent.readFrom(bytes.NewReader(data)); err != nil {
			return
		}
		fn(&newAdvertEvent)
	})
}

// OnPathUpdated subscribes to path updated events.
func (c *Conn) OnPathUpdated(fn func(*PathUpdatedEvent)) func() {
	return c.tx.Subscribe(NotificationTypePathUpdated, func(data []byte) {
		// TODO(kellegous): Errors should be propagated to the
		// receive error callback in the transport.
		var pathUpdatedEvent PathUpdatedEvent
		if err := pathUpdatedEvent.readFrom(bytes.NewReader(data)); err != nil {
			return
		}
		fn(&pathUpdatedEvent)
	})
}

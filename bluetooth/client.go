package bluetooth

import (
	"context"
	"iter"
	"strings"

	"github.com/kellegous/poop"
	"tinygo.org/x/bluetooth"

	"github.com/kellegous/meshcore"
)

var (
	serviceUUID  = mustParseUUID("6E400001-B5A3-F393-E0A9-E50E24DCCA9E")
	toDeviceUUID = mustParseUUID("6E400002-B5A3-F393-E0A9-E50E24DCCA9E")
	frDeviceUUID = mustParseUUID("6E400003-B5A3-F393-E0A9-E50E24DCCA9E")
)

func mustParseUUID(s string) bluetooth.UUID {
	uuid, err := bluetooth.ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return uuid
}

type Client struct {
	adapter *bluetooth.Adapter
}

func NewClient(adapter *bluetooth.Adapter) (*Client, error) {
	if err := adapter.Enable(); err != nil {
		return nil, poop.Chain(err)
	}
	return &Client{adapter: adapter}, nil
}

func isMeshcoreDevice(result *bluetooth.ScanResult) bool {
	return strings.HasPrefix(result.LocalName(), "MeshCore-")
}

func (c *Client) LookupDevice(ctx context.Context, name string) (*bluetooth.ScanResult, error) {
	for device, err := range c.DiscoverDevices(ctx) {
		if err != nil {
			return nil, poop.Chain(err)
		}

		if device.LocalName() == name {
			return device, nil
		}
	}
	return nil, poop.Newf("device %s not found", name)
}

func (c *Client) DiscoverDevices(ctx context.Context) iter.Seq2[*bluetooth.ScanResult, error] {
	return func(yield func(*bluetooth.ScanResult, error) bool) {
		seen := make(map[string]bool)

		go func() {
			<-ctx.Done()
			c.adapter.StopScan()
		}()

		if err := c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			if !isMeshcoreDevice(&result) || seen[result.Address.String()] {
				return
			}

			seen[result.Address.String()] = true
			if !yield(&result, nil) {
				c.adapter.StopScan()
			}
		}); err != nil {
			yield(nil, poop.Chain(err))
		}
	}
}

func (c *Client) Connect(ctx context.Context, address bluetooth.Address) (*meshcore.Conn, error) {
	device, err := c.adapter.Connect(address, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, poop.Chain(err)
	}

	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		return nil, poop.Chain(err)
	}
	if len(services) != 1 {
		return nil, poop.Newf("expected 1 service, got %d", len(services))
	}

	characteristics, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{toDeviceUUID, frDeviceUUID})
	if err != nil {
		return nil, poop.Chain(err)
	}
	if len(characteristics) != 2 {
		return nil, poop.Newf("expected 2 characteristics, got %d", len(characteristics))
	}

	toDevice, frDevice := characteristics[0], characteristics[1]

	transport := &Transport{
		device:   device,
		toDevice: toDevice,
		notifier: meshcore.NewNotifier(),
	}

	frDevice.EnableNotifications(func(data []byte) {
		code := meshcore.ResponseCode(data[0])
		transport.notifier.Notify(code, data[1:])
	})

	return meshcore.NewConnection(transport), nil
}

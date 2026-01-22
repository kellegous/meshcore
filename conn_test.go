package meshcore

import "io"

type fakeTransport struct {
	io.Writer
	notifier *Notifier
}

var _ Transport = (*fakeTransport)(nil)

func (t *fakeTransport) Write(p []byte) (n int, err error) {
	return t.Writer.Write(p)
}

func (t *fakeTransport) Disconnect() error {
	return nil
}

func (t *fakeTransport) Notifier() *Notifier {
	return t.notifier
}

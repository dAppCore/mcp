package mcp

import (
	"context"
)

// moved AX-7 triplet TestTransportStdio_Service_ServeStdio_Good
func TestTransportStdio_Service_ServeStdio_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeStdio(ctx)
	AssertError(t, err)
}

// moved AX-7 triplet TestTransportStdio_Service_ServeStdio_Bad
func TestTransportStdio_Service_ServeStdio_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.ServeStdio(context.Background()) })
	AssertNil(t, svc)
}

// moved AX-7 triplet TestTransportStdio_Service_ServeStdio_Ugly
func TestTransportStdio_Service_ServeStdio_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeStdio(ctx)
	AssertError(t, err)
	AssertNotNil(t, svc.Server())
}

package sourceaccess

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

type testRESTResolver struct {
	addresses []netip.Addr
	err       error
}

func (r testRESTResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return r.addresses, r.err
}

type testRESTConn struct {
	net.Conn
	remote net.Addr
	closed bool
}

func (c *testRESTConn) RemoteAddr() net.Addr { return c.remote }
func (c *testRESTConn) Close() error         { c.closed = true; return nil }

func TestQuestionTestClientDoesNotUseProxies(t *testing.T) {
	transport, ok := RestrictedQuestionTestOptions().Client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || transport.DialTLS != nil || transport.DialTLSContext != nil || transport.DialContext == nil {
		t.Fatal("REST egress must use the restricted direct dialer")
	}
}

func TestRESTSafeDialPinsPublicDNSAndRejectsRebinding(t *testing.T) {
	var dialed string
	dial := func(_ context.Context, network, address string) (net.Conn, error) {
		dialed = address
		return &testRESTConn{remote: &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 443}}, nil
	}
	connect := restrictedRESTDial(testRESTResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, dial)
	if _, err := connect(context.Background(), "tcp", "bank.example:443"); err != nil {
		t.Fatal(err)
	}
	if dialed != "8.8.8.8:443" {
		t.Fatalf("DNS was not pinned: %q", dialed)
	}

	redirected := &testRESTConn{remote: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 443}}
	connect = restrictedRESTDial(testRESTResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, func(context.Context, string, string) (net.Conn, error) { return redirected, nil })
	if _, err := connect(context.Background(), "tcp", "bank.example:443"); !errors.Is(err, ErrConnection) || !redirected.closed {
		t.Fatalf("rebinding was not blocked: %v", err)
	}
}

func TestRESTSafeDialRejectsUnsafeOrMixedDestinations(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "10.1.2.3", "169.254.169.254", "100.64.0.1", "192.0.2.2", "::1", "fc00::1", "::ffff:127.0.0.1", "2001:db8::1"} {
		t.Run(strings.ReplaceAll(address, ":", "_"), func(t *testing.T) {
			called := false
			connect := restrictedRESTDial(testRESTResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr(address)}}, func(context.Context, string, string) (net.Conn, error) { called = true; return nil, nil })
			if _, err := connect(context.Background(), "tcp", "bank.example:443"); !errors.Is(err, ErrConnection) || called {
				t.Fatalf("unsafe DNS result was not blocked: %v, called %v", err, called)
			}
		})
	}
}

package sourceaccess

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"time"
)

// RestrictedQuestionTestOptions confines a live question test to directly
// addressed public destinations. Existing catalog connections intentionally
// retain their separate transport and network configuration.
func RestrictedQuestionTestOptions() RESTJSONOptions {
	options := DefaultRESTJSONOptions()
	transport := options.Client.Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	transport.DialContext = restrictedRESTDial(net.DefaultResolver, (&net.Dialer{Timeout: 5 * time.Second}).DialContext)
	options.Client.Transport = transport
	return options
}

// Question tests must never use an ambient proxy or re-resolve the host after
// its addresses have been checked. Custom clients are for controlled tests.
type restResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type restDialFunc func(context.Context, string, string) (net.Conn, error)

func restrictedRESTDial(resolver restResolver, dial restDialFunc) restDialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if (network != "tcp" && network != "tcp4" && network != "tcp6") || resolver == nil || dial == nil {
			return nil, ErrConnection
		}
		host, portText, err := net.SplitHostPort(address)
		port, portErr := strconv.Atoi(portText)
		if err != nil || portErr != nil || host == "" || port < 1 || port > 65535 {
			return nil, ErrConnection
		}
		addresses, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil || len(addresses) == 0 || len(addresses) > 64 {
			return nil, fmt.Errorf("%w: source destination unavailable", ErrConnection)
		}
		for _, ip := range addresses {
			if !safeRESTPublicAddress(ip.Unmap()) {
				return nil, fmt.Errorf("%w: source destination is not permitted", ErrConnection)
			}
		}
		for _, value := range addresses {
			ip := value.Unmap()
			conn, dialErr := dial(ctx, "tcp", net.JoinHostPort(ip.String(), portText))
			if dialErr != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				continue
			}
			if conn == nil {
				return nil, ErrConnection
			}
			if conn.RemoteAddr() == nil {
				_ = conn.Close()
				return nil, fmt.Errorf("%w: source destination unavailable", ErrConnection)
			}
			remote, parseErr := netip.ParseAddrPort(conn.RemoteAddr().String())
			if parseErr != nil || remote.Addr().Unmap() != ip || int(remote.Port()) != port {
				_ = conn.Close()
				return nil, fmt.Errorf("%w: source destination changed", ErrConnection)
			}
			return conn, nil
		}
		return nil, fmt.Errorf("%w: source destination unavailable", ErrConnection)
	}
}

func safeRESTPublicAddress(ip netip.Addr) bool {
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, prefix := range unsafeRESTPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

// IANA special-purpose IPv4/IPv6 ranges; kept consistent with the existing
// public vendor-brand fetch policy. Transition prefixes fail closed.
var unsafeRESTPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.31.196.0/24"), netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("192.175.48.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("100::/64"), netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("2620:4f:8000::/48"),
	netip.MustParsePrefix("3ffe::/16"), netip.MustParsePrefix("3fff::/20"), netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("::/96"), netip.MustParsePrefix("fec0::/10"),
}

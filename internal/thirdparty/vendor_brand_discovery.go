package thirdparty

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sergeymakinen/go-ico"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
	"golang.org/x/net/html"
)

const (
	vendorBrandHTMLLimit         = 256 * 1024
	vendorBrandImageLimit        = 512 * 1024
	vendorBrandHeaderLimit       = 32 * 1024
	vendorBrandMaxDimension      = 1024
	vendorBrandMaxPixels         = 1024 * 1024
	vendorBrandOutputDimension   = 256
	vendorBrandMaximumRedirects  = 3
	vendorBrandMaximumCandidates = 16
	vendorBrandMaximumURLLength  = 2048
	vendorBrandRequestTimeout    = 3 * time.Second
)

var (
	ErrUnsafeVendorBrandDestination = errors.New("vendor brand destination is not public")
	ErrUnsafeVendorBrandURL         = errors.New("vendor brand URL is not allowed")
	ErrVendorBrandResponseTooLarge  = errors.New("vendor brand response exceeds the allowed size")
	ErrUnsupportedVendorBrandMedia  = errors.New("vendor brand media type is not supported")
	ErrInvalidVendorBrandImage      = errors.New("vendor brand image is invalid")
	ErrVendorBrandTimeout           = errors.New("vendor brand request timed out")
	ErrVendorBrandUnavailable       = errors.New("vendor brand icon is unavailable")
)

type VendorBrandResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type VendorBrandDialContext func(context.Context, string, string) (net.Conn, error)

type vendorBrandTransportFactory func(VendorBrandDialContext, string) http.RoundTripper

type VendorBrandDiscoverer struct {
	resolver         VendorBrandResolver
	dialer           VendorBrandDialContext
	transportFactory vendorBrandTransportFactory
	timeout          time.Duration
	discover         func(context.Context, WebsiteDomain) (DiscoveredVendorBrand, error)
}

type DiscoveredVendorBrand struct {
	PNG          []byte
	MediaType    string
	PixelWidth   int
	PixelHeight  int
	SourceDigest string
}

type vendorBrandIconCandidate struct {
	URL   *url.URL
	Score int
}

type systemVendorBrandResolver struct{ resolver *net.Resolver }

func (r systemVendorBrandResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return r.resolver.LookupNetIP(ctx, network, host)
}

func NewVendorBrandDiscoverer(resolver VendorBrandResolver, dialer VendorBrandDialContext) *VendorBrandDiscoverer {
	if resolver == nil {
		resolver = systemVendorBrandResolver{resolver: net.DefaultResolver}
	}
	if dialer == nil {
		dialer = (&net.Dialer{Timeout: vendorBrandRequestTimeout}).DialContext
	}
	value := &VendorBrandDiscoverer{resolver: resolver, dialer: dialer, timeout: vendorBrandRequestTimeout}
	value.transportFactory = func(dial VendorBrandDialContext, serverName string) http.RoundTripper {
		return &http.Transport{
			Proxy:                  nil,
			DialContext:            dial,
			DisableKeepAlives:      true,
			MaxResponseHeaderBytes: vendorBrandHeaderLimit,
			ForceAttemptHTTP2:      false,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: serverName,
			},
		}
	}
	return value
}

func NewDefaultVendorBrandDiscoverer() *VendorBrandDiscoverer {
	dialer := &net.Dialer{Timeout: vendorBrandRequestTimeout, KeepAlive: -1}
	return NewVendorBrandDiscoverer(systemVendorBrandResolver{resolver: net.DefaultResolver}, dialer.DialContext)
}

func (d *VendorBrandDiscoverer) Discover(ctx context.Context, domain WebsiteDomain) (DiscoveredVendorBrand, error) {
	if d == nil {
		return DiscoveredVendorBrand{}, ErrVendorBrandUnavailable
	}
	if d.discover != nil {
		return d.discover(ctx, domain)
	}
	normalized, err := NormalizeWebsiteDomain(string(domain))
	if err != nil || normalized != domain || legacyNumericVendorBrandHost(string(normalized)) {
		return DiscoveredVendorBrand{}, ErrUnsafeVendorBrandURL
	}
	origin, _ := url.Parse("https://" + string(normalized) + "/")
	deadline := d.timeout
	if deadline <= 0 || deadline > vendorBrandRequestTimeout {
		deadline = vendorBrandRequestTimeout
	}
	requestContext, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	page, finalOrigin, err := d.fetch(requestContext, origin, vendorBrandHTMLLimit, "html")
	if err != nil {
		return DiscoveredVendorBrand{}, classifyVendorBrandContextError(err)
	}
	candidates, err := parseVendorBrandIconCandidates(bytes.NewReader(page), finalOrigin.String())
	if err != nil {
		return DiscoveredVendorBrand{}, ErrVendorBrandUnavailable
	}
	fallback, _ := url.Parse("/favicon.ico")
	candidates = append(candidates, vendorBrandIconCandidate{URL: finalOrigin.ResolveReference(fallback), Score: -1})
	var firstFailure error
	for _, candidate := range candidates {
		body, finalURL, fetchErr := d.fetch(requestContext, candidate.URL, vendorBrandImageLimit, "image")
		if fetchErr != nil {
			if firstFailure == nil && !errors.Is(fetchErr, ErrVendorBrandUnavailable) {
				firstFailure = fetchErr
			}
			continue
		}
		result, decodeErr := canonicalVendorBrandPNG(body, finalURL)
		if decodeErr == nil {
			return result, nil
		}
		if firstFailure == nil {
			firstFailure = decodeErr
		}
	}
	if firstFailure != nil {
		return DiscoveredVendorBrand{}, classifyVendorBrandContextError(firstFailure)
	}
	return DiscoveredVendorBrand{}, ErrVendorBrandUnavailable
}

func (d *VendorBrandDiscoverer) fetch(ctx context.Context, target *url.URL, limit int64, kind string) ([]byte, *url.URL, error) {
	current, err := validatedVendorBrandURL(target)
	if err != nil {
		return nil, nil, err
	}
	for hop := 0; hop <= vendorBrandMaximumRedirects; hop++ {
		addresses, resolveErr := d.resolver.LookupNetIP(ctx, "ip", current.Hostname())
		if resolveErr != nil || len(addresses) == 0 {
			if resolveErr == nil {
				resolveErr = ErrVendorBrandUnavailable
			}
			return nil, nil, resolveErr
		}
		chosen, validateErr := validateVendorBrandAddresses(addresses)
		if validateErr != nil {
			return nil, nil, validateErr
		}
		port := 443
		if current.Port() != "" {
			port, _ = strconv.Atoi(current.Port())
		}
		dial := validatedVendorBrandDial(d.dialer, chosen, port)
		transport := d.transportFactory(dial, current.Hostname())
		client := &http.Client{
			Transport: transport,
			Timeout:   d.timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, current.String(), nil)
		if requestErr != nil {
			closeVendorBrandTransport(transport)
			return nil, nil, ErrUnsafeVendorBrandURL
		}
		request.Header.Set("Accept", vendorBrandAcceptHeader(kind))
		request.Header.Set("User-Agent", "ClearSight-Vendor-Brand/1.0")
		response, requestErr := client.Do(request)
		if requestErr != nil {
			closeVendorBrandTransport(transport)
			return nil, nil, classifyVendorBrandContextError(requestErr)
		}
		if response.StatusCode >= 300 && response.StatusCode <= 399 {
			location := response.Header.Get("Location")
			_ = response.Body.Close()
			closeVendorBrandTransport(transport)
			if hop == vendorBrandMaximumRedirects || location == "" {
				return nil, nil, ErrUnsafeVendorBrandURL
			}
			reference, parseErr := url.Parse(location)
			if parseErr != nil {
				return nil, nil, ErrUnsafeVendorBrandURL
			}
			current, parseErr = validatedVendorBrandURL(current.ResolveReference(reference))
			if parseErr != nil {
				return nil, nil, parseErr
			}
			continue
		}
		if response.StatusCode < 200 || response.StatusCode > 299 {
			_ = response.Body.Close()
			closeVendorBrandTransport(transport)
			return nil, nil, ErrVendorBrandUnavailable
		}
		if response.ContentLength > limit {
			_ = response.Body.Close()
			closeVendorBrandTransport(transport)
			return nil, nil, ErrVendorBrandResponseTooLarge
		}
		mediaType, _, mediaErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
		if mediaErr != nil && response.Header.Get("Content-Type") != "" {
			_ = response.Body.Close()
			closeVendorBrandTransport(transport)
			return nil, nil, ErrUnsupportedVendorBrandMedia
		}
		if kind == "html" && mediaType != "text/html" && mediaType != "application/xhtml+xml" {
			_ = response.Body.Close()
			closeVendorBrandTransport(transport)
			return nil, nil, ErrUnsupportedVendorBrandMedia
		}
		limited := &io.LimitedReader{R: response.Body, N: limit + 1}
		body, readErr := io.ReadAll(limited)
		closeErr := response.Body.Close()
		closeVendorBrandTransport(transport)
		if readErr != nil {
			return nil, nil, readErr
		}
		if closeErr != nil {
			return nil, nil, closeErr
		}
		if int64(len(body)) > limit {
			return nil, nil, ErrVendorBrandResponseTooLarge
		}
		if kind == "image" {
			if err := validateVendorBrandMedia(mediaType, body); err != nil {
				return nil, nil, err
			}
		}
		return body, current, nil
	}
	return nil, nil, ErrUnsafeVendorBrandURL
}

func closeVendorBrandTransport(transport http.RoundTripper) {
	if value, ok := transport.(interface{ CloseIdleConnections() }); ok {
		value.CloseIdleConnections()
	}
}

func vendorBrandAcceptHeader(kind string) string {
	if kind == "html" {
		return "text/html,application/xhtml+xml"
	}
	return "image/png,image/jpeg,image/webp,image/x-icon,image/vnd.microsoft.icon"
}

func validatedVendorBrandURL(input *url.URL) (*url.URL, error) {
	if input == nil || input.Scheme != "https" || input.Opaque != "" || input.User != nil || input.Hostname() == "" || len(input.String()) > vendorBrandMaximumURLLength {
		return nil, ErrUnsafeVendorBrandURL
	}
	if input.Port() != "" && input.Port() != "443" {
		return nil, ErrUnsafeVendorBrandURL
	}
	host, err := NormalizeWebsiteDomain(input.Hostname())
	if err != nil || legacyNumericVendorBrandHost(string(host)) {
		return nil, ErrUnsafeVendorBrandURL
	}
	copy := *input
	copy.Scheme = "https"
	copy.User = nil
	copy.Fragment = ""
	copy.Host = string(host)
	if input.Port() == "443" {
		copy.Host = net.JoinHostPort(string(host), "443")
	}
	return &copy, nil
}

func legacyNumericVendorBrandHost(host string) bool {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(host)), ".")
	if len(parts) < 1 || len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.ParseUint(part, 0, 32); err == nil {
			continue
		}
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return false
		}
	}
	return true
}

func validateVendorBrandAddresses(values []netip.Addr) (netip.Addr, error) {
	unique := make(map[netip.Addr]struct{}, len(values))
	addresses := make([]netip.Addr, 0, len(values))
	for _, value := range values {
		value = value.Unmap()
		if !safePublicVendorBrandAddress(value) {
			return netip.Addr{}, ErrUnsafeVendorBrandDestination
		}
		if _, exists := unique[value]; !exists {
			unique[value] = struct{}{}
			addresses = append(addresses, value)
		}
	}
	if len(addresses) == 0 {
		return netip.Addr{}, ErrUnsafeVendorBrandDestination
	}
	sort.Slice(addresses, func(i, j int) bool { return addresses[i].Less(addresses[j]) })
	return addresses[0], nil
}

func safePublicVendorBrandAddress(address netip.Addr) bool {
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range unsafeVendorBrandPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

// Sourced from the IANA IPv4 and IPv6 Special-Purpose Address Space
// registries, reviewed 2026-08-26. Explicit entries complement netip's
// semantic checks so registry-only and transition ranges fail closed.
var unsafeVendorBrandPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.31.196.0/24"), netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("192.175.48.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2620:4f:8000::/48"), netip.MustParsePrefix("3ffe::/16"), netip.MustParsePrefix("3fff::/20"), netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("::/96"), netip.MustParsePrefix("fec0::/10"),
}

func validatedVendorBrandDial(dialer VendorBrandDialContext, address netip.Addr, port int) VendorBrandDialContext {
	return func(ctx context.Context, network, _ string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, ErrUnsafeVendorBrandDestination
		}
		connection, err := dialer(ctx, "tcp", net.JoinHostPort(address.String(), strconv.Itoa(port)))
		if err != nil {
			return nil, err
		}
		remote, ok := remoteVendorBrandAddress(connection.RemoteAddr())
		if !ok || remote.Addr().Unmap() != address.Unmap() || int(remote.Port()) != port {
			_ = connection.Close()
			return nil, ErrUnsafeVendorBrandDestination
		}
		return connection, nil
	}
}

func remoteVendorBrandAddress(value net.Addr) (netip.AddrPort, bool) {
	switch address := value.(type) {
	case *net.TCPAddr:
		parsed, ok := netip.AddrFromSlice(address.IP)
		return netip.AddrPortFrom(parsed.Unmap(), uint16(address.Port)), ok && address.Port > 0 && address.Port <= 65535
	default:
		parsed, err := netip.ParseAddrPort(value.String())
		return parsed, err == nil
	}
}

func parseVendorBrandIconCandidates(reader io.Reader, baseURL string) ([]vendorBrandIconCandidate, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	document, err := html.Parse(io.LimitReader(reader, vendorBrandHTMLLimit+1))
	if err != nil {
		return nil, err
	}
	candidates := make([]vendorBrandIconCandidate, 0)
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "link" {
			attributes := make(map[string]string, len(node.Attr))
			for _, attribute := range node.Attr {
				attributes[strings.ToLower(attribute.Key)] = strings.TrimSpace(attribute.Val)
			}
			relations := strings.Fields(strings.ToLower(attributes["rel"]))
			isIcon := false
			for _, relation := range relations {
				if relation == "icon" || relation == "apple-touch-icon" || relation == "apple-touch-icon-precomposed" {
					isIcon = true
					break
				}
			}
			if isIcon && attributes["href"] != "" {
				reference, parseErr := url.Parse(attributes["href"])
				if parseErr == nil {
					resolved, validationErr := validatedVendorBrandURL(base.ResolveReference(reference))
					if validationErr == nil {
						candidate := vendorBrandIconCandidate{URL: resolved, Score: vendorBrandIconScore(attributes["sizes"], relations)}
						if len(candidates) < vendorBrandMaximumCandidates {
							candidates = append(candidates, candidate)
						} else {
							worst := 0
							for index := 1; index < len(candidates); index++ {
								if betterVendorBrandCandidate(candidates[worst], candidates[index]) {
									worst = index
								}
							}
							if betterVendorBrandCandidate(candidate, candidates[worst]) {
								candidates[worst] = candidate
							}
						}
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	sort.SliceStable(candidates, func(i, j int) bool {
		return betterVendorBrandCandidate(candidates[i], candidates[j])
	})
	return candidates, nil
}

func betterVendorBrandCandidate(left, right vendorBrandIconCandidate) bool {
	if left.Score == right.Score {
		return left.URL.String() < right.URL.String()
	}
	return left.Score > right.Score
}

func vendorBrandIconScore(sizes string, relations []string) int {
	best := 0
	for _, size := range strings.Fields(strings.ToLower(sizes)) {
		if size == "any" {
			best = vendorBrandMaxDimension * vendorBrandMaxDimension
			continue
		}
		parts := strings.Split(size, "x")
		if len(parts) != 2 {
			continue
		}
		width, widthErr := strconv.Atoi(parts[0])
		height, heightErr := strconv.Atoi(parts[1])
		if widthErr == nil && heightErr == nil && width > 0 && height > 0 && width <= vendorBrandMaxDimension && height <= vendorBrandMaxDimension {
			if area := width * height; area > best {
				best = area
			}
		}
	}
	if best == 0 {
		for _, relation := range relations {
			if strings.HasPrefix(relation, "apple-touch-icon") {
				return 180 * 180
			}
		}
	}
	return best
}

func validateVendorBrandMedia(declared string, body []byte) error {
	actual := vendorBrandMagicMedia(body)
	if actual == "" || declared == "image/svg+xml" || declared == "text/html" {
		return ErrUnsupportedVendorBrandMedia
	}
	declared = strings.ToLower(strings.TrimSpace(declared))
	if declared != "" && !vendorBrandMediaEquivalent(declared, actual) {
		return ErrUnsupportedVendorBrandMedia
	}
	return nil
}

func vendorBrandMediaEquivalent(declared, actual string) bool {
	if declared == actual {
		return true
	}
	return actual == "image/x-icon" && (declared == "image/vnd.microsoft.icon" || declared == "image/ico" || declared == "image/icon")
}

func vendorBrandMagicMedia(body []byte) string {
	switch {
	case len(body) >= 8 && bytes.Equal(body[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case len(body) >= 3 && body[0] == 0xff && body[1] == 0xd8 && body[2] == 0xff:
		return "image/jpeg"
	case len(body) >= 12 && string(body[:4]) == "RIFF" && string(body[8:12]) == "WEBP":
		return "image/webp"
	case len(body) >= 6 && binary.LittleEndian.Uint16(body[:2]) == 0 && binary.LittleEndian.Uint16(body[2:4]) == 1 && binary.LittleEndian.Uint16(body[4:6]) > 0:
		return "image/x-icon"
	default:
		return ""
	}
}

func canonicalVendorBrandPNG(body []byte, _ *url.URL) (DiscoveredVendorBrand, error) {
	mediaType := vendorBrandMagicMedia(body)
	if mediaType == "image/x-icon" {
		if err := validateVendorBrandICOContainer(body); err != nil {
			return DiscoveredVendorBrand{}, err
		}
	}
	config, err := decodeVendorBrandConfig(mediaType, body)
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > vendorBrandMaxDimension || config.Height > vendorBrandMaxDimension || config.Width*config.Height > vendorBrandMaxPixels {
		return DiscoveredVendorBrand{}, ErrInvalidVendorBrandImage
	}
	decoded, err := decodeVendorBrandImage(mediaType, body)
	if err != nil {
		return DiscoveredVendorBrand{}, ErrInvalidVendorBrandImage
	}
	canonical := resizeVendorBrandImage(decoded, vendorBrandOutputDimension)
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&encoded, canonical); err != nil || encoded.Len() > vendorBrandImageLimit {
		return DiscoveredVendorBrand{}, ErrInvalidVendorBrandImage
	}
	digest := sha256.Sum256(encoded.Bytes())
	bounds := canonical.Bounds()
	return DiscoveredVendorBrand{
		PNG: encoded.Bytes(), MediaType: "image/png", PixelWidth: bounds.Dx(), PixelHeight: bounds.Dy(),
		SourceDigest: hex.EncodeToString(digest[:]),
	}, nil
}

func validateVendorBrandICOContainer(body []byte) error {
	if len(body) < 6 {
		return ErrInvalidVendorBrandImage
	}
	count := int(binary.LittleEndian.Uint16(body[4:6]))
	if count < 1 || count > 64 || len(body) < 6+16*count {
		return ErrInvalidVendorBrandImage
	}
	for index := 0; index < count; index++ {
		start := 6 + 16*index
		size := uint64(binary.LittleEndian.Uint32(body[start+8 : start+12]))
		offset := uint64(binary.LittleEndian.Uint32(body[start+12 : start+16]))
		if size == 0 || offset < uint64(6+16*count) || offset > uint64(len(body)) || size > uint64(len(body))-offset {
			return ErrInvalidVendorBrandImage
		}
	}
	return nil
}

func decodeVendorBrandConfig(mediaType string, body []byte) (image.Config, error) {
	reader := bytes.NewReader(body)
	switch mediaType {
	case "image/png":
		return png.DecodeConfig(reader)
	case "image/jpeg":
		return jpeg.DecodeConfig(reader)
	case "image/webp":
		return webp.DecodeConfig(reader)
	case "image/x-icon":
		return ico.DecodeConfig(reader)
	default:
		return image.Config{}, ErrUnsupportedVendorBrandMedia
	}
}

func decodeVendorBrandImage(mediaType string, body []byte) (image.Image, error) {
	reader := bytes.NewReader(body)
	switch mediaType {
	case "image/png":
		return png.Decode(reader)
	case "image/jpeg":
		return jpeg.Decode(reader)
	case "image/webp":
		return webp.Decode(reader)
	case "image/x-icon":
		return ico.Decode(reader)
	default:
		return nil, ErrUnsupportedVendorBrandMedia
	}
}

func resizeVendorBrandImage(source image.Image, maximum int) image.Image {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maximum && height <= maximum {
		output := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				output.Set(x, y, source.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return output
	}
	scale := float64(maximum) / float64(width)
	if height > width {
		scale = float64(maximum) / float64(height)
	}
	targetWidth := max(1, int(float64(width)*scale))
	targetHeight := max(1, int(float64(height)*scale))
	output := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	xdraw.CatmullRom.Scale(output, output.Bounds(), source, bounds, xdraw.Over, nil)
	return output
}

func classifyVendorBrandContextError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
		return fmt.Errorf("%w: %v", ErrVendorBrandTimeout, err)
	}
	return err
}

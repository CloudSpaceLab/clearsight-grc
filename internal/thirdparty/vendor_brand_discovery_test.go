package thirdparty

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type brandResolverStub struct {
	answers map[string][][]netip.Addr
	calls   map[string]int
}

func (r *brandResolverStub) LookupNetIP(_ context.Context, network, host string) ([]netip.Addr, error) {
	if network != "ip" {
		return nil, errors.New("unexpected network")
	}
	if r.calls == nil {
		r.calls = map[string]int{}
	}
	answer := r.answers[host]
	index := r.calls[host]
	r.calls[host]++
	if len(answer) == 0 {
		return nil, errors.New("not found")
	}
	if index >= len(answer) {
		index = len(answer) - 1
	}
	return append([]netip.Addr(nil), answer[index]...), nil
}

type brandRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f brandRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func brandResponse(status int, contentType string, body []byte, headers ...[2]string) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	for _, pair := range headers {
		header.Set(pair[0], pair[1])
	}
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(bytes.NewReader(body))}
}

func brandPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	value := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value.Set(x, y, color.NRGBA{R: 20, G: 150, B: 180, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, value); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func brandJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	value := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value.Set(x, y, color.NRGBA{R: 180, G: 90, B: 20, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, value, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func brandICOWithPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	payload := brandPNG(t, width, height)
	result := make([]byte, 22+len(payload))
	binary.LittleEndian.PutUint16(result[2:4], 1)
	binary.LittleEndian.PutUint16(result[4:6], 1)
	result[6], result[7] = byte(width), byte(height)
	binary.LittleEndian.PutUint16(result[10:12], 1)
	binary.LittleEndian.PutUint16(result[12:14], 32)
	binary.LittleEndian.PutUint32(result[14:18], uint32(len(payload)))
	binary.LittleEndian.PutUint32(result[18:22], 22)
	copy(result[22:], payload)
	return result
}

func testBrandDiscoverer(resolver VendorBrandResolver, responses map[string]*http.Response) *VendorBrandDiscoverer {
	discoverer := NewVendorBrandDiscoverer(resolver, func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("test transport must not dial")
	})
	discoverer.transportFactory = func(_ VendorBrandDialContext, _ string) http.RoundTripper {
		return brandRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
			response, exists := responses[request.URL.String()]
			if !exists {
				return nil, errors.New("unexpected request: " + request.URL.String())
			}
			clone := *response
			clone.Header = response.Header.Clone()
			data, _ := io.ReadAll(response.Body)
			response.Body = io.NopCloser(bytes.NewReader(data))
			clone.Body = io.NopCloser(bytes.NewReader(data))
			return &clone, nil
		})
	}
	return discoverer
}

func TestVendorBrandDiscoveryRejectsUnsafeDNSAnswers(t *testing.T) {
	t.Parallel()
	unsafe := []string{
		"0.0.0.0", "10.0.0.1", "100.64.0.1", "127.0.0.1", "169.254.169.254", "172.16.0.1",
		"192.0.2.1", "192.168.1.1", "198.18.0.1", "198.51.100.1", "203.0.113.1", "224.0.0.1", "240.0.0.1",
		"::", "::1", "64:ff9b::7f00:1", "fc00::1", "fec0::1", "fe80::1", "ff02::1", "2001:db8::1", "2002:7f00:1::", "3fff::1",
	}
	for _, raw := range unsafe {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr(raw)}}}}
			discoverer := testBrandDiscoverer(resolver, nil)
			_, err := discoverer.Discover(context.Background(), "vendor.example")
			if !errors.Is(err, ErrUnsafeVendorBrandDestination) {
				t.Fatalf("Discover() error = %v, want unsafe destination", err)
			}
		})
	}
}

func TestVendorBrandDiscoveryRejectsAnyUnsafeAnswer(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{
		netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("127.0.0.1"),
	}}}}
	_, err := testBrandDiscoverer(resolver, nil).Discover(context.Background(), "vendor.example")
	if !errors.Is(err, ErrUnsafeVendorBrandDestination) {
		t.Fatalf("mixed DNS answer error = %v", err)
	}
}

func TestVendorBrandDiscoveryRejectsLegacyNumericHostFormsBeforeDNS(t *testing.T) {
	t.Parallel()
	for _, host := range []WebsiteDomain{"2130706433", "127.1", "0177.0.0.1", "0x7f000001", "0x7f.0.0.1"} {
		resolver := &brandResolverStub{answers: map[string][][]netip.Addr{string(host): {{netip.MustParseAddr("93.184.216.34")}}}}
		_, err := testBrandDiscoverer(resolver, nil).Discover(context.Background(), host)
		if !errors.Is(err, ErrUnsafeVendorBrandURL) {
			t.Fatalf("Discover(%q) error = %v, want unsafe URL", host, err)
		}
		if resolver.calls[string(host)] != 0 {
			t.Fatalf("legacy numeric host %q reached DNS", host)
		}
	}
}

func TestVendorBrandDiscoveryChoosesLargestDeclaredIconAndProducesBoundedPNG(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{
		"vendor.example": {{netip.MustParseAddr("93.184.216.34")}},
		"cdn.example":    {{netip.MustParseAddr("93.184.216.35")}},
	}}
	html := []byte(`<html><head>
		<link rel="shortcut icon" sizes="32x32" href="/small.png">
		<link rel="apple-touch-icon icon" sizes="512x512" href="https://cdn.example/large.png">
	</head></html>`)
	large := brandPNG(t, 512, 256)
	discoverer := testBrandDiscoverer(resolver, map[string]*http.Response{
		"https://vendor.example/":       brandResponse(http.StatusOK, "text/html", html),
		"https://cdn.example/large.png": brandResponse(http.StatusOK, "image/png", large),
	})
	result, err := discoverer.Discover(context.Background(), "vendor.example")
	if err != nil {
		t.Fatal(err)
	}
	if result.MediaType != "image/png" || result.PixelWidth > 256 || result.PixelHeight > 256 || len(result.PNG) == 0 {
		t.Fatalf("discovery result = %#v", result)
	}
	if resolver.calls["cdn.example"] != 1 {
		t.Fatalf("candidate DNS calls = %d, want 1", resolver.calls["cdn.example"])
	}
	config, err := png.DecodeConfig(bytes.NewReader(result.PNG))
	if err != nil || config.Width != result.PixelWidth || config.Height != result.PixelHeight {
		t.Fatalf("canonical PNG config = (%#v, %v)", config, err)
	}
}

func TestVendorBrandDiscoveryFallsBackToFavicon(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr("93.184.216.34")}}}}
	discoverer := testBrandDiscoverer(resolver, map[string]*http.Response{
		"https://vendor.example/":            brandResponse(http.StatusOK, "text/html", []byte(`<html><head></head></html>`)),
		"https://vendor.example/favicon.ico": brandResponse(http.StatusOK, "image/png", brandPNG(t, 24, 24)),
	})
	result, err := discoverer.Discover(context.Background(), "vendor.example")
	if err != nil || result.PixelWidth != 24 || result.PixelHeight != 24 {
		t.Fatalf("favicon fallback = (%#v, %v)", result, err)
	}
}

func TestVendorBrandDiscoveryAcceptsDecodedJPEGAndICOAsCanonicalPNG(t *testing.T) {
	t.Parallel()
	for _, item := range []struct {
		name      string
		mediaType string
		body      func(*testing.T) []byte
	}{
		{name: "jpeg", mediaType: "image/jpeg", body: func(t *testing.T) []byte { return brandJPEG(t, 20, 10) }},
		{name: "ico png", mediaType: "image/vnd.microsoft.icon", body: func(t *testing.T) []byte { return brandICOWithPNG(t, 32, 32) }},
		{name: "ico dib", mediaType: "image/x-icon", body: func(t *testing.T) []byte { return brandICODIB(t, 2, 2) }},
	} {
		item := item
		t.Run(item.name, func(t *testing.T) {
			resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr("93.184.216.34")}}}}
			discoverer := testBrandDiscoverer(resolver, map[string]*http.Response{
				"https://vendor.example/":     brandResponse(http.StatusOK, "text/html", []byte(`<link rel="icon" href="/icon">`)),
				"https://vendor.example/icon": brandResponse(http.StatusOK, item.mediaType, item.body(t)),
			})
			result, err := discoverer.Discover(context.Background(), "vendor.example")
			if err != nil || result.MediaType != "image/png" || vendorBrandMagicMedia(result.PNG) != "image/png" {
				t.Fatalf("canonical result = (%#v, %v)", result, err)
			}
		})
	}
}

func brandICODIB(t *testing.T, width, height int) []byte {
	t.Helper()
	rowBytes := width * 4
	maskRowBytes := (width + 31) / 32 * 4
	payloadSize := 40 + rowBytes*height + maskRowBytes*height
	result := make([]byte, 22+payloadSize)
	binary.LittleEndian.PutUint16(result[2:4], 1)
	binary.LittleEndian.PutUint16(result[4:6], 1)
	result[6], result[7] = byte(width), byte(height)
	binary.LittleEndian.PutUint16(result[10:12], 1)
	binary.LittleEndian.PutUint16(result[12:14], 32)
	binary.LittleEndian.PutUint32(result[14:18], uint32(payloadSize))
	binary.LittleEndian.PutUint32(result[18:22], 22)
	payload := result[22:]
	binary.LittleEndian.PutUint32(payload[0:4], 40)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(width))
	binary.LittleEndian.PutUint32(payload[8:12], uint32(height*2))
	binary.LittleEndian.PutUint16(payload[12:14], 1)
	binary.LittleEndian.PutUint16(payload[14:16], 32)
	binary.LittleEndian.PutUint32(payload[20:24], uint32(rowBytes*height))
	for index := 40; index < 40+rowBytes*height; index += 4 {
		payload[index+0], payload[index+1], payload[index+2], payload[index+3] = 180, 90, 20, 255
	}
	return result
}

func TestVendorBrandSourceDigestIsContentAddressed(t *testing.T) {
	t.Parallel()
	source, _ := url.Parse("https://vendor.example/icon.png")
	first, err := canonicalVendorBrandPNG(brandPNG(t, 16, 16), source)
	if err != nil {
		t.Fatal(err)
	}
	secondBody := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			secondBody.Set(x, y, color.NRGBA{R: 200, G: 20, B: 90, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, secondBody); err != nil {
		t.Fatal(err)
	}
	second, err := canonicalVendorBrandPNG(encoded.Bytes(), source)
	if err != nil {
		t.Fatal(err)
	}
	if first.SourceDigest == second.SourceDigest {
		t.Fatalf("different canonical bytes share digest %q", first.SourceDigest)
	}
}

func TestVendorBrandResizeUsesFilteredResampling(t *testing.T) {
	t.Parallel()
	source := image.NewNRGBA(image.Rect(0, 0, 512, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 512; x++ {
			shade := uint8(0)
			if x%2 == 1 {
				shade = 255
			}
			source.Set(x, y, color.NRGBA{R: shade, G: shade, B: shade, A: 255})
		}
	}
	resized := resizeVendorBrandImage(source, 256)
	r, _, _, _ := resized.At(100, 0).RGBA()
	if r == 0 || r == 0xffff {
		t.Fatalf("resampled checkerboard retained nearest-neighbor value %d", r)
	}
}

func TestVendorBrandDiscoveryRevalidatesRedirectAndRejectsRebinding(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {
		{netip.MustParseAddr("93.184.216.34")},
		{netip.MustParseAddr("127.0.0.1")},
	}}}
	discoverer := testBrandDiscoverer(resolver, map[string]*http.Response{
		"https://vendor.example/": brandResponse(http.StatusFound, "text/html", nil, [2]string{"Location", "https://vendor.example/home"}),
	})
	_, err := discoverer.Discover(context.Background(), "vendor.example")
	if !errors.Is(err, ErrUnsafeVendorBrandDestination) {
		t.Fatalf("redirect rebinding error = %v", err)
	}
}

func TestVendorBrandDiscoveryRejectsUnsafeURLs(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr("93.184.216.34")}}}}
	for name, location := range map[string]string{
		"scheme downgrade": "http://vendor.example/home",
		"credentials":      "https://user:secret@vendor.example/home",
		"unsafe port":      "https://vendor.example:8443/home",
	} {
		t.Run(name, func(t *testing.T) {
			discoverer := testBrandDiscoverer(resolver, map[string]*http.Response{
				"https://vendor.example/": brandResponse(http.StatusFound, "", nil, [2]string{"Location", location}),
			})
			_, err := discoverer.Discover(context.Background(), "vendor.example")
			if !errors.Is(err, ErrUnsafeVendorBrandURL) {
				t.Fatalf("unsafe redirect error = %v", err)
			}
		})
	}
}

func TestVendorBrandDiscoveryEnforcesBodyAndImageLimits(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr("93.184.216.34")}}}}
	tests := []struct {
		name      string
		responses map[string]*http.Response
		want      error
	}{
		{name: "html", responses: map[string]*http.Response{
			"https://vendor.example/": brandResponse(http.StatusOK, "text/html", bytes.Repeat([]byte("x"), vendorBrandHTMLLimit+1)),
		}, want: ErrVendorBrandResponseTooLarge},
		{name: "image", responses: map[string]*http.Response{
			"https://vendor.example/":            brandResponse(http.StatusOK, "text/html", []byte(`<link rel="icon" href="/icon.png">`)),
			"https://vendor.example/icon.png":    brandResponse(http.StatusOK, "image/png", bytes.Repeat([]byte("x"), vendorBrandImageLimit+1)),
			"https://vendor.example/favicon.ico": brandResponse(http.StatusNotFound, "", nil),
		}, want: ErrVendorBrandResponseTooLarge},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			_, err := testBrandDiscoverer(resolver, item.responses).Discover(context.Background(), "vendor.example")
			if !errors.Is(err, item.want) {
				t.Fatalf("limit error = %v", err)
			}
		})
	}
}

func TestVendorBrandDiscoveryRejectsMediaMagicAndDimensions(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr("93.184.216.34")}}}}
	tests := []struct {
		name string
		body []byte
		kind string
		want error
	}{
		{name: "remote svg", body: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), kind: "image/svg+xml", want: ErrUnsupportedVendorBrandMedia},
		{name: "media mismatch", body: brandPNG(t, 16, 16), kind: "image/jpeg", want: ErrUnsupportedVendorBrandMedia},
		{name: "malformed png", body: []byte("\x89PNG\r\n\x1a\nnot-an-image"), kind: "image/png", want: ErrInvalidVendorBrandImage},
		{name: "dimensions", body: brandPNG(t, vendorBrandMaxDimension+1, 1), kind: "image/png", want: ErrInvalidVendorBrandImage},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			discoverer := testBrandDiscoverer(resolver, map[string]*http.Response{
				"https://vendor.example/":     brandResponse(http.StatusOK, "text/html", []byte(`<link rel="icon" href="/icon">`)),
				"https://vendor.example/icon": brandResponse(http.StatusOK, item.kind, item.body),
			})
			_, err := discoverer.Discover(context.Background(), "vendor.example")
			if !errors.Is(err, item.want) {
				t.Fatalf("image validation error = %v", err)
			}
		})
	}
}

func TestVendorBrandDiscovererUsesBoundedClientWithoutSensitiveHeaders(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr("93.184.216.34")}}}}
	requestSeen := false
	discoverer := NewVendorBrandDiscoverer(resolver, func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("unused") })
	discoverer.transportFactory = func(_ VendorBrandDialContext, _ string) http.RoundTripper {
		return brandRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
			requestSeen = true
			if request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" || request.URL.User != nil {
				t.Fatalf("sensitive request = %#v", request)
			}
			if _, ok := request.Context().Deadline(); !ok {
				t.Fatal("request has no end-to-end deadline")
			}
			return brandResponse(http.StatusOK, "text/html", []byte(`<html></html>`)), nil
		})
	}
	_, _ = discoverer.Discover(context.Background(), "vendor.example")
	if !requestSeen || discoverer.timeout != 3*time.Second {
		t.Fatalf("requestSeen=%v timeout=%v", requestSeen, discoverer.timeout)
	}
}

func TestVendorBrandDiscoveryEnforcesEndToEndTimeout(t *testing.T) {
	t.Parallel()
	resolver := &brandResolverStub{answers: map[string][][]netip.Addr{"vendor.example": {{netip.MustParseAddr("93.184.216.34")}}}}
	discoverer := NewVendorBrandDiscoverer(resolver, func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("unused") })
	discoverer.timeout = 10 * time.Millisecond
	discoverer.transportFactory = func(_ VendorBrandDialContext, _ string) http.RoundTripper {
		return brandRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()
			return nil, request.Context().Err()
		})
	}
	_, err := discoverer.Discover(context.Background(), "vendor.example")
	if !errors.Is(err, ErrVendorBrandTimeout) {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestVendorBrandValidatedDialRejectsUnexpectedRemoteAddress(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	defer server.Close()
	connection := &brandRemoteConn{Conn: client, remote: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 443}}
	dial := validatedVendorBrandDial(func(context.Context, string, string) (net.Conn, error) { return connection, nil }, netip.MustParseAddr("93.184.216.34"), 443)
	_, err := dial(context.Background(), "tcp", "ignored:443")
	if !errors.Is(err, ErrUnsafeVendorBrandDestination) {
		t.Fatalf("remote address error = %v", err)
	}
}

type brandRemoteConn struct {
	net.Conn
	remote net.Addr
}

func (c *brandRemoteConn) RemoteAddr() net.Addr { return c.remote }

func TestVendorBrandDiscoveryCandidateOrderingIsDeterministic(t *testing.T) {
	t.Parallel()
	html := `<link rel="icon" sizes="64x64" href="/z.png"><link rel="icon" sizes="64x64" href="/a.png">`
	candidates, err := parseVendorBrandIconCandidates(strings.NewReader(html), "https://vendor.example/")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].URL.String() != "https://vendor.example/a.png" {
		t.Fatalf("candidate order = %#v", candidates)
	}
}

func TestVendorBrandCandidateBoundStillSelectsLargestLateDeclaration(t *testing.T) {
	t.Parallel()
	var markup strings.Builder
	for index := 0; index < vendorBrandMaximumCandidates; index++ {
		markup.WriteString(`<link rel="icon" sizes="16x16" href="/small-` + strconv.Itoa(index) + `.png">`)
	}
	markup.WriteString(`<link rel="icon" sizes="512x512" href="/largest.png">`)
	candidates, err := parseVendorBrandIconCandidates(strings.NewReader(markup.String()), "https://vendor.example/")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != vendorBrandMaximumCandidates || candidates[0].URL.Path != "/largest.png" {
		t.Fatalf("bounded candidates did not retain largest late declaration: %#v", candidates)
	}
}

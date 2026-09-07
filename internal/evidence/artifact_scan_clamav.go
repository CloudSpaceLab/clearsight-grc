package evidence

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"time"
)

// ClamAVScanner implements the documented clamd IDSESSION / INSTREAM protocol.
// The endpoint is operator configuration, never a document-supplied address.
// clamd TCP has no authentication or encryption: deploy it on an isolated network.
type ClamAVScanner struct {
	network, address string
	timeout          time.Duration
}

func NewClamAVScanner(network, address string, timeout time.Duration) (*ClamAVScanner, error) {
	if (network != "tcp" && network != "unix") || strings.TrimSpace(address) == "" || timeout <= 0 || timeout > 20*time.Second {
		return nil, ErrArtifactScannerUnavailable
	}
	if network == "tcp" {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return nil, ErrArtifactScannerUnavailable
		}
	}
	return &ClamAVScanner{network: network, address: address, timeout: timeout}, nil
}

func (s *ClamAVScanner) Scan(ctx context.Context, reader io.Reader, size int64) (ArtifactScanResult, error) {
	unavailable := ArtifactScanResult{}
	if reader == nil || size <= 0 || size > 100<<20 {
		return unavailable, ErrArtifactScannerUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, s.network, s.address)
	if err != nil {
		return unavailable, ErrArtifactScannerUnavailable
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return unavailable, ErrArtifactScannerUnavailable
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if _, err := io.WriteString(conn, "zIDSESSION\x00zVERSION\x00"); err != nil {
		return unavailable, ErrArtifactScannerUnavailable
	}
	replies := bufio.NewReaderSize(conn, 4096)
	version, err := clamAVReply(replies)
	if err != nil || !strings.HasPrefix(version, "1: ClamAV ") || !strings.Contains(version, "/") {
		return unavailable, ErrArtifactScannerUnavailable
	}
	version = strings.TrimPrefix(version, "1: ClamAV ")
	if len(version) > 512 {
		return unavailable, ErrArtifactScannerUnavailable
	}
	if _, err := io.WriteString(conn, "zINSTREAM\x00"); err != nil {
		return unavailable, ErrArtifactScannerUnavailable
	}
	// Read concurrently while streaming: clamd can return a size/error response
	// before accepting all input. A reply closes the connection to unblock writes.
	type replyResult struct {
		value string
		err   error
	}
	reply := make(chan replyResult, 1)
	go func() { value, err := clamAVReply(replies); reply <- replyResult{value, err}; _ = conn.Close() }()
	streamErr := streamClamAV(conn, reader, size)
	if streamErr != nil {
		_ = conn.Close()
	}
	response := <-reply
	if streamErr != nil || response.err != nil || ctx.Err() != nil {
		return unavailable, ErrArtifactScannerUnavailable
	}
	result := ArtifactScanResult{Scanner: "ClamAV", Version: version}
	switch {
	case response.value == "2: stream: OK":
		result.Verdict = "CLEAN"
	case strings.HasPrefix(response.value, "2: stream: ") && strings.HasSuffix(response.value, " FOUND") && len(response.value) > len("2: stream:  FOUND"):
		result.Verdict = "INFECTED"
	default:
		return unavailable, ErrArtifactScannerUnavailable
	}
	return result, nil
}

func clamAVReply(reader *bufio.Reader) (string, error) {
	bytes, err := reader.ReadSlice(0)
	if err != nil || len(bytes) > 4096 {
		return "", ErrArtifactScannerUnavailable
	}
	result := string(bytes[:len(bytes)-1])
	if strings.ContainsAny(result, "\r\n") {
		return "", ErrArtifactScannerUnavailable
	}
	return result, nil
}

func streamClamAV(conn io.Writer, reader io.Reader, size int64) error {
	reader = io.LimitReader(reader, size+1)
	buffer := make([]byte, 32768)
	var total int64
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			total += int64(n)
			if total > size {
				return ErrArtifactTooLarge
			}
			if err := binary.Write(conn, binary.BigEndian, uint32(n)); err != nil {
				return err
			}
			if _, err := io.Copy(conn, bytes.NewReader(buffer[:n])); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	if total != size {
		return io.ErrUnexpectedEOF
	}
	return binary.Write(conn, binary.BigEndian, uint32(0))
}

package worker

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Scanner validates stored binaries before a document is made available.
type Scanner interface {
	// Scan reads content and reports whether it is safe. Implementations must
	// drain or close content as needed; the caller owns closing.
	Scan(ctx context.Context, content io.Reader) (bool, error)
}

// NoopScanner trusts uploads without external checks (development default;
// the checksum-presence gate in the worker still applies).
type NoopScanner struct{}

// Scan implements Scanner.
func (NoopScanner) Scan(context.Context, io.Reader) (bool, error) { return true, nil }

// ClamAVScanner streams content to a clamd daemon using the INSTREAM
// protocol (zINSTREAM\0, length-prefixed chunks, zero-length terminator).
type ClamAVScanner struct {
	addr    string
	timeout time.Duration
	dial    func(ctx context.Context, addr string) (net.Conn, error)
}

// NewClamAVScanner wires a clamd client.
func NewClamAVScanner(addr string, timeout time.Duration) *ClamAVScanner {
	return &ClamAVScanner{
		addr:    addr,
		timeout: timeout,
		dial: func(ctx context.Context, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "tcp", addr)
		},
	}
}

// ErrVirusFound is returned when clamd reports a threat.
var ErrVirusFound = errors.New("virus found")

// Scan implements Scanner via the clamd INSTREAM command.
func (s *ClamAVScanner) Scan(ctx context.Context, content io.Reader) (bool, error) {
	conn, err := s.dial(ctx, s.addr)
	if err != nil {
		return false, fmt.Errorf("connect clamd: %w", err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(s.timeout))

	if _, err := conn.Write(append([]byte("zINSTREAM"), 0)); err != nil {
		return false, fmt.Errorf("send instream: %w", err)
	}
	buf := make([]byte, 32*1024)
	for {
		n, readErr := content.Read(buf)
		if n > 0 {
			var hdr [4]byte
			binary.BigEndian.PutUint32(hdr[:], uint32(n)) //nolint:gosec // G115: n is capped by the 32KB read buffer
			if _, err := conn.Write(append(hdr[:], buf[:n]...)); err != nil {
				return false, fmt.Errorf("stream chunk: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return false, fmt.Errorf("read content: %w", readErr)
		}
	}
	// Zero-length terminator.
	if _, err := conn.Write(make([]byte, 4)); err != nil {
		return false, fmt.Errorf("terminate stream: %w", err)
	}

	resp, err := io.ReadAll(io.LimitReader(conn, 4096))
	if err != nil {
		return false, fmt.Errorf("read clamd response: %w", err)
	}
	msg := strings.TrimRight(string(resp), "\r\n\x00")
	switch {
	case strings.HasPrefix(msg, "stream: OK"):
		return true, nil
	case strings.Contains(msg, "FOUND"):
		return false, fmt.Errorf("%w: %s", ErrVirusFound, msg)
	default:
		return false, fmt.Errorf("unexpected clamd response: %q", msg)
	}
}

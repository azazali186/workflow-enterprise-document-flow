package worker

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/objectstore"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
)

// fakeClamd serves the clamd INSTREAM protocol: it consumes the length-prefixed
// chunks and replies with verdict.
func fakeClamd(t *testing.T, verdict string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
				hdr := make([]byte, 4)
				// zINSTREAM\0 (9 chars + NUL).
				if _, err := io.CopyN(io.Discard, c, 10); err != nil {
					return
				}
				for {
					if _, err := io.ReadFull(c, hdr); err != nil {
						return
					}
					n := binary.BigEndian.Uint32(hdr)
					if n == 0 {
						break
					}
					if _, err := io.CopyN(io.Discard, c, int64(n)); err != nil {
						return
					}
				}
				_, _ = c.Write([]byte(verdict))
			}(conn)
		}
	}()
	return ln.Addr().String()
}

func TestClamAVScannerClean(t *testing.T) {
	addr := fakeClamd(t, "stream: OK\x00")
	s := NewClamAVScanner(addr, 5*time.Second)
	clean, err := s.Scan(context.Background(), strings.NewReader("harmless bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Fatal("expected clean verdict")
	}
}

func TestClamAVScannerFindsVirus(t *testing.T) {
	addr := fakeClamd(t, "stream: Eicar-Test-Signature FOUND\x00")
	s := NewClamAVScanner(addr, 5*time.Second)
	clean, err := s.Scan(context.Background(), strings.NewReader("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"))
	if err == nil {
		t.Fatal("expected an error for an infected file")
	}
	if clean {
		t.Fatal("infected file must not be clean")
	}
	if !errorsIsVirus(err) {
		t.Fatalf("expected virus-found error, got %v", err)
	}
}

func errorsIsVirus(err error) bool {
	return strings.Contains(err.Error(), "virus found")
}

func TestClamAVScannerUnreachable(t *testing.T) {
	// Nothing listens on this port.
	s := NewClamAVScanner("127.0.0.1:1", time.Second)
	if _, err := s.Scan(context.Background(), strings.NewReader("x")); err == nil {
		t.Fatal("expected a connection error")
	}
}

// TestWorkerVirusScanFailsOnInfectedObject proves a real scanner failure
// drives the saga to failed.
func TestWorkerVirusScanFailsOnInfectedObject(t *testing.T) {
	w, db, co := newTestWorker(t)
	_ = crypto.Init("") // dev key for encrypting the object key
	ctx := context.Background()
	docID := "20000000-0000-7000-8000-000000000005"
	seedDocument(t, db, docID, "draft")

	// Put an object in the local store and register it with a checksum so the
	// only thing that can fail the step is the scanner itself.
	store := &objectstore.LocalStore{Dir: t.TempDir()}
	key := objectstore.NewKey("bad.bin")
	if _, err := store.Put(ctx, key, strings.NewReader("EICAR"), 5, "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	encKey, err := crypto.Encrypt(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Storage{
		DocumentID: docID, Provider: "local", ObjectKey: encKey, Checksum: "abc", Status: "stored",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Wire a scanner that reports the object as infected.
	w.scanner = NewClamAVScanner(fakeClamd(t, "stream: Eicar-Test-Signature FOUND\x00"), 5*time.Second)
	w.store = store

	if _, err := co.Start(ctx, saga.DocumentUpload, "document", docID, saga.DocumentUploadSteps); err != nil {
		t.Fatal(err)
	}
	if err := w.HandleEvent(ctx, outbox.Event{AggregateType: "document", AggregateID: docID, EventType: "document_uploaded"}); err != nil {
		t.Fatal(err)
	}
	if err := w.HandleEvent(ctx, outbox.Event{AggregateType: "document", AggregateID: docID, EventType: "saga.upload"}); err != nil {
		t.Fatal(err)
	}
	s, err := co.FindByAggregate(ctx, "document", docID)
	if err != nil || s.Status != saga.StatusFailed {
		t.Fatalf("saga should be failed after infection, status=%+v err=%v", s, err)
	}
}

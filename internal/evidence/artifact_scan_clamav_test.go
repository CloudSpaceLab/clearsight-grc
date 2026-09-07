package evidence

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClamAVWorkerDistinguishesIntegrityFromTransportFailure(t *testing.T) {
	for _, tc := range []struct{ name, content, code, state string }{
		{"truncated", "short", "INTEGRITY_MISMATCH", "FAILED"},
		{"oversize", "safe bytes plus", "INTEGRITY_MISMATCH", "FAILED"},
		{"transport closed", "safe bytes", "SCANNER_UNAVAILABLE", "PENDING"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, store, a, now := scanFixture(t)
			if _, err := store.Put(context.Background(), a.StorageKey, strings.NewReader(tc.content), 100); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(time.Second))
				reader := bufio.NewReader(conn)
				for _, command := range []string{"zIDSESSION\x00", "zVERSION\x00"} {
					got, err := reader.ReadString(0)
					if err != nil || got != command {
						done <- io.ErrUnexpectedEOF
						return
					}
				}
				if _, err := io.WriteString(conn, "1: ClamAV 1.4.3/28000/Mon Sep 7 00:00:00 2026\x00"); err != nil {
					done <- err
					return
				}
				if command, err := reader.ReadString(0); err != nil || command != "zINSTREAM\x00" {
					done <- io.ErrUnexpectedEOF
					return
				}
				for {
					var size uint32
					if err := binary.Read(reader, binary.BigEndian, &size); err != nil {
						if errors.Is(err, io.EOF) && tc.code == "INTEGRITY_MISMATCH" {
							done <- nil
						} else {
							done <- err
						}
						return
					}
					if size == 0 {
						if tc.code == "INTEGRITY_MISMATCH" {
							done <- errors.New("invalid stream reached clean terminator")
						} else {
							done <- nil
						}
						return
					}
					if size > 32768 {
						done <- ErrArtifactTooLarge
						return
					}
					if _, err := io.CopyN(io.Discard, reader, int64(size)); err != nil {
						done <- err
						return
					}
				}
			}()
			scanner, err := NewClamAVScanner("tcp", listener.Addr().String(), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			worker := NewArtifactScanWorker(repo, store, scanner, "scanner-worker", 100)
			worker.now = func() time.Time { return now }
			_, _ = worker.Maintain(context.Background(), now, 1)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if len(repo.scanReceipts) != 1 {
				t.Fatalf("receipts=%+v", repo.scanReceipts)
			}
			receipt, job := repo.scanReceipts[0], repo.scanJobs[a.ID]
			if receipt.FailureCode != tc.code || receipt.Verdict != "UNAVAILABLE" || job.State != tc.state || job.Attempt != 1 || repo.artifacts[a.ID].Status != ArtifactStoredUnscanned {
				t.Fatalf("receipt=%+v job=%+v", receipt, job)
			}
		})
	}
}

func TestClamAVStreamUsesExplicitVerdictAndVersion(t *testing.T) {
	for _, tc := range []struct{ reply, want string }{{"2: stream: OK", "CLEAN"}, {"2: stream: Eicar-Test FOUND", "INFECTED"}, {"2: stream: size limit exceeded ERROR", ""}, {"2: other: OK", ""}, {"2: stream: OK\nextra", ""}} {
		t.Run(tc.reply, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(time.Second))
				r := bufio.NewReader(conn)
				for _, command := range []string{"zIDSESSION\x00", "zVERSION\x00"} {
					got, e := r.ReadString(0)
					if e != nil || got != command {
						done <- io.ErrUnexpectedEOF
						return
					}
				}
				_, _ = io.WriteString(conn, "1: ClamAV 1.4.3/28000/Mon Sep 7 00:00:00 2026\x00")
				command, e := r.ReadString(0)
				if e != nil || command != "zINSTREAM\x00" {
					done <- io.ErrUnexpectedEOF
					return
				}
				var received bytes.Buffer
				for {
					var n uint32
					if e := binary.Read(r, binary.BigEndian, &n); e != nil {
						done <- e
						return
					}
					if n == 0 {
						break
					}
					if n > 32768 {
						done <- ErrArtifactTooLarge
						return
					}
					if _, e := io.CopyN(&received, r, int64(n)); e != nil {
						done <- e
						return
					}
				}
				if received.String() != "test content" {
					done <- io.ErrUnexpectedEOF
					return
				}
				_, _ = io.WriteString(conn, tc.reply+"\x00")
				done <- nil
			}()
			scanner, err := NewClamAVScanner("tcp", listener.Addr().String(), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			result, err := scanner.Scan(context.Background(), strings.NewReader("test content"), 12)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("unsafe scanner reply accepted: %+v", result)
				}
			} else if err != nil || result.Verdict != tc.want || result.Scanner != "ClamAV" || !strings.Contains(result.Version, "28000") {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClamAVRejectsInvalidConnectionAndSize(t *testing.T) {
	for _, network := range []string{"", "http", "udp"} {
		if _, err := NewClamAVScanner(network, "unused", time.Second); err == nil {
			t.Fatal("invalid scanner network accepted")
		}
	}
	s, err := NewClamAVScanner("tcp", "127.0.0.1:1", 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int64{0, -1, 101 << 20} {
		if _, err := s.Scan(context.Background(), strings.NewReader("content"), n); err == nil {
			t.Fatal("invalid size accepted")
		}
	}
	if _, err := s.Scan(context.Background(), strings.NewReader("content"), 7); err == nil {
		t.Fatal("unreachable scanner reported clean")
	}
}

func TestClamAVNeverTerminatesAnIncompleteOrOversizeStreamAsClean(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		size          int64
	}{{"truncated", "short", 10}, {"oversize", "longer than manifest", 10}} {
		t.Run(tc.name, func(t *testing.T) {
			var wire bytes.Buffer
			if err := streamClamAV(&wire, strings.NewReader(tc.content), tc.size); err == nil {
				t.Fatal("invalid stream was terminated for inspection")
			}
			reader := bytes.NewReader(wire.Bytes())
			for reader.Len() > 0 {
				var n uint32
				if err := binary.Read(reader, binary.BigEndian, &n); err != nil {
					t.Fatal(err)
				}
				if n == 0 {
					t.Fatal("sent successful end marker for incomplete stream")
				}
				if _, err := io.CopyN(io.Discard, reader, int64(n)); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestClamAVTimeoutCannotProduceCleanReceipt(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(io.Discard, conn)
	}()
	scanner, err := NewClamAVScanner("tcp", listener.Addr().String(), 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := scanner.Scan(context.Background(), strings.NewReader("bytes"), 5); err == nil || result.Verdict == "CLEAN" {
		t.Fatalf("timeout result=%+v err=%v", result, err)
	}
	<-done
}

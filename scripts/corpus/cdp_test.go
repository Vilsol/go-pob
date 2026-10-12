package main

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// TestWSFrameRoundTrip exercises all three payload-length encodings through the masking path.
func TestWSFrameRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int
	}{
		{"7-bit length", 5},
		{"16-bit length", 4096},
		{"64-bit length", 70000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := net.Pipe()
			defer a.Close()
			defer b.Close()

			writer := &wsConn{conn: a, br: bufio.NewReaderSize(a, 1<<16)}
			reader := &wsConn{conn: b, br: bufio.NewReaderSize(b, 1<<16)}

			want := make([]byte, tc.size)
			for i := range want {
				want[i] = byte(i%251 + 1)
			}

			errc := make(chan error, 1)
			go func() { errc <- writer.write(0x1, want) }()

			got, err := reader.read()
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if err := <-errc; err != nil {
				t.Fatalf("write: %v", err)
			}
			if len(got) != len(want) {
				t.Fatalf("read %d bytes, wrote %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got %d, want %d — the mask was not applied symmetrically", i, got[i], want[i])
				}
			}
		})
	}
}

// TestWSFramesAreMasked pins the one rule a client cannot get wrong quietly.
func TestWSFramesAreMasked(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	writer := &wsConn{conn: a, br: bufio.NewReaderSize(a, 1<<16)}
	go func() { _ = writer.write(0x1, []byte("hello")) }()

	hdr := make([]byte, 2)
	if _, err := b.Read(hdr); err != nil {
		t.Fatalf("reading header: %v", err)
	}
	if hdr[0]&0x80 == 0 {
		t.Error("FIN is not set on a single-frame message")
	}
	if hdr[1]&0x80 == 0 {
		t.Error("the mask bit is NOT set — every client frame must be masked")
	}
}

// TestPAReadyPredicatesRejectTheChallenge is the assertion that keeps a transport failure from being recorded as a thin source.
func TestPAReadyPredicatesRejectTheChallenge(t *testing.T) {
	challenge := `<!DOCTYPE html><html><head><title>Just a moment...</title></head>` +
		`<body><div id="challenge-running">Verify you are human</div></body></html>`

	if paReadyListing(challenge) {
		t.Error("the listing predicate accepted a Cloudflare interstitial")
	}
	if paReadyBuild(challenge) {
		t.Error("the build predicate accepted a Cloudflare interstitial")
	}
	if _, blocked := looksBlocked([]byte(challenge), true); !blocked {
		t.Error("looksBlocked did not recognise the interstitial, so render() would treat it as content")
	}

	if !paReadyListing(`<a href="/build/abc123">x</a>`) {
		t.Error("the listing predicate rejected a real listing row")
	}
	if !paReadyBuild(`<a href="https://pobb.in/aBcD1234">pob</a>`) {
		t.Error("the build predicate rejected a real build page")
	}
	if strings.Contains(challenge, "pobb.in/") {
		t.Fatal("fixture is not a control: the challenge page must not contain the marker under test")
	}
}

package frameworkcatalog

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func fixtureReaders(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	manifest, err := os.Open("testdata/synthetic_manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open("testdata/synthetic_source.txt")
	if err != nil {
		manifest.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		manifest.Close()
		source.Close()
	})
	return manifest, source
}

func TestVerifyPublicationSourceFromLocalFixture(t *testing.T) {
	manifest, source := fixtureReaders(t)
	publication, release, size, err := VerifyPublicationSource(manifest, source)
	if err != nil {
		t.Fatal(err)
	}
	if size != 159 || publication.SourceSHA256 != "03b8522b5f96aae0d63748beef09fa930062343d0b1b4d353ccf357f820ddb5e" ||
		publication.EditionCode != "2026" || len(publication.Clauses) != 2 || len(release) != 64 {
		t.Fatalf("unexpected receipt: %+v %q %d", publication, release, size)
	}
}

func TestVerifyPublicationSourceFailsForModifiedArtifact(t *testing.T) {
	manifest, _ := fixtureReaders(t)
	_, _, _, err := VerifyPublicationSource(manifest, strings.NewReader("modified source"))
	if !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("modified source accepted: %v", err)
	}
}

func TestManifestRejectsDuplicateAndUnexpectedFields(t *testing.T) {
	invalid := []string{
		`{"publisher_code":"ONE","publisher_code":"TWO"}`,
		`{"publisher_code":"ONE","clauses":[{"code":"A","code":"B"}]}`,
		`{"publisher_code":"ONE","unknown_field":1}`,
		`{"publisher_code":"ONE"} {"publisher_code":"TWO"}`,
		`{"publisher_code":"ONE"} garbage`,
		`[]`,
	}
	for _, raw := range invalid {
		if _, _, err := ReadPublicationManifest(strings.NewReader(raw)); !errors.Is(err, ErrInvalidPublication) {
			t.Errorf("ambiguous or incomplete manifest accepted %s: %v", raw, err)
		}
	}
}

func TestVerifySourceChecksumBounds(t *testing.T) {
	if _, err := VerifySourceChecksum(strings.NewReader(""), strings.Repeat("a", 64)); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("empty source accepted: %v", err)
	}
	if _, err := VerifySourceChecksum(strings.NewReader("nonempty"), "invalid"); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("bad declared digest accepted: %v", err)
	}
	if _, err := VerifySourceChecksum(bytes.NewReader([]byte("a")), strings.Repeat("0", 64)); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("wrong hash accepted: %v", err)
	}
}

func TestManifestRejectsExcessiveJSONNesting(t *testing.T) {
	deep := strings.Repeat("[", 66) + "0" + strings.Repeat("]", 66)
	if _, _, err := ReadPublicationManifest(strings.NewReader(deep)); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("unbounded JSON depth accepted: %v", err)
	}
}

func TestManifestReaderFailureIsHandled(t *testing.T) {
	if _, _, err := ReadPublicationManifest(failingReader{}); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("read failure not handled: %v", err)
	}
	if _, err := VerifySourceChecksum(failingReader{}, strings.Repeat("a", 64)); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("source read failure not handled: %v", err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

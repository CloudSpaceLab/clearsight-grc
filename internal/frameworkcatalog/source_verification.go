package frameworkcatalog

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Publication manifests and source files are operator-supplied artifacts.
// Verification is deterministic and makes no external requests or database writes.
const (
	MaxPublicationManifestBytes int64 = 16 << 20
	MaxPublicationSourceBytes   int64 = 64 << 20
)

// VerifyPublicationSource validates the complete edition contract and proves
// that its declared source checksum matches the provided local artifact.
// This is not publication approval, issuer authentication or license verification.
func VerifyPublicationSource(manifest, source io.Reader) (Publication, string, int64, error) {
	publication, releaseSHA256, err := ReadPublicationManifest(manifest)
	if err != nil {
		return Publication{}, "", 0, err
	}
	size, err := VerifySourceChecksum(source, publication.SourceSHA256)
	if err != nil {
		return Publication{}, "", 0, err
	}
	return publication, releaseSHA256, size, nil
}

func ReadPublicationManifest(reader io.Reader) (Publication, string, error) {
	if reader == nil {
		return Publication{}, "", fmt.Errorf("%w: missing manifest", ErrInvalidPublication)
	}
	raw, err := io.ReadAll(io.LimitReader(reader, MaxPublicationManifestBytes+1))
	if err != nil {
		return Publication{}, "", fmt.Errorf("%w: read manifest: %v", ErrInvalidPublication, err)
	}
	if len(raw) == 0 || int64(len(raw)) > MaxPublicationManifestBytes {
		return Publication{}, "", fmt.Errorf("%w: manifest size out of bounds", ErrInvalidPublication)
	}
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return Publication{}, "", err
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var publication Publication
	if err := decoder.Decode(&publication); err != nil {
		return Publication{}, "", fmt.Errorf("%w: decode manifest: %v", ErrInvalidPublication, err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Publication{}, "", fmt.Errorf("%w: extra manifest content", ErrInvalidPublication)
	}
	releaseSHA256, err := ValidatePublication(publication)
	if err != nil {
		return Publication{}, "", err
	}
	return publication, releaseSHA256, nil
}

// VerifySourceChecksum bounds the source file and hashes its exact bytes.
// A matching digest does not itself establish an official source or content rights.
func VerifySourceChecksum(reader io.Reader, expected string) (int64, error) {
	if reader == nil || !publicationSHA256Pattern.MatchString(expected) {
		return 0, fmt.Errorf("%w: invalid source checksum", ErrInvalidPublication)
	}
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(reader, MaxPublicationSourceBytes+1))
	if err != nil {
		return 0, fmt.Errorf("%w: read source: %v", ErrInvalidPublication, err)
	}
	if count == 0 || count > MaxPublicationSourceBytes {
		return 0, fmt.Errorf("%w: source size out of bounds", ErrInvalidPublication)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return 0, fmt.Errorf("%w: source checksum mismatch", ErrInvalidPublication)
	}
	return count, nil
}

// Reject ambiguous JSON keys before decoding typed manifests. Go's normal
// decoder accepts duplicate object members and otherwise retains the last one.
func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := consumeUniqueJSONValue(decoder, 0); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPublication, err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: extra JSON value", ErrInvalidPublication)
	}
	return nil
}

func consumeUniqueJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 64 {
		return fmt.Errorf("manifest nesting exceeds safe depth")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			next, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := next.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, exists := seen[name]; exists {
				return fmt.Errorf("duplicate JSON property %q", name)
			}
			seen[name] = struct{}{}
			if err := consumeUniqueJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := consumeUniqueJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

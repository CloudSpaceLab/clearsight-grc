package monitoring

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// UnmarshalJSON keeps persisted/saved form filters on the same bounded typed
// expression contract used by live library queries. Scope fields remain
// non-serializable and unknown wire fields are rejected rather than ignored.
func (filter *FormLibraryFilter) UnmarshalJSON(data []byte) error {
	type wireFilter FormLibraryFilter
	var decoded wireFilter
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return errors.Join(ErrInvalid, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("unexpected trailing form filter data")
		}
		return errors.Join(ErrInvalid, err)
	}
	normalized, err := NormalizeFormFilterExpression(decoded.Expression)
	if err != nil {
		return err
	}
	decoded.Expression = normalized
	*filter = FormLibraryFilter(decoded)
	return nil
}

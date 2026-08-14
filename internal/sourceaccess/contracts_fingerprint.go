package sourceaccess

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

func ViewFingerprint(view View) (string, error) {
	canonical, err := canonicalView(view)
	if err != nil {
		return "", err
	}
	return fingerprintJSON(canonical)
}

func BindingFingerprint(view View, binding Binding) (string, error) {
	canonicalView, err := canonicalView(view)
	if err != nil {
		return "", err
	}
	limits, err := binding.Limits.Normalized()
	if err != nil {
		return "", err
	}
	canonical := binding
	canonical.Operations = append([]Operation{}, binding.Operations...)
	sort.Slice(canonical.Operations, func(i, j int) bool { return canonical.Operations[i] < canonical.Operations[j] })
	canonical.SelectedFields = append([]string{}, binding.SelectedFields...)
	canonical.KeyFields = append([]string{}, binding.KeyFields...)
	canonical.Limits = limits
	payload := struct {
		View    View    `json:"view"`
		Binding Binding `json:"binding"`
	}{View: canonicalView, Binding: canonical}
	return fingerprintJSON(payload)
}

func canonicalView(view View) (View, error) {
	definition, err := canonicalRawJSON(view.Definition)
	if err != nil {
		return View{}, err
	}
	view.Definition = definition
	view.StableKeys = append([]string{}, view.StableKeys...)
	return view, nil
}

func canonicalRawJSON(value json.RawMessage) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: source definition cannot be canonicalized", ErrDefinitionInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("%w: source definition has trailing data", ErrDefinitionInvalid)
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("%w: source definition cannot be canonicalized", ErrDefinitionInvalid)
	}
	return encoded, nil
}

func fingerprintJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("%w: source definition cannot be fingerprinted", ErrDefinitionInvalid)
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:]), nil
}

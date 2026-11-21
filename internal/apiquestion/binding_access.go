package apiquestion

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type BindingCatalog interface {
	Binding(context.Context, string, string, int64) (sourceaccess.BindingRevision, error)
	View(context.Context, string, string, int64) (sourceaccess.ViewRevision, error)
	Connection(context.Context, string, string, int64) (sourceaccess.ConnectionRevision, error)
}
type ActiveSourceScope interface {
	ValidateActiveSourcesForEntity(context.Context, string, string, []string) error
}
type catalogBindingVerifier struct {
	catalog BindingCatalog
	sources ActiveSourceScope
}

func NewCatalogBindingVerifier(catalog BindingCatalog, sources ActiveSourceScope) catalogBindingVerifier {
	return catalogBindingVerifier{catalog: catalog, sources: sources}
}

// AuthorizeQuestionBinding performs a conservative non-executing catalog
// check. It does not establish transaction-time network or activation policy.
func (v catalogBindingVerifier) AuthorizeQuestionBinding(ctx context.Context, tenant, entity, form string, formVersion int64, field, bindingID string, bindingVersion int64, resultType, endpoint string) error {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.TenantID != tenant || actor.LegalEntityID != entity || entity == "*" || form == "" || formVersion < 1 || field == "" || bindingID == "" || bindingVersion < 1 || (resultType != "BOOLEAN" && resultType != "NUMBER") || v.catalog == nil || v.sources == nil {
		return ErrUnavailable
	}
	binding, err := v.catalog.Binding(ctx, tenant, bindingID, bindingVersion)
	if err != nil || binding.BindingID != bindingID || binding.Version != bindingVersion || binding.TenantID == "" || binding.SourceID == "" || binding.ViewID == "" || binding.ViewVersion < 1 || !effectiveCatalogRevision(binding.RevisionLifecycle) || binding.RequiredFreshnessMinutes < 1 || len(binding.Operations) != 1 || binding.Operations[0] != sourceaccess.OperationPage || len(binding.KeyFields) != 0 || !questionSelectedFields(binding.SelectedFields) || !emptyJSONObject(binding.Mapping) || !emptyJSONObject(binding.ParameterSchema) {
		return ErrUnavailable
	}
	if err = v.sources.ValidateActiveSourcesForEntity(ctx, tenant, entity, []string{binding.SourceID}); err != nil {
		return ErrUnavailable
	}
	view, err := v.catalog.View(ctx, tenant, binding.ViewID, binding.ViewVersion)
	if err != nil || view.ViewID != binding.ViewID || view.Version != binding.ViewVersion || view.TenantID != binding.TenantID || view.SourceID != binding.SourceID || view.ConnectionID == "" || view.ConnectionVersion < 1 || view.OutputKind != sourceaccess.OutputRecords || !effectiveCatalogRevision(view.RevisionLifecycle) || !questionNativeSchema(view.NativeSchema, resultType) {
		return ErrUnavailable
	}
	var viewDef sourceaccess.RESTJSONViewDefinition
	if decodeCatalogDefinition(view.Definition, &viewDef) != nil || !viewDef.Singleton || viewDef.ResultType != resultType || viewDef.RecordsPointer != "" || len(viewDef.FixedQuery) != 0 || (viewDef.Pagination.Mode != "" && viewDef.Pagination.Mode != sourceaccess.RESTJSONPaginationNone) || viewDef.Pagination.CursorQueryParam != "" || viewDef.Pagination.NextCursorPointer != "" || viewDef.Pagination.PageSizeQueryParam != "" || viewDef.Lookup != nil || !strings.HasPrefix(viewDef.Path, "/") {
		return ErrUnavailable
	}
	connection, err := v.catalog.Connection(ctx, tenant, view.ConnectionID, view.ConnectionVersion)
	if err != nil || connection.ConnectionID != view.ConnectionID || connection.Version != view.ConnectionVersion || connection.TenantID != binding.TenantID || connection.SourceID != binding.SourceID || connection.AdapterKind != sourceaccess.AdapterRESTJSON || !effectiveCatalogRevision(connection.RevisionLifecycle) || !containsPageCapability(connection.VerifiedCapabilities) {
		return ErrUnavailable
	}
	var connectionDef sourceaccess.RESTJSONConnectionDefinition
	if decodeCatalogDefinition(connection.Definition, &connectionDef) != nil {
		return ErrUnavailable
	}
	base, err := url.Parse(connectionDef.BaseURL)
	if err != nil || base.Scheme != "https" || base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" {
		return ErrUnavailable
	}
	base.Path = strings.TrimRight(base.Path, "/") + viewDef.Path
	base.RawPath = ""
	if base.String() != endpoint {
		return ErrUnavailable
	}
	return nil
}

func effectiveCatalogRevision(lifecycle sourceaccess.RevisionLifecycle) bool {
	now := time.Now().UTC()
	return lifecycle.Status == sourceaccess.RevisionActive && lifecycle.IsCurrent && lifecycle.Version > 0 && lifecycle.EffectiveFrom != nil && !now.Before(*lifecycle.EffectiveFrom) && (lifecycle.EffectiveUntil == nil || now.Before(*lifecycle.EffectiveUntil))
}
func containsPageCapability(values []sourceaccess.Capability) bool {
	for _, value := range values {
		if value == sourceaccess.CapabilityPage {
			return true
		}
	}
	return false
}
func questionSelectedFields(fields []string) bool {
	if len(fields) < 1 || len(fields) > 2 {
		return false
	}
	seen := map[string]bool{}
	for _, field := range fields {
		if (field != "result" && field != "message") || seen[field] {
			return false
		}
		seen[field] = true
	}
	return seen["result"]
}
func questionNativeSchema(fields []sourceaccess.NativeField, resultType string) bool {
	if len(fields) < 1 || len(fields) > 2 {
		return false
	}
	expected := "json:boolean"
	if resultType == "NUMBER" {
		expected = "json:number"
	}
	seen := map[string]bool{}
	for _, field := range fields {
		if seen[field.Name] {
			return false
		}
		seen[field.Name] = true
		if field.Name == "result" {
			if field.NativeType != expected || field.Nullable {
				return false
			}
		} else if field.Name == "message" {
			if field.NativeType != "json:string" {
				return false
			}
		} else {
			return false
		}
	}
	return seen["result"]
}
func emptyJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var obj map[string]json.RawMessage
	return json.Unmarshal(raw, &obj) == nil && obj != nil && len(obj) == 0
}
func decodeCatalogDefinition(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return ErrUnavailable
	}
	return nil
}

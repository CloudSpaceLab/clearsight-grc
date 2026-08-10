package scimapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
)

const maxRequestBytes = 1 << 20

var (
	ErrNotFound = errors.New("SCIM resource not found")
	ErrConflict = errors.New("SCIM resource conflicts with existing state")
	ErrInvalid  = errors.New("SCIM resource is invalid")
)

type Source struct {
	ID               string
	TenantID         string
	IdentityIssuer   string
	SubjectAttribute string
}

type User struct {
	ID          string
	PrincipalID string
	ExternalID  string
	UserName    string
	DisplayName string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Group struct {
	ID          string
	ExternalID  string
	DisplayName string
	Members     []GroupMember
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GroupMember struct {
	UserID      string
	DisplayName string
}

type UserFilter struct {
	UserName   string
	ExternalID string
}

type GroupFilter struct {
	DisplayName string
	ExternalID  string
}

type Repository interface {
	AuthenticateSource(context.Context, []byte) (Source, error)
	CreateUser(context.Context, Source, User) (User, error)
	GetUser(context.Context, Source, string) (User, error)
	ListUsers(context.Context, Source, UserFilter, int, int) ([]User, int, error)
	ReplaceUser(context.Context, Source, string, User) (User, error)
	DeleteUser(context.Context, Source, string) error
	CreateGroup(context.Context, Source, Group) (Group, error)
	GetGroup(context.Context, Source, string) (Group, error)
	ListGroups(context.Context, Source, GroupFilter, int, int) ([]Group, int, error)
	ReplaceGroup(context.Context, Source, string, Group) (Group, error)
	DeleteGroup(context.Context, Source, string) error
}

type sourceContextKey struct{}

type Service struct {
	repository Repository
	logger     *slog.Logger
	handler    http.Handler
}

func New(repository Repository, logger *slog.Logger) (*Service, error) {
	if repository == nil {
		return nil, errors.New("SCIM repository is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	userType := scim.ResourceType{
		ID:          optional.NewString("User"),
		Name:        "User",
		Endpoint:    "/Users",
		Description: optional.NewString("ClearSight provisioned user"),
		Schema:      userSchema(),
		Handler:     &userHandler{repository: repository},
	}
	groupType := scim.ResourceType{
		ID:          optional.NewString("Group"),
		Name:        "Group",
		Endpoint:    "/Groups",
		Description: optional.NewString("ClearSight directory group"),
		Schema:      schema.CoreGroupSchema(),
		Handler:     &groupHandler{repository: repository},
	}
	server, err := scim.NewServer(&scim.ServerArgs{
		ServiceProviderConfig: &scim.ServiceProviderConfig{
			MaxResults:       100,
			SupportFiltering: true,
			SupportPatch:     true,
			AuthenticationSchemes: []scim.AuthenticationScheme{{
				Type: scim.AuthenticationTypeOauthBearerToken, Name: "Bearer Token",
				Description: "Tenant-scoped ClearSight SCIM provisioning token", Primary: true,
			}},
		},
		ResourceTypes: []scim.ResourceType{userType, groupType},
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		repository: repository,
		logger:     logger,
		handler:    http.StripPrefix("/scim", server),
	}, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/scim/v2/") {
		http.NotFound(w, r)
		return
	}
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeSCIMError(w, http.StatusUnauthorized, "valid bearer token required")
		return
	}
	digest := sha256.Sum256([]byte(token))
	source, err := s.repository.AuthenticateSource(r.Context(), digest[:])
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.ErrorContext(r.Context(), "SCIM source authentication failed", "error", err)
		}
		writeSCIMError(w, http.StatusUnauthorized, "valid bearer token required")
		return
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	}
	ctx := context.WithValue(r.Context(), sourceContextKey{}, source)
	s.handler.ServeHTTP(w, r.WithContext(ctx))
}

func SourceFromContext(ctx context.Context) (Source, bool) {
	source, ok := ctx.Value(sourceContextKey{}).(Source)
	return source, ok
}

func HashToken(token string) [32]byte { return sha256.Sum256([]byte(token)) }

func bearerToken(header string) (string, bool) {
	if len(header) > 4096 {
		return "", false
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) < 32 || len(parts[1]) > 2048 {
		return "", false
	}
	return parts[1], true
}

func writeSCIMError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(scimerrors.ScimError{Status: status, Detail: detail})
}

func userSchema() schema.Schema {
	return schema.Schema{
		ID: schema.UserSchema, Name: optional.NewString("User"), Description: optional.NewString("ClearSight provisioned user"),
		Attributes: []schema.CoreAttribute{
			schema.SimpleCoreAttribute(schema.SimpleStringParams(schema.StringParams{Name: "userName", Required: true, Uniqueness: schema.AttributeUniquenessServer()})),
			schema.SimpleCoreAttribute(schema.SimpleStringParams(schema.StringParams{Name: "displayName"})),
			schema.SimpleCoreAttribute(schema.SimpleBooleanParams(schema.BooleanParams{Name: "active"})),
		},
	}
}

func sourceForRequest(r *http.Request) (Source, error) {
	source, ok := SourceFromContext(r.Context())
	if !ok || source.ID == "" || source.TenantID == "" {
		return Source{}, scimerrors.ScimErrorInternal
	}
	return source, nil
}

func externalID(value scim.ResourceAttributes) string {
	return stringAttribute(value, "externalId")
}

func stringAttribute(attributes scim.ResourceAttributes, key string) string {
	value, _ := attributes[key].(string)
	return strings.TrimSpace(value)
}

func boolAttribute(attributes scim.ResourceAttributes, key string, fallback bool) bool {
	value, ok := attributes[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func resourceFromUser(user User) scim.Resource {
	created, updated := user.CreatedAt, user.UpdatedAt
	resource := scim.Resource{
		ID: user.ID,
		Attributes: scim.ResourceAttributes{
			"userName": user.UserName, "displayName": user.DisplayName, "active": user.Active,
		},
		Meta: scim.Meta{Created: &created, LastModified: &updated},
	}
	if user.ExternalID != "" {
		resource.ExternalID = optional.NewString(user.ExternalID)
	}
	return resource
}

func resourceFromGroup(group Group) scim.Resource {
	created, updated := group.CreatedAt, group.UpdatedAt
	members := make([]map[string]any, 0, len(group.Members))
	for _, member := range group.Members {
		members = append(members, map[string]any{"value": member.UserID, "type": "User", "display": member.DisplayName})
	}
	resource := scim.Resource{
		ID: group.ID,
		Attributes: scim.ResourceAttributes{"displayName": group.DisplayName, "members": members},
		Meta: scim.Meta{Created: &created, LastModified: &updated},
	}
	if group.ExternalID != "" {
		resource.ExternalID = optional.NewString(group.ExternalID)
	}
	return resource
}

func constantTimeBytesEqual(left, right []byte) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare(left, right) == 1
}

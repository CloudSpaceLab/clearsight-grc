package sourceaccess

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const RESTJSONAdapterVersion = "rest-json-v1"

type RESTJSONAuthKind string

const (
	RESTJSONAuthNone   RESTJSONAuthKind = "NONE"
	RESTJSONAuthBearer RESTJSONAuthKind = "BEARER"
	RESTJSONAuthHeader RESTJSONAuthKind = "HEADER"
)

type RESTJSONAuthentication struct {
	Kind       RESTJSONAuthKind `json:"kind"`
	HeaderName string           `json:"header_name,omitempty"`
}

type RESTJSONConnectionDefinition struct {
	BaseURL        string                 `json:"base_url"`
	Authentication RESTJSONAuthentication `json:"authentication"`
}

type RESTJSONOptions struct {
	Client *http.Client
}

func DefaultRESTJSONOptions() RESTJSONOptions {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 16
	transport.MaxIdleConnsPerHost = 4
	transport.IdleConnTimeout = 90 * time.Second
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = 10 * time.Second
	transport.ExpectContinueTimeout = time.Second
	return RESTJSONOptions{Client: &http.Client{Transport: transport}}
}

type RESTJSONAdapter struct {
	client *http.Client
}

func NewRESTJSONAdapter(options RESTJSONOptions) RESTJSONAdapter {
	client := options.Client
	if client == nil {
		client = DefaultRESTJSONOptions().Client
	}
	clone := *client
	clone.Timeout = 0 // operation contexts own the deadline
	clone.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return RESTJSONAdapter{client: &clone}
}

func (a RESTJSONAdapter) Open(ctx context.Context, connection Connection, resolver SecretResolver) (Session, error) {
	if err := connection.Validate(); err != nil {
		return nil, err
	}
	if connection.AdapterKind != AdapterRESTJSON || connection.AdapterVersion != RESTJSONAdapterVersion {
		return nil, fmt.Errorf("%w: matching REST/JSON adapter version is required", ErrDefinitionInvalid)
	}
	definition, err := decodeRESTJSONConnection(connection)
	if err != nil {
		return nil, err
	}
	baseURL, err := url.Parse(definition.BaseURL)
	if err != nil {
		return nil, ErrDefinitionInvalid
	}

	secret := ""
	switch definition.Authentication.Kind {
	case RESTJSONAuthNone:
		if strings.TrimSpace(connection.SecretRef) != "" {
			return nil, fmt.Errorf("%w: unauthenticated REST connections cannot carry a secret reference", ErrDefinitionInvalid)
		}
	case RESTJSONAuthBearer, RESTJSONAuthHeader:
		if resolver == nil || strings.TrimSpace(connection.SecretRef) == "" {
			return nil, ErrCredentials
		}
		secret, err = resolver.Resolve(ctx, connection.SecretRef)
		if err != nil || secret == "" || secret != strings.TrimSpace(secret) || len(secret) > HardMaxDefinitionBytes || containsControl(secret) {
			secret = ""
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, ErrCredentials
		}
	default:
		return nil, ErrDefinitionInvalid
	}

	return &RESTJSONSession{
		connection:     connection,
		baseURL:        baseURL,
		authentication: definition.Authentication,
		secret:         secret,
		client:         a.client,
		now:            time.Now,
	}, nil
}

type RESTJSONSession struct {
	connection     Connection
	baseURL        *url.URL
	authentication RESTJSONAuthentication
	secret         string
	client         *http.Client
	now            func() time.Time
}

func (s *RESTJSONSession) Connection() Connection {
	if s == nil {
		return Connection{}
	}
	return s.connection
}

func (s *RESTJSONSession) Capabilities() CapabilitySet {
	return NewCapabilitySet(CapabilityInspect, CapabilityPage, CapabilityLookup)
}

func (s *RESTJSONSession) Close() error {
	if s != nil {
		s.secret = ""
	}
	return nil
}

func (s *RESTJSONSession) ready(view View) error {
	if s == nil || s.client == nil || s.baseURL == nil {
		return ErrConnection
	}
	if err := s.connection.Validate(); err != nil {
		return err
	}
	return view.Validate(s.connection)
}

func (s *RESTJSONSession) authenticate(request *http.Request) error {
	if s == nil || request == nil {
		return ErrConnection
	}
	switch s.authentication.Kind {
	case RESTJSONAuthNone:
		return nil
	case RESTJSONAuthBearer:
		if s.secret == "" {
			return ErrCredentials
		}
		request.Header.Set("Authorization", "Bearer "+s.secret)
		return nil
	case RESTJSONAuthHeader:
		if s.secret == "" || !validRESTHeaderName(s.authentication.HeaderName) {
			return ErrCredentials
		}
		request.Header.Set(s.authentication.HeaderName, s.secret)
		return nil
	default:
		return ErrCredentials
	}
}

package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type SMTPConfig struct {
	Enabled     bool
	Host        string
	Port        int
	Username    string
	SecretRef   string
	FromAddress string
	TLSMode     string
}

func LoadSMTPConfig(environment string) (SMTPConfig, error) {
	host := strings.TrimSpace(os.Getenv("CLEARSIGHT_SMTP_HOST"))
	if host == "" {
		return SMTPConfig{}, nil
	}
	port := 587
	if raw := strings.TrimSpace(os.Getenv("CLEARSIGHT_SMTP_PORT")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 65535 {
			return SMTPConfig{}, fmt.Errorf("CLEARSIGHT_SMTP_PORT must be between 1 and 65535")
		}
		port = value
	}
	value := SMTPConfig{
		Enabled:     true,
		Host:        host,
		Port:        port,
		Username:    strings.TrimSpace(os.Getenv("CLEARSIGHT_SMTP_USERNAME")),
		SecretRef:   strings.TrimSpace(os.Getenv("CLEARSIGHT_SMTP_SECRET_REF")),
		FromAddress: strings.TrimSpace(os.Getenv("CLEARSIGHT_SMTP_FROM")),
		TLSMode:     strings.ToUpper(strings.TrimSpace(env("CLEARSIGHT_SMTP_TLS_MODE", "STARTTLS"))),
	}
	if strings.ContainsAny(value.Host, "/\r\n\t ") || value.FromAddress == "" {
		return SMTPConfig{}, fmt.Errorf("SMTP host and from address are required and must be valid")
	}
	if value.Username != "" && value.SecretRef == "" {
		return SMTPConfig{}, fmt.Errorf("SMTP username requires CLEARSIGHT_SMTP_SECRET_REF")
	}
	if value.Username == "" && value.SecretRef != "" {
		return SMTPConfig{}, fmt.Errorf("CLEARSIGHT_SMTP_SECRET_REF requires CLEARSIGHT_SMTP_USERNAME")
	}
	switch value.TLSMode {
	case "STARTTLS", "TLS":
	case "PLAIN":
		if !strings.EqualFold(environment, "development") || value.Username != "" {
			return SMTPConfig{}, fmt.Errorf("plaintext SMTP is restricted to unauthenticated development use")
		}
	default:
		return SMTPConfig{}, fmt.Errorf("CLEARSIGHT_SMTP_TLS_MODE must be STARTTLS, TLS or PLAIN")
	}
	if value.SecretRef != "" && !validEnvironmentSecretReference(value.SecretRef) {
		return SMTPConfig{}, fmt.Errorf("CLEARSIGHT_SMTP_SECRET_REF must use env:VARIABLE_NAME")
	}
	return value, nil
}

type EnvironmentSecretResolver struct{}

func (EnvironmentSecretResolver) ResolveSecret(ctx context.Context, reference string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !validEnvironmentSecretReference(reference) {
		return "", fmt.Errorf("invalid environment secret reference")
	}
	name := strings.TrimPrefix(reference, "env:")
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("configured secret is unavailable")
	}
	return value, nil
}

func validEnvironmentSecretReference(reference string) bool {
	if !strings.HasPrefix(reference, "env:") {
		return false
	}
	name := strings.TrimPrefix(reference, "env:")
	if name == "" || len(name) > 128 {
		return false
	}
	for index, char := range name {
		if (char >= 'A' && char <= 'Z') || char == '_' || (index > 0 && char >= '0' && char <= '9') {
			continue
		}
		return false
	}
	return true
}

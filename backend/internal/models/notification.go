package models

import (
	"time"
)

type Notification struct {
	ID        string     `db:"id" json:"id"`
	UserID    int64      `db:"user_id" json:"-"`
	Title     string     `db:"title" json:"title"`
	Body      string     `db:"body" json:"body"`
	URL       *string    `db:"url" json:"url"`
	Level     string     `db:"level" json:"level"`
	DedupeKey *string    `db:"dedupe_key" json:"-"`
	ReadAt    *time.Time `db:"read_at" json:"read_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

func (n Notification) IsUnread() bool {
	return n.ReadAt == nil
}

type StorageSettings struct {
	ID                        int64     `db:"id" json:"-"`
	Provider                  string    `db:"provider" json:"provider"`
	MaxUploadMB               int64     `db:"max_upload_mb" json:"max_upload_mb"`
	AwsAccessKey              *string   `db:"aws_access_key" json:"-"`
	AwsSecretKey              *string   `db:"aws_secret_key" json:"-"`
	AwsRegion                 string    `db:"aws_region" json:"aws_region"`
	AwsBucket                 *string   `db:"aws_bucket" json:"aws_bucket"`
	AwsEndpoint               *string   `db:"aws_endpoint" json:"aws_endpoint"`
	AwsPublicURL              *string   `db:"aws_public_url" json:"aws_public_url"`
	AwsUsePathStyle           bool      `db:"aws_use_path_style" json:"aws_use_path_style"`
	CloudflareAccountID       *string   `db:"cloudflare_account_id" json:"cloudflare_account_id"`
	CloudflareAccessKeyID     *string   `db:"cloudflare_access_key_id" json:"-"`
	CloudflareSecretAccessKey *string   `db:"cloudflare_secret_access_key" json:"-"`
	CloudflareBucket          *string   `db:"cloudflare_bucket" json:"cloudflare_bucket"`
	CloudflarePublicURL       *string   `db:"cloudflare_public_url" json:"cloudflare_public_url"`
	UpdatedAt                 time.Time `db:"updated_at" json:"updated_at"`
}

func (s StorageSettings) HasAWSSecret() bool { return s.AwsSecretKey != nil && *s.AwsSecretKey != "" }
func (s StorageSettings) HasCFSecret() bool {
	return s.CloudflareSecretAccessKey != nil && *s.CloudflareSecretAccessKey != ""
}
func (s StorageSettings) HasAWSAccess() bool { return s.AwsAccessKey != nil && *s.AwsAccessKey != "" }
func (s StorageSettings) HasCFAccess() bool {
	return s.CloudflareAccessKeyID != nil && *s.CloudflareAccessKeyID != ""
}
func (s StorageSettings) HasAWSBucket() bool { return s.AwsBucket != nil && *s.AwsBucket != "" }
func (s StorageSettings) HasCFBucket() bool {
	return s.CloudflareBucket != nil && *s.CloudflareBucket != ""
}

func (s StorageSettings) Configured(provider string) bool {
	switch provider {
	case "local":
		return true
	case "aws":
		return s.HasAWSAccess() && s.HasAWSSecret() && s.HasAWSBucket()
	case "cloudflare":
		return s.HasCFAccess() && s.HasCFSecret() && s.HasCFBucket()
	default:
		return false
	}
}

// EffectiveMaxUploadMB falls back to a sane default when the stored value is
// missing or invalid.
func (s StorageSettings) EffectiveMaxUploadMB(fallback int64) int64 {
	if s.MaxUploadMB > 0 {
		return s.MaxUploadMB
	}
	if fallback > 0 {
		return fallback
	}
	return 20
}

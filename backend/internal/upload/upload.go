package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/crypto"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

var ErrInvalidImage = errors.New("invalid image")
var ErrInvalidFile = errors.New("invalid file")

var allowedDocumentTypes = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".zip": true, ".rar": true, ".txt": true,
	".csv": true, ".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
	".kml": true, ".kmz": true, ".dwg": true, ".dxf": true, ".ics": true,
	".7z": true, ".gz": true, ".tar": true, ".json": true, ".xml": true,
	".md": true, ".odt": true, ".ods": true, ".odp": true, ".epub": true,
}

type Service struct {
	store  *store.Store
	config config.Config
	cipher *crypto.Cipher
}

func New(s *store.Store, cfg config.Config, cipher *crypto.Cipher) *Service {
	return &Service{store: s, config: cfg, cipher: cipher}
}

type Result struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

var allowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
	"image/bmp":  ".bmp",
}

func (s *Service) MaxUploadMB() int64 {
	settings, err := s.store.GetStorageSettings()
	if err != nil {
		return s.config.UploadMaxMB
	}
	return settings.EffectiveMaxUploadMB(s.config.UploadMaxMB)
}

// Provider returns the effective storage provider ("local", "aws" or
// "cloudflare"). It is used by modules that need to know whether a stored file
// is served by the application or by a remote bucket.
func (s *Service) Provider() string {
	settings, err := s.store.GetStorageSettings()
	if err != nil || !settings.Configured(settings.Provider) {
		return "local"
	}
	return settings.Provider
}

func (s *Service) Save(ctx context.Context, file multipart.File, header *multipart.FileHeader) (Result, error) {
	head := make([]byte, 512)
	read, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Result{}, err
	}
	head = head[:read]
	contentType := http.DetectContentType(head)
	extension, ok := allowedTypes[contentType]
	if !ok {
		return Result{}, ErrInvalidImage
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Result{}, err
	}

	now := time.Now()
	key := fmt.Sprintf("uploads/%d/%02d/%s%s", now.Year(), int(now.Month()), uuid.NewString(), extension)

	settings, err := s.store.GetStorageSettings()
	if err != nil {
		return Result{}, err
	}
	provider := settings.Provider
	if !settings.Configured(provider) {
		provider = "local"
	}

	if provider == "local" {
		return s.saveLocal(key, file)
	}
	return s.uploadToS3(ctx, key, contentType, file)
}

func (s *Service) SaveDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader, prefix string) (Result, error) {
	extension := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedDocumentTypes[extension] {
		return Result{}, ErrInvalidFile
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	now := time.Now()
	key := fmt.Sprintf("%s/%d/%02d/%s%s", strings.Trim(prefix, "/"), now.Year(), int(now.Month()), uuid.NewString(), extension)
	settings, err := s.store.GetStorageSettings()
	if err != nil {
		return Result{}, err
	}
	provider := settings.Provider
	if !settings.Configured(provider) {
		provider = "local"
	}
	if provider == "local" {
		return s.saveLocal(key, file)
	}
	return s.uploadToS3(ctx, key, contentType, file)
}

func (s *Service) DeleteLocal(key string) {
	if key == "" {
		return
	}
	relative := strings.TrimPrefix(key, "uploads/")
	_ = os.Remove(filepath.Join(s.config.UploadDir, filepath.FromSlash(relative)))
}

func (s *Service) saveLocal(key string, file multipart.File) (Result, error) {
	relative := strings.TrimPrefix(key, "uploads/")
	target := filepath.Join(s.config.UploadDir, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Result{}, err
	}
	output, err := os.Create(target)
	if err != nil {
		return Result{}, err
	}
	defer output.Close()
	if _, err := io.Copy(output, file); err != nil {
		return Result{}, err
	}
	return Result{
		URL:  strings.TrimRight(s.config.PublicURL, "/") + "/" + key,
		Path: key,
	}, nil
}

// uploadToS3 sends the file to AWS S3 or Cloudflare R2.
func (s *Service) uploadToS3(ctx context.Context, key, contentType string, file multipart.File) (Result, error) {
	settings, err := s.store.GetStorageSettings()
	if err != nil {
		return Result{}, err
	}
	target, err := s.s3Target(ctx, settings)
	if err != nil {
		return Result{}, err
	}
	_, err = target.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(target.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("upload: s3 put failed: %v", err)
		return Result{}, err
	}
	return Result{URL: target.publicURLFor(key), Path: key}, nil
}

// ExportObjects streams every application object stored in the configured
// remote bucket (uploads, images and module attachments). It is a no-op when
// the provider is local.
func (s *Service) ExportObjects(ctx context.Context, add func(key string, body io.Reader) error) error {
	settings, err := s.store.GetStorageSettings()
	if err != nil {
		return err
	}
	if !settings.Configured(settings.Provider) || settings.Provider == "local" {
		return nil
	}
	target, err := s.s3Target(ctx, settings)
	if err != nil {
		return err
	}
	paginator := s3.NewListObjectsV2Paginator(target.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(target.bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, object := range page.Contents {
			key := aws.ToString(object.Key)
			if !isApplicationObject(key) {
				continue
			}
			result, err := target.client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(target.bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return fmt.Errorf("download %s: %w", key, err)
			}
			exportErr := add(key, result.Body)
			_ = result.Body.Close()
			if exportErr != nil {
				return exportErr
			}
		}
	}
	return nil
}

// ImportObject writes an object back to the active storage. For remote
// providers it uploads to the bucket; otherwise it saves under UPLOAD_DIR.
func (s *Service) ImportObject(ctx context.Context, key string, body io.Reader) error {
	settings, err := s.store.GetStorageSettings()
	if err != nil {
		return err
	}
	if !settings.Configured(settings.Provider) || settings.Provider == "local" {
		relative := strings.TrimPrefix(key, "uploads/")
		target := filepath.Join(s.config.UploadDir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		output, err := os.Create(target)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, body)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	target, err := s.s3Target(ctx, settings)
	if err != nil {
		return err
	}
	contentType := mime.TypeByExtension(filepath.Ext(key))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err = target.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(target.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	return err
}

type s3Target struct {
	client    *s3.Client
	bucket    string
	endpoint  string
	region    string
	publicURL string
}

func (t s3Target) publicURLFor(key string) string {
	publicURL := t.publicURL
	if publicURL == "" {
		if t.endpoint != "" {
			publicURL = strings.TrimRight(t.endpoint, "/") + "/" + t.bucket
		} else {
			publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", t.bucket, t.region)
		}
	}
	return strings.TrimRight(publicURL, "/") + "/" + key
}

func (s *Service) s3Target(ctx context.Context, settings models.StorageSettings) (s3Target, error) {
	var (
		accessKey, secretKey, region, bucket, endpoint, publicURL string
		pathStyle                                                 bool
	)
	switch settings.Provider {
	case "aws":
		accessKey = deref(settings.AwsAccessKey)
		secretKey = deref(settings.AwsSecretKey)
		region = settings.AwsRegion
		bucket = deref(settings.AwsBucket)
		endpoint = deref(settings.AwsEndpoint)
		publicURL = deref(settings.AwsPublicURL)
		pathStyle = settings.AwsUsePathStyle
	case "cloudflare":
		accessKey = deref(settings.CloudflareAccessKeyID)
		secretKey = deref(settings.CloudflareSecretAccessKey)
		region = "auto"
		bucket = deref(settings.CloudflareBucket)
		account := deref(settings.CloudflareAccountID)
		if account != "" {
			endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", account)
		}
		publicURL = deref(settings.CloudflarePublicURL)
		pathStyle = true
	default:
		return s3Target{}, fmt.Errorf("unsupported storage provider %q", settings.Provider)
	}

	secretKey, err := s.cipher.Decrypt(secretKey)
	if err != nil {
		return s3Target{}, err
	}
	accessKey, err = s.cipher.Decrypt(accessKey)
	if err != nil {
		return s3Target{}, err
	}
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return s3Target{}, err
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
		options.UsePathStyle = pathStyle
	})
	return s3Target{
		client:    client,
		bucket:    bucket,
		endpoint:  endpoint,
		region:    region,
		publicURL: publicURL,
	}, nil
}

// applicationPrefixes lists the key prefixes used by images and module
// attachments. Other objects in the bucket are ignored by backups.
var applicationPrefixes = []string{"uploads/", "demands/", "links/", "nokia/", "stations/", "projects/"}

func isApplicationObject(key string) bool {
	for _, prefix := range applicationPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

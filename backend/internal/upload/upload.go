package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

var ErrInvalidImage = errors.New("invalid image")

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
	provider := settings.Provider

	var (
		accessKey, secretKey, region, bucket, endpoint, publicURL string
		pathStyle                                                 bool
	)
	switch provider {
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
		return Result{}, fmt.Errorf("unsupported storage provider %q", provider)
	}

	secretKey, err = s.cipher.Decrypt(secretKey)
	if err != nil {
		return Result{}, err
	}
	accessKey, err = s.cipher.Decrypt(accessKey)
	if err != nil {
		return Result{}, err
	}
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return Result{}, err
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
		options.UsePathStyle = pathStyle
	})

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("upload: s3 put failed: %v", err)
		return Result{}, err
	}

	if publicURL == "" {
		if endpoint != "" {
			publicURL = strings.TrimRight(endpoint, "/") + "/" + bucket
		} else {
			publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
		}
	}
	return Result{URL: strings.TrimRight(publicURL, "/") + "/" + key, Path: key}, nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

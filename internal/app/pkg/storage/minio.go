package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIO struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

// NewMinIO создает клиента MinIO. hostPort например "127.0.0.1:9000".
func NewMinIO(hostPort, accessKey, secretKey, bucket string, useSSL bool, publicBase string) (*MinIO, error) {
	endpoint := hostPort
	c, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: useSSL})
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := c.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := c.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &MinIO{client: c, bucket: bucket, publicBase: strings.TrimRight(publicBase, "/")}, nil
}

// sanitizeFileName латинизирует имя файла: только [a-z0-9-_].
var nonSafe = regexp.MustCompile(`[^a-z0-9\-_.]+`)

func sanitizeFileName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = nonSafe.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-_")
	if name == "" {
		name = "file"
	}
	return name
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// UploadImage загружает файл из multipart и возвращает относительный путь key и публичный URL.
func (m *MinIO) UploadImage(ctx context.Context, fileHeader *multipart.FileHeader, prefix string) (key string, publicURL string, err error) {
	f, err := fileHeader.Open()
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, f); err != nil {
		return "", "", err
	}

	base := sanitizeFileName(prefix)
	ext := path.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".bin"
	}
	key = fmt.Sprintf("%s-%s%s", base, randomHex(4), ext)

	_, err = m.client.PutObject(ctx, m.bucket, key, bytes.NewReader(buf.Bytes()), int64(buf.Len()), minio.PutObjectOptions{ContentType: fileHeader.Header.Get("Content-Type")})
	if err != nil {
		return "", "", err
	}

	u, _ := url.Parse(m.publicBase)
	u.Path = path.Join(u.Path, m.bucket, key)
	return key, u.String(), nil
}

func (m *MinIO) DeleteImage(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

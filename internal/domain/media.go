package domain

import (
	"context"
	"io"
)

// MaxImageBytes caps uploaded images (avatars and similar).
const MaxImageBytes = 5 << 20

// imageTypes maps sniffed content types to stored file extensions.
var imageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// ImageExtension returns the file extension for an allowed image content type.
func ImageExtension(contentType string) (string, bool) {
	ext, ok := imageTypes[contentType]
	return ext, ok
}

type MediaObject struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
	ETag          string
}

// MediaStorage stores uploaded files under opaque keys.
type MediaStorage interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (*MediaObject, error)
	Delete(ctx context.Context, key string) error
}

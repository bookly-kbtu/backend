package platform

import (
	"bytes"
	"errors"
	"testing"

	"github.com/bookly-kbtu/backend/internal/domain"
)

func TestReadImage(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 32)...)

	for name, tc := range map[string]struct {
		body    []byte
		size    int64
		wantExt string
		wantErr bool
	}{
		"png":                {body: png, size: int64(len(png)), wantExt: "png"},
		"webp":               {body: webp, size: int64(len(webp)), wantExt: "webp"},
		"html is rejected":   {body: []byte("<html><script>alert(1)</script>"), size: 31, wantErr: true},
		"declared too large": {body: png, size: domain.MaxImageBytes + 1, wantErr: true},
		"actually too large": {body: append(png, make([]byte, domain.MaxImageBytes)...), size: 10, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, ext, err := readImage(Upload{Body: bytes.NewReader(tc.body), Size: tc.size})
			if tc.wantErr {
				if !errors.Is(err, domain.ErrValidation) {
					t.Fatalf("want validation error, got %v", err)
				}
				return
			}
			if err != nil || ext != tc.wantExt {
				t.Fatalf("got ext %q err %v, want %q", ext, err, tc.wantExt)
			}
		})
	}
}

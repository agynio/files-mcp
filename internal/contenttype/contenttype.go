package contenttype

import (
	"mime"
	"strings"
)

type Kind int

const (
	KindText Kind = iota
	KindImage
	KindResource
)

func Classify(contentType string) Kind {
	mediaType := strings.TrimSpace(contentType)
	if mediaType != "" {
		if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
			mediaType = parsed
		}
	}
	mediaType = strings.ToLower(mediaType)

	switch {
	case strings.HasPrefix(mediaType, "image/"):
		return KindImage
	case strings.HasPrefix(mediaType, "text/"):
		return KindText
	case mediaType == "application/json", mediaType == "application/xml", mediaType == "application/yaml":
		return KindText
	default:
		return KindResource
	}
}

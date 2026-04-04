package contenttype

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        Kind
	}{
		{name: "image png", contentType: "image/png", want: KindImage},
		{name: "image jpeg uppercase", contentType: "IMAGE/JPEG", want: KindImage},
		{name: "image params", contentType: "image/png; charset=binary", want: KindImage},
		{name: "text plain", contentType: "text/plain", want: KindText},
		{name: "text html uppercase", contentType: "TEXT/HTML", want: KindText},
		{name: "text params", contentType: "text/plain; charset=utf-8", want: KindText},
		{name: "application json", contentType: "application/json", want: KindText},
		{name: "application xml", contentType: "application/xml", want: KindText},
		{name: "application yaml", contentType: "application/yaml", want: KindText},
		{name: "application json params", contentType: "application/json; charset=utf-8", want: KindText},
		{name: "application json uppercase", contentType: "Application/JSON", want: KindText},
		{name: "application pdf", contentType: "application/pdf", want: KindResource},
		{name: "octet stream", contentType: "application/octet-stream", want: KindResource},
		{name: "empty", contentType: "", want: KindResource},
		{name: "whitespace", contentType: "  ", want: KindResource},
		{name: "malformed", contentType: "not a type", want: KindResource},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Classify(test.contentType)
			if got != test.want {
				t.Fatalf("Classify(%q) = %v, want %v", test.contentType, got, test.want)
			}
		})
	}
}

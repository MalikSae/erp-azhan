package media

import "testing"

func TestPublicMediaCacheControl(t *testing.T) {
	cases := map[string]string{
		"6b27a769-1a2b-4c3d-8e9f-0123456789ab.webp": "public, max-age=31536000, immutable",
		"6B27A769-1A2B-4C3D-8E9F-0123456789AB.PNG":  "public, max-age=31536000, immutable",
		"garuda-indonesia.webp":                     "public, max-age=86400",
		"6b27a769.webp":                             "public, max-age=86400",
	}
	for name, want := range cases {
		if got := publicMediaCacheControl(name); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}

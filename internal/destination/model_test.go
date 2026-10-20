package destination

import (
	"strings"
	"testing"
)

func TestValidSlug(t *testing.T) {
	ok := []string{"dubai", "abu-dhabi", "turki", "uzbekistan-2"}
	bad := []string{"", "Dubai", "abu--dhabi", "-dubai", "dubai-", "abu dhabi", "dubai/turki"}
	for _, s := range ok {
		if !ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = true, want false", s)
		}
	}
	if ValidSlug(strings.Repeat("a", maxSlugLen+1)) {
		t.Error("slug lebih dari batas panjang harus ditolak")
	}
}

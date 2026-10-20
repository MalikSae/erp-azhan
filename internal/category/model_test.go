package category

import "testing"

func TestNormalizeSEOTier(t *testing.T) {
	s := func(v string) *string { return &v }
	if v, ok := normalizeSEOTier(nil); v != nil || !ok {
		t.Error("nil harus (nil, true)")
	}
	if v, ok := normalizeSEOTier(s("  ")); v != nil || !ok {
		t.Error("kosong harus (nil, true)")
	}
	if v, ok := normalizeSEOTier(s(" Reguler ")); !ok || v == nil || *v != "reguler" {
		t.Errorf("Reguler harus dinormalkan ke reguler, dapat %v %v", v, ok)
	}
	for _, bad := range []string{"vip", "low cost", "plus"} {
		if _, ok := normalizeSEOTier(s(bad)); ok {
			t.Errorf("tier %q harus ditolak", bad)
		}
	}
}

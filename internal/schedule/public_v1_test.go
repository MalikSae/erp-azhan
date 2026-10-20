package schedule

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"
)

func strp(s string) *string { return &s }

func TestParseFilterV1(t *testing.T) {
	f, err := parseFilterV1(url.Values{
		"periode": {"2027-01"}, "destinasi": {"Dubai"}, "tier": {"Reguler"}, "program": {"plus"},
		"durasi": {"9"}, "hotel_bintang": {"4"}, "harga_maks": {"35000000"}, "registrasi": {"almost_full"},
		"updated_since": {"2026-10-01"}, "page": {"2"}, "per_page": {"20"},
	})
	if err != nil {
		t.Fatalf("parseFilterV1: %v", err)
	}
	if f.Periode != "2027-01" || f.Destinasi != "dubai" || f.Tier != "reguler" || f.Program != "plus" ||
		f.Durasi != 9 || f.HotelBintang != 4 || f.HargaMaks != 35000000 || f.Registrasi != RegAlmostFull ||
		f.Page != 2 || f.PerPage != 20 || f.UpdatedSince.IsZero() {
		t.Fatalf("filter tidak sesuai: %+v", f)
	}

	def, err := parseFilterV1(url.Values{})
	if err != nil || def.Page != 1 || def.PerPage != defaultPerPage {
		t.Fatalf("default salah: %+v err=%v", def, err)
	}

	bad := []url.Values{
		{"periode": {"2027-13"}}, {"periode": {"januari"}}, {"tier": {"vip"}}, {"registrasi": {"FULL"}},
		{"durasi": {"0"}}, {"durasi": {"x"}}, {"hotel_bintang": {"6"}}, {"harga_maks": {"-1"}},
		{"updated_since": {"kemarin"}}, {"page": {"0"}}, {"per_page": {"201"}}, {"per_page": {"0"}},
	}
	for _, q := range bad {
		if _, err := parseFilterV1(q); err == nil {
			t.Errorf("parseFilterV1(%v) harus gagal", q)
		}
	}
}

func samplePackage(id int64, berangkat, pulang string, sisa int) PackageV1 {
	ps := &PublicSchedule{
		ID: id, BrandID: 1, JadwalNama: "Umroh Plus Maroko", BerangkatTanggal: berangkat, PulangTanggal: pulang,
		SeatTotal: 45, SeatSisa: sisa, HargaQuad: 34900000, HargaTriple: 36900000, HargaDouble: 39900000,
		HotelMekkah: HotelRef{StarRating: 5}, UpdatedAt: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
	}
	k := Klasifikasi{
		JourneyType: JourneyUmroh, ProgramSlug: strp("plus"),
		Destination: &DestinationRef{ID: 3, Name: "Maroko", Slug: "maroko"},
		Tier:        &TierRef{SEO: strp("reguler"), CategoryID: 2, CategoryName: "Reguler", CategorySlug: "reguler"},
	}
	return newPackageV1(ps, k, "2026-11-19")
}

func TestNewPackageV1Derivatives(t *testing.T) {
	p := samplePackage(1, "2026-12-20", "2026-12-31", 25)
	if p.DurationDays != 12 {
		t.Errorf("DurationDays = %d, want 12", p.DurationDays)
	}
	if p.PriceFrom != 34900000 {
		t.Errorf("PriceFrom = %v, want 34900000", p.PriceFrom)
	}
	if p.RegistrationStatus != RegOpen {
		t.Errorf("RegistrationStatus = %s, want %s", p.RegistrationStatus, RegOpen)
	}
	if p.Visibility != "visible" {
		t.Errorf("Visibility = %s", p.Visibility)
	}
}

func TestPriceFromSkipsZero(t *testing.T) {
	s := &PublicSchedule{HargaQuad: 0, HargaTriple: 30000000, HargaDouble: 32000000}
	if got := priceFrom(s); got != 30000000 {
		t.Errorf("priceFrom = %v, want 30000000", got)
	}
	if got := priceFrom(&PublicSchedule{}); got != 0 {
		t.Errorf("priceFrom kosong = %v, want 0", got)
	}
}

func TestFilterV1Match(t *testing.T) {
	p := samplePackage(1, "2026-12-20", "2026-12-31", 25)
	cases := []struct {
		nama string
		f    filterV1
		want bool
	}{
		{"tanpa filter", filterV1{}, true},
		{"periode sama", filterV1{Periode: "2026-12"}, true},
		{"periode beda", filterV1{Periode: "2027-01"}, false},
		{"destinasi sama", filterV1{Destinasi: "maroko"}, true},
		{"destinasi beda", filterV1{Destinasi: "dubai"}, false},
		{"program sama", filterV1{Program: "plus"}, true},
		{"tier sama", filterV1{Tier: "reguler"}, true},
		{"tier beda", filterV1{Tier: "premium"}, false},
		{"durasi sama", filterV1{Durasi: 12}, true},
		{"durasi beda", filterV1{Durasi: 9}, false},
		{"hotel bintang cukup", filterV1{HotelBintang: 5}, true},
		{"hotel bintang kurang", filterV1{HotelBintang: 6}, false},
		{"harga di bawah batas", filterV1{HargaMaks: 35000000}, true},
		{"harga di atas batas", filterV1{HargaMaks: 30000000}, false},
		{"registrasi cocok", filterV1{Registrasi: RegOpen}, true},
		{"registrasi beda", filterV1{Registrasi: RegSoldOut}, false},
		{"updated_since sebelum", filterV1{UpdatedSince: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}, true},
		{"updated_since sesudah", filterV1{UpdatedSince: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)}, false},
	}
	for _, c := range cases {
		if got := c.f.match(p); got != c.want {
			t.Errorf("%s: match = %v, want %v", c.nama, got, c.want)
		}
	}

	// Paket tanpa destinasi/tier tidak cocok dengan filter yang meminta keduanya.
	tanpa := p
	tanpa.Destination = nil
	tanpa.Tier = nil
	if (filterV1{Destinasi: "maroko"}).match(tanpa) || (filterV1{Tier: "reguler"}).match(tanpa) {
		t.Error("paket tanpa klasifikasi tidak boleh cocok dengan filter destinasi/tier")
	}
}

func TestPaginate(t *testing.T) {
	items := make([]PackageV1, 5)
	for i := range items {
		items[i] = samplePackage(int64(i+1), "2026-12-20", "2026-12-31", 25)
	}
	page, total := paginate(items, 1, 2)
	if len(page) != 2 || total != 5 {
		t.Errorf("halaman 1: len=%d total=%d", len(page), total)
	}
	page, _ = paginate(items, 3, 2)
	if len(page) != 1 || page[0].ID != 5 {
		t.Errorf("halaman 3 harus berisi paket terakhir saja: %+v", page)
	}
	page, total = paginate(items, 4, 2)
	if len(page) != 0 || total != 5 {
		t.Errorf("halaman di luar jangkauan harus kosong, total tetap: len=%d total=%d", len(page), total)
	}
}

// Bentuk JSON adalah kontrak dengan CMS dan situs publik: field lama PublicSchedule tetap datar,
// klasifikasi dan turunan ditambahkan di tingkat yang sama.
func TestPackageV1JSONShape(t *testing.T) {
	raw, err := json.Marshal(samplePackage(7, "2026-12-20", "2026-12-31", 4))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"id", "brand_id", "jadwal_nama", "berangkat_tanggal", "seat_total", "seat_sisa", "harga_quad",
		"journey_type", "program", "destination", "tier", "visibility", "duration_days", "price_from",
		"registration_status",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("field %q hilang dari JSON v1", key)
		}
	}
	if m["registration_status"] != RegAlmostFull {
		t.Errorf("registration_status = %v, want %s", m["registration_status"], RegAlmostFull)
	}
	dest, _ := m["destination"].(map[string]any)
	if dest["slug"] != "maroko" {
		t.Errorf("destination.slug = %v", dest["slug"])
	}
	tier, _ := m["tier"].(map[string]any)
	if tier["seo"] != "reguler" {
		t.Errorf("tier.seo = %v", tier["seo"])
	}
	if _, leaked := m["status"]; leaked {
		t.Error("field status admin tidak boleh bocor ke API publik")
	}
}

func TestGonePackageJSON(t *testing.T) {
	s := &Schedule{ID: 9, BrandID: 1, JadwalNama: "Umroh Lama", BerangkatTanggal: "2026-10-30"}
	raw, _ := json.Marshal(gonePackage(s, Klasifikasi{JourneyType: JourneyUmroh}, "closed"))
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["visibility"] != "gone" || m["reason"] != "closed" || m["berangkat_tanggal"] != "2026-10-30" {
		t.Errorf("payload gone salah: %s", raw)
	}
	if _, ok := m["harga_quad"]; ok {
		t.Error("paket gone tidak boleh membawa harga")
	}
}

func TestNormalizeKlasifikasi(t *testing.T) {
	// Kosong = tidak diubah.
	if k, err := normalizeKlasifikasi("", strp("plus"), nil); k != nil || err != nil {
		t.Fatalf("journey kosong harus (nil, nil), dapat %+v %v", k, err)
	}
	// Program plus wajib destinasi.
	if _, err := normalizeKlasifikasi("umroh", strp("Plus"), nil); err == nil {
		t.Error("program plus tanpa destinasi harus ditolak")
	}
	dest := int64(3)
	k, err := normalizeKlasifikasi(" Umroh ", strp(" Plus "), &dest)
	if err != nil || k == nil || k.JourneyType != "umroh" || k.ProgramSlug == nil || *k.ProgramSlug != "plus" || k.DestinationID == nil {
		t.Fatalf("normalisasi salah: %+v err=%v", k, err)
	}
	// Tanpa program dan destinasi diperbolehkan.
	k, err = normalizeKlasifikasi("umroh", nil, nil)
	if err != nil || k == nil || k.ProgramSlug != nil || k.DestinationID != nil {
		t.Fatalf("umroh polos harus valid: %+v err=%v", k, err)
	}
	// Program kosong dianggap tidak ada; destinasi 0 dianggap tidak ada.
	zero := int64(0)
	k, err = normalizeKlasifikasi("haji", strp("  "), &zero)
	if err != nil || k.ProgramSlug != nil || k.DestinationID != nil {
		t.Fatalf("nilai kosong harus dianggap tidak ada: %+v err=%v", k, err)
	}
	for _, bad := range []struct {
		j string
		p *string
	}{{"umrah", nil}, {"umroh", strp("Plus Dubai")}, {"umroh", strp("-plus")}} {
		if _, err := normalizeKlasifikasi(bad.j, bad.p, &dest); err == nil {
			t.Errorf("normalizeKlasifikasi(%q, %v) harus gagal", bad.j, bad.p)
		}
	}
}

package schedule

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/benefit"
)

// Paket yang dijual wajib membawa empat highlight benefit; draf dan arsip tidak diperiksa
// (keputusan 13 November 2026).
func TestBenefitCukup(t *testing.T) {
	kasus := []struct {
		status string
		jumlah int
		lolos  bool
	}{
		{StatusDraft, 0, true},
		{StatusDraft, 2, true},
		{StatusArchived, 0, true},
		{StatusPublished, 3, false},
		{StatusPublished, 0, false},
		{StatusPublished, 5, false},
		{StatusPublished, 4, true},
	}
	for _, k := range kasus {
		w := httptest.NewRecorder()
		if got := benefitCukup(w, k.status, k.jumlah); got != k.lolos {
			t.Errorf("status %s dengan %d benefit: lolos=%v, seharusnya %v", k.status, k.jumlah, got, k.lolos)
		}
		if !k.lolos {
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), errBenefitKurang) {
				t.Errorf("status %s dengan %d benefit: jawaban %d %s", k.status, k.jumlah, w.Code, w.Body.String())
			}
		}
	}
}

// Klien lama yang tidak mengirim benefit_ids tidak boleh menghapus benefit yang tersimpan: kunci yang tidak
// ada harus terbaca nil, sedangkan daftar kosong terbaca sebagai daftar kosong.
func TestBenefitIDsMembedakanTidakDikirimDanKosong(t *testing.T) {
	var tanpa UpdateScheduleRequest
	if err := json.Unmarshal([]byte(`{"jadwal_nama":"x"}`), &tanpa); err != nil {
		t.Fatal(err)
	}
	if tanpa.BenefitIDs != nil {
		t.Fatalf("kunci tidak dikirim seharusnya nil, dapat %v", *tanpa.BenefitIDs)
	}

	var kosong UpdateScheduleRequest
	if err := json.Unmarshal([]byte(`{"benefit_ids":[]}`), &kosong); err != nil {
		t.Fatal(err)
	}
	if kosong.BenefitIDs == nil || len(*kosong.BenefitIDs) != 0 {
		t.Fatal("daftar kosong seharusnya terbaca sebagai daftar kosong, bukan nil")
	}

	var isi UpdateScheduleRequest
	if err := json.Unmarshal([]byte(`{"benefit_ids":[9,3,7,1]}`), &isi); err != nil {
		t.Fatal(err)
	}
	if isi.BenefitIDs == nil || len(*isi.BenefitIDs) != 4 || (*isi.BenefitIDs)[0] != 9 {
		t.Fatalf("urutan harus dipertahankan, dapat %v", isi.BenefitIDs)
	}
}

// Response publik selalu membawa "benefits" sebagai array, dalam urutan admin, tanpa id dan status aktif.
func TestToPublicMembawaBenefit(t *testing.T) {
	s := &Schedule{Benefits: []benefit.Ref{
		{ID: 2, Teks: "Ustadz pembimbing", Ikon: "solar:user-speak-bold-duotone", IsActive: true},
		{ID: 5, Teks: "Makan 3x sehari", Ikon: "solar:chef-hat-bold-duotone", IsActive: false},
	}}
	raw, err := json.Marshal(s.ToPublic())
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Benefits []map[string]any `json:"benefits"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Benefits) != 2 || out.Benefits[0]["teks"] != "Ustadz pembimbing" || out.Benefits[1]["ikon"] != "solar:chef-hat-bold-duotone" {
		t.Fatalf("benefits salah: %s", raw)
	}
	if _, ada := out.Benefits[0]["id"]; ada {
		t.Fatal("id benefit tidak boleh ikut di response publik")
	}

	// Paket tanpa benefit: array kosong, bukan null.
	raw, _ = json.Marshal((&Schedule{}).ToPublic())
	if !strings.Contains(string(raw), `"benefits":[]`) {
		t.Fatalf("paket tanpa benefit harus mengirim array kosong: %s", raw)
	}
}

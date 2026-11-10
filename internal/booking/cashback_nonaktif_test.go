package booking

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/komisi"
)

// Code review area uang, 10 November 2026: selama fitur kredit cashback dimatikan, kedua rute cashback
// menolak permintaan sebelum menyentuh data, supaya diskon tidak bisa ditambah lewat jalur ini.
func TestRuteCashbackDitolakSelamaFiturMati(t *testing.T) {
	awal := komisi.KomisiRepeatCashbackAktif
	t.Cleanup(func() { komisi.KomisiRepeatCashbackAktif = awal })
	komisi.KomisiRepeatCashbackAktif = false

	// Handler tanpa repository: bila pemeriksaan fitur terlewati, pemanggilan ini panik.
	h := &Handler{}
	for nama, panggil := range map[string]func(http.ResponseWriter, *http.Request){
		"GET":  h.GetKreditCashback,
		"POST": h.PakaiKreditCashback,
	} {
		rec := httptest.NewRecorder()
		panggil(rec, httptest.NewRequest(nama, "/api/admin/bookings/1/cashback", strings.NewReader(`{"nominal":1000}`)))
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), ErrCashbackNonaktif.Error()) {
			t.Errorf("%s: status %d, isi %s", nama, rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	if cashbackAktif(rec) {
		t.Fatal("cashbackAktif harus false saat fitur mati")
	}
	komisi.KomisiRepeatCashbackAktif = true
	if !cashbackAktif(httptest.NewRecorder()) {
		t.Fatal("cashbackAktif harus true saat fitur nyala")
	}
}

package airline

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func pesanGalat(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body["error"]
}

func ptr(s string) *string { return &s }

func TestValidateAirlineInput(t *testing.T) {
	kasus := []struct {
		nama string
		kode *string
		logo *string
		mau  string
	}{
		{" ", nil, nil, "Nama wajib diisi."},
		{strings.Repeat("a", maksNama+1), nil, nil, "Nama maksimal 150 karakter."},
		{"Saudia", ptr("SVX"), nil, "Kode IATA harus 2 karakter."},
		{"Saudia", ptr("sv"), ptr("https://x.test/" + strings.Repeat("a", maksLogoURL)), "URL logo maksimal 500 karakter."},
	}
	for _, k := range kasus {
		w := httptest.NewRecorder()
		if _, _, _, ok := validateAirlineInput(w, k.nama, k.kode, k.logo); ok || pesanGalat(t, w) != k.mau {
			t.Errorf("%q: ok=%v pesan=%q, mau %q", k.nama, ok, pesanGalat(t, w), k.mau)
		}
	}
	w := httptest.NewRecorder()
	name, code, logo, ok := validateAirlineInput(w, " saudia ", ptr(" sv "), ptr(" "))
	if !ok || name != "SAUDIA" || code == nil || *code != "SV" || logo != nil {
		t.Errorf("input sah: %q %v %v ok=%v", name, code, logo, ok)
	}
}

// Audit menu Paket R-15: kode IATA tidak boleh kembar (case-insensitive); kode kosong tidak dicek.
func TestCodeExists(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	res, err := tx.Exec(`INSERT INTO airlines (name, code) VALUES ('MASKAPAI UJI R15 QZP', 'Q9')`)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	cek := func(label string, code *string, exclude uint64, mau bool) {
		t.Helper()
		ada, err := codeExists(ctx, tx, code, exclude)
		if err != nil || ada != mau {
			t.Errorf("%s: ada=%v err=%v, mau %v", label, ada, err, mau)
		}
	}
	cek("kode sama", ptr("Q9"), 0, true)
	cek("kode beda huruf", ptr("q9"), 0, true)
	cek("maskapai sendiri", ptr("Q9"), uint64(id), false)
	cek("kode kosong", ptr(""), 0, false)
	cek("tanpa kode", nil, 0, false)
}

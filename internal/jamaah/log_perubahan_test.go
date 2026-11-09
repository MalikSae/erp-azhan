package jamaah

import "testing"

func TestBedaIsianEditAdmin(t *testing.T) {
	s := func(v string) *string { return &v }
	lama := &Jamaah{NamaLengkap: "Siti", Kota: s("Bogor"), TanggalLahir: s("1980-01-02"), NIK: s("3201010101010001"), Alamat: nil}
	req := &UpdateJamaahRequest{NamaLengkap: " Siti ", Kota: s("Depok"), TanggalLahir: s("1980-01-02T00:00:00Z"), NIK: s("3201010101010001"), Alamat: s("Jl. Mawar")}
	beda := bedaIsian(lama, req)
	got := map[string]isianLog{}
	for _, b := range beda {
		got[b.kolom] = b
	}
	if len(got) != 2 || got["kota"].lama != "Bogor" || got["kota"].baru != "Depok" || got["alamat"].lama != "" || got["alamat"].baru != "Jl. Mawar" {
		t.Fatalf("beda = %+v", beda)
	}
	if len(bedaIsian(lama, &UpdateJamaahRequest{NamaLengkap: "Siti", Kota: s("Bogor"), TanggalLahir: s("1980-01-02"), NIK: s("3201010101010001")})) != 0 {
		t.Fatal("tanpa perubahan harus kosong")
	}
}

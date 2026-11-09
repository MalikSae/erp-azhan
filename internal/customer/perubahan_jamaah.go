package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-azhan/api/internal/jamaah"
	"github.com/go-sql-driver/mysql"
)

// Perubahan data jamaah oleh pelanggan. Orang yang sama di beberapa booking adalah beberapa baris jamaah (tiap booking membuat baris baru);
// di sini baris-baris itu digabung menjadi satu "orang". Mengisi isian yang masih kosong langsung tersimpan, mengubah isian yang sudah
// terisi menjadi pengajuan yang menunggu persetujuan admin. Semuanya tercatat di jamaah_perubahan dan jamaah_perubahan_item.

type kolomJamaah struct {
	Key, Label string
	Tanggal    bool
	Sensitif   bool // tersamar di riwayat perubahan yang sudah diputuskan
}

// kolomEdit: isian yang boleh diubah pelanggan, urut seperti formulir. Nama lengkap, NIK, dan email sengaja tidak ada (identitas).
var kolomEdit = []kolomJamaah{
	{Key: "jenis_kelamin", Label: "Jenis kelamin"},
	{Key: "tempat_lahir", Label: "Tempat lahir"},
	{Key: "tanggal_lahir", Label: "Tanggal lahir", Tanggal: true},
	{Key: "nama_ayah_kandung", Label: "Nama ayah kandung"},
	{Key: "no_paspor", Label: "Nomor paspor", Sensitif: true},
	{Key: "tempat_paspor_keluar", Label: "Tempat paspor dikeluarkan"},
	{Key: "tanggal_paspor_keluar", Label: "Tanggal paspor dikeluarkan", Tanggal: true},
	{Key: "paspor_berlaku_sampai", Label: "Masa berlaku paspor", Tanggal: true},
	{Key: "no_hp", Label: "Nomor WhatsApp"},
	{Key: "pekerjaan", Label: "Pekerjaan"},
	{Key: "pendidikan_terakhir", Label: "Pendidikan terakhir"},
	{Key: "penjamin_kesehatan", Label: "Penjamin kesehatan"},
	{Key: "no_asuransi_bpjs", Label: "Nomor BPJS atau asuransi", Sensitif: true},
	{Key: "alamat", Label: "Alamat"},
	{Key: "kota", Label: "Kota"},
	{Key: "emergency_nama", Label: "Nama kontak darurat"},
	{Key: "emergency_hp", Label: "Nomor kontak darurat"},
	{Key: "emergency_hubungan", Label: "Hubungan kontak darurat"},
	{Key: "emergency_alamat", Label: "Alamat kontak darurat"},
}

var kolomIndeks = func() map[string]kolomJamaah {
	m := make(map[string]kolomJamaah, len(kolomEdit))
	for _, k := range kolomEdit {
		m[k.Key] = k
	}
	return m
}()

// kolomLog: label dan kepekaan semua isian yang bisa muncul di riwayat (isian yang boleh diubah pelanggan, ditambah identitas yang hanya diubah admin).
var kolomLog = func() map[string]kolomJamaah {
	m := map[string]kolomJamaah{
		"nama_lengkap": {Key: "nama_lengkap", Label: "Nama lengkap"},
		"nik":          {Key: "nik", Label: "NIK", Sensitif: true},
		"email":        {Key: "email", Label: "Email"},
	}
	for k, v := range kolomIndeks {
		m[k] = v
	}
	return m
}()

func ekspresiKolom(k kolomJamaah) string {
	if k.Tanggal {
		return fmt.Sprintf("COALESCE(DATE_FORMAT(j.%s,'%%Y-%%m-%%d'),'')", k.Key)
	}
	return fmt.Sprintf("COALESCE(j.%s,'')", k.Key)
}

type dbq interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// anggotaJamaah: satu baris jamaah milik akun beserta booking terbarunya.
type anggotaJamaah struct {
	ID            int64
	Nama          string
	Lahir         string
	PaxType       string
	BookingID     int64
	BookingKode   string
	StatusBooking string
	Paket         string
	Berangkat     string
}

func normalNama(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
}

// muatAnggota: semua baris jamaah pada booking akun (selain draft), terbaru dulu. Booking batal ikut.
func muatAnggota(ctx context.Context, q dbq, akunID int64) ([]anggotaJamaah, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, COALESCE(DATE_FORMAT(j.tanggal_lahir,'%Y-%m-%d'),''), bp.pax_type,
		       b.id, b.id_booking, b.status, s.jadwal_nama, COALESCE(DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'),'')
		FROM booking_pax bp
		JOIN bookings b ON b.id = bp.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE b.customer_account_id = ? AND b.status <> 'draft' AND bp.pax_status = 'aktif'
		ORDER BY j.id DESC, b.id DESC`, akunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []anggotaJamaah
	sudah := map[int64]bool{}
	for rows.Next() {
		var a anggotaJamaah
		if err := rows.Scan(&a.ID, &a.Nama, &a.Lahir, &a.PaxType, &a.BookingID, &a.BookingKode, &a.StatusBooking, &a.Paket, &a.Berangkat); err != nil {
			return nil, err
		}
		if sudah[a.ID] { // satu baris per jamaah: booking terbaru
			continue
		}
		sudah[a.ID] = true
		out = append(out, a)
	}
	return out, rows.Err()
}

// cocokOrang: nama sama dan tanggal lahir sama (atau salah satunya belum diisi).
func cocokOrang(a, b anggotaJamaah) bool {
	if normalNama(a.Nama) != normalNama(b.Nama) {
		return false
	}
	return a.Lahir == "" || b.Lahir == "" || a.Lahir == b.Lahir
}

// kelompokOrang menggabungkan baris jamaah menjadi orang. Wakil tiap orang: baris dengan id terbesar (urutan masuk: terbaru dulu).
func kelompokOrang(semua []anggotaJamaah) [][]anggotaJamaah {
	var hasil [][]anggotaJamaah
	terpakai := make([]bool, len(semua))
	for i, a := range semua {
		if terpakai[i] {
			continue
		}
		grup := []anggotaJamaah{a}
		terpakai[i] = true
		for j := i + 1; j < len(semua); j++ {
			if !terpakai[j] && cocokOrang(a, semua[j]) {
				grup = append(grup, semua[j])
				terpakai[j] = true
			}
		}
		hasil = append(hasil, grup)
	}
	return hasil
}

var errJamaahTidakAda = errors.New("jamaah tidak ditemukan")

// kelompokDari mengembalikan orang yang memuat baris jamaah jamaahID (harus milik akun), terbaru dulu.
func kelompokDari(ctx context.Context, q dbq, akunID, jamaahID int64) ([]anggotaJamaah, error) {
	semua, err := muatAnggota(ctx, q, akunID)
	if err != nil {
		return nil, err
	}
	for _, grup := range kelompokOrang(semua) {
		for _, a := range grup {
			if a.ID == jamaahID {
				return grup, nil
			}
		}
	}
	return nil, errJamaahTidakAda
}

func idGrup(grup []anggotaJamaah) []int64 {
	ids := make([]int64, len(grup))
	for i, a := range grup {
		ids[i] = a.ID
	}
	return ids
}

func placeholders(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }

func argsID(ids []int64) []any {
	a := make([]any, len(ids))
	for i, v := range ids {
		a[i] = v
	}
	return a
}

// dataOrang: isian gabungan satu orang (tiap isian diambil dari baris terbaru yang mengisinya) dan identitas terkunci.
type dataOrang struct {
	Nilai map[string]string
	Nama  string
	NIK   string
	Email string
}

func muatDataOrang(ctx context.Context, q dbq, ids []int64) (*dataOrang, error) {
	kolom := make([]string, len(kolomEdit))
	for i, k := range kolomEdit {
		kolom[i] = ekspresiKolom(k)
	}
	rows, err := q.QueryContext(ctx, `SELECT j.id, j.nama_lengkap, COALESCE(j.nik,''), COALESCE(j.email,''), `+strings.Join(kolom, ", ")+
		` FROM jamaah j WHERE j.id IN (`+placeholders(len(ids))+`) ORDER BY j.id DESC`, argsID(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d := &dataOrang{Nilai: map[string]string{}}
	for rows.Next() {
		var id int64
		var nama, nik, email string
		isi := make([]string, len(kolomEdit))
		dest := []any{&id, &nama, &nik, &email}
		for i := range isi {
			dest = append(dest, &isi[i])
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		if d.Nama == "" {
			d.Nama = nama
		}
		if d.NIK == "" {
			d.NIK = nik
		}
		if d.Email == "" {
			d.Email = email
		}
		for i, k := range kolomEdit {
			if d.Nilai[k.Key] == "" {
				d.Nilai[k.Key] = isi[i]
			}
		}
	}
	return d, rows.Err()
}

func kelengkapanOrang(nilai map[string]string, infant bool) (terisi, total int, kurang []string) {
	kurang = []string{}
	for _, f := range dataWajibJamaah {
		if infant && (strings.HasPrefix(f.key, "emergency_") || f.key == "no_hp") {
			continue
		}
		total++
		if strings.TrimSpace(nilai[f.key]) != "" {
			terisi++
		} else {
			kurang = append(kurang, f.label)
		}
	}
	return
}

// perubahanItem: satu isian yang berubah.
type perubahanItem struct {
	Kolom string `json:"kolom"`
	Label string `json:"label"`
	Lama  string `json:"lama"`
	Baru  string `json:"baru"`
}

// hasilSimpan: ringkasan satu kali simpan.
type hasilSimpan struct {
	Langsung    int // isian kosong yang langsung terisi
	Pengajuan   int // isian yang menunggu persetujuan
	PengajuanID int64
}

type errBentrokHP struct{}

func (errBentrokHP) Error() string { return "nomor WhatsApp ini sudah dipakai jamaah lain" }

// validasiPerubahan memeriksa isian hasil gabungan data lama dan perubahan. Mengembalikan pesan untuk pelanggan, atau "".
func validasiPerubahan(perubahan, gabungan map[string]string) string {
	for k, v := range perubahan {
		if _, ok := kolomIndeks[k]; !ok {
			return "isian tidak dikenal: " + k
		}
		if len([]rune(v)) > panjangMaksData {
			return "isian terlalu panjang (maksimal 255 karakter)"
		}
	}
	if jk := gabungan["jenis_kelamin"]; jk != "" && jk != "L" && jk != "P" {
		return "jenis kelamin tidak valid"
	}
	req := simpanDataJamaahRequest{
		NamaAyahKandung: gabungan["nama_ayah_kandung"], NoPaspor: gabungan["no_paspor"], NoAsuransiBPJS: gabungan["no_asuransi_bpjs"],
		EmergencyNama: gabungan["emergency_nama"], EmergencyHubungan: gabungan["emergency_hubungan"],
	}
	// Hanya isian yang diubah yang diperiksa formatnya, agar data lama yang tidak rapi tidak menghalangi perbaikan isian lain.
	if _, ok := perubahan["nama_ayah_kandung"]; !ok {
		req.NamaAyahKandung = ""
	}
	if _, ok := perubahan["no_paspor"]; !ok {
		req.NoPaspor = ""
	}
	if _, ok := perubahan["no_asuransi_bpjs"]; !ok {
		req.NoAsuransiBPJS = ""
	}
	if _, ok := perubahan["emergency_nama"]; !ok {
		req.EmergencyNama = ""
	}
	if _, ok := perubahan["emergency_hubungan"]; !ok {
		req.EmergencyHubungan = ""
	}
	if msg := periksaFormatData(&req); msg != "" {
		return msg
	}
	tl, tk, bs, hp, ehp := gabungan["tanggal_lahir"], gabungan["tanggal_paspor_keluar"], gabungan["paspor_berlaku_sampai"], gabungan["no_hp"], gabungan["emergency_hp"]
	if msg := jamaah.ValidasiInput(&jamaah.CreateJamaahRequest{
		TanggalLahir: &tl, TanggalPasporKeluar: &tk, PasporBerlakuSampai: &bs, NoHP: &hp, EmergencyHP: &ehp,
	}); msg != "" {
		return ramahPesanData(msg)
	}
	return ""
}

func nilaiNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// terapkanKolom menulis satu isian ke baris-baris jamaah. hanyaKosong: tulis hanya pada baris yang isiannya masih kosong.
func terapkanKolom(ctx context.Context, q dbq, ids []int64, k kolomJamaah, nilai string, hanyaKosong bool) error {
	sql := fmt.Sprintf("UPDATE jamaah SET %s=? WHERE id IN (%s)", k.Key, placeholders(len(ids)))
	if hanyaKosong {
		sql += fmt.Sprintf(" AND (%s IS NULL OR %s='')", k.Key, k.Key)
	}
	args := append([]any{nilaiNull(nilai)}, argsID(ids)...)
	_, err := q.ExecContext(ctx, sql, args...)
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) && myErr.Number == 1062 {
		return errBentrokHP{}
	}
	return err
}

func catatPerubahan(ctx context.Context, q dbq, jamaahID, akunID, adminID int64, sumber, status string, items []perubahanItem) (int64, error) {
	res, err := q.ExecContext(ctx, `INSERT INTO jamaah_perubahan (jamaah_id, akun_id, admin_id, sumber, status) VALUES (?,?,?,?,?)`,
		jamaahID, nullID(akunID), nullID(adminID), sumber, status)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, it := range items {
		if _, err := q.ExecContext(ctx, `INSERT INTO jamaah_perubahan_item (perubahan_id, kolom, nilai_lama, nilai_baru) VALUES (?,?,?,?)`,
			id, it.Kolom, nilaiNull(it.Lama), nilaiNull(it.Baru)); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// simpanPerubahanPelanggan: mengisi yang kosong langsung, sisanya jadi satu pengajuan (menggantikan pengajuan terbuka sebelumnya).
// Harus dipanggil di dalam transaksi. grup: orang yang dimaksud (wakil = grup[0]).
func simpanPerubahanPelanggan(ctx context.Context, tx dbq, akunID int64, grup []anggotaJamaah, perubahan map[string]string) (*hasilSimpan, error) {
	ids := idGrup(grup)
	wakil := ids[0]
	// Kunci baris wakil agar dua simpan bersamaan tidak membuat dua pengajuan terbuka.
	if _, err := tx.ExecContext(ctx, `SELECT id FROM jamaah WHERE id = ? FOR UPDATE`, wakil); err != nil {
		return nil, err
	}
	sekarang, err := muatDataOrang(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	gabungan := map[string]string{}
	for k, v := range sekarang.Nilai {
		gabungan[k] = v
	}
	for k, v := range perubahan {
		gabungan[k] = strings.TrimSpace(v)
	}
	bersih := map[string]string{}
	for k := range perubahan {
		bersih[k] = gabungan[k]
	}
	if msg := validasiPerubahan(bersih, gabungan); msg != "" {
		return nil, &ErrValidasi{Pesan: msg}
	}

	var langsung, ajukan []perubahanItem
	for _, k := range kolomEdit {
		baru, ada := bersih[k.Key]
		if !ada {
			continue
		}
		lama := sekarang.Nilai[k.Key]
		if baru == lama {
			continue
		}
		it := perubahanItem{Kolom: k.Key, Label: k.Label, Lama: lama, Baru: baru}
		if lama == "" {
			langsung = append(langsung, it)
		} else {
			ajukan = append(ajukan, it)
		}
	}
	if len(langsung) == 0 && len(ajukan) == 0 {
		return nil, ErrTidakBerubah
	}

	h := &hasilSimpan{Langsung: len(langsung), Pengajuan: len(ajukan)}
	if len(langsung) > 0 {
		for _, it := range langsung {
			if err := terapkanKolom(ctx, tx, ids, kolomIndeks[it.Kolom], it.Baru, true); err != nil {
				return nil, err
			}
		}
		if _, err := catatPerubahan(ctx, tx, wakil, akunID, 0, "pelanggan", "langsung", langsung); err != nil {
			return nil, err
		}
	}
	if len(ajukan) > 0 {
		// Pengajuan terbuka sebelumnya (atas baris mana pun dari orang ini) digantikan.
		if _, err := tx.ExecContext(ctx, `UPDATE jamaah_perubahan SET status='diganti', diputus_at=NOW()
			WHERE status='menunggu' AND akun_id=? AND jamaah_id IN (`+placeholders(len(ids))+`)`, append([]any{akunID}, argsID(ids)...)...); err != nil {
			return nil, err
		}
		id, err := catatPerubahan(ctx, tx, wakil, akunID, 0, "pelanggan", "menunggu", ajukan)
		if err != nil {
			return nil, err
		}
		h.PengajuanID = id
	}
	return h, nil
}

// pengajuanView: pengajuan pelanggan yang perlu dilihat pelanggan (menunggu, atau ditolak terakhir).
type pengajuanView struct {
	ID        int64           `json:"id"`
	Status    string          `json:"status"`
	DibuatAt  time.Time       `json:"dibuat_at"`
	Catatan   string          `json:"catatan,omitempty"`
	Perubahan []perubahanItem `json:"perubahan"`
}

func muatItem(ctx context.Context, q dbq, perubahanID int64, samarkan bool) ([]perubahanItem, error) {
	rows, err := q.QueryContext(ctx, `SELECT kolom, COALESCE(nilai_lama,''), COALESCE(nilai_baru,'') FROM jamaah_perubahan_item WHERE perubahan_id=? ORDER BY id`, perubahanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []perubahanItem{}
	for rows.Next() {
		var it perubahanItem
		if err := rows.Scan(&it.Kolom, &it.Lama, &it.Baru); err != nil {
			return nil, err
		}
		k := kolomLog[it.Kolom]
		it.Label = k.Label
		if it.Label == "" {
			it.Label = it.Kolom
		}
		if samarkan && k.Sensitif {
			it.Lama, it.Baru = samarkanNilai(it.Lama), samarkanNilai(it.Baru)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// samarkanNilai menyisakan 3 karakter terakhir.
func samarkanNilai(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	if len(r) <= 3 {
		return strings.Repeat("•", len(r))
	}
	return strings.Repeat("•", len(r)-3) + string(r[len(r)-3:])
}

func pengajuanUntukPelanggan(ctx context.Context, q dbq, akunID int64, ids []int64) (*pengajuanView, error) {
	var v pengajuanView
	var catatan sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id, status, dibuat_at, catatan FROM jamaah_perubahan
		WHERE akun_id=? AND sumber='pelanggan' AND status IN ('menunggu','disetujui','ditolak') AND jamaah_id IN (`+placeholders(len(ids))+`)
		ORDER BY id DESC LIMIT 1`, append([]any{akunID}, argsID(ids)...)...).Scan(&v.ID, &v.Status, &v.DibuatAt, &catatan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if v.Status == "disetujui" {
		return nil, nil
	}
	v.Catatan = catatan.String
	if v.Perubahan, err = muatItem(ctx, q, v.ID, false); err != nil {
		return nil, err
	}
	return &v, nil
}

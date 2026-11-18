package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/notifikasi"
	"erp-azhan/api/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

// Aktivasi akun jamaah (keputusan 18 November 2026): jamaah yang dibuatkan booking oleh orang lain (mis. agen)
// mengaktifkan akunnya sendiri dengan kode booking dan NIK. Satu fungsi inti (Aktivasi.Jalankan) dipakai dua pintu
// HTTP: publik (akun belum ada) dan akun masuk (menautkan akun yang sudah ada). Tanpa Google, captcha, atau OTP.
// Semua sebab gagal menghasilkan satu galat yang sama; sebab sebenarnya hanya masuk jamaah_aktivasi_log.

const (
	// KodeGalatAktivasiGagal: kode mesin 400 untuk semua sebab gagal.
	KodeGalatAktivasiGagal = "AKTIVASI_GAGAL"
	// KodeGalatAktivasiTerlaluSering: kode mesin 429; satu-satunya pembeda dari galat generik.
	KodeGalatAktivasiTerlaluSering = "AKTIVASI_TERLALU_SERING"

	PesanAktivasiGagal         = "Data tidak sesuai. Periksa kode booking dan NIK."
	PesanAktivasiTerlaluSering = "Terlalu banyak percobaan. Coba lagi nanti."

	// MinPepperAktivasi: panjang minimum AKTIVASI_PEPPER (byte) saat aktivasi dinyalakan.
	MinPepperAktivasi = 32
)

var (
	// ErrAktivasiGagal: kode booking dan NIK tidak cocok, atau aktivasi tidak bisa dilanjutkan (satu pesan untuk semua sebab).
	ErrAktivasiGagal = errors.New("aktivasi gagal")
	// ErrKodeAgenTidakAda: kode agen yang diketik tidak valid (hanya untuk jamaah yang belum terikat agen).
	ErrKodeAgenTidakAda = errors.New("kode agen tidak ditemukan")

	errPepperPendek = errors.New("aktivasi: pepper kurang dari 32 byte")
	polaKodeBooking = regexp.MustCompile(`^[A-Z0-9]{1,8}$`)
)

// ErrAktivasiTerlaluSering: batas percobaan terlampaui; Tunggu menjadi header Retry-After.
type ErrAktivasiTerlaluSering struct{ Tunggu time.Duration }

func (e *ErrAktivasiTerlaluSering) Error() string { return PesanAktivasiTerlaluSering }

// ErrAktivasiInput: isian email, WhatsApp, atau kata sandi tidak valid. Diperiksa sebelum penghitung disentuh.
type ErrAktivasiInput struct{ Pesan string }

func (e *ErrAktivasiInput) Error() string { return e.Pesan }

// AktivasiConfig: AKTIVASI_JAMAAH_AKTIF (bawaan mati) dan AKTIVASI_PEPPER untuk HMAC hash kode, NIK, IP, perangkat.
type AktivasiConfig struct {
	Aktif  bool
	Pepper []byte
}

// KonfigAktivasiDariEnv membaca konfigurasi dan gagal (fail closed) bila aktivasi menyala tanpa pepper minimal 32 byte.
func KonfigAktivasiDariEnv() (AktivasiConfig, error) {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AKTIVASI_JAMAAH_AKTIF")))
	cfg := AktivasiConfig{
		Aktif:  v == "true" || v == "1",
		Pepper: []byte(strings.TrimSpace(os.Getenv("AKTIVASI_PEPPER"))),
	}
	if cfg.Aktif && len(cfg.Pepper) < MinPepperAktivasi {
		return cfg, errors.New("AKTIVASI_JAMAAH_AKTIF aktif tetapi AKTIVASI_PEPPER kurang dari 32 byte")
	}
	return cfg, nil
}

// AktivasiInput: satu bentuk permintaan untuk cek (Simpan=false) dan simpan (Simpan=true).
type AktivasiInput struct {
	BrandID     int64
	KodeBooking string
	NIK         string
	Simpan      bool
	// Hanya jalur publik (AkunID = 0) saat simpan.
	NoHP, Email, Password  string
	KodeAgen, KodeReferral string
	// AkunID > 0: jalur akun masuk (brand dari sesi). 0: jalur publik.
	AkunID int64
	// IP dan Perangkat (header X-Perangkat-Id, boleh kosong) hanya dipakai untuk penghitung, di-hash.
	IP, Perangkat string
}

// DataJamaahAktivasi: data jamaah yang dikunci di layar aktivasi. Tidak pernah memuat NIK, kode, atau data orang lain.
type DataJamaahAktivasi struct {
	NamaLengkap  string `json:"nama_lengkap"`
	JenisKelamin string `json:"jenis_kelamin"`
	TanggalLahir string `json:"tanggal_lahir"`
}

// AktivasiHasil: hasil cek atau simpan. AkunID dan Token terisi hanya saat simpan; Token hanya di jalur publik.
type AktivasiHasil struct {
	Jamaah DataJamaahAktivasi
	AkunID int64
	Token  string
}

type Aktivasi struct {
	db            *sql.DB
	repo          *Repository
	batas         *pembatas
	kodeAgenGagal *shared.WindowLimiter
	// jedaGagal: jeda acak 150-400 ms di jalur gagal supaya waktu respons tidak membedakan sebab.
	jedaGagal func()
}

func NewAktivasi(db *sql.DB, repo *Repository, pepper []byte) *Aktivasi {
	return &Aktivasi{
		db: db, repo: repo,
		batas:         &pembatas{db: db, pepper: pepper, now: time.Now},
		kodeAgenGagal: shared.NewWindowLimiter(kodeAgenGagalLimit, time.Hour),
		jedaGagal: func() {
			time.Sleep(150*time.Millisecond + time.Duration(rand.Int64N(int64(250*time.Millisecond))))
		},
	}
}

// kandidat: jamaah yang memenuhi syarat untuk kode booking + NIK.
type kandidat struct {
	JamaahID  int64
	BookingID int64
	Nama      string
	JK        string
	Lahir     string
	Kaitan    string
	Kunci     int64
}

// galatAlasan: gagal dengan sebab internal (hanya untuk log); pengguna menerima ErrAktivasiGagal.
type galatAlasan struct {
	aksi, alasan string
	jamaahID     int64
}

func (g *galatAlasan) Error() string   { return "aktivasi gagal: " + g.alasan }
func (g *galatAlasan) Is(t error) bool { return t == ErrAktivasiGagal }

// percobaan: keadaan satu permintaan (slot yang sudah diambil, hash untuk log).
type percobaan struct {
	a       *Aktivasi
	in      AktivasiInput
	kode    string
	diambil []slotBatas
	kodeAda bool
	ipH     string
	devH    string
	kodeH   string
	nikH    string
}

func (p *percobaan) ambil(ctx context.Context, b jenisBatas, kunci string) error {
	s, urutan, ok, tunggu, err := p.a.batas.ambil(ctx, b, kunci)
	if err != nil {
		return err
	}
	p.diambil = append(p.diambil, s)
	if ok {
		return nil
	}
	if urutan == b.batas+1 {
		switch b.jenis {
		case batasKode24.jenis:
			p.catat(ctx, "kode_dikunci", b.jenis, 0, 0)
			log.Printf("[WARN] aktivasi: brand %d, satu kode booking melewati %d percobaan per hari", p.in.BrandID, b.batas)
		case batasNIK24.jenis:
			log.Printf("[WARN] aktivasi: brand %d, satu NIK melewati %d percobaan per hari", p.in.BrandID, b.batas)
		}
	}
	return &ErrAktivasiTerlaluSering{Tunggu: tunggu}
}

func (p *percobaan) kembalikanSemua(ctx context.Context) {
	for _, s := range p.diambil {
		if s.jenis == batasSimpanIP.jenis {
			continue
		}
		if err := p.a.batas.kembalikan(ctx, s); err != nil {
			log.Printf("[WARN] %v", err)
		}
	}
}

func (p *percobaan) catat(ctx context.Context, aksi, alasan string, jamaahID, akunID int64) {
	// kode_dikunci ditulis sekali per jendela dan tidak ikut batas tulis log kegagalan.
	if aksi != "kode_dikunci" && !bolehTulisLogGagal(ctx, p.a.db, p.kodeH) {
		return
	}
	err := catatLogAktivasi(ctx, p.a.db, entriLogAktivasi{
		BrandID: p.in.BrandID, Aksi: aksi, JamaahID: jamaahID, AkunID: akunID,
		KodeHash: p.kodeH, NIKHash: p.nikH, IPHash: p.ipH, DeviceHash: p.devH, Alasan: alasan,
	})
	if err != nil {
		log.Printf("[WARN] %v", err)
	}
}

// gagal mencatat sebab (hanya bila kode booking ada), menaikkan penghitung brand, dan mengembalikan galat generik
// setelah jeda acak.
func (p *percobaan) gagal(ctx context.Context, aksi, alasan string, jamaahID int64) error {
	if p.kodeAda {
		p.catat(ctx, aksi, alasan, jamaahID, p.in.AkunID)
	}
	if _, urutan, _, _, err := p.a.batas.ambil(ctx, batasBrand, p.a.batas.hash("brand", p.in.BrandID, "")); err == nil && urutan == batasBrand.batas {
		log.Printf("[WARN] aktivasi: brand %d melewati %d percobaan gagal per jam", p.in.BrandID, batasBrand.batas)
	}
	p.a.jedaGagal()
	return ErrAktivasiGagal
}

// Jalankan memeriksa kode booking + NIK dan, bila Simpan, membuat atau menautkan akun. Mengembalikan ErrAktivasiGagal
// (satu galat untuk semua sebab), *ErrAktivasiTerlaluSering, *ErrAktivasiInput, ErrKodeAgenTidakAda, ErrEmailTaken,
// ErrPhoneTaken, atau galat sistem.
func (a *Aktivasi) Jalankan(ctx context.Context, in AktivasiInput) (*AktivasiHasil, error) {
	if len(a.batas.pepper) < MinPepperAktivasi {
		return nil, errPepperPendek
	}
	in.Email, in.NoHP = strings.TrimSpace(in.Email), strings.TrimSpace(in.NoHP)
	if in.Simpan && in.AkunID == 0 {
		if msg := validasiSimpanAktivasi(in); msg != "" {
			return nil, &ErrAktivasiInput{Pesan: msg}
		}
	}
	if in.IP == "" {
		in.IP = "-"
	}
	in.IP = NormalisasiIPAktivasi(in.IP)
	p := &percobaan{a: a, in: in, kode: strings.ToUpper(strings.TrimSpace(in.KodeBooking))}
	nik := strings.TrimSpace(in.NIK)
	b := a.batas
	p.ipH = b.hash("ip", 0, in.IP)
	if in.Perangkat != "" {
		p.devH = b.hash("dev", 0, in.Perangkat)
	}

	if in.Simpan && in.AkunID == 0 {
		if err := p.ambil(ctx, batasSimpanIP, b.hash("simpan_ip", 0, in.IP)); err != nil {
			return nil, err
		}
	}
	if err := p.ambil(ctx, batasIP, p.ipH); err != nil {
		return nil, err
	}
	if p.devH != "" {
		if err := p.ambil(ctx, batasPerangkat, p.devH); err != nil {
			return nil, err
		}
	}
	if in.AkunID > 0 {
		if err := p.ambil(ctx, batasAkun, b.hash("akun", in.BrandID, fmt.Sprint(in.AkunID))); err != nil {
			return nil, err
		}
	}
	if in.BrandID <= 0 || !polaKodeBooking.MatchString(p.kode) || !ValidNIK(nik) {
		return nil, p.gagal(ctx, "cek_gagal", "format", 0)
	}
	p.kodeH = b.hash("kode", in.BrandID, p.kode)
	p.nikH = b.hash("nik", in.BrandID, nik)
	// Setiap kode berformat valid disentuh, ada atau tidak, supaya respons tidak membocorkan keberadaan kode.
	if err := p.ambil(ctx, batasPasangan, b.hash("pasangan", in.BrandID, p.kode+"|"+nik)); err != nil {
		return nil, err
	}
	if err := p.ambil(ctx, batasKode24, p.kodeH); err != nil {
		return nil, err
	}
	if err := p.ambil(ctx, batasNIK24, p.nikH); err != nil {
		return nil, err
	}
	ada, err := a.kodeAda(ctx, in.BrandID, p.kode)
	if err != nil {
		return nil, err
	}
	p.kodeAda = ada

	k, alasan, err := a.cariKandidat(ctx, a.db, in.BrandID, p.kode, nik, in.AkunID)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, p.gagal(ctx, "cek_gagal", alasan, 0)
	}
	hasil := &AktivasiHasil{Jamaah: DataJamaahAktivasi{NamaLengkap: k.Nama, JenisKelamin: k.JK, TanggalLahir: k.Lahir}}
	if !in.Simpan {
		p.kembalikanSemua(ctx)
		return hasil, nil
	}

	var hash string
	var kodeAgenID int64
	if in.AkunID == 0 {
		if k.Kaitan != "terikat_agen" {
			if kodeAgenID, err = a.uplineDariKode(ctx, in); err != nil {
				p.kembalikanSemua(ctx)
				return nil, err
			}
		}
		h, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
		if err != nil {
			p.kembalikanSemua(ctx)
			return nil, fmt.Errorf("aktivasi: hash password: %w", err)
		}
		hash = string(h)
	}
	akunID, err := a.simpanTx(ctx, in, k, p.kode, nik, hash, kodeAgenID, p)
	var ga *galatAlasan
	if errors.As(err, &ga) {
		return nil, p.gagal(ctx, ga.aksi, ga.alasan, ga.jamaahID)
	}
	if err != nil {
		p.kembalikanSemua(ctx)
		return nil, err
	}
	p.kembalikanSemua(ctx)
	hasil.AkunID = akunID
	if in.AkunID == 0 {
		token, err := a.repo.CreateSession(ctx, akunID)
		if err != nil {
			return nil, err
		}
		hasil.Token = token
	}
	return hasil, nil
}

// validasiSimpanAktivasi memeriksa isian akun baru (jalur publik); "" bila valid.
func validasiSimpanAktivasi(in AktivasiInput) string {
	if !ValidEmail(in.Email) {
		return "format email tidak valid"
	}
	if !ValidPhone(in.NoHP) {
		return "nomor WhatsApp harus 10–15 digit"
	}
	if ok, msg := ValidPassword(in.Password); !ok {
		return msg
	}
	return ""
}

// uplineDariKode: untuk jamaah yang belum terikat agen, kode agen berlaku seperti daftar biasa untuk upline akun.
// Kode ketik menang atas kode referral (cookie); kode ketik yang tidak valid ditolak, kode referral yang tidak valid diabaikan.
func (a *Aktivasi) uplineDariKode(ctx context.Context, in AktivasiInput) (int64, error) {
	if ketik := strings.TrimSpace(in.KodeAgen); ketik != "" {
		if !a.kodeAgenGagal.Take(in.IP) {
			return 0, &ErrAktivasiTerlaluSering{Tunggu: time.Hour}
		}
		id, err := agen.ResolveKodeReferral(ctx, a.db, in.BrandID, ketik)
		if err != nil {
			a.kodeAgenGagal.Refund(in.IP)
			return 0, err
		}
		if id == 0 {
			return 0, ErrKodeAgenTidakAda
		}
		a.kodeAgenGagal.Refund(in.IP)
		return id, nil
	}
	return agen.ResolveKodeReferral(ctx, a.db, in.BrandID, in.KodeReferral)
}

// kodeAda: kode booking ada di brand ini (status apa pun). Hanya menentukan apakah kegagalan ditulis ke log.
func (a *Aktivasi) kodeAda(ctx context.Context, brandID int64, kode string) (bool, error) {
	var n int
	err := a.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id_booking = ? AND s.brand_id = ?`, kode, brandID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("aktivasi: cek kode: %w", err)
	}
	return n > 0, nil
}

// cariKandidat: kode booking dan NIK cocok pada jamaah yang sama, pax aktif non-bayi di booking baru/dp/lunas, brand
// sama, dan jamaah belum tertaut ke akun mana pun. Jalur akun: akun itu sendiri juga harus belum tertaut. Mengembalikan
// nil dan sebab internal bila tidak memenuhi.
func (a *Aktivasi) cariKandidat(ctx context.Context, q dbq, brandID int64, kode, nik string, akunID int64) (*kandidat, string, error) {
	if akunID > 0 {
		var jamaah sql.NullInt64
		err := q.QueryRowContext(ctx, `SELECT jamaah_id FROM customer_accounts WHERE id = ? AND brand_id = ? AND dihapus_at IS NULL`, akunID, brandID).Scan(&jamaah)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "akun_tidak_ada", nil
		}
		if err != nil {
			return nil, "", fmt.Errorf("aktivasi: baca akun: %w", err)
		}
		if jamaah.Valid {
			return nil, "akun_tertaut", nil
		}
	}
	var k kandidat
	var jk, lahir sql.NullString
	var kunci sql.NullInt64
	err := q.QueryRowContext(ctx, `
		SELECT j.id, b.id, j.nama_lengkap, j.jenis_kelamin, DATE_FORMAT(j.tanggal_lahir, '%Y-%m-%d'), j.kaitan_status, j.direkrut_oleh_akun_id
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id AND s.brand_id = ?
		JOIN booking_pax bp ON bp.booking_id = b.id AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant'
		JOIN jamaah j ON j.id = bp.jamaah_id AND j.brand_id = s.brand_id AND j.nik = ?
		WHERE b.id_booking = ? AND b.status IN ('baru','dp','lunas')
		  AND NOT EXISTS (SELECT 1 FROM customer_accounts ca WHERE ca.jamaah_id = j.id)
		LIMIT 1`, brandID, nik, kode).Scan(&k.JamaahID, &k.BookingID, &k.Nama, &jk, &lahir, &k.Kaitan, &kunci)
	if errors.Is(err, sql.ErrNoRows) {
		alasan, err := a.alasanGagal(ctx, q, brandID, kode, nik)
		return nil, alasan, err
	}
	if err != nil {
		return nil, "", fmt.Errorf("aktivasi: cari kandidat: %w", err)
	}
	k.JK, k.Lahir, k.Kunci = jk.String, lahir.String, kunci.Int64
	return &k, "", nil
}

// alasanGagal menentukan sebab sebenarnya untuk log admin (tidak pernah dikirim ke pengguna).
func (a *Aktivasi) alasanGagal(ctx context.Context, q dbq, brandID int64, kode, nik string) (string, error) {
	var bookingID int64
	var status string
	err := q.QueryRowContext(ctx, `
		SELECT b.id, b.status FROM bookings b JOIN schedules s ON s.id = b.schedule_id AND s.brand_id = ?
		WHERE b.id_booking = ?`, brandID, kode).Scan(&bookingID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return "kode_salah", nil
	}
	if err != nil {
		return "", fmt.Errorf("aktivasi: alasan booking: %w", err)
	}
	if status != "baru" && status != "dp" && status != "lunas" {
		return "booking_" + status, nil
	}
	var jamaahID int64
	var paxType, paxStatus string
	err = q.QueryRowContext(ctx, `
		SELECT j.id, bp.pax_type, bp.pax_status FROM booking_pax bp JOIN jamaah j ON j.id = bp.jamaah_id AND j.brand_id = ?
		WHERE bp.booking_id = ? AND j.nik = ? ORDER BY (bp.pax_status = 'aktif') DESC LIMIT 1`, brandID, bookingID, nik).Scan(&jamaahID, &paxType, &paxStatus)
	if errors.Is(err, sql.ErrNoRows) {
		var n int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM jamaah WHERE brand_id = ? AND nik = ?`, brandID, nik).Scan(&n); err != nil {
			return "", fmt.Errorf("aktivasi: alasan nik: %w", err)
		}
		if n > 0 {
			return "bukan_peserta", nil
		}
		return "nik_salah", nil
	}
	if err != nil {
		return "", fmt.Errorf("aktivasi: alasan pax: %w", err)
	}
	switch {
	case paxStatus != "aktif":
		return "pax_batal", nil
	case paxType == "infant":
		return "bayi", nil
	}
	return "sudah_tertaut", nil
}

// simpanTx membuat atau menautkan akun dalam satu transaksi: kunci booking dan jamaah, ulang kelayakan, buat/tautkan
// akun, isi upline dari kunci jamaah, tulis log, notifikasi ke pemilik kunci. Mengembalikan *galatAlasan untuk sebab
// gagal yang dihitung sebagai percobaan.
func (a *Aktivasi) simpanTx(ctx context.Context, in AktivasiInput, k *kandidat, kode, nik, hash string, kodeAgenID int64, p *percobaan) (int64, error) {
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("aktivasi: mulai transaksi: %w", err)
	}
	defer tx.Rollback()

	// Urutan kunci sama dengan pembatalan booking: booking dulu, lalu jamaah.
	var x int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM bookings WHERE id = ? AND status IN ('baru','dp','lunas') FOR SHARE`, k.BookingID).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, &galatAlasan{aksi: "cek_gagal", alasan: "booking_batal", jamaahID: k.JamaahID}
	}
	if err != nil {
		return 0, fmt.Errorf("aktivasi: kunci booking: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM jamaah WHERE id = ? FOR UPDATE`, k.JamaahID).Scan(&x); err != nil {
		return 0, fmt.Errorf("aktivasi: kunci jamaah: %w", err)
	}
	k2, alasan, err := a.cariKandidat(ctx, tx, in.BrandID, kode, nik, in.AkunID)
	if err != nil {
		return 0, err
	}
	if k2 == nil || k2.JamaahID != k.JamaahID {
		if k2 != nil {
			alasan = "berubah"
		}
		return 0, &galatAlasan{aksi: "cek_gagal", alasan: alasan, jamaahID: k.JamaahID}
	}

	var akunID int64
	aksi := "tautan"
	if in.AkunID == 0 {
		aksi = "aktivasi"
		nama := k.Nama
		if utf8.RuneCountInString(nama) > 100 {
			nama = string([]rune(nama)[:100])
		}
		akunID, err = a.repo.CreateTx(ctx, tx, in.BrandID, nama, in.Email, in.NoHP, hash, kodeAgenID, k.JamaahID, Biodata{JenisKelamin: k.JK, TanggalLahir: k.Lahir})
		if errors.Is(err, ErrJamaahTertaut) {
			return 0, &galatAlasan{aksi: "cek_gagal", alasan: "sudah_tertaut", jamaahID: k.JamaahID}
		}
		if err != nil {
			return 0, err
		}
	} else {
		akunID = in.AkunID
		res, err := tx.ExecContext(ctx,
			`UPDATE customer_accounts SET jamaah_id = ? WHERE id = ? AND brand_id = ? AND jamaah_id IS NULL AND dihapus_at IS NULL`,
			k.JamaahID, akunID, in.BrandID)
		if err != nil {
			if err = mapDuplicate(err, "tautkan jamaah"); errors.Is(err, ErrJamaahTertaut) {
				return 0, &galatAlasan{aksi: "cek_gagal", alasan: "sudah_tertaut", jamaahID: k.JamaahID}
			}
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return 0, &galatAlasan{aksi: "cek_gagal", alasan: "akun_tertaut", jamaahID: k.JamaahID}
		}
	}
	if err := agen.IkatAkunDariKunciTx(ctx, tx, akunID, k.JamaahID); err != nil {
		if errors.Is(err, agen.ErrUplineBentrok) {
			return 0, &galatAlasan{aksi: "tolak", alasan: "upline_beda", jamaahID: k.JamaahID}
		}
		return 0, err
	}
	if err := catatLogAktivasi(ctx, tx, entriLogAktivasi{
		BrandID: in.BrandID, Aksi: aksi, JamaahID: k.JamaahID, AkunID: akunID,
		KodeHash: p.kodeH, NIKHash: p.nikH, IPHash: p.ipH, DeviceHash: p.devH,
	}); err != nil {
		return 0, err
	}
	if k.Kaitan == "terikat_agen" && k.Kunci > 0 && k.Kunci != akunID {
		notifikasi.JamaahAktivasi(ctx, tx, k.Kunci, k.JamaahID, k.Nama)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("aktivasi: commit: %w", err)
	}
	return akunID, nil
}

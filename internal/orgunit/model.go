package orgunit

// Pemimpin ringkas untuk tampilan pohon unit.
type Pemimpin struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"display_name"`
}

// Unit = node struktur organisasi (divisi atau tim). Dipakai untuk penugasan;
// tidak membawa izin dan tidak membawa metadata yang dibaca logika.
type Unit struct {
	ID            uint64     `json:"id"`
	ParentID      *uint64    `json:"parent_id"`
	Jenis         string     `json:"jenis"` // 'divisi' | 'tim'
	Nama          string     `json:"nama"`
	Urutan        int        `json:"urutan"`
	IsActive      bool       `json:"is_active"`
	JumlahAnggota int        `json:"jumlah_anggota"` // anggota + pemimpin, pengguna aktif saja
	Pemimpin      []Pemimpin `json:"pemimpin"`       // pengguna aktif berposisi pemimpin
	Role          []string   `json:"role,omitempty"` // divisi: role yang berdivisi asal di sini (R10)
	Tim           *[]Unit    `json:"tim,omitempty"`  // selalu ada (boleh []) pada divisi; tidak ada pada tim
}

// CreateRequest payload POST /api/admin/org-units.
type CreateRequest struct {
	Jenis    string  `json:"jenis"`
	Nama     string  `json:"nama"`
	ParentID *uint64 `json:"parent_id"` // wajib untuk tim, wajib kosong untuk divisi
	Urutan   int     `json:"urutan"`
}

// UpdateRequest payload PUT /api/admin/org-units/{id}. Jenis tidak bisa
// diubah; tim boleh dipindah ke divisi lain.
type UpdateRequest struct {
	Nama     string  `json:"nama"`
	ParentID *uint64 `json:"parent_id"`
	Urutan   int     `json:"urutan"`
}

// SetStatusRequest payload PUT /api/admin/org-units/{id}/status.
type SetStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

## Aturan proyek

@.agents/rules/AGENTS.md

Sebelum menyentuh UI dashboard ERP, baca `.agents/rules/design-system.md`. Teks antarmuka mengikuti `docs/ux-copy.md`.

### Aturan wajib

- **Paket = jadwal.** Satu paket adalah satu baris tabel `schedules`. Tidak ada entitas paket terpisah; `internal/paket` hanya placeholder. Jangan tanya "per paket atau per jadwal".
- **Tanpa kata "cicilan".** Tidak ada program cicilan umroh. Pakai DP, pelunasan, total biaya, atau tabungan.
- **Copy UI singkat.** Label, status, dan pesan sesingkat mungkin. Tanpa penjelasan, alasan, atau blok ketentuan. Pilihan yang tidak bisa dipilih disembunyikan, bukan dinonaktifkan dengan keterangan.
- **Tanpa tombol bertumpuk.** Satu tombol utama per kartu atau bagian; aksi kedua jadi tautan atau ikon kecil.
- **Istilah baku:** akun (bisa booking), jamaah (berangkat umroh), agen (punya kode referral), upline, hold seat / lepas seat / kuota seat. Jangan "blokir seat" atau "agen pengikat".

### Verifikasi sebelum PR

- Backend: `go build ./...` dan `go test ./...`
- Frontend: jalankan `npm run build` di dashboard yang diubah
- UI: nilai hasilnya lewat screenshot sebelum dinyatakan selesai

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues (MalikSae/erp-azhan) via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: one `GLOSSARY.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

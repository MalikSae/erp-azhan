# 0006. Dokumen, surat, dan notifikasi dokumen untuk peran jamaah

Tanggal: 18 November 2026
Status: digantikan ADR 0005 (revisi 18 November 2026).

Rute `GET /account/perjalanan-jamaah/{id}/dokumen` dan `/surat` dibuang. Peran jamaah memakai endpoint booking biasa: unggah dokumen dan unduh surat atas nama sendiri dibuka. Notifikasi dokumen untuk akun tertaut tetap berlaku.

package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"erp-azhan/api/internal/shared"
)

// Konsultan dari tautan agen (docs/spek/SPEK-KONSULTAN-TAUTAN-AGEN-2026-11-27.md): pengunjung yang membuka tautan ?ref
// melihat nama agen di halaman paket dan WhatsApp diarahkan ke agen itu. Hanya nama dan WA; foto dan domisili tetap privat.

// Batas kode berbeda per IP per jam. Kode yang sama dari IP yang sama dalam jam itu tidak menghabiskan jatah, jadi
// pengunjung biasa yang membuka banyak halaman paket tidak pernah kena; yang kena hanya penebakan kode.
const konsultanKodePerIPPerJam = 20

type KonsultanView struct {
	NamaLengkap string `json:"nama_lengkap"`
	WhatsApp    string `json:"whatsapp"` // angka saja berawalan 62, siap dipakai di wa.me
}

// CariKonsultan mengembalikan agen aktif pemilik kode di brand ini, atau nil (kode tidak valid, tidak dikenal,
// milik agen nonaktif atau brand lain, atau agen tanpa nomor).
func CariKonsultan(ctx context.Context, q querier, brandID int64, kode string) (*KonsultanView, error) {
	kode = strings.ToUpper(strings.TrimSpace(kode))
	if brandID <= 0 || !KodeAgenValid(kode) {
		return nil, nil
	}
	var v KonsultanView
	var hp string
	err := q.QueryRowContext(ctx,
		`SELECT nama_lengkap, no_hp FROM customer_accounts WHERE brand_id = ? AND kode_referral = ? AND status_agen = 'aktif'`,
		brandID, kode).Scan(&v.NamaLengkap, &hp)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agen: cari konsultan: %w", err)
	}
	v.NamaLengkap = strings.TrimSpace(v.NamaLengkap)
	v.WhatsApp, _ = shared.PhoneVariants(hp)
	if v.NamaLengkap == "" || len(v.WhatsApp) < 8 {
		return nil, nil
	}
	return &v, nil
}

// ambilJatahKonsultan: true bila IP ini boleh menanyakan kode ini sekarang.
func (h *Handler) ambilJatahKonsultan(ip string, brandID int64, kode string) bool {
	kunci := kunciKlikKode(ip, brandID, kode)
	if !h.konsultanIPKode.Take(kunci) {
		return true // kode ini sudah ditanyakan IP ini dalam jam ini
	}
	if !h.konsultanIP.Take(ip) {
		h.konsultanIPKode.Refund(kunci)
		return false
	}
	return true
}

// KonsultanPublik GET /api/public/v1/agen/konsultan?brand_id=&kode=: {"konsultan": {...} | null}. Selalu 200 dengan
// bentuk yang sama; kode tak dikenal dan lewat batas sama-sama null, supaya endpoint tidak bisa dipakai menebak kode.
func (h *Handler) KonsultanPublik(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	q := r.URL.Query()
	brandID, _ := strconv.ParseInt(q.Get("brand_id"), 10, 64)
	kode := strings.ToUpper(strings.TrimSpace(q.Get("kode")))
	var hasil *KonsultanView
	if brandID > 0 && KodeAgenValid(kode) && h.ambilJatahKonsultan(shared.ClientIP(r), brandID, kode) {
		v, err := CariKonsultan(r.Context(), h.repo.db, brandID, kode)
		if err != nil {
			log.Printf("[ERROR] konsultan publik: %v", err)
		}
		hasil = v
	}
	shared.WriteJSON(w, http.StatusOK, map[string]any{"konsultan": hasil})
}

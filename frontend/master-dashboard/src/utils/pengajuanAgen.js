import { useEffect, useState } from 'react';
import { listPengajuanAgen, AKSI_KEPUTUSAN_AGEN as AKSI } from 'shared';

// Aturan dan teks antrean pengajuan agen, dipakai tab Pengajuan (menu Agen), halaman agen, dan sidebar.

export const BAYAR_META = {
  menunggu_verifikasi: { label: 'Menunggu verifikasi', dot: 'bg-warning-500' },
  terverifikasi: { label: 'Lunas', dot: 'bg-success-500' },
  dibebaskan: { label: 'Dibebaskan', dot: 'bg-success-500' },
  ditolak: { label: 'Bukti ditolak, menunggu unggah ulang', dot: 'bg-danger-500' },
};

export const AKSI_META = {
  [AKSI.SETUJUI]: {
    label: 'Setujui',
    desc: 'Bukti valid atau biaya Rp0. Agen aktif, tagihan lunas, kode referral dibuat.',
    descSusulan: 'Bukti valid. Tagihan pendaftaran dicatat lunas; status agen tidak berubah.',
    variant: 'primary',
    wajibCatatan: false,
  },
  [AKSI.BEBASKAN]: {
    label: 'Bebaskan biaya',
    desc: 'Agen aktif, tagihan dihapus tanpa pembayaran.',
    descSusulan: 'Tagihan pendaftaran dihapus tanpa pembayaran; status agen tidak berubah.',
    labelCatatan: 'Catatan internal (tidak dikirim ke pemohon)',
    variant: 'primary',
    wajibCatatan: true,
    placeholder: 'Mis. keluarga pemilik travel, biaya dibebaskan',
  },
  [AKSI.MINTA_UNGGAH_ULANG]: {
    label: 'Minta unggah ulang',
    desc: 'Bukti kurang jelas. Pengajuan tetap menunggu, kembali ke antrean setelah pemohon mengunggah lagi.',
    descSusulan: 'Bukti kurang jelas. Tagihan tetap terbuka sampai agen mengunggah bukti baru.',
    labelCatatan: 'Pesan untuk pemohon',
    variant: 'secondary',
    wajibCatatan: true,
    placeholder: 'Mis. foto buram, nominal tidak terbaca (dibaca pemohon di portal)',
  },
  [AKSI.TOLAK]: {
    label: 'Tolak',
    desc: 'Pengajuan ditutup. Pemohon boleh mengajukan ulang dari awal.',
    variant: 'danger',
    wajibCatatan: true,
    placeholder: 'Alasan penolakan (dibaca pemohon)',
    labelCatatan: 'Alasan penolakan (dibaca pemohon)',
  },
  [AKSI.SETUJUI_BAYAR_MENYUSUL]: {
    label: 'Setujui, bayar menyusul',
    desc: 'Agen aktif sekarang, tagihan tetap tercatat. Lunas setelah bukti diverifikasi, atau bebaskan kemudian.',
    variant: 'primary',
    wajibCatatan: true,
    placeholder: 'Mis. transfer menyusul pekan depan, dikonfirmasi lewat WhatsApp',
    labelCatatan: 'Catatan internal (tidak dikirim ke pemohon)',
  },
};

// Aksi yang tersedia mengikuti aturan keputusanTx di backend (internal/agen/keputusan.go).
// Mengembalikan [{ aksi, alasanNonaktif }]; alasanNonaktif terisi bila aksi ada tetapi belum bisa dipilih.
export function aksiTersedia(row) {
  const p = row.pembayaran;
  if (!p) return [];
  const final = p.status === 'terverifikasi' || p.status === 'dibebaskan';
  const berbayar = Number(p.nominal_tagihan) > 0;
  const adaBukti = p.status === 'menunggu_verifikasi' && Boolean(p.bukti_transfer_url);
  const pengajuan = row.status_agen === 'pengajuan';
  // Tagihan agen yang sudah disetujui tetap bisa diputuskan walau agennya dinonaktifkan.
  const susulan = (row.status_agen === 'aktif' || row.status_agen === 'nonaktif') && p.keputusan_agen === 'disetujui' && !final && berbayar;
  if (!pengajuan && !susulan) return [];

  const setujui = { aksi: AKSI.SETUJUI };
  if (!final && !adaBukti) setujui.alasanNonaktif = 'Bukti transfer belum ada atau sudah ditolak. Pilih bebaskan, bayar menyusul, atau tunggu unggahan.';
  const daftar = [setujui];
  if (!final) daftar.push({ aksi: AKSI.BEBASKAN });
  if ((pengajuan || susulan) && !final && berbayar) {
    daftar.push({ aksi: AKSI.MINTA_UNGGAH_ULANG, ...(adaBukti ? {} : { alasanNonaktif: 'Hanya untuk bukti yang menunggu verifikasi.' }) });
  }
  if (pengajuan) daftar.push({ aksi: AKSI.TOLAK });
  if (pengajuan && !final && berbayar) daftar.push({ aksi: AKSI.SETUJUI_BAYAR_MENYUSUL });
  return daftar;
}

// Dikirim setiap kali antrean dimuat ulang (termasuk setelah keputusan), membawa jumlah barisnya, supaya
// angka di sidebar dan di tab ikut berubah tanpa pindah halaman.
const EVENT_PENGAJUAN_AGEN = 'pengajuan-agen-dimuat';

export function umumkanPengajuanAgen(items) {
  window.dispatchEvent(new CustomEvent(EVENT_PENGAJUAN_AGEN, { detail: (items || []).length }));
}

// Memuat antrean pengajuan sekaligus mengumumkan jumlahnya.
export async function muatPengajuanAgen() {
  const data = await listPengajuanAgen();
  umumkanPengajuanAgen(data);
  return data;
}

// Jumlah baris antrean pengajuan agen. null selama belum diketahui atau bila `aktif` false (pengguna tanpa
// izin agen.approve tidak memanggil API-nya).
export function useJumlahPengajuanAgen(aktif) {
  const [jumlah, setJumlah] = useState(null);
  useEffect(() => {
    if (!aktif) return undefined;
    let batal = false;
    listPengajuanAgen().then((d) => { if (!batal) setJumlah((d || []).length); }).catch(() => {});
    const dengar = (e) => setJumlah(e.detail);
    window.addEventListener(EVENT_PENGAJUAN_AGEN, dengar);
    return () => {
      batal = true;
      window.removeEventListener(EVENT_PENGAJUAN_AGEN, dengar);
    };
  }, [aktif]);
  return jumlah;
}

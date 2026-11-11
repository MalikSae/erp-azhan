import { useCallback, useEffect, useState } from 'react';
import { TriangleAlert } from 'lucide-react';
import {
  keputusanAgen,
  AKSI_KEPUTUSAN_AGEN as AKSI,
  uploadBuktiPembayaranAgen,
  uploadMedia,
  openProtectedMedia,
  useProtectedImage,
  dateLabel,
  money,
} from 'shared';
import ActionMenu from 'shared/ui/ActionMenu';
import Alert from 'shared/ui/Alert';
import Button from 'shared/ui/Button';
import { AKSI_META, BAYAR_META, aksiTersedia, muatPengajuanAgen } from '../utils/pengajuanAgen';

// Semua tombol di kartu ini setinggi tombol menu "Lainnya".
const TINGGI = 'h-[38px]';

// Label dan nilai berdampingan, rata kiri: mata tidak perlu menyeberang dari tepi kiri ke tepi kanan.
const Baris = ({ label, children }) => (
  <div className="flex gap-3 text-sm">
    <span className="w-24 shrink-0 text-[13px] font-body text-neutral-500">{label}</span>
    <span className="min-w-0 text-neutral-900 font-heading font-semibold break-words">{children}</span>
  </div>
);

// Satu pengajuan agen beserta keputusan admin (spek keagenan bagian 7): setujui, bebaskan biaya, minta unggah
// ulang, tolak, atau setujui dengan bayar menyusul. Tampil di halaman agen selama akun itu masih ada di
// antrean (pengajuan baru, atau agen yang tagihan pendaftarannya belum selesai). Di luar itu tidak tampil.
// onBerubah dipanggil setelah keputusan atau unggahan tersimpan; onAda memberi tahu halaman apakah akun ini
// ada di antrean.
export default function TinjauPengajuanAgen({ akunId, onBerubah, onAda }) {
  const [row, setRow] = useState(undefined); // undefined = memuat, null = tidak di antrean
  const [gagalMuat, setGagalMuat] = useState(false);
  const [aksi, setAksi] = useState(null);
  const [catatan, setCatatan] = useState('');
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');
  const fotoSrc = useProtectedImage(row?.foto_url);
  // Bukti transfer terakhir, ditampilkan langsung di samping data. Data lama bisa punya bukti di tagihan
  // tanpa baris riwayat unggahan, jadi tagihan dipakai sebagai cadangan.
  const tagihan = row?.pembayaran;
  const buktiTerbaru = !tagihan || !(Number(tagihan.nominal_tagihan) > 0) ? null
    : tagihan.uploads?.length > 0 ? tagihan.uploads[tagihan.uploads.length - 1]
      : tagihan.bukti_transfer_url ? { bukti_transfer_url: tagihan.bukti_transfer_url } : null;
  const buktiPdf = /\.pdf$/i.test(buktiTerbaru?.bukti_transfer_url || '');
  const buktiSrc = useProtectedImage(buktiPdf ? null : buktiTerbaru?.bukti_transfer_url);

  const muat = useCallback(async () => {
    try {
      const data = await muatPengajuanAgen();
      const ketemu = data.find((it) => String(it.akun_id) === String(akunId)) || null;
      setRow(ketemu);
      setGagalMuat(false);
      onAda?.(Boolean(ketemu), ketemu);
    } catch {
      setRow(null);
      setGagalMuat(true);
      onAda?.(false, null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [akunId]);

  useEffect(() => {
    setRow(undefined);
    setAksi(null);
    setCatatan('');
    setActionError('');
    muat();
  }, [muat]);

  const selesai = async () => {
    setAksi(null);
    setCatatan('');
    await muat();
    onBerubah?.();
  };

  const unggah = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file || !row?.pembayaran) return;
    setSaving(true);
    setActionError('');
    try {
      const url = await uploadMedia(file, 'payment-proofs');
      await uploadBuktiPembayaranAgen(row.pembayaran.id, url);
      await selesai();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Bukti transfer gagal diunggah.');
    } finally {
      setSaving(false);
    }
  };

  // kodeLangsung: kode aksi bila dipanggil tanpa menunggu state `aksi`.
  const simpan = async (kodeLangsung) => {
    const kode = kodeLangsung || aksi;
    const meta = AKSI_META[kode];
    if (!meta) return;
    if (meta.wajibCatatan && catatan.trim().length < 3) {
      setActionError(kode === AKSI.TOLAK ? 'Alasan wajib diisi (minimal 3 karakter).' : 'Catatan wajib diisi (minimal 3 karakter).');
      return;
    }
    setSaving(true);
    setActionError('');
    try {
      // Catatan hanya dikirim untuk aksi yang memintanya; teks dari pilihan aksi sebelumnya tidak ikut.
      await keputusanAgen(row.akun_id, kode, meta.wajibCatatan ? catatan.trim() : '');
      await selesai();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Keputusan gagal disimpan.');
    } finally {
      setSaving(false);
    }
  };

  if (gagalMuat) return <Alert variant="error" message="Pengajuan agen gagal dimuat. Silakan coba kembali." />;
  if (!row) return null;

  const p = row.pembayaran;
  const berbayar = p && Number(p.nominal_tagihan) > 0;
  const bukaBukti = (u) => {
    setActionError('');
    return openProtectedMedia(u.bukti_transfer_url).catch((err) => setActionError(
      err?.response?.status === 404 ? 'Berkas bukti transfer tidak ditemukan di server.' : 'Bukti transfer gagal dibuka.'
    ));
  };
  const bayarLabel = !p ? null
    : p.status === 'menunggu_verifikasi' && !buktiTerbaru ? { label: 'Belum ada bukti', dot: 'bg-neutral-300' }
      : (BAYAR_META[p.status] || { label: p.status });
  const pilihan = aksiTersedia(row);
  const aksiMeta = aksi ? AKSI_META[aksi] : null;
  const bisaUnggah = p && berbayar && p.status !== 'terverifikasi' && p.status !== 'dibebaskan';

  const setujui = pilihan.find((x) => x.aksi === AKSI.SETUJUI);
  const tolak = pilihan.find((x) => x.aksi === AKSI.TOLAK);
  const lainnya = pilihan.filter((x) => x.aksi !== AKSI.SETUJUI && x.aksi !== AKSI.TOLAK);

  // Memilih aksi belum menyimpan: konfirmasi (dan catatan, bila diminta) muncul di bawah tombol.
  const pilihAksi = (kode) => {
    setActionError('');
    setCatatan('');
    setAksi(kode);
  };

  // Hanya isian yang terisi, dalam urutan memeriksa: identitas dulu, lalu jaringan, lalu sisanya.
  const data = [
    ['NIK', row.nik_masked ? <span className="font-mono">{row.nik_masked}</span> : null],
    ['Upline', row.direkrut_oleh_nama],
    ['Domisili', row.domisili],
    ['Tanggal lahir', row.tanggal_lahir ? dateLabel(row.tanggal_lahir) : null],
    ['Jenis kelamin', row.jenis_kelamin === 'L' ? 'Laki-laki' : row.jenis_kelamin === 'P' ? 'Perempuan' : null],
    ['Setuju S&K', row.menyetujui_syarat_ketentuan_agen_at ? dateLabel(row.menyetujui_syarat_ketentuan_agen_at) : null],
  ].filter(([, nilai]) => nilai);

  return (
    <div className="max-w-xl space-y-4">
      {actionError && <Alert variant="error" message={actionError} />}

      {/* Satu kartu sempit, satu kolom: data pemohon, lalu biaya dengan pratinjau bukti, lalu keputusan.
          Tidak ada dua kolom, jadi tidak ada sisi yang kosong saat isinya sedikit. */}
      <div className="bg-white border border-neutral-200/80 rounded-2xl shadow-card overflow-hidden">
        <div className="p-5 space-y-4">
          {row.peringatan_nik?.length > 0 && (
            <div role="alert" className="rounded-xl border border-warning-300 bg-warning-50 px-3.5 py-2.5 text-xs text-warning-900">
              <p className="flex items-center gap-2 font-bold"><TriangleAlert size={14} />Periksa NIK</p>
              <ul className="mt-1 list-disc pl-6 space-y-0.5">
                {row.peringatan_nik.map((w) => <li key={w}>{w}</li>)}
              </ul>
            </div>
          )}

          <div className="flex gap-4">
            <div className="min-w-0 flex-1 space-y-2.5">
              {data.map(([label, nilai]) => <Baris key={label} label={label}>{nilai}</Baris>)}
              {!berbayar && <Baris label="Biaya">Tanpa biaya</Baris>}
            </div>
            {fotoSrc && (
              <div className="h-20 w-20 shrink-0 overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50">
                <img src={fotoSrc} alt={`Foto ${row.nama_lengkap}`} className="h-full w-full object-cover" />
              </div>
            )}
          </div>

          {berbayar && (
            <div className="flex items-center gap-4 border-t border-neutral-200/80 pt-4">
              {/* Pratinjau bukti berukuran tetap; klik untuk memperbesar. */}
              {buktiTerbaru && !buktiPdf ? (
                <button
                  type="button"
                  onClick={() => bukaBukti(buktiTerbaru)}
                  title="Perbesar bukti transfer"
                  className="h-20 w-20 shrink-0 overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 cursor-zoom-in"
                >
                  {buktiSrc
                    ? <img src={buktiSrc} alt="Bukti transfer" className="h-full w-full object-cover object-top" />
                    : <span className="flex h-full items-center justify-center px-1 text-center text-[10px] text-neutral-400">Tidak bisa ditampilkan</span>}
                </button>
              ) : (
                <div className="flex h-20 w-20 shrink-0 items-center justify-center rounded-xl border border-dashed border-neutral-300 bg-neutral-50 px-1 text-center text-[11px] text-neutral-500">
                  {buktiTerbaru ? 'PDF' : 'Belum ada bukti'}
                </div>
              )}

              <div className="min-w-0 flex-1">
                <p className="text-[13px] font-body text-neutral-500">Biaya Pendaftaran</p>
                <p className="text-lg font-heading font-bold text-neutral-900 leading-tight">{money(p.nominal_tagihan)}</p>
                {bayarLabel && (
                  <span className="mt-0.5 inline-flex items-center gap-1.5 text-[12px] font-body text-neutral-600">
                    <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${bayarLabel.dot || 'bg-neutral-400'}`} />
                    {bayarLabel.label}
                  </span>
                )}
              </div>

              <div className="flex shrink-0 flex-col items-end gap-1 text-xs font-semibold">
                {buktiTerbaru && <button type="button" onClick={() => bukaBukti(buktiTerbaru)} className="text-primary-600 hover:underline cursor-pointer">Lihat bukti</button>}
                {bisaUnggah && (
                  <label className={`text-primary-600 hover:underline ${saving ? 'opacity-50 pointer-events-none' : 'cursor-pointer'}`}>
                    {buktiTerbaru ? 'Unggah bukti baru' : 'Unggah bukti'}
                    <input type="file" accept="image/*,.pdf" className="hidden" onChange={unggah} />
                  </label>
                )}
              </div>
            </div>
          )}

          {/* Bukti lama hanya tampil bila pemohon pernah mengunggah lebih dari sekali. */}
          {berbayar && p.uploads.length > 1 && (
            <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs">
              {p.uploads.map((u, i) => (
                <li key={u.id}>
                  <button type="button" onClick={() => bukaBukti(u)} className="text-primary-600 font-semibold hover:underline cursor-pointer">Bukti {i + 1}</button>
                  <span className="text-neutral-500"> · {dateLabel(u.created_at)}{u.diupload_oleh_tipe === 'admin' ? ' · admin' : ''}</span>
                </li>
              ))}
            </ul>
          )}
          {p?.status === 'ditolak' && p.catatan_penolakan && <p className="text-xs text-danger-700">Catatan unggah ulang: {p.catatan_penolakan}</p>}
          {p?.catatan_admin && <p className="text-xs text-neutral-600">Catatan admin: {p.catatan_admin}</p>}

          {p?.riwayat_keputusan?.length > 0 && (
            <ul className="space-y-1 border-t border-neutral-200/80 pt-3 text-xs text-neutral-600">
              {p.riwayat_keputusan.map((k, i) => (
                <li key={i}>
                  <span className="font-semibold text-neutral-900">{AKSI_META[k.aksi]?.label || k.aksi}</span> · {k.oleh_nama || 'admin'} · {dateLabel(k.created_at)}{k.catatan ? ` · ${k.catatan}` : ''}
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Kaki kartu: keputusan. Memilih belum menyimpan; konfirmasi muncul di tempat yang sama. */}
        {pilihan.length > 0 && (
          <div className="border-t border-neutral-200/80 bg-neutral-50/70 px-5 py-3.5 space-y-3">
            {aksiMeta?.wajibCatatan && (
              <div className="space-y-1.5">
                <label htmlFor="catatan-keputusan" className="block text-xs font-bold text-neutral-700">
                  {aksiMeta.labelCatatan || 'Catatan admin'} <span className="text-danger-600">*</span>
                </label>
                <textarea
                  id="catatan-keputusan"
                  value={catatan}
                  onChange={(e) => setCatatan(e.target.value)}
                  rows={2}
                  placeholder={aksiMeta.placeholder}
                  disabled={saving}
                  className="w-full rounded-lg border border-neutral-300 bg-white p-2.5 text-xs text-neutral-800 placeholder-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
                />
              </div>
            )}
            <div className="flex flex-wrap items-center justify-end gap-2">
              {aksiMeta ? (
                <>
                  <Button variant="secondary" className={TINGGI} onClick={() => setAksi(null)} disabled={saving}>Batal</Button>
                  <Button variant={aksiMeta.variant === 'danger' ? 'danger' : 'primary'} className={TINGGI} onClick={() => simpan()} isLoading={saving}>
                    Simpan: {aksiMeta.label}
                  </Button>
                </>
              ) : (
                <>
                  {lainnya.length > 0 && (
                    <ActionMenu
                      label="Lainnya"
                      align="right"
                      items={lainnya.map(({ aksi: kode, alasanNonaktif }) => ({
                        label: AKSI_META[kode].label,
                        disabled: Boolean(alasanNonaktif),
                        disabledTooltip: alasanNonaktif,
                        onClick: () => pilihAksi(kode),
                      }))}
                    />
                  )}
                  {tolak && <Button variant="danger-light" className={TINGGI} disabled={saving} onClick={() => pilihAksi(AKSI.TOLAK)}>Tolak</Button>}
                  {setujui && (
                    <span title={setujui.alasanNonaktif || undefined}>
                      <Button variant="primary" className={TINGGI} disabled={saving || Boolean(setujui.alasanNonaktif)} onClick={() => pilihAksi(AKSI.SETUJUI)}>Setujui</Button>
                    </span>
                  )}
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

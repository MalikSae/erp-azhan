import { useEffect, useState } from 'react';
import {
  listPengajuanAgen,
  keputusanAgen,
  AKSI_KEPUTUSAN_AGEN as AKSI,
  uploadBuktiPembayaranAgen,
  uploadMedia,
  openProtectedMedia,
  useProtectedImage,
} from 'shared';
import Alert from 'shared/ui/Alert';
import ActionMenu from 'shared/ui/ActionMenu';
import StatTile from 'shared/ui/StatTile';
import Button from 'shared/ui/Button';
import DataTable from 'shared/ui/DataTable';
import Modal from 'shared/ui/Modal';
import PageHeader from 'shared/ui/PageHeader';
import { Eye, UserCheck, ClipboardList, Banknote, TriangleAlert } from 'lucide-react';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(value));
const bayarMeta = {
  menunggu_verifikasi: { label: 'Menunggu verifikasi', dot: 'bg-warning-500' },
  terverifikasi: { label: 'Lunas', dot: 'bg-success-500' },
  dibebaskan: { label: 'Dibebaskan', dot: 'bg-success-500' },
  ditolak: { label: 'Bukti ditolak, menunggu unggah ulang', dot: 'bg-danger-500' },
};

// Status = titik berwarna kecil + teks (bukan badge berwarna).
const StatusDot = ({ meta }) => (
  <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-neutral-700">
    <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${meta.dot || 'bg-neutral-400'}`} />
    {meta.label}
  </span>
);

const AKSI_META = {
  [AKSI.SETUJUI]: {
    label: 'Setujui',
    desc: 'Bukti valid atau biaya Rp0. Agen aktif, tagihan lunas, kode referral dibuat.',
    variant: 'primary',
    wajibCatatan: false,
  },
  [AKSI.BEBASKAN]: {
    label: 'Bebaskan biaya',
    desc: 'Agen aktif, tagihan dihapus tanpa pembayaran.',
    variant: 'primary',
    wajibCatatan: true,
    placeholder: 'Mis. keluarga pemilik travel, biaya dibebaskan',
  },
  [AKSI.MINTA_UNGGAH_ULANG]: {
    label: 'Minta unggah ulang',
    desc: 'Bukti kurang jelas. Pengajuan tetap menunggu, kembali ke antrean setelah pemohon mengunggah lagi.',
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
  },
  [AKSI.SETUJUI_BAYAR_MENYUSUL]: {
    label: 'Setujui, bayar menyusul',
    desc: 'Agen aktif sekarang, tagihan tetap tercatat. Lunas setelah bukti diverifikasi, atau bebaskan kemudian.',
    variant: 'primary',
    wajibCatatan: true,
    placeholder: 'Mis. transfer menyusul pekan depan, dikonfirmasi lewat WhatsApp',
  },
};

// Aksi yang tersedia mengikuti aturan keputusanTx di backend (internal/agen/keputusan.go).
// Mengembalikan [{ aksi, alasanNonaktif }]; alasanNonaktif terisi bila aksi ada tetapi belum bisa dipilih.
function aksiTersedia(row) {
  const p = row.pembayaran;
  if (!p) return [];
  const final = p.status === 'terverifikasi' || p.status === 'dibebaskan';
  const berbayar = Number(p.nominal_tagihan) > 0;
  const adaBukti = p.status === 'menunggu_verifikasi' && Boolean(p.bukti_transfer_url);
  const pengajuan = row.status_agen === 'pengajuan';
  const susulan = row.status_agen === 'aktif' && p.keputusan_agen === 'disetujui' && !final && berbayar;
  if (!pengajuan && !susulan) return [];

  const setujui = { aksi: AKSI.SETUJUI };
  if (!final && !adaBukti) setujui.alasanNonaktif = 'Bukti transfer belum ada atau sudah ditolak. Pilih bebaskan, bayar menyusul, atau tunggu unggahan.';
  const daftar = [setujui];
  if (!final) daftar.push({ aksi: AKSI.BEBASKAN });
  if (pengajuan && !final && berbayar) {
    daftar.push({ aksi: AKSI.MINTA_UNGGAH_ULANG, ...(adaBukti ? {} : { alasanNonaktif: 'Hanya untuk bukti yang menunggu verifikasi.' }) });
  }
  if (pengajuan) daftar.push({ aksi: AKSI.TOLAK });
  if (pengajuan && !final && berbayar) daftar.push({ aksi: AKSI.SETUJUI_BAYAR_MENYUSUL });
  return daftar;
}

// Screen B1 — satu kartu pengajuan, satu keputusan admin dengan lima aksi (spek keagenan bagian 7).
export default function PersetujuanAgenPage() {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [aksi, setAksi] = useState(null);
  const [catatan, setCatatan] = useState('');
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');
  const fotoSrc = useProtectedImage(selected?.foto_agen_url);

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const data = await listPengajuanAgen();
      setItems(data);
      return data;
    } catch {
      setError('Pengajuan agen gagal dimuat. Silakan coba kembali.');
      return [];
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const open = (row) => {
    setSelected(row);
    setAksi(null);
    setCatatan('');
    setActionError('');
  };

  const close = () => {
    if (saving) return;
    setSelected(null);
  };

  const refreshSelected = async () => {
    const data = await load();
    const fresh = data.find((it) => it.akun_id === selected?.akun_id);
    setSelected(fresh || null);
    setAksi(null);
    setCatatan('');
  };

  const handleUploadAdmin = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file || !selected?.pembayaran) return;
    setSaving(true);
    setActionError('');
    try {
      const url = await uploadMedia(file, 'payment-proofs');
      await uploadBuktiPembayaranAgen(selected.pembayaran.id, url);
      await refreshSelected();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Bukti transfer gagal diunggah.');
    } finally {
      setSaving(false);
    }
  };

  const submitKeputusan = async () => {
    const meta = AKSI_META[aksi];
    if (!meta) return;
    if (meta.wajibCatatan && catatan.trim().length < 3) {
      setActionError(aksi === AKSI.TOLAK ? 'Alasan wajib diisi (minimal 3 karakter).' : 'Catatan wajib diisi (minimal 3 karakter).');
      return;
    }
    setSaving(true);
    setActionError('');
    try {
      await keputusanAgen(selected.akun_id, aksi, catatan.trim());
      await refreshSelected();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Keputusan gagal disimpan.');
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    { header: 'Pemohon', key: 'nama_lengkap', sortable: true },
    { header: 'Domisili', key: 'domisili', sortable: true },
    { header: 'Diajukan', key: 'diajukan_agen_at', sortable: true },
    { header: 'Pembayaran', key: 'pembayaran' },
    { header: 'Aksi', key: 'actions' },
  ];

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') {
      return (
        <div className="min-w-[160px]">
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); open(row); }}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors cursor-pointer"
            title="Tinjau Pengajuan"
          >
            {row.nama_lengkap}
          </button>
          <p className="mt-0.5 text-[11px] text-neutral-400 font-mono">{row.nik_masked || '-'}{row.no_hp ? ` • ${row.no_hp}` : ''}</p>
          {row.peringatan_nik?.length > 0 && (
            <p className="mt-1 inline-flex items-center gap-1 text-[11px] font-semibold text-warning-700"><TriangleAlert size={12} />Periksa NIK</p>
          )}
          {row.status_agen !== 'pengajuan' && (
            <div className="mt-1 inline-flex items-center gap-1.5 text-[11px] font-body text-success-700"><span className="w-1.5 h-1.5 rounded-full bg-success-500" />Agen {row.status_agen} • tinggal pembayaran</div>
          )}
          {row.direkrut_oleh_nama && <p className="mt-0.5 text-[11px] text-neutral-400">Upline: {row.direkrut_oleh_nama}</p>}
        </div>
      );
    }
    if (key === 'diajukan_agen_at') return dateLabel(row.diajukan_agen_at);
    if (key === 'pembayaran') {
      const p = row.pembayaran;
      if (!p) return '-';
      if (Number(p.nominal_tagihan) === 0) return <span className="text-xs text-neutral-500">Tanpa biaya</span>;
      const meta = bayarMeta[p.status] || { label: p.status };
      return (
        <div>
          <p className="text-[13px] font-semibold text-neutral-900">{money(p.nominal_tagihan)}</p>
          <div className="mt-1"><StatusDot meta={meta} /></div>
        </div>
      );
    }
    if (key === 'actions') {
      return <ActionMenu items={[{ label: 'Tinjau', icon: Eye, tooltip: 'Tinjau Pengajuan', onClick: () => open(row) }]} align="right" />;
    }
    return row[key] ?? '-';
  };

  const p = selected?.pembayaran;
  const berbayar = p && Number(p.nominal_tagihan) > 0;
  const bayarLabel = p ? (bayarMeta[p.status] || { label: p.status }) : null;
  const pilihan = selected ? aksiTersedia(selected) : [];
  const aksiMeta = aksi ? AKSI_META[aksi] : null;
  const bisaUnggah = p && berbayar && p.status !== 'terverifikasi' && p.status !== 'dibebaskan';

  return (
    <div className="space-y-5">
      <PageHeader title="Persetujuan Agen" />
      {error && <Alert variant="error" message={error} />}

      {!loading && (
        <div className="grid grid-cols-2 lg:grid-cols-3 gap-4">
          <StatTile icon={ClipboardList} variant="primary" value={items.length} label="Total Pengajuan" />
          <StatTile icon={UserCheck} variant="warning" value={items.filter((it) => it.status_agen === 'pengajuan').length} label="Menunggu Keputusan" />
          <StatTile icon={Banknote} variant="info" value={items.filter((it) => it.pembayaran?.status === 'menunggu_verifikasi').length} label="Bukti Perlu Diperiksa" />
        </div>
      )}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        itemsPerPage={10}
        onRowClick={open}
        searchPlaceholder="Cari nama, nomor HP, atau domisili..."
        emptyMessage={loading ? 'Memuat pengajuan...' : 'Belum ada pengajuan agen'}
      />

      <Modal
        isOpen={Boolean(selected)}
        onClose={close}
        title="Tinjau Pengajuan Agen"
        size="2xl"
        footer={
          <div className="flex flex-col-reverse sm:flex-row sm:items-center sm:justify-between gap-2 w-full">
            <Button variant="secondary" onClick={close} disabled={saving}>Tutup</Button>
            {pilihan.length > 0 && (
              <Button
                variant={aksiMeta?.variant === 'danger' ? 'danger' : 'primary'}
                onClick={submitKeputusan}
                isLoading={saving}
                disabled={!aksi}
              >
                {aksiMeta ? `Simpan keputusan: ${aksiMeta.label}` : 'Pilih satu keputusan'}
              </Button>
            )}
          </div>
        }
      >
        {selected && (
          <div className="space-y-4">
            {actionError && <Alert variant="error" message={actionError} />}

            {selected.peringatan_nik?.length > 0 && (
              <div role="alert" className="rounded-xl border border-warning-300 bg-warning-50 p-3.5 space-y-1.5">
                <p className="flex items-center gap-2 text-sm font-bold text-warning-900"><TriangleAlert size={16} />Periksa NIK sebelum menyetujui</p>
                <ul className="list-disc pl-6 text-xs text-warning-900 space-y-0.5">
                  {selected.peringatan_nik.map((w) => <li key={w}>{w}</li>)}
                </ul>
              </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-3">
                <div className="p-3.5 rounded-xl border border-neutral-200/70 space-y-2 text-xs">
                  <div className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Nama</span>
                    <span className="font-semibold text-neutral-900 text-right">{selected.nama_lengkap}</span>
                  </div>
                  <div className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">NIK</span>
                    <span className="font-mono text-neutral-800 text-right">{selected.nik_masked || '-'}</span>
                  </div>
                  <div className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Nomor WA</span>
                    <span className="font-mono text-neutral-800 text-right">{selected.no_hp || '-'}</span>
                  </div>
                  <div className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Domisili</span>
                    <span className="font-semibold text-neutral-800 text-right">{selected.domisili || '-'}</span>
                  </div>
                  <div className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Setuju S&K</span>
                    <span className="text-neutral-800">{dateLabel(selected.menyetujui_syarat_ketentuan_agen_at)}</span>
                  </div>
                  <div className="flex justify-between gap-2">
                    <span className="text-neutral-500">Upline</span>
                    <span className="text-neutral-800 text-right">{selected.direkrut_oleh_nama || 'Tanpa agen (tidak ada upline)'}</span>
                  </div>
                </div>

                <div className="p-3.5 rounded-xl bg-neutral-50 border border-neutral-200/80 space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[10px] font-bold uppercase tracking-wider text-neutral-400">Biaya pendaftaran</span>
                    {berbayar && bayarLabel && <StatusDot meta={bayarLabel} />}
                  </div>
                  <p className="text-xl font-black text-neutral-900">{berbayar ? money(p.nominal_tagihan) : 'Tanpa biaya'}</p>
                  {p?.status === 'ditolak' && p.catatan_penolakan && (
                    <p className="text-xs text-danger-700 bg-danger-50 p-2 rounded-lg border border-danger-200/60">
                      <strong>Catatan unggah ulang:</strong> {p.catatan_penolakan}
                    </p>
                  )}
                  {p?.catatan_admin && (
                    <p className="text-xs text-neutral-700 bg-white p-2 rounded-lg border border-neutral-200">
                      <strong>Catatan admin:</strong> {p.catatan_admin}
                    </p>
                  )}

                  {berbayar && (
                    <div className="space-y-1.5 pt-1">
                      <span className="text-xs font-bold text-neutral-700">Riwayat bukti transfer</span>
                      {p.uploads.length === 0 ? (
                        <p className="text-xs text-neutral-500">Belum ada bukti diunggah (mungkin konfirmasi lewat WhatsApp).</p>
                      ) : (
                        <ul className="divide-y divide-neutral-200/70 text-xs">
                          {p.uploads.map((u, i) => (
                            <li key={u.id} className="py-1.5 flex items-center justify-between gap-2">
                              <span className="text-neutral-700">
                                #{i + 1} • {dateLabel(u.created_at)} • {u.diupload_oleh_tipe === 'admin' ? 'oleh admin' : 'oleh pemohon'}
                              </span>
                              <button
                                type="button"
                                onClick={() => openProtectedMedia(u.bukti_transfer_url).catch(() => setActionError('Bukti transfer gagal dibuka.'))}
                                className="text-primary-600 font-semibold hover:underline cursor-pointer"
                              >
                                Lihat
                              </button>
                            </li>
                          ))}
                        </ul>
                      )}
                      {bisaUnggah && (
                        <label className={`inline-flex items-center justify-center rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-50 ${saving ? 'opacity-50 pointer-events-none' : 'cursor-pointer'}`}>
                          Unggah bukti atas nama pemohon
                          <input type="file" accept="image/*,.pdf" className="hidden" onChange={handleUploadAdmin} />
                        </label>
                      )}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex flex-col">
                <span className="text-xs font-bold text-neutral-700 mb-2 block">Foto agen</span>
                <div className="flex-1 min-h-[240px] max-h-[340px] rounded-xl border border-neutral-200 bg-neutral-50 overflow-hidden flex items-center justify-center p-2">
                  {fotoSrc ? (
                    <img src={fotoSrc} alt={`Foto ${selected.nama_lengkap}`} className="max-h-[320px] w-auto max-w-full object-contain rounded-lg" />
                  ) : (
                    <p className="text-xs text-neutral-400">Memuat foto...</p>
                  )}
                </div>
              </div>
            </div>

            {pilihan.length === 0 ? (
              <p className="text-xs text-neutral-500">Tidak ada keputusan yang bisa diambil pada status ini.</p>
            ) : (
              <fieldset className="space-y-2" disabled={saving}>
                <legend className="text-xs font-bold text-neutral-700 mb-1">
                  {selected.status_agen === 'pengajuan' ? 'Keputusan admin' : 'Keputusan atas tagihan yang tersisa'}
                </legend>
                {pilihan.map(({ aksi: kode, alasanNonaktif }) => {
                  const meta = AKSI_META[kode];
                  const aktif = aksi === kode;
                  return (
                    <label
                      key={kode}
                      className={`flex items-start gap-3 rounded-xl border p-3 text-xs transition-colors ${
                        alasanNonaktif ? 'opacity-60 cursor-not-allowed border-neutral-200 bg-neutral-50'
                          : aktif ? 'cursor-pointer border-primary-400 bg-primary-50' : 'cursor-pointer border-neutral-200 hover:bg-neutral-50'
                      }`}
                    >
                      <input
                        type="radio"
                        name="keputusan-agen"
                        value={kode}
                        checked={aktif}
                        disabled={Boolean(alasanNonaktif)}
                        onChange={() => { setAksi(kode); setActionError(''); }}
                        className="mt-0.5 w-4 h-4 text-primary-500 border-neutral-300 focus:ring-primary-500"
                      />
                      <span className="min-w-0">
                        <span className="block font-semibold text-neutral-900">{meta.label}</span>
                        <span className="block text-neutral-500 mt-0.5">{meta.desc}</span>
                        {alasanNonaktif && <span className="block text-neutral-500 mt-0.5 italic">{alasanNonaktif}</span>}
                        {kode === AKSI.SETUJUI && selected.peringatan_nik?.length > 0 && !alasanNonaktif && (
                          <span className="mt-1 flex items-center gap-1 font-semibold text-warning-700"><TriangleAlert size={12} />Ada peringatan NIK di atas. Periksa dulu.</span>
                        )}
                      </span>
                    </label>
                  );
                })}

                {aksiMeta?.wajibCatatan && (
                  <div className="space-y-1.5 pt-1">
                    <label htmlFor="catatan-keputusan" className="block text-xs font-bold text-neutral-700">
                      {aksi === AKSI.TOLAK ? 'Alasan penolakan' : 'Catatan admin'} <span className="text-danger-600">*</span>
                    </label>
                    <textarea
                      id="catatan-keputusan"
                      value={catatan}
                      onChange={(e) => setCatatan(e.target.value)}
                      rows={3}
                      placeholder={aksiMeta.placeholder}
                      className="w-full rounded-lg border border-neutral-300 bg-white p-2.5 text-xs text-neutral-800 placeholder-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
                      autoFocus
                    />
                  </div>
                )}
              </fieldset>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

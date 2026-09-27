import { useEffect, useState } from 'react';
import {
  client,
  listPengajuanAgen,
  setujuiAgen,
  tolakAgen,
  verifikasiPembayaranAgen,
  tolakPembayaranAgen,
  uploadBuktiPembayaranAgen,
  uploadMedia,
  openProtectedMedia,
} from 'shared';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import DataTable from '../components/ui/DataTable';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(value));
const bayarMeta = {
  menunggu_verifikasi: { label: 'Menunggu verifikasi', variant: 'pending' },
  terverifikasi: { label: 'Terverifikasi', variant: 'success' },
  ditolak: { label: 'Ditolak', variant: 'danger' },
};

// Memuat file dari endpoint media terproteksi sebagai blob URL (foto agen).
function useProtectedImage(url) {
  const [src, setSrc] = useState(null);
  useEffect(() => {
    let alive = true;
    let objectUrl = null;
    setSrc(null);
    if (!url) return undefined;
    client.get(url, { responseType: 'blob' })
      .then((res) => {
        if (!alive) return;
        objectUrl = URL.createObjectURL(res.data);
        setSrc(objectUrl);
      })
      .catch(() => alive && setSrc(null));
    return () => {
      alive = false;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [url]);
  return src;
}

// Screen B1 — antrian pengajuan agen. Approve keagenan TIDAK PERNAH dikunci
// oleh status pembayaran (agen-azhan.md §5 poin 20).
export default function PersetujuanAgenPage() {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [mode, setMode] = useState(null); // null | 'tolak-agen' | 'tolak-bayar'
  const [alasan, setAlasan] = useState('');
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
    setMode(null);
    setAlasan('');
    setActionError('');
  };

  const close = () => {
    if (saving) return;
    setSelected(null);
  };

  // Jalankan aksi lalu muat ulang. Kalau pengajuan masih ada (mis. hanya
  // verifikasi pembayaran), modal tetap terbuka dengan data terbaru.
  const run = async (fn) => {
    setSaving(true);
    setActionError('');
    try {
      await fn();
      const data = await load();
      const fresh = data.find((it) => it.jamaah_id === selected?.jamaah_id);
      setSelected(fresh || null);
      setMode(null);
      setAlasan('');
    } catch (err) {
      setActionError(err.response?.data?.error || 'Aksi gagal diproses.');
    } finally {
      setSaving(false);
    }
  };

  const handleUploadAdmin = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file || !selected?.pembayaran) return;
    await run(async () => {
      const url = await uploadMedia(file, 'payment-proofs');
      await uploadBuktiPembayaranAgen(selected.pembayaran.id, url);
    });
  };

  const submitAlasan = () => {
    if (alasan.trim().length < 3) {
      setActionError('Alasan wajib diisi (minimal 3 karakter).');
      return;
    }
    if (mode === 'tolak-agen') run(() => tolakAgen(selected.jamaah_id, alasan.trim()));
    if (mode === 'tolak-bayar') run(() => tolakPembayaranAgen(selected.pembayaran.id, alasan.trim()));
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
          <p className="font-semibold text-neutral-900">{row.nama_lengkap}</p>
          <p className="mt-0.5 text-xs text-neutral-500 font-mono">{row.id_jamaah || '-'}{row.no_hp ? ` • ${row.no_hp}` : ''}</p>
          {row.status_agen !== 'pengajuan' && (
            <div className="mt-1"><Badge variant="success">Agen {row.status_agen} • tinggal pembayaran</Badge></div>
          )}
          {row.direkrut_oleh_nama && <p className="mt-0.5 text-xs text-neutral-500">Direkrut: {row.direkrut_oleh_nama}</p>}
        </div>
      );
    }
    if (key === 'diajukan_agen_at') return dateLabel(row.diajukan_agen_at);
    if (key === 'pembayaran') {
      const p = row.pembayaran;
      if (!p) return '-';
      if (Number(p.nominal_tagihan) === 0) return <span className="text-xs text-neutral-500">Tanpa biaya</span>;
      const meta = bayarMeta[p.status] || { label: p.status, variant: 'neutral' };
      return (
        <div>
          <p className="font-semibold text-neutral-900">{money(p.nominal_tagihan)}</p>
          <div className="mt-1"><Badge variant={meta.variant}>{meta.label}</Badge></div>
        </div>
      );
    }
    if (key === 'actions') {
      return (
        <button
          type="button"
          onClick={() => open(row)}
          className="rounded-full border border-neutral-200 bg-white hover:bg-neutral-50 px-3.5 py-1.5 text-xs font-semibold text-neutral-700 shadow-2xs hover:border-neutral-300 transition-colors cursor-pointer"
        >
          Tinjau
        </button>
      );
    }
    return row[key] ?? '-';
  };

  const p = selected?.pembayaran;
  const berbayar = p && Number(p.nominal_tagihan) > 0;
  const bayarLabel = p ? (bayarMeta[p.status] || { label: p.status, variant: 'neutral' }) : null;

  return (
    <div className="space-y-5">
      <PageHeader title="Persetujuan Agen" subtitle="Tinjau pengajuan agen Syiar dan verifikasi pembayaran pendaftarannya." />
      {error && <Alert variant="error" message={error} />}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        itemsPerPage={10}
        searchPlaceholder="Cari nama, ID jamaah, atau domisili..."
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
            <div className="flex flex-col sm:flex-row gap-2">
              {mode ? (
                <>
                  <Button variant="secondary" onClick={() => { setMode(null); setActionError(''); }} disabled={saving}>Batal</Button>
                  <Button variant="danger" onClick={submitAlasan} isLoading={saving}>
                    {mode === 'tolak-agen' ? 'Konfirmasi Tolak Keagenan' : 'Konfirmasi Tolak Pembayaran'}
                  </Button>
                </>
              ) : selected?.status_agen === 'pengajuan' ? (
                <>
                  <Button variant="danger-light" onClick={() => setMode('tolak-agen')} disabled={saving}>Tolak Keagenan</Button>
                  <Button variant="primary" onClick={() => run(() => setujuiAgen(selected.jamaah_id))} isLoading={saving}>
                    Setujui Agen
                  </Button>
                </>
              ) : (
                <span className="text-xs text-neutral-500 self-center">Keagenan sudah disetujui. Tinggal verifikasi pembayaran.</span>
              )}
            </div>
          </div>
        }
      >
        {selected && (
          <div className="space-y-4">
            {actionError && <Alert variant="error" message={actionError} />}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-3">
                <div className="p-3.5 rounded-xl border border-neutral-200/70 space-y-2 text-xs">
                  <div className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Nama</span>
                    <span className="font-semibold text-neutral-900 text-right">{selected.nama_lengkap}</span>
                  </div>
                  <div className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">ID Jamaah / HP</span>
                    <span className="font-mono text-neutral-800 text-right">{selected.id_jamaah || '-'} • {selected.no_hp || '-'}</span>
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
                    <span className="text-neutral-500">Direkrut oleh</span>
                    <span className="text-neutral-800 text-right">{selected.direkrut_oleh_nama || 'Tanpa agen (tidak ada upline)'}</span>
                  </div>
                </div>

                <div className="p-3.5 rounded-xl bg-neutral-50 border border-neutral-200/80 space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[10px] font-bold uppercase tracking-wider text-neutral-400">Pembayaran pendaftaran</span>
                    {berbayar && bayarLabel && <Badge variant={bayarLabel.variant}>{bayarLabel.label}</Badge>}
                  </div>
                  <p className="text-xl font-black text-neutral-900">{berbayar ? money(p.nominal_tagihan) : 'Tanpa biaya'}</p>
                  {p?.status === 'ditolak' && p.catatan_penolakan && (
                    <p className="text-xs text-danger-700 bg-danger-50 p-2 rounded-lg border border-danger-200/60">
                      <strong>Alasan ditolak:</strong> {p.catatan_penolakan}
                    </p>
                  )}
                  <p className="text-[11px] text-neutral-500">Status pembayaran tidak menghalangi persetujuan keagenan.</p>

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
                      {p.status !== 'terverifikasi' && (
                        <div className="flex flex-col sm:flex-row gap-2 pt-1">
                          <Button size="sm" variant="primary" onClick={() => run(() => verifikasiPembayaranAgen(p.id))} disabled={saving || Boolean(mode)}>
                            Tandai Terverifikasi
                          </Button>
                          {p.status === 'menunggu_verifikasi' && (
                            <Button size="sm" variant="danger-light" onClick={() => setMode('tolak-bayar')} disabled={saving || Boolean(mode)}>
                              Tolak Pembayaran
                            </Button>
                          )}
                          <label className={`inline-flex items-center justify-center rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-50 ${saving || mode ? 'opacity-50 pointer-events-none' : 'cursor-pointer'}`}>
                            Upload Bukti Transfer
                            <input type="file" accept="image/*,.pdf" className="hidden" onChange={handleUploadAdmin} />
                          </label>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex flex-col">
                <span className="text-xs font-bold text-neutral-700 mb-2 block">Foto diri (ID card)</span>
                <div className="flex-1 min-h-[240px] max-h-[340px] rounded-xl border border-neutral-200 bg-neutral-50 overflow-hidden flex items-center justify-center p-2">
                  {fotoSrc ? (
                    <img src={fotoSrc} alt={`Foto ${selected.nama_lengkap}`} className="max-h-[320px] w-auto max-w-full object-contain rounded-lg" />
                  ) : (
                    <p className="text-xs text-neutral-400">Memuat foto...</p>
                  )}
                </div>
              </div>
            </div>

            {mode && (
              <div className="p-3.5 rounded-xl bg-danger-50/70 border border-danger-200/80 space-y-2">
                <label htmlFor="alasan-tolak" className="block text-xs font-bold text-danger-900">
                  {mode === 'tolak-agen' ? 'Alasan penolakan keagenan' : 'Catatan penolakan pembayaran'} <span className="text-danger-600">*</span>
                </label>
                <textarea
                  id="alasan-tolak"
                  value={alasan}
                  onChange={(e) => setAlasan(e.target.value)}
                  rows={3}
                  placeholder={mode === 'tolak-agen' ? 'Mis. domisili di luar area layanan' : 'Mis. nominal transfer kurang Rp10.000 (dibaca pemohon di portal)'}
                  className="w-full rounded-lg border border-danger-300 bg-white p-2.5 text-xs text-neutral-800 placeholder-neutral-400 focus:border-danger-500 focus:outline-none focus:ring-1 focus:ring-danger-500"
                  autoFocus
                />
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

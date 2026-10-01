import { useEffect, useState } from 'react';
import {
  listPencairanAdmin,
  setujuiPencairan,
  tolakPencairan,
  uploadMedia,
  openProtectedMedia,
  listBrands,
} from 'shared';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';
import Textarea from '../components/ui/Textarea';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(value));
const statusMeta = {
  pending: { label: 'Menunggu', variant: 'pending' },
  disetujui: { label: 'Disetujui', variant: 'success' },
  ditolak: { label: 'Ditolak', variant: 'danger' },
};
const STATUS_OPTIONS = [
  { value: 'pending', label: 'Menunggu' },
  { value: 'disetujui', label: 'Disetujui' },
  { value: 'ditolak', label: 'Ditolak' },
  { value: 'semua', label: 'Semua status' },
];

const Row = ({ label, children }) => (
  <div className="flex justify-between gap-3 border-b border-neutral-100 pb-1.5 last:border-b-0 text-sm">
    <span className="text-neutral-500">{label}</span>
    <span className="text-neutral-900 text-right font-medium">{children}</span>
  </div>
);

// Screen C1 — Persetujuan Pencairan lintas brand (agen-azhan.md §5 poin 11).
// Menyetujui wajib mengunggah bukti transfer keluar; saldo dicek ulang di
// server. Pajak diproses di luar sistem (D7).
export default function KomisiReferralPage() {
  const [status, setStatus] = useState('pending');
  const [brandId, setBrandId] = useState('');
  const [brands, setBrands] = useState([]);
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [mode, setMode] = useState(null); // null | 'setujui' | 'tolak'
  const [bukti, setBukti] = useState(null);
  const [alasan, setAlasan] = useState('');
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');

  useEffect(() => {
    listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, []);

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      setItems(await listPencairanAdmin({ status: status === 'semua' ? '' : status, brandId }));
    } catch {
      setError('Pengajuan pencairan gagal dimuat. Silakan coba kembali.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, brandId]);

  const open = (row) => {
    setSelected(row);
    setMode(null);
    setBukti(null);
    setAlasan('');
    setActionError('');
  };

  const close = () => !saving && setSelected(null);

  const submit = async () => {
    setActionError('');
    if (mode === 'setujui' && !bukti) {
      setActionError('Unggah bukti transfer keluar terlebih dahulu.');
      return;
    }
    if (mode === 'tolak' && alasan.trim().length < 3) {
      setActionError('Catatan penolakan wajib diisi (minimal 3 karakter).');
      return;
    }
    setSaving(true);
    try {
      if (mode === 'setujui') {
        const url = await uploadMedia(bukti, 'payment-proofs');
        await setujuiPencairan(selected.id, url);
      } else {
        await tolakPencairan(selected.id, alasan.trim());
      }
      setSelected(null);
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Aksi gagal diproses.');
    } finally {
      setSaving(false);
    }
  };

  const lihatBukti = async (url) => {
    try {
      await openProtectedMedia(url);
    } catch {
      setActionError('Bukti transfer gagal dibuka.');
    }
  };

  const columns = [
    { header: 'Agen', key: 'nama_agen', sortable: true },
    { header: 'Nominal', key: 'nominal_diajukan', sortable: true },
    { header: 'Rekening Tujuan', key: 'rekening' },
    { header: 'Diajukan', key: 'diajukan_at', sortable: true },
    { header: 'Status', key: 'status', sortable: true },
    { header: 'Aksi', key: 'actions' },
  ];

  const renderCell = (row, key) => {
    if (key === 'nama_agen') {
      return (
        <div className="min-w-[150px]">
          <p className="font-semibold text-neutral-900">{row.nama_agen}</p>
          <p className="mt-0.5 text-xs text-neutral-500">{row.brand_name}{row.kode_referral ? ` • ${row.kode_referral}` : ''}</p>
        </div>
      );
    }
    if (key === 'nominal_diajukan') return <span className="font-semibold text-neutral-900">{money(row.nominal_diajukan)}</span>;
    if (key === 'rekening') {
      return (
        <div className="min-w-[150px] text-xs">
          <p className="font-semibold text-neutral-800">{row.rekening_bank} • {row.rekening_nomor}</p>
          <p className="text-neutral-500">a.n. {row.rekening_atas_nama}</p>
        </div>
      );
    }
    if (key === 'diajukan_at') return dateLabel(row.diajukan_at);
    if (key === 'status') {
      const meta = statusMeta[row.status] || { label: row.status, variant: 'neutral' };
      return <Badge variant={meta.variant}>{meta.label}</Badge>;
    }
    if (key === 'actions') {
      return (
        <button
          type="button"
          onClick={() => open(row)}
          className="rounded-full border border-neutral-200 bg-white hover:bg-neutral-50 px-3.5 py-1.5 text-xs font-semibold text-neutral-700 shadow-2xs transition-colors cursor-pointer"
        >
          {row.status === 'pending' ? 'Proses' : 'Detail'}
        </button>
      );
    }
    return row[key] ?? '-';
  };

  const saldoKurang = selected?.status === 'pending' && Number(selected.saldo_tersedia) < 0;

  const toolbarActions = (
    <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
      <CustomDropdown
        name="status"
        options={STATUS_OPTIONS}
        value={status}
        onChange={(v) => setStatus(v?.target ? v.target.value : v)}
        placeholder="Semua status"
        className="!mb-0 w-full sm:w-44"
      />
      <CustomDropdown
        name="brand_id"
        options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
        value={brandId}
        onChange={(v) => setBrandId(v?.target ? v.target.value : v)}
        placeholder="Semua brand"
        className="!mb-0 w-full sm:w-44"
      />
    </div>
  );
  return (
    <div className="space-y-5">
      <PageHeader title="Persetujuan Pencairan" subtitle="Pengajuan tarik saldo agen Syiar dari seluruh brand." />
      {error && <Alert variant="error" message={error} />}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        itemsPerPage={15}
        toolbarActions={toolbarActions}
        searchPlaceholder="Cari agen, brand, atau rekening..."
        emptyMessage={loading ? 'Memuat pengajuan...' : 'Tidak ada pengajuan pencairan'}
      />

      <Modal
        isOpen={Boolean(selected)}
        onClose={close}
        title="Pengajuan Pencairan"
        footer={
          <div className="flex flex-col-reverse sm:flex-row sm:justify-between gap-2 w-full">
            <Button variant="secondary" onClick={close} disabled={saving}>Tutup</Button>
            {selected?.status === 'pending' && (
              <div className="flex flex-col sm:flex-row gap-2">
                {mode ? (
                  <>
                    <Button variant="secondary" onClick={() => { setMode(null); setActionError(''); }} disabled={saving}>Batal</Button>
                    <Button variant={mode === 'tolak' ? 'danger' : 'primary'} onClick={submit} isLoading={saving}>
                      {mode === 'tolak' ? 'Konfirmasi Tolak' : 'Konfirmasi Setujui'}
                    </Button>
                  </>
                ) : (
                  <>
                    <Button variant="danger-light" onClick={() => setMode('tolak')}>Tolak</Button>
                    <Button variant="primary" onClick={() => setMode('setujui')} disabled={saldoKurang}>Setujui & Unggah Bukti</Button>
                  </>
                )}
              </div>
            )}
          </div>
        }
      >
        {selected && (
          <div className="space-y-4">
            {actionError && <Alert variant="error" message={actionError} />}
            <div className="p-3.5 rounded-xl border border-neutral-200/70 space-y-2">
              <Row label="Agen">{selected.nama_agen}</Row>
              <Row label="Brand">{selected.brand_name}</Row>
              <Row label="Nominal">{money(selected.nominal_diajukan)}</Row>
              <Row label="Bank">{selected.rekening_bank}</Row>
              <Row label="Nomor rekening"><span className="font-mono">{selected.rekening_nomor}</span></Row>
              <Row label="Atas nama">{selected.rekening_atas_nama}</Row>
              <Row label="Diajukan">{dateLabel(selected.diajukan_at)}</Row>
              {selected.status !== 'pending' && (
                <>
                  <Row label="Diproses">{dateLabel(selected.diproses_at)}</Row>
                  <Row label="Diproses oleh">{selected.diproses_oleh_nama || '-'}</Row>
                </>
              )}
            </div>

            {selected.status === 'pending' && (
              <div className={`p-3 rounded-xl border text-sm ${saldoKurang ? 'border-danger-200 bg-danger-50 text-danger-700' : 'border-neutral-200 bg-neutral-50 text-neutral-700'}`}>
                Saldo tersedia setelah pengajuan ini: <span className="font-bold">{money(selected.saldo_tersedia)}</span>
                {saldoKurang && <p className="mt-1 text-xs">Saldo tidak lagi mencukupi. Tolak pengajuan ini.</p>}
              </div>
            )}

            {selected.status === 'ditolak' && selected.catatan_penolakan && (
              <Alert variant="error" message={`Alasan ditolak: ${selected.catatan_penolakan}`} />
            )}

            {selected.status === 'disetujui' && selected.bukti_transfer_keluar_url && (
              <Button variant="secondary" onClick={() => lihatBukti(selected.bukti_transfer_keluar_url)}>Lihat Bukti Transfer</Button>
            )}

            {mode === 'setujui' && (
              <div className="space-y-1.5">
                <label className="text-sm font-semibold text-neutral-900 font-heading">Bukti transfer keluar *</label>
                <input
                  type="file"
                  accept="image/jpeg,image/png,image/webp,application/pdf"
                  onChange={(e) => setBukti(e.target.files?.[0] || null)}
                  className="block w-full text-sm text-neutral-700 file:mr-3 file:rounded-lg file:border-0 file:bg-neutral-100 file:px-3 file:py-2 file:text-sm file:font-semibold file:text-neutral-800 hover:file:bg-neutral-200"
                />
                <p className="text-xs text-neutral-500">Transfer dulu ke rekening di atas, lalu unggah buktinya. Pajak (bila ada) dicatat manual oleh keuangan.</p>
              </div>
            )}

            {mode === 'tolak' && (
              <Textarea
                label="Catatan penolakan"
                value={alasan}
                onChange={(e) => setAlasan(e.target.value)}
                rows={3}
                placeholder="Contoh: nama pemilik rekening tidak sesuai"
              />
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

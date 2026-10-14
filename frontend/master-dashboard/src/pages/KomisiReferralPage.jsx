import { useEffect, useMemo, useState } from 'react';
import { Clock3, CheckCircle2, XCircle, Banknote, Eye } from 'lucide-react';
import {
  listPencairanAdmin,
  setujuiPencairan,
  tolakPencairan,
  uploadMedia,
  openProtectedMedia,
  listBrands,
} from 'shared';
import Alert from 'shared/ui/Alert';
import Button from 'shared/ui/Button';
import CustomDropdown from 'shared/ui/CustomDropdown';
import DataTable from 'shared/ui/DataTable';
import Modal from 'shared/ui/Modal';
import PageHeader from 'shared/ui/PageHeader';
import StatTile from 'shared/ui/StatTile';
import ActionMenu from 'shared/ui/ActionMenu';
import Textarea from 'shared/ui/Textarea';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(value));
const statusMeta = {
  pending: { label: 'Menunggu', chip: 'bg-warning-50 text-warning-700 border-warning-200', dot: 'bg-warning-500' },
  disetujui: { label: 'Disetujui', chip: 'bg-success-50 text-success-700 border-success-200', dot: 'bg-success-500' },
  ditolak: { label: 'Ditolak', chip: 'bg-danger-50 text-danger-700 border-danger-200', dot: 'bg-danger-500' },
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
  const [allItems, setAllItems] = useState([]);
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
      setAllItems((await listPencairanAdmin({ status: '', brandId })) || []);
    } catch {
      setError('Pengajuan pencairan gagal dimuat. Silakan coba kembali.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [brandId]);

  // Ringkasan memakai semua status (mengikuti filter brand); tabel disaring menurut status.
  const items = useMemo(
    () => allItems.filter((row) => status === 'semua' || row.status === status),
    [allItems, status]
  );
  const stats = useMemo(() => {
    const by = (st) => allItems.filter((row) => row.status === st);
    const sum = (list) => list.reduce((t, row) => t + (Number(row.nominal_diajukan) || 0), 0);
    return {
      pending: by('pending').length, pendingNominal: sum(by('pending')),
      disetujui: by('disetujui').length, dicairkan: sum(by('disetujui')),
      ditolak: by('ditolak').length,
    };
  }, [allItems]);

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
    { header: 'Nominal', key: 'nominal_diajukan', sortable: true, sortFn: (a, b) => Number(a.nominal_diajukan) - Number(b.nominal_diajukan) },
    { header: 'Rekening Tujuan', key: 'rekening' },
    { header: 'Diajukan', key: 'diajukan_at', sortable: true },
    { header: 'Status', key: 'status', sortable: true },
    { header: 'Aksi', key: 'actions' },
  ];

  const renderCell = (row, key) => {
    if (key === 'nama_agen') {
      return (
        <div className="min-w-[150px]">
          <button
            type="button"
            onClick={() => open(row)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
          >
            {row.nama_agen}
          </button>
          <p className="text-[11px] font-body text-neutral-400">
            {row.brand_name}{row.kode_referral ? ' · ' : ''}{row.kode_referral && <span className="font-mono">{row.kode_referral}</span>}
          </p>
        </div>
      );
    }
    if (key === 'nominal_diajukan') return <span className="text-[13px] font-body font-semibold text-neutral-900 whitespace-nowrap">{money(row.nominal_diajukan)}</span>;
    if (key === 'rekening') {
      return (
        <div className="min-w-[150px]">
          <p className="text-[13px] font-body font-medium text-neutral-800">{row.rekening_bank} · <span className="font-mono">{row.rekening_nomor}</span></p>
          <p className="text-[11px] font-body text-neutral-400">a.n. {row.rekening_atas_nama}</p>
        </div>
      );
    }
    if (key === 'diajukan_at') return <span className="text-[13px] font-body text-neutral-500 whitespace-nowrap">{dateLabel(row.diajukan_at)}</span>;
    if (key === 'status') {
      const meta = statusMeta[row.status] || { label: row.status, chip: 'bg-neutral-50 text-neutral-600 border-neutral-200', dot: 'bg-neutral-400' };
      return (
        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium font-body border ${meta.chip}`}>
          <span className={`w-1.5 h-1.5 rounded-full ${meta.dot}`} />
          {meta.label}
        </span>
      );
    }
    if (key === 'actions') {
      return (
        <ActionMenu
          align="right"
          items={[{
            label: row.status === 'pending' ? 'Proses' : 'Lihat Detail',
            icon: Eye,
            tooltip: row.status === 'pending' ? 'Setujui atau tolak pengajuan ini' : 'Lihat detail pengajuan',
            onClick: () => open(row),
          }]}
        />
      );
    }
    return row[key] ?? '-';
  };

  const saldoKurang = selected?.status === 'pending' && Number(selected.saldo_tersedia) < 0;

  const toolbarActions = (
    <div className="flex flex-wrap items-center gap-2">
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
    <div className="space-y-6">
      <PageHeader title="Persetujuan Pencairan" />
      {error && <Alert variant="error" message={error} />}

      {!loading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Clock3} variant="warning" value={stats.pending} label="Menunggu" hint={money(stats.pendingNominal)} />
          <StatTile icon={CheckCircle2} variant="success" value={stats.disetujui} label="Disetujui" />
          <StatTile icon={XCircle} variant="danger" value={stats.ditolak} label="Ditolak" />
          <StatTile icon={Banknote} variant="primary" value={money(stats.dicairkan)} label="Total dicairkan" />
        </div>
      )}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        onRowClick={open}
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

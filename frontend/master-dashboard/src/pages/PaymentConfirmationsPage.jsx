import { useEffect, useMemo, useState } from 'react';
import { listPayments, verifyPayment } from '../api/payments';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import DataTable from '../components/ui/DataTable';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(`${String(value).slice(0, 10)}T00:00:00`));
const statusMeta = { pending: ['Menunggu', 'pending'], confirmed: ['Terkonfirmasi', 'success'], rejected: ['Ditolak', 'danger'] };
const controlClass = 'mt-1.5 h-11 w-full rounded-md border border-neutral-300 bg-white px-3 text-sm text-neutral-800 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500';

export default function PaymentConfirmationsPage() {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filters, setFilters] = useState({ status: '', brand: '', bank: '', from: '', to: '' });
  const [dialog, setDialog] = useState(null);
  const [reason, setReason] = useState('');
  const [saving, setSaving] = useState(false);

  const load = async () => {
    setLoading(true); setError('');
    try { setItems((await listPayments()) || []); }
    catch { setError('Pembayaran gagal dimuat. Silakan coba kembali.'); }
    finally { setLoading(false); }
  };
  useEffect(() => { load(); }, []);

  const options = useMemo(() => ({
    brands: [...new Set(items.map((item) => item.brand_name).filter(Boolean))].sort(),
    banks: [...new Set(items.map((item) => item.destination_bank_name).filter(Boolean))].sort(),
  }), [items]);
  const filteredItems = useMemo(() => items.filter((item) => {
    const date = String(item.tanggal || '').slice(0, 10);
    return (!filters.status || item.status === filters.status)
      && (!filters.brand || item.brand_name === filters.brand)
      && (!filters.bank || item.destination_bank_name === filters.bank)
      && (!filters.from || date >= filters.from)
      && (!filters.to || date <= filters.to);
  }).map((item) => ({
    ...item,
    invoice: item.booking_id_booking || item.id_booking || `ID: ${item.booking_id}`,
    search_detail: [item.brand_name, item.jamaah_name, item.schedule_name, item.sender_name, item.sender_bank, item.destination_bank_name, item.destination_account_number].filter(Boolean).join(' '),
  })), [items, filters]);

  const resetFilters = () => setFilters({ status: '', brand: '', bank: '', from: '', to: '' });
  const openDialog = (payment, action) => { setError(''); setReason(''); setDialog({ payment, action }); };
  const submit = async () => {
    if (!dialog) return;
    if (dialog.action === 'rejected' && !reason.trim()) { setError('Alasan penolakan wajib diisi agar dapat dibaca jamaah.'); return; }
    setSaving(true); setError('');
    try { await verifyPayment(dialog.payment.id, dialog.action, dialog.action === 'rejected' ? reason.trim() : null); setDialog(null); await load(); }
    catch (err) { setError(err.response?.data?.error || 'Status pembayaran gagal diperbarui.'); }
    finally { setSaving(false); }
  };

  const hasFilters = Object.values(filters).some(Boolean);
  const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');
  const proofUrl = (url) => /^https?:\/\//i.test(url) ? url : `${apiBase}${url.startsWith('/') ? '' : '/'}${url}`;
  const columns = [
    { header: 'Brand & Invoice', key: 'brand_name', sortable: true },
    { header: 'Jamaah & Paket', key: 'jamaah_name', sortable: true },
    { header: 'Transfer', key: 'tanggal', sortable: true },
    { header: 'Rekening Tujuan', key: 'destination_bank_name', sortable: true },
    { header: 'Nominal', key: 'jumlah', sortable: true, sortFn: (a, b) => Number(a.jumlah) - Number(b.jumlah) },
    { header: 'Status', key: 'status', sortable: true },
    { header: 'Aksi', key: 'actions' },
  ];
  const renderCell = (row, key) => {
    if (key === 'brand_name') return <div className="min-w-[150px]"><p className="font-semibold text-neutral-900">{row.brand_name || '-'}</p><p className="mt-0.5 text-xs text-neutral-500">{row.invoice}</p></div>;
    if (key === 'jamaah_name') return <div className="max-w-[220px] whitespace-normal"><p className="font-semibold text-neutral-900">{row.jamaah_name || '-'}</p><p className="mt-0.5 text-xs text-neutral-500">{row.schedule_name || '-'}</p></div>;
    if (key === 'tanggal') return <div><p className="font-medium text-neutral-800">{dateLabel(row.tanggal)}</p><p className="mt-0.5 text-xs text-neutral-500">{row.sender_name || 'Pengirim belum diisi'}{row.sender_bank ? ` · ${row.sender_bank}` : ''}</p></div>;
    if (key === 'destination_bank_name') return <div><p className="font-medium text-neutral-800">{row.destination_bank_name || '-'}</p><p className="mt-0.5 text-xs text-neutral-500">{row.destination_account_number || '-'}{row.destination_account_holder ? ` · ${row.destination_account_holder}` : ''}</p></div>;
    if (key === 'jumlah') return <span className="font-semibold text-neutral-900">{money(row.jumlah)}</span>;
    if (key === 'status') { const meta = statusMeta[row.status] || [row.status || '-', 'neutral']; return <div className="max-w-[170px] whitespace-normal"><Badge variant={meta[1]}>{meta[0]}</Badge>{row.rejection_reason && <p className="mt-1.5 text-xs text-danger-700">{row.rejection_reason}</p>}</div>; }
    if (key === 'actions') return <div className="flex items-center gap-2">{row.bukti_url && <a href={proofUrl(row.bukti_url)} target="_blank" rel="noreferrer" className="rounded-full border border-neutral-200 px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-50">Bukti</a>}{row.status === 'pending' && <><button onClick={() => openDialog(row, 'confirmed')} className="rounded-full bg-success-50 px-3 py-1.5 text-xs font-semibold text-success-700 hover:bg-success-100">Konfirmasi</button><button onClick={() => openDialog(row, 'rejected')} className="rounded-full bg-danger-50 px-3 py-1.5 text-xs font-semibold text-danger-700 hover:bg-danger-100">Tolak</button></>}</div>;
    return row[key] ?? '-';
  };

  return <div className="space-y-5">
    <PageHeader title="Konfirmasi Pembayaran" subtitle="Cari dan verifikasi transfer jamaah dari seluruh brand." />
    {error && <Alert variant="error">{error}</Alert>}
    <section className="rounded-lg border border-neutral-200 bg-white p-4">
      <div className="mb-3 flex items-center justify-between gap-3"><div><h2 className="font-semibold text-neutral-900">Filter pembayaran</h2><p className="text-xs text-neutral-500">Persempit data berdasarkan brand, status, rekening, atau tanggal transfer.</p></div>{hasFilters && <button onClick={resetFilters} className="text-sm font-semibold text-primary-600 hover:underline">Reset filter</button>}</div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <label className="text-xs font-medium text-neutral-600">Status<select value={filters.status} onChange={(e) => setFilters((prev) => ({ ...prev, status: e.target.value }))} className={controlClass}><option value="">Semua status</option><option value="pending">Menunggu</option><option value="confirmed">Terkonfirmasi</option><option value="rejected">Ditolak</option></select></label>
        <label className="text-xs font-medium text-neutral-600">Brand<select value={filters.brand} onChange={(e) => setFilters((prev) => ({ ...prev, brand: e.target.value }))} className={controlClass}><option value="">Semua brand</option>{options.brands.map((brand) => <option key={brand} value={brand}>{brand}</option>)}</select></label>
        <label className="text-xs font-medium text-neutral-600">Rekening tujuan<select value={filters.bank} onChange={(e) => setFilters((prev) => ({ ...prev, bank: e.target.value }))} className={controlClass}><option value="">Semua rekening</option>{options.banks.map((bank) => <option key={bank} value={bank}>{bank}</option>)}</select></label>
        <label className="text-xs font-medium text-neutral-600">Dari tanggal<input type="date" value={filters.from} onChange={(e) => setFilters((prev) => ({ ...prev, from: e.target.value }))} className={controlClass} /></label>
        <label className="text-xs font-medium text-neutral-600">Sampai tanggal<input type="date" value={filters.to} min={filters.from || undefined} onChange={(e) => setFilters((prev) => ({ ...prev, to: e.target.value }))} className={controlClass} /></label>
      </div>
    </section>
    <DataTable columns={columns} data={filteredItems} renderCell={renderCell} itemsPerPage={10} searchPlaceholder="Cari brand, invoice, jamaah, paket, pengirim, atau rekening..." emptyMessage={loading ? 'Memuat pembayaran...' : hasFilters ? 'Tidak ada pembayaran yang sesuai filter' : 'Belum ada pembayaran'} />
    <Modal isOpen={Boolean(dialog)} onClose={() => !saving && setDialog(null)} title={dialog?.action === 'confirmed' ? 'Konfirmasi pembayaran' : 'Tolak pembayaran'} footer={<><Button variant="secondary" onClick={() => setDialog(null)} disabled={saving}>Batal</Button><Button variant={dialog?.action === 'confirmed' ? 'primary' : 'danger'} onClick={submit} isLoading={saving}>{dialog?.action === 'confirmed' ? 'Ya, konfirmasi' : 'Tolak pembayaran'}</Button></>}>
      {dialog && <div className="space-y-4"><div className="rounded-lg bg-neutral-50 p-4"><p className="text-sm font-semibold text-neutral-900">{dialog.payment.brand_name} · {dialog.payment.jamaah_name}</p><p className="mt-1 text-sm text-neutral-600">{dialog.payment.invoice} · {money(dialog.payment.jumlah)}</p></div>{dialog.action === 'confirmed' ? <p className="text-sm leading-6 text-neutral-600">Pembayaran akan tercatat sebagai terkonfirmasi dan mengurangi sisa tagihan jamaah.</p> : <label className="block text-sm font-medium text-neutral-700">Alasan penolakan <span className="text-danger-600">*</span><textarea value={reason} onChange={(e) => setReason(e.target.value)} rows={4} placeholder="Contoh: nominal atau bukti transfer tidak sesuai" className="mt-2 w-full rounded-md border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500" /></label>}</div>}
    </Modal>
  </div>;
}

import { useEffect, useMemo, useState } from 'react';
import { SlidersHorizontal } from 'lucide-react';
import client from '../api/client';
import { listAllPayments, verifyPayment } from '../api/bookings';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import Input from '../components/ui/Input';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(`${String(value).slice(0, 10)}T00:00:00`));
const statusMeta = {
  pending: { label: 'Menunggu', variant: 'pending' },
  confirmed: { label: 'Terkonfirmasi', variant: 'success' },
  rejected: { label: 'Ditolak', variant: 'danger' },
};
const STATUS_OPTIONS = [
  { value: '', label: 'Semua status' },
  { value: 'pending', label: 'Menunggu' },
  { value: 'confirmed', label: 'Terkonfirmasi' },
  { value: 'rejected', label: 'Ditolak' },
];
const EMPTY_FILTERS = { status: '', brand: '', bank: '', from: '', to: '' };
// CustomDropdown mengirim nilai langsung atau objek event.
const valueOf = (v) => (v?.target ? v.target.value : v);

// Konfirmasi pembayaran manual (Master & Travel). showBrandColumn: Admin Master,
// lintas brand dengan filter brand. Filter status selalu tampil di toolbar tabel;
// brand, rekening, dan rentang tanggal ada di panel "Filter" agar tidak menumpuk.
export default function PaymentConfirmationsPage({ showBrandColumn = false }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [panelOpen, setPanelOpen] = useState(false);

  // Modal Detail & Verifikasi State
  const [selectedPayment, setSelectedPayment] = useState(null);
  const [proofBlobUrl, setProofBlobUrl] = useState(null);
  const [proofLoading, setProofLoading] = useState(false);
  const [rejectMode, setRejectMode] = useState(false);
  const [rejectionReason, setRejectionReason] = useState('');
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      setItems((await listAllPayments()) || []);
    } catch {
      setError('Pembayaran gagal dimuat. Silakan coba kembali.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  // Fetch Protected Media Blob ketika selectedPayment berubah
  useEffect(() => {
    let isMounted = true;
    let objectUrl = null;

    if (selectedPayment?.bukti_url) {
      setProofLoading(true);
      const normalizedUrl = String(selectedPayment.bukti_url || '').replace(
        /^\/uploads\/(dokumen-jamaah|payment-proofs)\/(.+)$/,
        '/api/admin/media/$1/$2',
      );

      client
        .get(normalizedUrl, { responseType: 'blob' })
        .then((res) => {
          if (!isMounted) return;
          objectUrl = URL.createObjectURL(res.data);
          setProofBlobUrl(objectUrl);
        })
        .catch((err) => {
          console.error('Gagal memuat gambar bukti transfer:', err);
          if (isMounted) setProofBlobUrl(null);
        })
        .finally(() => {
          if (isMounted) setProofLoading(false);
        });
    } else {
      setProofBlobUrl(null);
      setProofLoading(false);
    }

    return () => {
      isMounted = false;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [selectedPayment]);

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
    search_detail: [
      showBrandColumn ? item.brand_name : null,
      item.jamaah_name,
      item.schedule_name,
      item.sender_name,
      item.sender_bank,
      item.destination_bank_name,
      item.destination_account_number,
    ].filter(Boolean).join(' '),
  })), [items, filters, showBrandColumn]);

  const setFilter = (key, value) => setFilters((prev) => ({ ...prev, [key]: value }));
  const resetFilters = () => setFilters(EMPTY_FILTERS);
  const hasFilters = Object.values(filters).some(Boolean);
  // Jumlah filter aktif di panel (status tampil terpisah di toolbar).
  const panelActive = ['brand', 'bank', 'from', 'to'].filter((k) => filters[k]).length;

  const openDetail = (payment) => {
    setActionError('');
    setRejectMode(false);
    setRejectionReason('');
    setSelectedPayment(payment);
  };

  const closeDetail = () => {
    if (saving) return;
    setSelectedPayment(null);
    setProofBlobUrl(null);
    setRejectMode(false);
    setRejectionReason('');
    setActionError('');
  };

  const handleConfirmPayment = async () => {
    if (!selectedPayment) return;
    setSaving(true);
    setActionError('');
    try {
      await verifyPayment(selectedPayment.id, 'confirmed', null);
      closeDetail();
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Status pembayaran gagal diperbarui.');
    } finally {
      setSaving(false);
    }
  };

  const handleRejectPayment = async () => {
    if (!selectedPayment) return;
    if (!rejectionReason.trim()) {
      setActionError('Alasan penolakan wajib diisi agar dapat dibaca jamaah.');
      return;
    }
    setSaving(true);
    setActionError('');
    try {
      await verifyPayment(selectedPayment.id, 'rejected', rejectionReason.trim());
      closeDetail();
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Status pembayaran gagal diperbarui.');
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    { header: 'Invoice & Jamaah', key: 'jamaah_name', sortable: true },
    { header: 'Paket', key: 'schedule_name', sortable: true },
    { header: 'Transfer', key: 'tanggal', sortable: true },
    { header: 'Rekening Tujuan', key: 'destination_bank_name', sortable: true },
    { header: 'Nominal', key: 'jumlah', sortable: true, sortFn: (a, b) => Number(a.jumlah) - Number(b.jumlah) },
    { header: 'Status', key: 'status', sortable: true },
    { header: 'Aksi', key: 'actions' },
  ];

  const renderCell = (row, key) => {
    if (key === 'jamaah_name') {
      return (
        <div className="min-w-40">
          <p className="font-semibold text-neutral-900">{row.jamaah_name || '-'}</p>
          {showBrandColumn && <p className="text-xs text-neutral-500">{row.brand_name || '-'}</p>}
          {row.booking_id ? (
            <a
              href={`/bookings/${row.booking_id}`}
              target="_blank"
              rel="noreferrer"
              className="mt-1 inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-neutral-100 hover:bg-neutral-200 border border-neutral-200/80 text-neutral-800 hover:text-neutral-950 font-mono text-xs font-bold transition-all cursor-pointer group"
              title="Buka Detail Booking di tab baru"
            >
              <span>{row.invoice}</span>
              <svg className="w-3 h-3 text-neutral-400 group-hover:text-neutral-700 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                <path strokeLinecap="round" strokeLinejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
              </svg>
            </a>
          ) : (
            <p className="mt-0.5 text-xs text-neutral-500 font-mono">{row.invoice}</p>
          )}
        </div>
      );
    }
    if (key === 'schedule_name') {
      return (
        <div className="min-w-36 max-w-56 whitespace-normal">
          <p className="font-medium text-neutral-700 leading-tight">{row.schedule_name || '-'}</p>
          {row.departure_date && <p className="mt-0.5 text-xs text-neutral-400">{dateLabel(row.departure_date)}</p>}
        </div>
      );
    }
    if (key === 'tanggal') {
      return (
        <div className="min-w-28">
          <p className="font-medium text-neutral-800 whitespace-nowrap">{dateLabel(row.tanggal)}</p>
          <p className="mt-0.5 text-xs text-neutral-500">
            {row.sender_name || 'Pengirim belum diisi'}
            {row.sender_bank ? ` • ${row.sender_bank}` : ''}
          </p>
        </div>
      );
    }
    if (key === 'destination_bank_name') {
      return (
        <div className="min-w-28">
          <p className="font-medium text-neutral-800">{row.destination_bank_name || '-'}</p>
          <p className="mt-0.5 text-xs text-neutral-500">
            {row.destination_account_number || '-'}
            {row.destination_account_holder ? ` • ${row.destination_account_holder}` : ''}
          </p>
        </div>
      );
    }
    if (key === 'jumlah') {
      return <span className="font-semibold text-neutral-900 whitespace-nowrap">{money(row.jumlah)}</span>;
    }
    if (key === 'status') {
      const meta = statusMeta[row.status] || { label: row.status || '-', variant: 'neutral' };
      return (
        <div className="max-w-44 whitespace-normal">
          <Badge variant={meta.variant}>{meta.label}</Badge>
          {row.rejection_reason && <p className="mt-1.5 text-xs text-danger-700 leading-tight">{row.rejection_reason}</p>}
        </div>
      );
    }
    if (key === 'actions') {
      return (
        <button
          type="button"
          onClick={() => openDetail(row)}
          className="whitespace-nowrap rounded-full border border-neutral-200 bg-white hover:bg-neutral-50 px-3.5 py-1.5 text-xs font-semibold text-neutral-700 shadow-2xs hover:border-neutral-300 transition-colors cursor-pointer"
        >
          Lihat Detail
        </button>
      );
    }
    return row[key] ?? '-';
  };

  const selectedMeta = selectedPayment ? (statusMeta[selectedPayment.status] || { label: selectedPayment.status || '-', variant: 'neutral' }) : null;

  const toolbarActions = (
    <div className="flex w-full items-center gap-2 sm:w-auto">
      <CustomDropdown
        value={filters.status}
        onChange={(v) => setFilter('status', valueOf(v))}
        options={STATUS_OPTIONS}
        placeholder="Semua status"
        className="!mb-0 flex-1 sm:w-44 sm:flex-none"
      />
      <Button
        type="button"
        variant={panelOpen || panelActive ? 'dark' : 'secondary'}
        onClick={() => setPanelOpen((open) => !open)}
        className="h-11 shrink-0 whitespace-nowrap"
        aria-expanded={panelOpen}
      >
        <SlidersHorizontal className="w-4 h-4" />
        <span>Filter{panelActive ? ` (${panelActive})` : ''}</span>
      </Button>
    </div>
  );

  const toolbarPanel = panelOpen && (
    <div className="space-y-3">
      <div className={`grid grid-cols-1 gap-3 sm:grid-cols-2 ${showBrandColumn ? 'lg:grid-cols-4' : 'lg:grid-cols-3'}`}>
        {showBrandColumn && (
          <CustomDropdown
            label="Brand"
            value={filters.brand}
            onChange={(v) => setFilter('brand', valueOf(v))}
            options={[{ value: '', label: 'Semua brand' }, ...options.brands.map((b) => ({ value: b, label: b }))]}
            placeholder="Semua brand"
          />
        )}
        <CustomDropdown
          label="Rekening tujuan"
          value={filters.bank}
          onChange={(v) => setFilter('bank', valueOf(v))}
          options={[{ value: '', label: 'Semua rekening' }, ...options.banks.map((b) => ({ value: b, label: b }))]}
          placeholder="Semua rekening"
        />
        <Input label="Transfer dari" type="date" name="dari" value={filters.from} onChange={(e) => setFilter('from', e.target.value)} />
        <Input label="Transfer sampai" type="date" name="sampai" value={filters.to} min={filters.from || undefined} onChange={(e) => setFilter('to', e.target.value)} />
      </div>
      {hasFilters && (
        <div className="flex justify-end">
          <button type="button" onClick={resetFilters} className="text-sm font-semibold text-neutral-900 underline underline-offset-2">
            Reset semua filter
          </button>
        </div>
      )}
    </div>
  );

  return (
    <div className="space-y-5">
      <PageHeader
        title={showBrandColumn ? 'Konfirmasi Pembayaran' : 'Pembayaran'}
        subtitle={showBrandColumn ? 'Cari dan verifikasi transfer jamaah dari seluruh brand.' : 'Cari dan verifikasi transfer manual jamaah dalam satu tabel.'}
      />

      {error && <Alert variant="error" message={error} />}

      <DataTable
        columns={columns}
        data={filteredItems}
        renderCell={renderCell}
        itemsPerPage={10}
        searchPlaceholder={showBrandColumn ? 'Cari brand, invoice, jamaah, paket, pengirim...' : 'Cari invoice, jamaah, paket, pengirim...'}
        emptyMessage={loading ? 'Memuat pembayaran...' : hasFilters ? 'Tidak ada pembayaran yang sesuai filter' : 'Belum ada pembayaran'}
        toolbarActions={toolbarActions}
        toolbarPanel={toolbarPanel}
      />

      {/* Modal Detail Pembayaran & Verifikasi */}
      <Modal
        isOpen={Boolean(selectedPayment)}
        onClose={closeDetail}
        title="Detail Pembayaran"
        size="2xl"
        footer={
          <div className="flex items-center justify-between w-full">
            <div>
              {rejectMode && (
                <Button
                  variant="secondary"
                  onClick={() => {
                    setRejectMode(false);
                    setActionError('');
                  }}
                  disabled={saving}
                >
                  Batal Tolak
                </Button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Button variant="secondary" onClick={closeDetail} disabled={saving}>
                Tutup
              </Button>

              {selectedPayment?.status === 'pending' && !rejectMode && (
                <>
                  <Button
                    variant="danger-light"
                    onClick={() => {
                      setRejectMode(true);
                      setActionError('');
                    }}
                    disabled={saving}
                  >
                    Tolak
                  </Button>
                  <Button
                    variant="primary"
                    className="!bg-emerald-600 hover:!bg-emerald-700 !text-white !border-emerald-600 shadow-xs"
                    onClick={handleConfirmPayment}
                    isLoading={saving}
                  >
                    Pembayaran Diterima
                  </Button>
                </>
              )}

              {selectedPayment?.status === 'pending' && rejectMode && (
                <Button variant="danger" onClick={handleRejectPayment} isLoading={saving}>
                  Konfirmasi Tolak Pembayaran
                </Button>
              )}
            </div>
          </div>
        }
      >
        {selectedPayment && (
          <div className="space-y-4">
            {actionError && <Alert variant="error" message={actionError} />}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Kolom Kiri: Info Ringkas Pembayaran */}
              <div className="space-y-3">
                <div className="p-3.5 rounded-xl bg-neutral-50 border border-neutral-200/80">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-bold uppercase tracking-wider text-neutral-400">Nominal Pembayaran</span>
                    {selectedMeta && <Badge variant={selectedMeta.variant}>{selectedMeta.label}</Badge>}
                  </div>
                  <div className="text-2xl font-black text-neutral-900 tracking-tight mt-1">{money(selectedPayment.jumlah)}</div>
                  {selectedPayment.status === 'rejected' && selectedPayment.rejection_reason && (
                    <p className="mt-1.5 text-xs text-danger-700 bg-danger-50 p-2 rounded-lg border border-danger-200/60">
                      <strong>Alasan ditolak:</strong> {selectedPayment.rejection_reason}
                    </p>
                  )}
                </div>

                <div className="p-3.5 rounded-xl border border-neutral-200/70 space-y-2 text-xs">
                  {showBrandColumn && (
                    <div className="flex justify-between border-b border-neutral-100 pb-1.5">
                      <span className="text-neutral-500">Brand</span>
                      <span className="font-semibold text-neutral-800">{selectedPayment.brand_name || '-'}</span>
                    </div>
                  )}
                  <div className="flex justify-between items-center border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Kode Booking</span>
                    <div className="flex items-center gap-2">
                      <span className="font-mono font-bold text-neutral-900">#{selectedPayment.invoice}</span>
                      {selectedPayment.booking_id && (
                        <a
                          href={`/bookings/${selectedPayment.booking_id}`}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-neutral-100 hover:bg-neutral-200 text-neutral-700 hover:text-neutral-900 text-xs font-medium transition-colors cursor-pointer"
                          title="Buka Detail Booking di tab baru"
                        >
                          <span>Buka Booking</span>
                          <svg className="w-3 h-3 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                          </svg>
                        </a>
                      )}
                    </div>
                  </div>
                  <div className="flex justify-between border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Nama Jamaah</span>
                    <span className="font-semibold text-neutral-900">{selectedPayment.jamaah_name || '-'}</span>
                  </div>
                  <div className="flex justify-between items-start gap-3">
                    <span className="text-neutral-500 shrink-0">Paket</span>
                    <div className="text-right">
                      <span className="font-medium text-neutral-800 block leading-tight">{selectedPayment.schedule_name || '-'}</span>
                      {selectedPayment.departure_date && (
                        <span className="text-xs text-neutral-700 font-semibold block mt-1">{dateLabel(selectedPayment.departure_date)}</span>
                      )}
                    </div>
                  </div>
                </div>

                <div className="p-3.5 rounded-xl border border-neutral-200/70 space-y-2 text-xs">
                  <div className="flex justify-between border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Tanggal Transfer</span>
                    <span className="font-medium text-neutral-800">{dateLabel(selectedPayment.tanggal)}</span>
                  </div>
                  <div className="flex justify-between border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Pengirim</span>
                    <span className="font-semibold text-neutral-800">
                      {selectedPayment.sender_name || 'Belum diisi'}
                      {selectedPayment.sender_bank ? ` (${selectedPayment.sender_bank})` : ''}
                    </span>
                  </div>
                  <div className="flex justify-between items-start">
                    <span className="text-neutral-500">Rekening Tujuan</span>
                    <div className="text-right">
                      <p className="font-semibold text-neutral-800">{selectedPayment.destination_bank_name || '-'}</p>
                      <p className="font-mono text-xs text-neutral-600">{selectedPayment.destination_account_number || '-'}</p>
                      {selectedPayment.destination_account_holder && (
                        <p className="text-xs text-neutral-500">a.n. {selectedPayment.destination_account_holder}</p>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              {/* Kolom Kanan: Foto Bukti Transfer */}
              <div className="flex flex-col">
                <span className="text-xs font-bold text-neutral-700 mb-2 block">Foto Bukti Transfer</span>

                <div className="flex-1 min-h-64 max-h-80 rounded-xl border border-neutral-200 bg-neutral-50 overflow-hidden flex flex-col items-center justify-center p-2">
                  {proofLoading ? (
                    <div className="text-center py-10 space-y-2">
                      <div className="w-7 h-7 border-2 border-neutral-300 border-t-primary-500 rounded-full animate-spin mx-auto" />
                      <p className="text-xs text-neutral-500 font-medium">Memuat bukti transfer...</p>
                    </div>
                  ) : proofBlobUrl ? (
                    <div className="relative w-full h-full flex flex-col items-center justify-center">
                      <img
                        src={proofBlobUrl}
                        alt="Bukti Transfer"
                        className="max-h-72 w-auto max-w-full object-contain rounded-lg shadow-2xs cursor-zoom-in hover:opacity-90 transition-opacity"
                        onClick={() => window.open(proofBlobUrl, '_blank')}
                        title="Klik untuk melihat ukuran penuh"
                      />
                      <div className="mt-2 text-center">
                        <button
                          type="button"
                          onClick={() => window.open(proofBlobUrl, '_blank')}
                          className="text-xs font-semibold text-neutral-900 hover:underline"
                        >
                          Buka Gambar Penuh ↗
                        </button>
                      </div>
                    </div>
                  ) : (
                    <div className="text-center py-12 px-4 space-y-1 text-neutral-400">
                      <svg className="w-10 h-10 mx-auto text-neutral-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                      <p className="text-xs font-medium text-neutral-500">Tidak Ada Bukti Transfer</p>
                      <p className="text-xs text-neutral-400">Pembayaran ini dicatat tanpa unggahan bukti transfer.</p>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Input Alasan Penolakan Jika Masuk Mode Tolak */}
            {rejectMode && (
              <div className="p-3.5 rounded-xl bg-danger-50/70 border border-danger-200/80 space-y-2">
                <label className="block text-xs font-bold text-danger-900">
                  Alasan Penolakan <span className="text-danger-600">*</span>
                </label>
                <textarea
                  value={rejectionReason}
                  onChange={(e) => setRejectionReason(e.target.value)}
                  rows={3}
                  placeholder="Tuliskan alasan mengapa pembayaran ini ditolak (akan dibaca jamaah di portal)..."
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

import { useEffect, useMemo, useState } from 'react';
import { PAYMENT_STATUS_META } from '../utils/bookingStatus';
import { SlidersHorizontal, Eye } from 'lucide-react';
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
import ActionMenu from '../components/ui/ActionMenu';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(`${String(value).slice(0, 10)}T00:00:00`));
const statusMeta = PAYMENT_STATUS_META;
const STATUS_STRONG = Object.fromEntries(
  Object.entries(PAYMENT_STATUS_META).map(([key, meta]) => [key, meta.className]),
);
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
  const [successMessage, setSuccessMessage] = useState('');
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

  // Hitungan ditampilkan di label filter status (bukan kartu KPI); mengikuti filter lain.
  const statusOptions = useMemo(() => {
    const base = items.filter((item) => {
      const date = String(item.tanggal || '').slice(0, 10);
      return (!filters.brand || item.brand_name === filters.brand)
        && (!filters.bank || item.destination_bank_name === filters.bank)
        && (!filters.from || date >= filters.from)
        && (!filters.to || date <= filters.to);
    });
    const count = (status) => base.filter((i) => i.status === status).length;
    return [
      { value: '', label: `Semua status (${base.length})` },
      { value: 'pending', label: `Menunggu (${count('pending')})` },
      { value: 'confirmed', label: `Terkonfirmasi (${count('confirmed')})` },
      { value: 'rejected', label: `Ditolak (${count('rejected')})` },
    ];
  }, [items, filters.brand, filters.bank, filters.from, filters.to]);

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
      setSuccessMessage(`Pembayaran ${selectedPayment.jamaah_name || ''} sebesar ${money(selectedPayment.jumlah)} dikonfirmasi.`.replace('  ', ' '));
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
      setSuccessMessage(`Pembayaran ${selectedPayment.jamaah_name || ''} ditolak.`.replace('  ', ' '));
      closeDetail();
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Status pembayaran gagal diperbarui.');
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    { header: 'Jamaah', key: 'jamaah_name', sortable: true },
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
          <button
            type="button"
            onClick={() => openDetail(row)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
          >
            {row.jamaah_name || '-'}
          </button>
          <div className="mt-0.5 text-[11px] font-body text-neutral-400">
            {showBrandColumn && row.brand_name ? `${row.brand_name} · ` : ''}
            {row.booking_id ? (
              <a
                href={`/bookings/${row.booking_id}`}
                target="_blank"
                rel="noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="font-mono hover:text-primary-600 hover:underline"
                title="Buka Detail Booking di tab baru"
              >
                {row.invoice}
              </a>
            ) : (
              <span className="font-mono">{row.invoice}</span>
            )}
          </div>
        </div>
      );
    }
    if (key === 'schedule_name') {
      return (
        <div className="min-w-36 max-w-56 whitespace-normal">
          <p className="text-[13px] font-body font-medium text-neutral-800 leading-tight">{row.schedule_name || '-'}</p>
          {row.departure_date && <p className="mt-0.5 text-[11px] font-body text-neutral-400">{dateLabel(row.departure_date)}</p>}
        </div>
      );
    }
    if (key === 'tanggal') {
      return (
        <div className="min-w-28">
          <p className="text-[13px] font-body font-medium text-neutral-800 whitespace-nowrap">{dateLabel(row.tanggal)}</p>
          <p className="mt-0.5 text-[11px] font-body text-neutral-400">
            {row.sender_name || 'Pengirim belum diisi'}
            {row.sender_bank ? ` · ${row.sender_bank}` : ''}
          </p>
        </div>
      );
    }
    if (key === 'destination_bank_name') {
      // Tunai tidak punya rekening: tampilkan staff yang menerima uangnya.
      if (String(row.metode || '').toLowerCase() === 'cash') {
        return (
          <div className="min-w-28">
            <p className="text-[13px] font-body font-medium text-neutral-800">Tunai</p>
            <p className="mt-0.5 text-[11px] font-body text-neutral-400">
              {row.received_by ? `Diterima: ${row.received_by}` : 'Penerima belum dicatat'}
            </p>
          </div>
        );
      }
      return (
        <div className="min-w-28">
          <p className="text-[13px] font-body font-medium text-neutral-800">{row.destination_bank_name || '-'}</p>
          <p className="mt-0.5 text-[11px] font-body text-neutral-400">
            {row.destination_account_number || '-'}
            {row.destination_account_holder ? ` · ${row.destination_account_holder}` : ''}
          </p>
        </div>
      );
    }

    if (key === 'jumlah') {
      return <span className="text-[13px] font-body font-semibold text-neutral-900 whitespace-nowrap">{money(row.jumlah)}</span>;
    }
    if (key === 'status') {
      const meta = statusMeta[row.status] || { label: row.status || '-', variant: 'archived' };
      return (
        <div className="max-w-44 whitespace-normal">
          <Badge variant={meta.variant} hideIcon className={STATUS_STRONG[row.status]}>{meta.label}</Badge>
          {row.rejection_reason && <p className="mt-1 text-[11px] font-body text-danger-700 leading-tight">{row.rejection_reason}</p>}
        </div>
      );
    }
    if (key === 'actions') {
      return (
        <ActionMenu
          align="right"
          items={[{ label: 'Lihat Detail', icon: Eye, tooltip: 'Lihat detail dan verifikasi pembayaran', onClick: () => openDetail(row) }]}
        />
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
        options={statusOptions}
        placeholder="Semua status"
        className="!mb-0 flex-1 sm:w-52 sm:flex-none"
      />
      <Button
        type="button"
        variant="secondary"
        onClick={() => setPanelOpen((open) => !open)}
        className={`h-10 shrink-0 whitespace-nowrap ${panelOpen || panelActive ? '!border-primary-200 !bg-primary-50 !text-primary-700' : ''}`}
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
    <div className="space-y-6">
      <PageHeader title="Pembayaran" />

      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage('')}>{successMessage}</Alert>}
      {error && <Alert variant="error" onClose={() => setError('')}>{error}</Alert>}

      <DataTable
        columns={columns}
        data={filteredItems}
        renderCell={renderCell}
        onRowClick={openDetail}
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
        size={proofLoading || proofBlobUrl ? 'lg' : 'xl'}
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
                    className="!bg-success-600 hover:!bg-success-700 !text-white !border-success-600 shadow-xs"
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
        {selectedPayment && (() => {
          const hasProof = proofLoading || Boolean(proofBlobUrl);
          const row = 'flex items-start justify-between gap-6 py-2.5';
          const dt = 'shrink-0 text-neutral-500';
          const dd = 'text-right font-medium text-neutral-900';

          const details = (
            <div className="min-w-0">
              <span className="block text-[11px] font-body text-neutral-400">Nominal Pembayaran</span>
              <div className="mt-0.5 flex items-center justify-between gap-3">
                <span className="font-heading text-3xl font-bold tracking-tight text-neutral-900">
                  {money(selectedPayment.jumlah)}
                </span>
                {selectedMeta && <Badge variant={selectedMeta.variant} hideIcon className={STATUS_STRONG[selectedPayment.status]}>{selectedMeta.label}</Badge>}
              </div>

              {selectedPayment.status === 'rejected' && selectedPayment.rejection_reason && (
                <p className="mt-2 text-xs font-body text-danger-700">
                  <span className="font-semibold">Alasan ditolak:</span> {selectedPayment.rejection_reason}
                </p>
              )}

              <dl className="mt-4 divide-y divide-neutral-200/80 border-t border-neutral-200/80 text-[13px] font-body">
                <div className={row}>
                  <dt className={dt}>Tanggal Transfer</dt>
                  <dd className={dd}>{dateLabel(selectedPayment.tanggal)}</dd>
                </div>
                {showBrandColumn && (
                  <div className={row}>
                    <dt className={dt}>Brand</dt>
                    <dd className={dd}>{selectedPayment.brand_name || '-'}</dd>
                  </div>
                )}
                <div className={row}>
                  <dt className={dt}>Kode Booking</dt>
                  <dd className={dd}>
                    <span className="font-mono">#{selectedPayment.invoice}</span>
                    {selectedPayment.booking_id && (
                      <a
                        href={`/bookings/${selectedPayment.booking_id}`}
                        target="_blank"
                        rel="noreferrer"
                        className="ml-2 text-xs font-semibold text-primary-600 hover:text-primary-700 hover:underline"
                        title="Buka Detail Booking di tab baru"
                      >
                        Buka ↗
                      </a>
                    )}
                  </dd>
                </div>
                <div className={row}>
                  <dt className={dt}>Nama Jamaah</dt>
                  <dd className={dd}>{selectedPayment.jamaah_name || '-'}</dd>
                </div>
                <div className={row}>
                  <dt className={dt}>Paket</dt>
                  <dd className={dd}>
                    {selectedPayment.schedule_name || '-'}
                    {selectedPayment.departure_date && (
                      <span className="block text-xs font-normal text-neutral-500">{dateLabel(selectedPayment.departure_date)}</span>
                    )}
                  </dd>
                </div>
                {String(selectedPayment.metode || '').toLowerCase() === 'cash' ? (
                  <>
                    <div className={row}>
                      <dt className={dt}>Metode</dt>
                      <dd className={dd}>Tunai</dd>
                    </div>
                    <div className={row}>
                      <dt className={dt}>Diterima Oleh</dt>
                      <dd className={dd}>{selectedPayment.received_by || 'Belum dicatat'}</dd>
                    </div>
                  </>
                ) : (
                  <>
                    <div className={row}>
                      <dt className={dt}>Pengirim</dt>
                      <dd className={dd}>
                        {selectedPayment.sender_name || 'Belum diisi'}
                        {selectedPayment.sender_bank ? ` (${selectedPayment.sender_bank})` : ''}
                      </dd>
                    </div>
                    <div className={row}>
                      <dt className={dt}>Rekening Tujuan</dt>
                      <dd className={dd}>
                        {selectedPayment.destination_bank_name || (selectedPayment.destination_account_number ? '' : '-')}
                        {selectedPayment.destination_account_number && (
                          <span className="block font-mono text-xs font-normal text-neutral-600">{selectedPayment.destination_account_number}</span>
                        )}
                        {selectedPayment.destination_account_holder && (
                          <span className="block text-xs font-normal text-neutral-500">a.n. {selectedPayment.destination_account_holder}</span>
                        )}
                      </dd>
                    </div>
                  </>
                )}
              </dl>
            </div>
          );

          return (
            <div className="space-y-5">
              {actionError && <Alert variant="error" message={actionError} />}

              {hasProof ? (
                <div className="grid grid-cols-1 gap-x-8 gap-y-6 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                  {/* Bukti transfer jadi fokus verifikasi */}
                  <div className="min-w-0">
                    <span className="mb-1.5 block text-[11px] font-body text-neutral-400">Bukti Transfer</span>
                    <div className="flex min-h-72 items-center justify-center overflow-hidden rounded-lg border border-neutral-200 bg-neutral-50 p-2">
                      {proofLoading ? (
                        <div className="py-10 text-center space-y-2">
                          <div className="mx-auto h-7 w-7 animate-spin rounded-full border-2 border-neutral-300 border-t-primary-500" />
                          <p className="text-xs font-body text-neutral-500">Memuat bukti transfer...</p>
                        </div>
                      ) : (
                        <img
                          src={proofBlobUrl}
                          alt="Bukti Transfer"
                          className="max-h-96 w-auto max-w-full cursor-zoom-in rounded-md object-contain transition-opacity hover:opacity-90"
                          onClick={() => window.open(proofBlobUrl, '_blank')}
                          title="Klik untuk melihat ukuran penuh"
                        />
                      )}
                    </div>
                    {proofBlobUrl && (
                      <button
                        type="button"
                        onClick={() => window.open(proofBlobUrl, '_blank')}
                        className="mt-2 cursor-pointer text-xs font-semibold text-primary-600 hover:text-primary-700 hover:underline"
                      >
                        Buka gambar penuh ↗
                      </button>
                    )}
                  </div>
                  {details}
                </div>
              ) : (
                <>
                  {details}
                  <p className="border-t border-neutral-200/80 pt-3 text-xs font-body text-neutral-500">
                    Tidak ada bukti transfer. Pembayaran ini dicatat tanpa unggahan bukti.
                  </p>
                </>
              )}

              {/* Alasan penolakan (mode tolak) */}
              {rejectMode && (
                <div className="border-t border-neutral-200/80 pt-4">
                  <label className="mb-1.5 block text-xs font-semibold font-body text-neutral-800">
                    Alasan Penolakan <span className="text-danger-600">*</span>
                  </label>
                  <textarea
                    value={rejectionReason}
                    onChange={(e) => setRejectionReason(e.target.value)}
                    rows={3}
                    placeholder="Tuliskan alasan mengapa pembayaran ini ditolak (akan dibaca jamaah di portal)..."
                    className="w-full rounded-lg border border-neutral-200 bg-white px-3.5 py-2.5 text-[13px] font-body text-neutral-900 placeholder:text-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
                    autoFocus
                  />
                </div>
              )}
            </div>
          );
        })()}
      </Modal>
    </div>
  );
}

import { useEffect, useMemo, useState } from 'react';
import client from '../api/client';
import { listPayments, verifyPayment } from '../api/payments';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import DataTable from '../components/ui/DataTable';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';

const money = (value) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);
const dateLabel = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(`${String(value).slice(0, 10)}T00:00:00`));
const statusMeta = {
  pending: ['Menunggu', 'pending'],
  confirmed: ['Terkonfirmasi', 'success'],
  rejected: ['Ditolak', 'danger'],
};
const controlClass = 'mt-1.5 h-11 w-full rounded-md border border-neutral-300 bg-white px-3 text-sm text-neutral-800 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500';

export default function PaymentConfirmationsPage() {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filters, setFilters] = useState({ status: '', brand: '', bank: '', from: '', to: '' });

  // Modal Detail & Verifikasi State
  const [selectedPayment, setSelectedPayment] = useState(null);
  const [proofBlobUrl, setProofBlobUrl] = useState(null);
  const [proofLoading, setProofLoading] = useState(false);
  const [rejectMode, setRejectMode] = useState(false);
  const [rejectionReason, setRejectionReason] = useState('');
  const [actionSaving, setActionSaving] = useState(false);
  const [actionError, setActionError] = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      setItems((await listPayments()) || []);
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
      const rawUrl = selectedPayment.bukti_url;
      const normalizedUrl = String(rawUrl || '').replace(
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
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
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
      item.brand_name,
      item.jamaah_name,
      item.schedule_name,
      item.sender_name,
      item.sender_bank,
      item.destination_bank_name,
      item.destination_account_number,
    ].filter(Boolean).join(' '),
  })), [items, filters]);

  const resetFilters = () => setFilters({ status: '', brand: '', bank: '', from: '', to: '' });

  const openDetail = (payment) => {
    setActionError('');
    setRejectMode(false);
    setRejectionReason('');
    setSelectedPayment(payment);
  };

  const closeDetail = () => {
    if (actionSaving) return;
    setSelectedPayment(null);
    setProofBlobUrl(null);
    setRejectMode(false);
    setRejectionReason('');
    setActionError('');
  };

  // Konfirmasi Terima Pembayaran
  const handleConfirmPayment = async () => {
    if (!selectedPayment) return;
    setActionSaving(true);
    setActionError('');
    try {
      await verifyPayment(selectedPayment.id, 'confirmed', null);
      closeDetail();
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Status pembayaran gagal diperbarui.');
    } finally {
      setActionSaving(false);
    }
  };

  // Tolak Pembayaran
  const handleRejectPayment = async () => {
    if (!selectedPayment) return;
    if (!rejectionReason.trim()) {
      setActionError('Alasan penolakan wajib diisi agar dapat dibaca jamaah.');
      return;
    }
    setActionSaving(true);
    setActionError('');
    try {
      await verifyPayment(selectedPayment.id, 'rejected', rejectionReason.trim());
      closeDetail();
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Status pembayaran gagal diperbarui.');
    } finally {
      setActionSaving(false);
    }
  };

  const hasFilters = Object.values(filters).some(Boolean);

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
    if (key === 'brand_name') {
      return (
        <div className="min-w-[150px]">
          <p className="font-semibold text-neutral-900">{row.brand_name || '-'}</p>
          {row.booking_id ? (
            <a
              href={`/bookings/${row.booking_id}`}
              target="_blank"
              rel="noreferrer"
              className="mt-1 inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-neutral-100 hover:bg-neutral-200 border border-neutral-200/80 text-neutral-800 hover:text-neutral-950 font-mono text-[11.5px] font-bold transition-all cursor-pointer group"
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
    if (key === 'jamaah_name') {
      return (
        <div className="max-w-[220px] whitespace-normal">
          <p className="font-semibold text-neutral-900">{row.jamaah_name || '-'}</p>
          <p className="mt-0.5 text-xs text-neutral-500">{row.schedule_name || '-'}</p>
          {row.departure_date && (
            <p className="mt-0.5 text-[11px] text-neutral-400">
              {dateLabel(row.departure_date)}
            </p>
          )}
        </div>
      );
    }
    if (key === 'tanggal') {
      return (
        <div>
          <p className="font-medium text-neutral-800">{dateLabel(row.tanggal)}</p>
          <p className="mt-0.5 text-xs text-neutral-500">
            {row.sender_name || 'Pengirim belum diisi'}
            {row.sender_bank ? ` • ${row.sender_bank}` : ''}
          </p>
        </div>
      );
    }
    if (key === 'destination_bank_name') {
      return (
        <div>
          <p className="font-medium text-neutral-800">{row.destination_bank_name || '-'}</p>
          <p className="mt-0.5 text-xs text-neutral-500">
            {row.destination_account_number || '-'}
            {row.destination_account_holder ? ` • ${row.destination_account_holder}` : ''}
          </p>
        </div>
      );
    }
    if (key === 'jumlah') {
      return <span className="font-semibold text-neutral-900">{money(row.jumlah)}</span>;
    }
    if (key === 'status') {
      const meta = statusMeta[row.status] || [row.status || '-', 'neutral'];
      return (
        <div className="max-w-[170px] whitespace-normal">
          <Badge variant={meta[1]}>{meta[0]}</Badge>
          {row.rejection_reason && (
            <p className="mt-1.5 text-xs text-danger-700 leading-tight">{row.rejection_reason}</p>
          )}
        </div>
      );
    }
    if (key === 'actions') {
      return (
        <button
          type="button"
          onClick={() => openDetail(row)}
          className="rounded-full border border-neutral-200 bg-white hover:bg-neutral-50 px-3.5 py-1.5 text-xs font-semibold text-neutral-700 shadow-2xs hover:border-neutral-300 transition-colors cursor-pointer"
        >
          Lihat Detail
        </button>
      );
    }
    return row[key] ?? '-';
  };

  const selectedMeta = selectedPayment ? (statusMeta[selectedPayment.status] || [selectedPayment.status || '-', 'neutral']) : null;

  return (
    <div className="space-y-5">
      <PageHeader
        title="Konfirmasi Pembayaran"
        subtitle="Cari dan verifikasi transfer jamaah dari seluruh brand."
      />

      {error && <Alert variant="error">{error}</Alert>}

      <section className="rounded-lg border border-neutral-200 bg-white p-4">
        <div className="mb-3 flex items-center justify-between gap-3">
          <div>
            <h2 className="font-semibold text-neutral-900">Filter pembayaran</h2>
            <p className="text-xs text-neutral-500">Persempit data berdasarkan brand, status, rekening, atau tanggal transfer.</p>
          </div>
          {hasFilters && (
            <button onClick={resetFilters} className="text-sm font-semibold text-primary-600 hover:underline">
              Reset filter
            </button>
          )}
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          <label className="text-xs font-medium text-neutral-600">
            Status
            <select
              value={filters.status}
              onChange={(e) => setFilters((prev) => ({ ...prev, status: e.target.value }))}
              className={controlClass}
            >
              <option value="">Semua status</option>
              <option value="pending">Menunggu</option>
              <option value="confirmed">Terkonfirmasi</option>
              <option value="rejected">Ditolak</option>
            </select>
          </label>
          <label className="text-xs font-medium text-neutral-600">
            Brand
            <select
              value={filters.brand}
              onChange={(e) => setFilters((prev) => ({ ...prev, brand: e.target.value }))}
              className={controlClass}
            >
              <option value="">Semua brand</option>
              {options.brands.map((brand) => (
                <option key={brand} value={brand}>{brand}</option>
              ))}
            </select>
          </label>
          <label className="text-xs font-medium text-neutral-600">
            Rekening tujuan
            <select
              value={filters.bank}
              onChange={(e) => setFilters((prev) => ({ ...prev, bank: e.target.value }))}
              className={controlClass}
            >
              <option value="">Semua rekening</option>
              {options.banks.map((bank) => (
                <option key={bank} value={bank}>{bank}</option>
              ))}
            </select>
          </label>
          <label className="text-xs font-medium text-neutral-600">
            Dari tanggal
            <input
              type="date"
              value={filters.from}
              onChange={(e) => setFilters((prev) => ({ ...prev, from: e.target.value }))}
              className={controlClass}
            />
          </label>
          <label className="text-xs font-medium text-neutral-600">
            Sampai tanggal
            <input
              type="date"
              value={filters.to}
              min={filters.from || undefined}
              onChange={(e) => setFilters((prev) => ({ ...prev, to: e.target.value }))}
              className={controlClass}
            />
          </label>
        </div>
      </section>

      <DataTable
        columns={columns}
        data={filteredItems}
        renderCell={renderCell}
        itemsPerPage={10}
        searchPlaceholder="Cari brand, invoice, jamaah, paket, pengirim, atau rekening..."
        emptyMessage={loading ? 'Memuat pembayaran...' : hasFilters ? 'Tidak ada pembayaran yang sesuai filter' : 'Belum ada pembayaran'}
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
                  disabled={actionSaving}
                >
                  Batal Tolak
                </Button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="secondary"
                onClick={closeDetail}
                disabled={actionSaving}
              >
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
                    disabled={actionSaving}
                  >
                    Tolak
                  </Button>
                  <Button
                    variant="primary"
                    className="!bg-emerald-600 hover:!bg-emerald-700 !text-white !border-emerald-600 shadow-xs"
                    onClick={handleConfirmPayment}
                    isLoading={actionSaving}
                  >
                    Pembayaran Diterima
                  </Button>
                </>
              )}

              {selectedPayment?.status === 'pending' && rejectMode && (
                <Button
                  variant="danger"
                  onClick={handleRejectPayment}
                  isLoading={actionSaving}
                >
                  Konfirmasi Tolak Pembayaran
                </Button>
              )}
            </div>
          </div>
        }
      >
        {selectedPayment && (
          <div className="space-y-4">
            {actionError && <Alert variant="error">{actionError}</Alert>}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Kolom Kiri: Info Ringkas Pembayaran */}
              <div className="space-y-3">
                {/* Box Nominal */}
                <div className="p-3.5 rounded-xl bg-neutral-50 border border-neutral-200/80">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                      Nominal Pembayaran
                    </span>
                    {selectedMeta && (
                      <Badge variant={selectedMeta[1]}>{selectedMeta[0]}</Badge>
                    )}
                  </div>
                  <div className="text-2xl font-black text-neutral-900 tracking-tight mt-1">
                    {money(selectedPayment.jumlah)}
                  </div>
                  {selectedPayment.status === 'rejected' && selectedPayment.rejection_reason && (
                    <p className="mt-1.5 text-xs text-danger-700 bg-danger-50 p-2 rounded-lg border border-danger-200/60">
                      <strong>Alasan ditolak:</strong> {selectedPayment.rejection_reason}
                    </p>
                  )}
                </div>

                {/* Detail Paket & Jamaah */}
                <div className="p-3.5 rounded-xl border border-neutral-200/70 space-y-2 text-xs">
                  <div className="flex justify-between border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Brand</span>
                    <span className="font-semibold text-neutral-800">{selectedPayment.brand_name || '-'}</span>
                  </div>
                  <div className="flex justify-between items-center border-b border-neutral-100 pb-1.5">
                    <span className="text-neutral-500">Kode Booking</span>
                    <div className="flex items-center gap-2">
                      <span className="font-mono font-bold text-neutral-900">#{selectedPayment.invoice}</span>
                      {selectedPayment.booking_id && (
                        <a
                          href={`/bookings/${selectedPayment.booking_id}`}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-neutral-100 hover:bg-neutral-200 text-neutral-700 hover:text-neutral-900 text-[11px] font-medium transition-colors cursor-pointer"
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
                  <div className="flex justify-between items-start">
                    <span className="text-neutral-500">Paket</span>
                    <div className="text-right max-w-[60%]">
                      <span className="font-medium text-neutral-800 block leading-tight">
                        {selectedPayment.schedule_name || '-'}
                      </span>
                      {selectedPayment.departure_date && (
                        <span className="text-[11px] text-neutral-500 block mt-1 font-normal">
                          <strong className="text-neutral-700 font-semibold">{dateLabel(selectedPayment.departure_date)}</strong>
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* Detail Transfer */}
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
                      <p className="font-mono text-[11px] text-neutral-600">{selectedPayment.destination_account_number || '-'}</p>
                      {selectedPayment.destination_account_holder && (
                        <p className="text-[11px] text-neutral-500">a.n. {selectedPayment.destination_account_holder}</p>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              {/* Kolom Kanan: Photo Bukti Transfer */}
              <div className="flex flex-col">
                <span className="text-xs font-bold text-neutral-700 mb-2 block">
                  Foto Bukti Transfer
                </span>

                <div className="flex-1 min-h-[260px] max-h-[340px] rounded-xl border border-neutral-200 bg-neutral-50 overflow-hidden flex flex-col items-center justify-center p-2">
                  {proofLoading ? (
                    <div className="text-center py-10 space-y-2">
                      <div className="w-7 h-7 border-2 border-neutral-300 border-t-primary-500 rounded-full animate-spin mx-auto" />
                      <p className="text-xs text-neutral-500 font-medium">Memuat bukti transfer...</p>
                    </div>
                  ) : proofBlobUrl ? (
                    <div className="relative w-full h-full flex flex-col items-center justify-center group">
                      <img
                        src={proofBlobUrl}
                        alt="Bukti Transfer"
                        className="max-h-[300px] w-auto max-w-full object-contain rounded-lg shadow-2xs cursor-zoom-in transition-transform group-hover:scale-[1.01]"
                        onClick={() => window.open(proofBlobUrl, '_blank')}
                        title="Klik untuk melihat ukuran penuh"
                      />
                      <div className="mt-2 text-center">
                        <button
                          type="button"
                          onClick={() => window.open(proofBlobUrl, '_blank')}
                          className="text-[11px] font-semibold text-primary-600 hover:text-primary-700 hover:underline"
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
                      <p className="text-[11px] text-neutral-400">Pembayaran ini dicatat tanpa unggahan bukti transfer.</p>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Input Alasan Penolakan Jika Masuk Mode Tolak */}
            {rejectMode && (
              <div className="p-3.5 rounded-xl bg-danger-50/70 border border-danger-200/80 space-y-2 animate-in fade-in duration-150">
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

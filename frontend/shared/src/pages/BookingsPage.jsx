import { useState, useEffect, useMemo } from "react";
import { useNavigate, useSearchParams, useLocation } from 'react-router-dom';
import { Eye, Pencil, Trash2, SlidersHorizontal } from "lucide-react";
import { listBookings, deleteDraftBooking } from "../api/bookings";
import { listBrands } from "../api/brands";
import { getSeatLockIcon } from "../utils/bookingStatus";
import PageHeader from "../components/ui/PageHeader";
import DataTable from "../components/ui/DataTable";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import CustomDropdown from "../components/ui/CustomDropdown";
import Modal from "../components/ui/Modal";
import Badge from "../components/ui/Badge";
import ActionMenu from "../components/ui/ActionMenu";

const formatRupiah = (angka) => {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(angka || 0);
};

const formatTanggal = (dateStr) => {
  if (!dateStr) return "-";
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
  } catch (e) {
    return dateStr;
  }
};

const formatPaxDetail = (row) => {
  const reg = row.regular_pax_count ?? (row.pax ? row.pax.filter(p => p.pax_type !== 'infant').length : 0);
  const inf = row.infant_pax_count ?? (row.pax ? row.pax.filter(p => p.pax_type === 'infant').length : 0);

  if (reg === 0 && inf === 0) {
    const total = row.pax_count || 1;
    return `${total} reguler`;
  }

  if (inf > 0) {
    return `${reg} reguler, ${inf} infant`;
  }
  return `${reg} reguler`;
};

// Status booking memakai Badge bersama; warna dipertegas agar terbaca di atas latar abu-abu.
const STATUS_BADGE = {
  draft: { label: 'Draft', variant: 'draft', cls: '!border-neutral-400 !text-neutral-800' },
  baru: { label: 'Baru', variant: 'neutral', cls: '!bg-info-100 !border-info-500 !text-info-600' },
  dp: { label: 'DP', variant: 'warning', cls: '!bg-warning-100 !border-warning-300 !text-warning-800' },
  lunas: { label: 'Lunas', variant: 'success', cls: '!bg-success-100 !border-success-300 !text-success-800' },
  batal: { label: 'Batal', variant: 'danger', cls: '!bg-danger-100 !border-danger-200 !text-danger-700' },
};

const StatusPill = ({ status }) => {
  const s = STATUS_BADGE[status] || { label: status || '-', variant: 'archived', cls: '!bg-neutral-200 !border-neutral-300 !text-neutral-800' };
  return <Badge variant={s.variant} hideIcon className={s.cls}>{s.label}</Badge>;
};

// Chip brand: warna brand hanya penanda (dot), bukan warna UI.
const BrandChip = ({ brand, brandId }) => (
  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[12px] font-medium font-body bg-white border border-neutral-200 text-neutral-700 whitespace-nowrap">
    <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: brand?.primary_color || '#9CA3AF' }} />
    {brand?.name || `Brand #${brandId}`}
  </span>
);

export const BookingsPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [data, setData] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);

  // Delete Draft Modal State
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [selectedDraft, setSelectedDraft] = useState(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // Filter State
  const [filterBrandId, setFilterBrandId] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const [filterSeat, setFilterSeat] = useState("");
  const [filterDate, setFilterDate] = useState("");
  const [panelOpen, setPanelOpen] = useState(false);

  // Tautan dari detail paket: /bookings?schedule=<id> hanya menampilkan booking paket itu.
  const [searchParams, setSearchParams] = useSearchParams();
  const filterScheduleId = searchParams.get('schedule') || '';
  const clearScheduleFilter = () => setSearchParams({}, { replace: true });

  useEffect(() => {
    fetchData();
  }, [showBrandColumn]);

  useEffect(() => {
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, "");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const promises = [listBookings()];
      if (showBrandColumn) {
        promises.push(listBrands());
      }

      const [resBookings, resBrands] = await Promise.all(promises);
      setData(resBookings || []);

      if (resBrands) {
        setBrands(resBrands || []);
        const bMap = {};
        (resBrands || []).forEach(b => {
          bMap[b.id] = b;
        });
        setBrandsMap(bMap);
      }
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memuat data booking");
    } finally {
      setLoading(false);
    }
  };

  const confirmDeleteDraft = async () => {
    if (!selectedDraft) return;
    try {
      setDeleteLoading(true);
      await deleteDraftBooking(selectedDraft.id);
      setIsDeleteModalOpen(false);
      setSuccessMessage("Draft booking berhasil dihapus.");
      setSelectedDraft(null);
      await fetchData();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menghapus draft booking");
      setIsDeleteModalOpen(false);
    } finally {
      setDeleteLoading(false);
    }
  };

  // Available unique departure dates sorted chronologically
  const availableDates = useMemo(() => {
    const map = new Map();
    data.forEach(b => {
      if (b.berangkat_tanggal) {
        const d = new Date(b.berangkat_tanggal);
        if (!isNaN(d.getTime())) {
          const val = d.toISOString().split('T')[0];
          const label = d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
          map.set(val, label);
        }
      }
    });
    return Array.from(map.entries())
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([value, label]) => ({ value, label }));
  }, [data]);

  // Client-side filtering
  const filteredBookings = useMemo(() => {
    return data.filter(b => {
      // 0. Filter Brand
      if (showBrandColumn && filterBrandId) {
        if (b.brand_id?.toString() !== filterBrandId) return false;
      }

      // 1. Filter Status
      if (filterScheduleId && String(b.schedule_id) !== filterScheduleId) return false;

      if (filterStatus && b.status !== filterStatus) {
        return false;
      }

      // 2. Filter Seat
      if (filterSeat) {
        if (filterSeat === 'blocked' && !b.is_seat_blocked) return false;
        if (filterSeat === 'unblocked' && b.is_seat_blocked) return false;
      }

      // 3. Filter Tanggal Keberangkatan
      if (filterDate) {
        const dateVal = b.berangkat_tanggal ? new Date(b.berangkat_tanggal).toISOString().split('T')[0] : '';
        if (dateVal !== filterDate) return false;
      }

      return true;
    });
  }, [data, showBrandColumn, filterBrandId, filterStatus, filterSeat, filterDate, filterScheduleId]);

  // Hitungan ditampilkan di label filter (bukan kartu KPI); mengikuti filter brand/paket.
  const baseForCounts = useMemo(
    () => data.filter(b => !(showBrandColumn && filterBrandId && b.brand_id?.toString() !== filterBrandId) && !(filterScheduleId && String(b.schedule_id) !== filterScheduleId)),
    [data, showBrandColumn, filterBrandId, filterScheduleId]
  );
  const statusOptions = useMemo(() => {
    const count = (st) => baseForCounts.filter(b => b.status === st).length;
    return [
      { value: '', label: `Semua Status (${baseForCounts.length})` },
      { value: 'draft', label: `Draft (${count('draft')})` },
      { value: 'baru', label: `Baru (${count('baru')})` },
      { value: 'dp', label: `DP (${count('dp')})` },
      { value: 'lunas', label: `Lunas (${count('lunas')})` },
      { value: 'batal', label: `Batal (${count('batal')})` },
    ];
  }, [baseForCounts]);
  const seatOptions = useMemo(() => {
    const unlocked = baseForCounts.filter(b => ['baru', 'dp', 'lunas'].includes(b.status) && !b.is_seat_blocked).length;
    return [
      { value: '', label: 'Semua Seat' },
      { value: 'blocked', label: 'Seat Terkunci' },
      { value: 'unblocked', label: `Seat Belum Terkunci (${unlocked})` },
    ];
  }, [baseForCounts]);

  // Jumlah filter aktif di panel (status tampil terpisah di toolbar).
  const panelActive = [showBrandColumn && filterBrandId, filterSeat, filterDate].filter(Boolean).length;

  const columns = useMemo(() => {
    const cols = [{ header: "Booking", key: "nama_jamaah", sortable: true }];
    if (showBrandColumn) cols.push({ header: "Brand", key: "brand" });
    cols.push(
      {
        header: "Paket",
        key: "jadwal_nama",
        sortable: true,
        sortFn: (a, b) => String(a.berangkat_tanggal || "").localeCompare(String(b.berangkat_tanggal || "")),
      },
      {
        header: "Tagihan",
        key: "total_harga",
        sortable: true,
        sortFn: (a, b) => (a.total_harga || 0) - (b.total_harga || 0),
      },
      { header: "Status", key: "status" },
      { header: "Seat", key: "is_seat_blocked" },
      { header: "Aksi", key: "aksi" },
    );
    return cols;
  }, [showBrandColumn]);

  const renderCell = (row, key) => {
    if (key === "nama_jamaah") {
      const idLabel = row.id_booking || (row.status === 'draft' ? 'Draft' : `ID ${row.id}`);
      return (
        <div className="min-w-0">
          <div className="font-heading font-semibold text-neutral-900 leading-snug">{row.nama_jamaah || "-"}</div>
          <div className="mt-0.5 text-[11px] font-body text-neutral-400">
            <span className={row.id_booking ? "font-mono" : ""}>{idLabel}</span> · {formatPaxDetail(row)}
          </div>
        </div>
      );
    }

    if (key === "brand") return <BrandChip brand={brandsMap[row.brand_id]} brandId={row.brand_id} />;

    if (key === "jadwal_nama") {
      return (
        <div className="min-w-0">
          <div className="text-[13px] font-body font-medium text-neutral-800">{row.jadwal_nama || "-"}</div>
          <div className="mt-0.5 text-[11px] font-body text-neutral-400">
            {formatTanggal(row.berangkat_tanggal)}
          </div>
        </div>
      );
    }

    if (key === "total_harga") {
      return (
        <div className="min-w-0 whitespace-nowrap">
          <div className="text-[13px] font-body font-semibold text-neutral-900">{formatRupiah(row.total_harga)}</div>
          {row.total_dibayar > 0 && (
            <div className="mt-0.5 text-[11px] font-body text-neutral-400">Dibayar {formatRupiah(row.total_dibayar)}</div>
          )}
        </div>
      );
    }

    if (key === "status") {
      return (
        <div>
          <StatusPill status={row.status} />
          {row.perlu_refund && <div className="mt-1 text-[11px] font-body text-warning-700">Perlu refund</div>}
        </div>
      );
    }

    if (key === "is_seat_blocked") {
      const lockInfo = getSeatLockIcon(row.status, row.is_seat_blocked);
      if (!lockInfo) return <span className="text-xs text-neutral-400 font-body">-</span>;
      return row.is_seat_blocked ? (
        <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-success-700 whitespace-nowrap">
          <span className="w-2 h-2 rounded-full bg-success-500" />Terkunci
        </span>
      ) : (
        <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-warning-700 whitespace-nowrap">
          <span className="w-2 h-2 rounded-full bg-warning-500" />Belum terkunci
        </span>
      );
    }

    if (key === "aksi") {
      const isDraft = row.status === 'draft';
      const menuItems = isDraft
        ? [
            { label: 'Lanjutkan', icon: Pencil, tooltip: 'Lanjutkan Draft Booking', onClick: () => navigate(`/bookings/${row.id}/edit`) },
            { label: 'Hapus Draft', icon: Trash2, danger: true, tooltip: 'Hapus Draft Booking', onClick: () => { setSelectedDraft(row); setIsDeleteModalOpen(true); } },
          ]
        : [
            { label: 'Lihat', icon: Eye, tooltip: 'Lihat Detail Booking', onClick: () => navigate(`/bookings/${row.id}`) },
          ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Booking"
        actionLabel="+ Buat Booking"
        onAction={() => navigate("/bookings/new")}
      />

      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}
      {error && <Alert variant="error" onClose={() => setError(null)}>{error}</Alert>}

      {filterScheduleId && (
        <div className="flex items-center justify-between gap-3 rounded-lg border border-neutral-200 bg-white px-4 py-2.5 text-sm font-body">
          <span className="text-neutral-700">
            Menampilkan booking untuk paket{' '}
            <span className="font-semibold text-neutral-900">
              {data.find(b => String(b.schedule_id) === filterScheduleId)?.jadwal_nama || `#${filterScheduleId}`}
            </span>
          </span>
          <button
            type="button"
            onClick={clearScheduleFilter}
            className="shrink-0 cursor-pointer text-xs font-semibold text-primary-600 hover:text-primary-700"
          >
            Tampilkan semua
          </button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={filteredBookings}
          itemsPerPage={15}
          onRowClick={(row) => navigate(row.status === 'draft' ? `/bookings/${row.id}/edit` : `/bookings/${row.id}`)}
          searchPlaceholder="Cari no. booking, nama jamaah, atau paket..."
          emptyMessage={filterBrandId || filterStatus || filterSeat || filterDate ? "Data booking tidak ditemukan" : "Belum ada data booking. Klik \"+ Buat Booking\" untuk menambahkan."}
          toolbarActions={
            <div className="flex w-full items-center gap-2 sm:w-auto">
              <CustomDropdown
                value={filterStatus}
                onChange={(val) => setFilterStatus(val)}
                className="!mb-0 flex-1 sm:w-52 sm:flex-none"
                placeholder="Semua Status"
                options={statusOptions}
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
          }
          toolbarPanel={panelOpen && (
            <div className="space-y-3">
              <div className={`grid grid-cols-1 gap-3 sm:grid-cols-2 ${showBrandColumn && brands.length > 0 ? 'lg:grid-cols-3' : 'lg:grid-cols-2'}`}>
                {showBrandColumn && brands.length > 0 && (
                  <CustomDropdown
                    label="Brand"
                    value={filterBrandId}
                    onChange={(val) => setFilterBrandId(val)}
                    placeholder="Semua Brand"
                    options={[
                      { value: '', label: 'Semua Brand' },
                      ...brands.map(b => ({ value: b.id.toString(), label: b.name }))
                    ]}
                  />
                )}
                <CustomDropdown
                  label="Seat"
                  value={filterSeat}
                  onChange={(val) => setFilterSeat(val)}
                  placeholder="Semua Seat"
                  options={seatOptions}
                />
                <CustomDropdown
                  label="Tanggal Keberangkatan"
                  value={filterDate}
                  onChange={(val) => setFilterDate(val)}
                  placeholder="Semua Tanggal"
                  options={[
                    { value: '', label: 'Semua Tanggal' },
                    ...availableDates.map(d => ({ value: d.value, label: d.label }))
                  ]}
                />
              </div>
              {panelActive > 0 && (
                <div className="flex justify-end">
                  <button
                    type="button"
                    onClick={() => { setFilterBrandId(""); setFilterSeat(""); setFilterDate(""); }}
                    className="text-sm font-semibold text-neutral-900 underline underline-offset-2 cursor-pointer"
                  >
                    Reset filter
                  </button>
                </div>
              )}
            </div>
          )}
          renderCell={renderCell}
        />
      )}

      {/* Modal Konfirmasi Hapus Draft */}
      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => !deleteLoading && setIsDeleteModalOpen(false)}
        title="Konfirmasi Hapus Draft"
      >
        <p className="text-neutral-600 mb-6 font-body text-sm">
          Hapus draft booking ini? Data pax yang sudah diisi akan ikut terhapus dan tidak bisa dikembalikan.
        </p>
        <div className="flex justify-end gap-3">
          <Button
            variant="ghost"
            onClick={() => setIsDeleteModalOpen(false)}
            disabled={deleteLoading}
          >
            Batal
          </Button>
          <Button
            variant="danger"
            onClick={confirmDeleteDraft}
            disabled={deleteLoading}
          >
            {deleteLoading ? "Menghapus..." : "Hapus"}
          </Button>
        </div>
      </Modal>
    </div>
  );
};

export default BookingsPage;

import { useState, useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { listBookings, deleteDraftBooking } from "../api/bookings";
import { listBrands } from "../api/brands";
import { getStatusBadgeConfig, getSeatLockIcon } from "../utils/bookingStatus";
import PageHeader from "../components/ui/PageHeader";
import DataTable from "../components/ui/DataTable";
import Button from "../components/ui/Button";
import Badge from "../components/ui/Badge";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import CustomDropdown from "../components/ui/CustomDropdown";
import Modal from "../components/ui/Modal";
import BrandCell from "../components/BrandCell";
import { Eye, Trash2, Loader, CircleCheckBig } from "lucide-react";

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

export const BookingsPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();
  const [data, setData] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Delete Draft Modal State
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [selectedDraft, setSelectedDraft] = useState(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // Filter State
  const [filterBrandId, setFilterBrandId] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const [filterSeat, setFilterSeat] = useState("");
  const [filterDate, setFilterDate] = useState("");

  useEffect(() => {
    fetchData();
  }, [showBrandColumn]);

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
  }, [data, showBrandColumn, filterBrandId, filterStatus, filterSeat, filterDate]);

  const columns = useMemo(() => {
    const cols = [];

    if (showBrandColumn) {
      cols.push({
        header: "Brand",
        key: "brand",
        accessor: (row) => {
          const b = brandsMap[row.brand_id];
          return <BrandCell brand={b} brandId={row.brand_id} />;
        }
      });
    }

    cols.push(
      { 
        header: "ID Booking", 
        key: "id_booking", 
        accessor: (row) => {
          if (row.id_booking) {
            return (
              <span className="font-mono font-bold text-neutral-900 tracking-wider">
                {row.id_booking}
              </span>
            );
          }
          if (row.status === 'draft') {
            return <span className="text-xs text-neutral-400 italic">Draft</span>;
          }
          return <span className="text-xs text-neutral-400 font-medium">ID: {row.id}</span>;
        }
      },
      { 
        header: "Kontak Utama (PIC)", 
        key: "nama_jamaah", 
        accessor: (row) => (
          <div className="flex flex-col">
            <span className="font-semibold text-neutral-900">{row.nama_jamaah || "-"}</span>
            <span className="text-xs text-neutral-500 font-mono">
              {formatPaxDetail(row)}
            </span>
          </div>
        )
      },
      { 
        header: "Paket & Keberangkatan", 
        key: "jadwal_nama", 
        accessor: (row) => (
          <div className="flex flex-col">
            <span className="font-medium text-neutral-900">{row.jadwal_nama || "-"}</span>
            <span className="text-xs text-neutral-500">
              {formatTanggal(row.berangkat_tanggal)}
            </span>
          </div>
        )
      },
      { 
        header: "Total Tagihan", 
        key: "total_harga", 
        accessor: (row) => (
          <span className="font-semibold text-neutral-800">
            {formatRupiah(row.total_harga)}
          </span>
        )
      },
      { 
        header: "Status", 
        key: "status", 
        accessor: (row) => {
          const [statusVariant, statusLabel] = getStatusBadgeConfig(row.status);
          return <Badge variant={statusVariant} hideIcon={true}>{statusLabel}</Badge>;
        }
      },
      { 
        header: "Seat", 
        key: "is_seat_blocked", 
        accessor: (row) => {
          const lockInfo = getSeatLockIcon(row.status, row.is_seat_blocked);
          if (!lockInfo) {
            return <span className="text-xs text-neutral-400 font-medium">–</span>;
          }
          const IconComponent = lockInfo.icon === 'CircleCheckBig' ? CircleCheckBig : Loader;
          return (
            <span title={lockInfo.label} className={`inline-flex items-center justify-center ${lockInfo.colorClass}`}>
              <IconComponent size={16} />
            </span>
          );
        }
      },
      {
        header: "Aksi",
        key: "aksi",
        accessor: (row) => (
          <div className="flex items-center gap-1">
            <Button 
              variant="ghost" 
              size="sm" 
              onClick={() => navigate(row.status === 'draft' ? `/bookings/${row.id}/edit` : `/bookings/${row.id}`)}
              title={row.status === 'draft' ? "Lanjutkan Draft" : "Detail"}
              className="p-1.5 text-neutral-400 hover:text-neutral-800 hover:bg-neutral-100 rounded-lg"
            >
              <Eye size={16} />
            </Button>
            {row.status === 'draft' && (
              <Button 
                variant="ghost" 
                size="sm" 
                onClick={() => {
                  setSelectedDraft(row);
                  setIsDeleteModalOpen(true);
                }}
                title="Hapus Draft"
                className="p-1.5 text-danger-400 hover:text-danger-700 hover:bg-danger-50 rounded-lg"
              >
                <Trash2 size={16} />
              </Button>
            )}
          </div>
        )
      }
    );

    return cols;
  }, [showBrandColumn, brandsMap, navigate]);

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Booking" 
        actionLabel="Buat Booking Baru"
        onAction={() => navigate("/bookings/new")}
      />

      {error && <Alert variant="error">{error}</Alert>}

      {loading ? (
        <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <div className="space-y-4">
          <DataTable
            columns={columns}
            data={filteredBookings}
            itemsPerPage={15}
            searchPlaceholder="Cari no. booking, nama jamaah, atau paket..."
            emptyMessage={filterBrandId || filterStatus || filterSeat || filterDate ? "Data booking tidak ditemukan" : "Belum ada data booking. Klik \"Buat Booking Baru\" untuk menambahkan."}
            toolbarActions={
              <div className="flex flex-wrap items-center gap-2">
                {showBrandColumn && brands.length > 0 && (
                  <CustomDropdown
                    value={filterBrandId}
                    onChange={(val) => setFilterBrandId(val)}
                    className="!mb-0 w-36"
                    placeholder="Semua Brand"
                    options={[
                      { value: '', label: 'Semua Brand' },
                      ...brands.map(b => ({ value: b.id.toString(), label: b.name }))
                    ]}
                  />
                )}

                <CustomDropdown
                  value={filterStatus}
                  onChange={(val) => setFilterStatus(val)}
                  className="!mb-0 w-36"
                  placeholder="Semua Status"
                  options={[
                    { value: '', label: 'Semua Status' },
                    { value: 'draft', label: 'Draft' },
                    { value: 'baru', label: 'Baru' },
                    { value: 'dp', label: 'DP' },
                    { value: 'lunas', label: 'Lunas' },
                    { value: 'batal', label: 'Batal' }
                  ]}
                />

                <CustomDropdown
                  value={filterSeat}
                  onChange={(val) => setFilterSeat(val)}
                  className="!mb-0 w-36"
                  placeholder="Semua Seat"
                  options={[
                    { value: '', label: 'Semua Seat' },
                    { value: 'blocked', label: 'Seat Terkunci' },
                    { value: 'unblocked', label: 'Seat Belum Terkunci' }
                  ]}
                />

                <CustomDropdown
                  value={filterDate}
                  onChange={(val) => setFilterDate(val)}
                  className="!mb-0 w-40"
                  placeholder="Semua Tanggal"
                  options={[
                    { value: '', label: 'Semua Tanggal' },
                    ...availableDates.map(d => ({ value: d.value, label: d.label }))
                  ]}
                />
              </div>
            }
          />
        </div>
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

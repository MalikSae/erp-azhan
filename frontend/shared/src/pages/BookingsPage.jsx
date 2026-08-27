import { useState, useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { listBookings } from "../api/bookings";
import { listBrands } from "../api/brands";
import { getStatusBadgeConfig, getSeatLockIcon } from "../utils/bookingStatus";
import PageHeader from "../components/ui/PageHeader";
import DataTable from "../components/ui/DataTable";
import Button from "../components/ui/Button";
import Badge from "../components/ui/Badge";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import CustomDropdown from "../components/ui/CustomDropdown";
import BrandCell from "../components/BrandCell";
import { Eye, Loader, CircleCheckBig } from "lucide-react";

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

export const BookingsPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();
  const [data, setData] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

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

      // 2. Filter Seat (is_seat_blocked)
      if (filterSeat) {
        if (filterSeat === 'blocked' && !b.is_seat_blocked) return false;
        if (filterSeat === 'unblocked' && b.is_seat_blocked) return false;
      }

      // 3. Filter Specific Departure Date
      if (filterDate) {
        if (!b.berangkat_tanggal) return false;
        const d = new Date(b.berangkat_tanggal);
        if (isNaN(d.getTime())) return false;
        const val = d.toISOString().split('T')[0];
        if (val !== filterDate) return false;
      }

      return true;
    });
  }, [data, showBrandColumn, filterBrandId, filterStatus, filterSeat, filterDate]);

  const columns = useMemo(() => {
    const cols = [
      {
        header: "ID Booking",
        key: "id_booking",
        accessor: (row) => (
          <button
            type="button"
            onClick={() => navigate(`/bookings/${row.id}`)}
            className="font-mono text-sm font-semibold text-neutral-900 hover:text-primary-600 hover:underline transition-colors"
          >
            {row.id_booking || `ID: ${row.id}`}
          </button>
        )
      },
      { 
        header: "Jamaah", 
        key: "nama_jamaah",
        accessor: (row) => (
          row.jamaah_id ? (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                navigate(`/jamaah/${row.jamaah_id}`);
              }}
              className="font-medium text-neutral-900 hover:text-primary-600 hover:underline text-left transition-colors font-body"
              title="Lihat Detail Jamaah"
            >
              {row.nama_jamaah || row.jamaah?.nama_lengkap || `ID: ${row.jamaah_id}`}
            </button>
          ) : (
            <span className="text-neutral-700 font-body">{row.nama_jamaah || "-"}</span>
          )
        )
      },
    ];

    if (showBrandColumn) {
      cols.push({
        header: "Brand",
        key: "brand_id",
        accessor: (row) => {
          const brand = brandsMap[row.brand_id];
          return <BrandCell brand={brand} brandId={row.brand_id} />;
        }
      });
    }

    cols.push(
      { 
        header: "Paket", 
        key: "jadwal_nama", 
        accessor: (row) => row.jadwal_nama || row.schedule?.jadwal_nama || `ID: ${row.schedule_id}` 
      },
      { 
        header: "Tgl Berangkat", 
        key: "berangkat_tanggal", 
        accessor: (row) => formatTanggal(row.berangkat_tanggal) 
      },
      { 
        header: "Kamar", 
        key: "room_type", 
        accessor: (row) => (row.room_type || row.tipe_kamar || '-').toUpperCase() 
      },
      { 
        header: "Total Harga", 
        key: "total_harga", 
        accessor: (row) => formatRupiah(row.total_harga) 
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
        key: "seat",
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
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={() => navigate(`/bookings/${row.id}`)}
            title="Detail"
          >
            <Eye size={16} className="text-primary-600" />
          </Button>
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
            searchPlaceholder="Cari data..."
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
    </div>
  );
};

export default BookingsPage;

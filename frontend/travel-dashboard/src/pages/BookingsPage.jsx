import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { listBookings } from "../api/bookings";
import PageHeader from "../components/ui/PageHeader";
import Table from "../components/ui/Table";
import Button from "../components/ui/Button";
import Badge from "../components/ui/Badge";
import Alert from "../components/ui/Alert";
import { Eye } from "lucide-react";

const formatRupiah = (angka) => {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(angka);
};

const getStatusBadge = (status) => {
  switch (status) {
    case 'baru': return <Badge variant="warning">Baru</Badge>;
    case 'dp': return <Badge variant="primary">Seat Blocked</Badge>;
    case 'lunas': return <Badge variant="success">Lunas</Badge>;
    case 'batal': return <Badge variant="archived">Batal</Badge>;
    default: return <Badge>{status}</Badge>;
  }
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

const BookingsPage = () => {
  const navigate = useNavigate();
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const res = await listBookings();
      setData(res || []);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memuat data booking");
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    { 
      header: "Jamaah", 
      accessor: (row) => (
        row.jamaah_id ? (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/jamaah/${row.jamaah_id}`);
            }}
            className="font-medium text-neutral-900 hover:text-primary-600 hover:underline text-left transition-colors"
            title="Lihat Detail Jamaah"
          >
            {row.nama_jamaah || row.jamaah?.nama_lengkap || `ID: ${row.jamaah_id}`}
          </button>
        ) : (
          <span className="text-neutral-700">{row.nama_jamaah || "-"}</span>
        )
      )
    },
    { header: "Paket", accessor: (row) => row.jadwal_nama || row.schedule?.jadwal_nama || `ID: ${row.schedule_id}` },
    { header: "Tgl Berangkat", accessor: (row) => formatTanggal(row.berangkat_tanggal) },
    { header: "Kamar", accessor: (row) => row.room_type || row.tipe_kamar },
    { header: "Total Harga", accessor: (row) => formatRupiah(row.total_harga) },
    { header: "Status", accessor: (row) => getStatusBadge(row.status) },
    {
      header: "Aksi",
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
  ];

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Booking" 
        actionLabel="Buat Booking Baru"
        onAction={() => navigate("/bookings/new")}
      />

      {error && <Alert variant="error" message={error} onClose={() => setError(null)} />}

      <div className="bg-white rounded-lg shadow-sm overflow-x-auto">
        <Table 
          columns={columns}
          data={data}
          loading={loading}
          emptyMessage="Belum ada data booking"
        />
      </div>
    </div>
  );
};

export default BookingsPage;

import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { listJamaah, deleteJamaah } from "../api/jamaah";
import PageHeader from "../components/ui/PageHeader";
import DataTable from "../components/ui/DataTable";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import Modal from "../components/ui/Modal";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import { Eye, Trash2 } from "lucide-react";

const JamaahPage = () => {
  const navigate = useNavigate();
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState(null);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const res = await listJamaah();
      setData(res || []);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memuat data jamaah");
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteClick = (item) => {
    setSelectedItem(item);
    setIsDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    try {
      await deleteJamaah(selectedItem.id);
      setIsDeleteModalOpen(false);
      setSelectedItem(null);
      fetchData();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menghapus data");
      setIsDeleteModalOpen(false);
    }
  };

  const columns = [
    {
      header: "ID Jamaah",
      key: "id_jamaah",
      accessor: (row) => (
        <button
          type="button"
          onClick={() => navigate(`/jamaah/${row.id}`)}
          className="font-mono text-xs font-bold text-primary-700 bg-primary-50 px-2 py-0.5 rounded border border-primary-200 hover:bg-primary-100 transition-colors"
          title="Lihat Detail Jamaah"
        >
          {row.id_jamaah || "-"}
        </button>
      ),
    },
    { 
      header: "Nama Lengkap", 
      key: "nama_lengkap",
      accessor: (row) => (
        <button
          type="button"
          onClick={() => navigate(`/jamaah/${row.id}`)}
          className="font-medium text-neutral-900 hover:text-primary-600 hover:underline text-left transition-colors"
          title="Lihat Detail Jamaah"
        >
          {row.nama_lengkap}
        </button>
      )
    },
    { header: "NIK", key: "nik", accessor: "nik" },
    { header: "No HP", key: "no_hp", accessor: "no_hp" },
    {
      header: "Aksi",
      key: "aksi",
      accessor: (row) => (
        <div className="flex gap-2">
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={() => navigate(`/jamaah/${row.id}`)}
            title="Lihat Detail"
          >
            <Eye size={16} className="text-primary-600" />
          </Button>
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={() => handleDeleteClick(row)}
            title="Hapus"
          >
            <Trash2 size={16} className="text-danger-600" />
          </Button>
        </div>
      )
    }
  ];

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Jamaah" 
        actionLabel="+ Tambah Jamaah"
        onAction={() => navigate("/jamaah/new")}
      />

      {error && <Alert variant="error" message={error} onClose={() => setError(null)} />}

      {loading ? (
        <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <div className="space-y-4">
          <DataTable 
            columns={columns}
            data={data}
            itemsPerPage={15}
            searchPlaceholder="Cari data..."
            emptyMessage='Belum ada data jamaah. Klik "+ Tambah Jamaah" untuk menambahkan.'
          />
        </div>
      )}

      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => setIsDeleteModalOpen(false)}
        title="Konfirmasi Hapus"
      >
        <p className="text-neutral-600 mb-6 font-body text-sm">
          Apakah Anda yakin ingin menghapus jamaah <strong>{selectedItem?.nama_lengkap}</strong>?
          Data yang dihapus tidak dapat dikembalikan.
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="ghost" onClick={() => setIsDeleteModalOpen(false)}>Batal</Button>
          <Button variant="danger" onClick={confirmDelete}>Hapus</Button>
        </div>
      </Modal>
    </div>
  );
};

export default JamaahPage;

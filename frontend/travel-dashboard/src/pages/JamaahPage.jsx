import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { listJamaah, deleteJamaah } from "../api/jamaah";
import PageHeader from "../components/ui/PageHeader";
import Table from "../components/ui/Table";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import Modal from "../components/ui/Modal";
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
      accessor: (row) => (
        <span className="font-mono text-xs font-bold text-primary-700 bg-primary-50 px-2 py-0.5 rounded border border-primary-200">
          {row.id_jamaah || "-"}
        </span>
      ),
    },
    { header: "Nama Lengkap", accessor: "nama_lengkap" },
    { header: "NIK", accessor: "nik" },
    { header: "No HP", accessor: "no_hp" },
    {
      header: "Aksi",
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

      <div className="bg-white rounded-lg shadow-sm overflow-x-auto">
        <Table 
          columns={columns}
          data={data}
          loading={loading}
          emptyMessage="Belum ada data jamaah"
        />
      </div>

      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => setIsDeleteModalOpen(false)}
        title="Konfirmasi Hapus"
      >
        <p className="text-neutral-600 mb-6">
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

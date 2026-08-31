import { useState, useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { listJamaah, deleteJamaah } from "../api/jamaah";
import { listBrands } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import DataTable from "../components/ui/DataTable";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import Modal from "../components/ui/Modal";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import CustomDropdown from "../components/ui/CustomDropdown";
import BrandCell from "../components/BrandCell";
import { Eye, Trash2 } from "lucide-react";

export const JamaahPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();
  const [data, setData] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const [filterBrandId, setFilterBrandId] = useState("");
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState(null);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      if (showBrandColumn) {
        const [resJamaah, resBrands] = await Promise.all([
          listJamaah(),
          listBrands()
        ]);
        setData(resJamaah || []);
        const bList = resBrands || [];
        setBrands(bList);
        const bMap = {};
        bList.forEach(b => { bMap[b.id] = b; });
        setBrandsMap(bMap);
      } else {
        const resJamaah = await listJamaah();
        setData(resJamaah || []);
      }
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

  const filteredData = useMemo(() => {
    return data.filter(j => {
      if (showBrandColumn && filterBrandId && j.brand_id?.toString() !== filterBrandId) return false;
      return true;
    });
  }, [data, showBrandColumn, filterBrandId]);

  const columns = useMemo(() => {
    const cols = [];

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
        header: "ID Jamaah",
        key: "id_jamaah",
        accessor: (row) => (
          <button
            type="button"
            onClick={() => navigate(`/jamaah/${row.id}`)}
            className="font-mono text-xs font-bold text-neutral-800 bg-neutral-100 px-2.5 py-1 rounded-lg border border-neutral-200/90 hover:bg-neutral-200/80 hover:border-neutral-300 transition-colors cursor-pointer"
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
            className="font-semibold text-neutral-900 hover:text-neutral-600 hover:underline text-left transition-colors font-body"
            title="Lihat Detail Jamaah"
          >
            {row.nama_lengkap}
          </button>
        )
      },
      { header: "NIK", key: "nik", accessor: "nik" },
      { header: "No HP", key: "no_hp", accessor: "no_hp" }
    );

    cols.push({
      header: "Aksi",
      key: "aksi",
      accessor: (row) => (
        <div className="flex gap-1">
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={() => navigate(`/jamaah/${row.id}`)}
            title="Lihat Detail"
            className="p-1.5 text-neutral-400 hover:text-neutral-800 hover:bg-neutral-100 rounded-lg"
          >
            <Eye size={16} />
          </Button>
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={() => handleDeleteClick(row)}
            title="Hapus Jamaah"
            className="p-1.5 text-danger-400 hover:text-danger-700 hover:bg-danger-50 rounded-lg"
          >
            <Trash2 size={16} />
          </Button>
        </div>
      )
    });

    return cols;
  }, [showBrandColumn, brandsMap, navigate]);

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Jamaah" 
        actionLabel="+ Tambah Jamaah"
        onAction={() => navigate("/jamaah/new")}
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
            data={filteredData}
            itemsPerPage={15}
            searchPlaceholder="Cari nama, NIK, atau nomor HP..."
            emptyMessage='Belum ada data jamaah. Klik "+ Tambah Jamaah" untuk menambahkan.'
            toolbarActions={
              showBrandColumn && brands.length > 0 ? (
                <div className="flex flex-wrap items-center gap-2">
                  <CustomDropdown
                    value={filterBrandId}
                    onChange={(val) => setFilterBrandId(val)}
                    options={[
                      { value: '', label: 'Semua Brand' },
                      ...brands.map(b => ({ value: b.id.toString(), label: b.name }))
                    ]}
                    placeholder="Semua Brand"
                    className="!mb-0 min-w-40"
                  />
                </div>
              ) : null
            }
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

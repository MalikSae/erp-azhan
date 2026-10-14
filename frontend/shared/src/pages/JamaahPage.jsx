import { useState, useEffect, useMemo } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Eye, Trash2 } from "lucide-react";
import { listJamaah, deleteJamaah } from "../api/jamaah";
import { listBrands } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import DataTable from "../components/ui/DataTable";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import Modal from "../components/ui/Modal";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import CustomDropdown from "../components/ui/CustomDropdown";
import ActionMenu from "../components/ui/ActionMenu";

const terikatAgen = (j) => j.kaitan_status === "terikat_agen" && !!j.direkrut_oleh_nama;

// Chip brand: warna brand hanya penanda (dot), bukan warna UI.
const BrandChip = ({ brand, brandId }) => (
  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[12px] font-medium font-body bg-white border border-neutral-200 text-neutral-700 whitespace-nowrap">
    <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: brand?.primary_color || "#9CA3AF" }} />
    {brand?.name || `Brand #${brandId}`}
  </span>
);

const ProgressCell = ({ value, title }) => {
  const pct = value || 0;
  const color = pct === 100 ? "bg-success-500" : pct >= 50 ? "bg-warning-500" : "bg-danger-500";
  return (
    <div className="flex items-center gap-2" title={title}>
      <div className="w-16 h-1.5 bg-neutral-100 rounded-full overflow-hidden">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs font-body text-neutral-600 w-8">{pct}%</span>
    </div>
  );
};

export const JamaahPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [data, setData] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);

  const [filterBrandId, setFilterBrandId] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState(null);

  useEffect(() => {
    fetchData();
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, "");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
      setSuccessMessage(`Jamaah "${selectedItem.nama_lengkap}" berhasil dihapus.`);
      setSelectedItem(null);
      fetchData();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menghapus data");
      setIsDeleteModalOpen(false);
    }
  };

  // Ringkasan mengikuti filter brand, supaya angkanya cocok dengan tabel.
  const baseData = useMemo(
    () => data.filter(j => !(showBrandColumn && filterBrandId && j.brand_id?.toString() !== filterBrandId)),
    [data, showBrandColumn, filterBrandId]
  );

  // Hitungan ditampilkan di label filter (bukan kartu KPI).
  const statusOptions = useMemo(() => [
    { value: "", label: `Semua Jamaah (${baseData.length})` },
    { value: "data", label: `Data belum lengkap (${baseData.filter(j => j.pct_data !== 100).length})` },
    { value: "dokumen", label: `Dokumen belum lengkap (${baseData.filter(j => j.pct_dokumen !== 100).length})` },
    { value: "tanpa_agen", label: `Tanpa agen (${baseData.filter(j => !terikatAgen(j)).length})` },
  ], [baseData]);

  const rows = useMemo(() => baseData
    .filter(j => {
      if (filterStatus === "data") return j.pct_data !== 100;
      if (filterStatus === "dokumen") return j.pct_dokumen !== 100;
      if (filterStatus === "tanpa_agen") return !terikatAgen(j);
      return true;
    })
    // Kolom bantu pencarian: nama brand ikut bisa dicari.
    .map(j => ({ ...j, _cari: brandsMap[j.brand_id]?.name || "" })), [baseData, filterStatus, brandsMap]);

  const columns = useMemo(() => {
    const cols = [{ header: "Jamaah", key: "nama_lengkap", sortable: true }];
    if (showBrandColumn) cols.push({ header: "Brand", key: "brand_id" });
    cols.push(
      { header: "Kontak", key: "no_hp" },
      { header: "Agen", key: "direkrut_oleh_nama", sortable: true },
      { header: "Data", key: "pct_data", sortable: true, sortFn: (a, b) => (a.pct_data || 0) - (b.pct_data || 0) },
      { header: "Dokumen", key: "pct_dokumen", sortable: true, sortFn: (a, b) => (a.pct_dokumen || 0) - (b.pct_dokumen || 0) },
      { header: "Aksi", key: "aksi" },
    );
    return cols;
  }, [showBrandColumn]);

  const renderCell = (row, key) => {
    if (key === "nama_lengkap") {
      const gender = row.jenis_kelamin === "L" ? "Laki-laki" : row.jenis_kelamin === "P" ? "Perempuan" : "";
      const sub = [row.id_jamaah, gender].filter(Boolean).join(" · ");
      return (
        <div className="min-w-0">
          <button
            type="button"
            onClick={() => navigate(`/jamaah/${row.id}`)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
            title="Lihat Detail Jamaah"
          >
            {row.nama_lengkap}
          </button>
          {sub && <div className="mt-0.5 text-[11px] font-body text-neutral-400">{sub}</div>}
        </div>
      );
    }

    if (key === "brand_id") return <BrandChip brand={brandsMap[row.brand_id]} brandId={row.brand_id} />;

    if (key === "no_hp") {
      return (
        <div className="min-w-0">
          <div className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.no_hp || "-"}</div>
          {row.nik && <div className="mt-0.5 text-[11px] font-body text-neutral-400">NIK {row.nik}</div>}
        </div>
      );
    }

    if (key === "direkrut_oleh_nama") {
      return terikatAgen(row)
        ? <span className="text-[13px] font-body font-medium text-neutral-800 whitespace-nowrap">{row.direkrut_oleh_nama}</span>
        : <span className="text-[13px] font-body text-neutral-400 whitespace-nowrap">{row.kaitan_status === "tanpa_agen" ? "Tanpa agen" : "Belum ditentukan"}</span>;
    }

    if (key === "pct_data") return <ProgressCell value={row.pct_data} title="Kelengkapan Data Profil" />;
    if (key === "pct_dokumen") return <ProgressCell value={row.pct_dokumen} title="Kelengkapan Dokumen Wajib" />;

    if (key === "aksi") {
      const menuItems = [
        { label: "Lihat", icon: Eye, tooltip: "Lihat Detail Jamaah", onClick: () => navigate(`/jamaah/${row.id}`) },
        { label: "Hapus", icon: Trash2, danger: true, tooltip: "Hapus Jamaah", onClick: () => handleDeleteClick(row) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Jamaah"
        actionLabel="+ Tambah Jamaah"
        onAction={() => navigate("/jamaah/new")}
      />

      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}
      {error && <Alert variant="error" onClose={() => setError(null)}>{error}</Alert>}

      {loading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={15}
          searchPlaceholder="Cari nama, NIK, atau nomor HP..."
          emptyMessage='Belum ada data jamaah. Klik "+ Tambah Jamaah" untuk menambahkan.'
          toolbarActions={
            <div className="flex flex-wrap items-center gap-2">
              <CustomDropdown
                value={filterStatus}
                onChange={(val) => setFilterStatus(val)}
                options={statusOptions}
                placeholder="Semua Jamaah"
                className="!mb-0 w-60"
              />
              {showBrandColumn && brands.length > 0 && (
                <CustomDropdown
                  value={filterBrandId}
                  onChange={(val) => setFilterBrandId(val)}
                  options={[
                    { value: "", label: "Semua Brand" },
                    ...brands.map(b => ({ value: b.id.toString(), label: b.name }))
                  ]}
                  placeholder="Semua Brand"
                  className="!mb-0 w-40"
                />
              )}
            </div>
          }
          renderCell={renderCell}
        />
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

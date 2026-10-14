import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Pencil, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import ActionMenu from 'shared/ui/ActionMenu';
import { listBrands, deleteBrand } from '../api/brands';
import { BrandCell } from 'shared';

const ColorCell = ({ color }) => {
  if (!color) return <span className="text-xs text-neutral-400 font-body">-</span>;
  return (
    <div className="flex items-center gap-2">
      <div
        className="w-4 h-4 rounded-full border border-neutral-200 shrink-0"
        style={{ backgroundColor: color }}
      />
      <span className="text-[12px] font-mono uppercase text-neutral-600">{color}</span>
    </div>
  );
};

const BrandsPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [successMessage, setSuccessMessage] = useState(null);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchBrands = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listBrands();
      setBrands(data || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data brand.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchBrands();
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, '');
    }
  }, []);

  const confirmDelete = (id) => {
    setDeleteConfirmId(id);
  };

  const cancelDelete = () => {
    setDeleteConfirmId(null);
  };

  const handleDelete = async () => {
    try {
      await deleteBrand(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchBrands();
    } catch (error) {
      setDeleteConfirmId(null);
      if (error.response?.status === 409) {
        setErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, masih dipakai.");
      } else {
        setErrorMessage("Terjadi kesalahan saat menghapus data.");
      }
    }
  };

  const brandToDelete = brands.find(b => b.id === deleteConfirmId);

  const columns = [
    { header: 'Brand', key: 'name', sortable: true },
    { header: 'Domain', key: 'domain', sortable: true },
    { header: 'Warna', key: 'primary_color' },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderCell = (row, key) => {
    if (key === 'name') {
      return (
        <div className="flex items-center gap-3 min-w-0">
          <BrandCell key={row.icon_url || row.id} brand={row} />
          <div className="min-w-0">
            <button
              type="button"
              onClick={() => navigate(`/brands/${row.id}/edit`)}
              className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
            >
              {row.name}
            </button>
            <div className="text-[11px] font-body text-neutral-400">
              Kode <span className="font-mono">{row.kode_brand || '-'}</span>
            </div>
          </div>
        </div>
      );
    }
    if (key === 'domain') {
      return row.domain
        ? <span className="text-[13px] font-mono text-neutral-700">{row.domain}</span>
        : <span className="text-xs text-neutral-400 font-body">-</span>;
    }
    if (key === 'primary_color') return <ColorCell color={row.primary_color} />;
    if (key === 'aksi') {
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Brand', onClick: () => navigate(`/brands/${row.id}/edit`) },
        { label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus Brand', onClick: () => confirmDelete(row.id) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }
    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Brand"
        actionLabel="+ Tambah Brand"
        onAction={() => navigate("/brands/new")}
      />

      {successMessage && (
        <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>
      )}

      {errorMessage && (
        <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={brands}
          itemsPerPage={10}
          emptyMessage='Belum ada brand. Klik "+ Tambah Brand" untuk menambahkan.'
          searchPlaceholder="Cari nama brand, kode, atau domain..."
          renderCell={renderCell}
        />
      )}

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteConfirmId !== null}
        onClose={cancelDelete}
        title="Hapus Brand?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={cancelDelete}>Batal</Button>
            <Button variant="danger" onClick={handleDelete}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus {brandToDelete?.name}? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>
    </div>
  );
};

export default BrandsPage;

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listBrands, deleteBrand } from '../api/brands';

const LogoCell = ({ url, name }) => {
  const [error, setError] = React.useState(false);
  
  const placeholder = (
    <div className="w-8 h-8 bg-neutral-100 rounded border border-neutral-200 flex items-center justify-center text-neutral-400 text-xs font-medium">
      —
    </div>
  );
  
  if (!url || error) return placeholder;
  
  return (
    <img 
      src={url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL}${url}` : url} 
      alt={`${name} logo`}
      className="w-8 h-8 object-contain rounded bg-white"
      onError={() => setError(true)}
    />
  );
};

const ColorCell = ({ color }) => {
  if (!color) return <span>-</span>;
  return (
    <div className="flex items-center gap-2">
      <div 
        className="w-4 h-4 rounded border border-neutral-200" 
        style={{ backgroundColor: color }}
      />
      <span className="text-sm font-medium uppercase text-neutral-700">{color}</span>
    </div>
  );
};

const BrandsPage = () => {
  const navigate = useNavigate();
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchBrands = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listBrands();
      setBrands(data);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data brand.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchBrands();
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

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Brand" 
        actionLabel="+ Tambah Brand" 
        onAction={() => navigate("/brands/new")}
      />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={[
            { header: 'Logo', key: 'logo_url' },
            { header: 'Kode', key: 'kode_brand' },
            { header: 'Nama Brand', key: 'name' },
            { header: 'Domain', key: 'domain' },
            { header: 'Warna', key: 'primary_color' },
            { header: 'Aksi', key: 'aksi' },
          ]}
          data={brands}
          itemsPerPage={10}
          emptyMessage='Belum ada brand. Klik "+ Tambah Brand" untuk menambahkan.'
          renderCell={(row, key) => {
            if (key === 'logo_url') {
              return <LogoCell url={row.logo_url} name={row.name} />;
            }
            if (key === 'kode_brand') {
              return row.kode_brand ? (
                <span className="font-mono text-xs font-bold text-neutral-800 bg-neutral-100 border border-neutral-200/90 px-2 py-0.5 rounded-lg">
                  {row.kode_brand}
                </span>
              ) : (
                <span className="text-neutral-400 font-mono text-xs">-</span>
              );
            }
            if (key === 'domain') {
              return row.domain ? (
                <span className="font-mono text-xs text-neutral-700">{row.domain}</span>
              ) : (
                <span className="text-neutral-400">-</span>
              );
            }
            if (key === 'primary_color') {
              return <ColorCell color={row.primary_color} />;
            }
            if (key === 'aksi') return (
              <div className="flex gap-3">
                <button 
                  onClick={() => navigate(`/brands/${row.id}/edit`)} 
                  title="Edit"
                  className="text-neutral-400 hover:text-neutral-700 transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                  </svg>
                </button>
                <button 
                  onClick={() => confirmDelete(row.id)} 
                  title="Hapus"
                  className="text-neutral-400 hover:text-neutral-700 transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            );
            return row[key];
          }}
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

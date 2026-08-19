import React, { useState, useEffect } from 'react';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listAirlines, createAirline, updateAirline, deleteAirline } from '../api/airlines';
import { uploadMedia } from '../api/media';

const initialForm = {
  name: '',
  logo_url: ''
};

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

const AirlinesPage = () => {
  const [airlines, setAirlines] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingAirline, setEditingAirline] = useState(null);
  const [formData, setFormData] = useState(initialForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isUploading, setIsUploading] = useState(false);
  const [localPreview, setLocalPreview] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchAirlines = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listAirlines();
      setAirlines(data);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data maskapai.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchAirlines();
  }, []);

  const handleOpenModal = (airline = null) => {
    if (airline) {
      setEditingAirline(airline);
      setFormData({
        name: airline.name,
        logo_url: airline.logo_url || ''
      });
      setLocalPreview(null);
    } else {
      setEditingAirline(null);
      setFormData(initialForm);
      setLocalPreview(null);
    }
    setFormErrors(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (!isSubmitting) {
      setIsModalOpen(false);
      setEditingAirline(null);
      setFormData(initialForm);
      setLocalPreview(null);
    }
  };

  const handleFileChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    // Local preview
    const objectUrl = URL.createObjectURL(file);
    setLocalPreview(objectUrl);
    setFormErrors(null);
    setIsUploading(true);

    try {
      const url = await uploadMedia(file, 'airline-logos');
      setFormData(prev => ({ ...prev, logo_url: url }));
    } catch (err) {
      console.error(err);
      setFormErrors('Gagal upload logo, coba lagi');
    } finally {
      setIsUploading(false);
    }
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);
    setIsSubmitting(true);

    try {
      const payload = {
        name: formData.name,
        logo_url: formData.logo_url === '' ? null : formData.logo_url
      };

      if (editingAirline) {
        await updateAirline(editingAirline.id, payload);
      } else {
        await createAirline(payload);
      }

      handleCloseModal();
      fetchAirlines();
    } catch (error) {
      const msg = error.response?.data?.error || "Terjadi kesalahan, coba lagi.";
      setFormErrors(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const confirmDelete = (id) => {
    setDeleteConfirmId(id);
  };

  const cancelDelete = () => {
    setDeleteConfirmId(null);
  };

  const handleDelete = async () => {
    try {
      await deleteAirline(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchAirlines();
    } catch (error) {
      setDeleteConfirmId(null);
      if (error.response?.status === 409) {
        setErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, masih dipakai oleh paket lain.");
      } else {
        setErrorMessage("Terjadi kesalahan saat menghapus data.");
      }
    }
  };

  const airlineToDelete = airlines.find(a => a.id === deleteConfirmId);

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Maskapai" 
        actionLabel="+ Tambah Maskapai" 
        onAction={() => handleOpenModal()}
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
            { header: 'Nama Maskapai', key: 'name' },
            { header: 'Aksi', key: 'aksi' },
          ]}
          data={airlines}
          itemsPerPage={10}
          emptyMessage='Belum ada maskapai. Klik "+ Tambah Maskapai" untuk menambahkan.'
          renderCell={(row, key) => {
            if (key === 'logo_url') {
              return <LogoCell url={row.logo_url} name={row.name} />;
            }
            if (key === 'aksi') return (
              <div className="flex gap-3">
                <button 
                  onClick={() => handleOpenModal(row)} 
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

      {/* Form Modal */}
      <Modal 
        isOpen={isModalOpen} 
        onClose={handleCloseModal}
        title={editingAirline ? "Edit Maskapai" : "Tambah Maskapai"}
        size="md"
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <FormField label="Nama Maskapai" required>
            <Input 
              name="name"
              value={formData.name}
              onChange={handleChange}
              required
              placeholder="Contoh: Saudia Airlines"
            />
          </FormField>

          <FormField label="Logo Maskapai (Opsional)">
            {(localPreview || formData.logo_url) && (
              <div className="mb-2 relative inline-block">
                <img 
                  src={localPreview || (formData.logo_url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL}${formData.logo_url}` : formData.logo_url)}
                  alt="Preview"
                  className="h-16 object-contain rounded bg-white border border-neutral-200"
                />
                {(localPreview || formData.logo_url) && (
                  <button
                    type="button"
                    onClick={() => {
                      setLocalPreview(null);
                      setFormData(prev => ({ ...prev, logo_url: '' }));
                    }}
                    className="absolute -top-2 -right-2 bg-danger-600 text-white rounded-full w-5 h-5 flex items-center justify-center shadow-sm text-[10px]"
                  >
                    X
                  </button>
                )}
              </div>
            )}
            
            <Input 
              type="file"
              accept="image/*"
              onChange={handleFileChange}
              disabled={isUploading}
            />
            {isUploading && <p className="text-sm text-neutral-500 mt-1">Mengupload...</p>}
          </FormField>

          {formErrors && (
            <Alert variant="error">{formErrors}</Alert>
          )}

          <div className="pt-4 flex justify-end space-x-3 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={handleCloseModal} disabled={isSubmitting}>
              Batal
            </Button>
            <Button type="submit" variant="primary" isLoading={isSubmitting}>
              {editingAirline ? "Update" : "Simpan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteConfirmId !== null}
        onClose={cancelDelete}
        title="Hapus Maskapai?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={cancelDelete}>Batal</Button>
            <Button variant="danger" onClick={handleDelete}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus {airlineToDelete?.name}? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>
    </div>
  );
};

export default AirlinesPage;

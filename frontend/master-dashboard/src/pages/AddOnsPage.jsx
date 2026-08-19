import React, { useState, useEffect } from 'react';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listAddOns, createAddOn, updateAddOn, deleteAddOn } from '../api/addons';

const initialForm = {
  name: ''
};

const AddOnsPage = () => {
  const [addons, setAddOns] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingAddOn, setEditingAddOn] = useState(null);
  const [formData, setFormData] = useState(initialForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchAddOns = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listAddOns();
      setAddOns(data);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data add-on.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchAddOns();
  }, []);

  const handleOpenModal = (addon = null) => {
    if (addon) {
      setEditingAddOn(addon);
      setFormData({
        name: addon.name
      });
    } else {
      setEditingAddOn(null);
      setFormData(initialForm);
    }
    setFormErrors(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (!isSubmitting) {
      setIsModalOpen(false);
      setEditingAddOn(null);
      setFormData(initialForm);
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
        name: formData.name
      };

      if (editingAddOn) {
        await updateAddOn(editingAddOn.id, payload);
      } else {
        await createAddOn(payload);
      }

      handleCloseModal();
      fetchAddOns();
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
      await deleteAddOn(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchAddOns();
    } catch (error) {
      setDeleteConfirmId(null);
      if (error.response?.status === 409) {
        setErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, masih dipakai oleh paket lain.");
      } else {
        setErrorMessage("Terjadi kesalahan saat menghapus data.");
      }
    }
  };

  const addOnToDelete = addons.find(a => a.id === deleteConfirmId);

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Layanan Tambahan (Add-On)" 
        actionLabel="+ Tambah Add-On" 
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
            { header: 'Nama Layanan/Add-On', key: 'name', sortable: true },
            { header: 'Aksi', key: 'aksi' },
          ]}
          data={addons}
          itemsPerPage={10}
          emptyMessage='Belum ada add-on. Klik "+ Tambah Add-On" untuk menambahkan.'
          renderCell={(row, key) => {
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
        title={editingAddOn ? "Edit Add-On" : "Tambah Add-On"}
        size="md"
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <FormField label="Nama Layanan/Add-On" required>
            <Input 
              name="name"
              value={formData.name}
              onChange={handleChange}
              required
              placeholder="Contoh: Kereta Cepat Haramain"
            />
          </FormField>

          {formErrors && (
            <Alert variant="error">{formErrors}</Alert>
          )}

          <div className="pt-4 flex justify-end space-x-3 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={handleCloseModal} disabled={isSubmitting}>
              Batal
            </Button>
            <Button type="submit" variant="primary" isLoading={isSubmitting}>
              {editingAddOn ? "Update" : "Simpan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteConfirmId !== null}
        onClose={cancelDelete}
        title="Hapus Add-On?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={cancelDelete}>Batal</Button>
            <Button variant="danger" onClick={handleDelete}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus {addOnToDelete?.name}? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>
    </div>
  );
};

export default AddOnsPage;

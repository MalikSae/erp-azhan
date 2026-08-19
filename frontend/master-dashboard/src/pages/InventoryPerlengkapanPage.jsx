import React, { useState, useEffect } from 'react';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import {
  listPerlengkapanItems,
  createPerlengkapanItem,
  updatePerlengkapanItem,
  deletePerlengkapanItem
} from '../api/perlengkapan';

const initialItemForm = {
  nama: '',
  qty_per_set: '0'
};

const InventoryPerlengkapanPage = () => {
  const [items, setItems] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);

  // Modal state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [formData, setFormData] = useState(initialItemForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Delete modal state
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchItems = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listPerlengkapanItems();
      setItems(data || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat katalog item perlengkapan.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchItems();
  }, []);

  const handleOpenModal = (item = null) => {
    if (item) {
      setEditingItem(item);
      setFormData({
        nama: item.nama || '',
        qty_per_set: String(item.qty_per_set ?? 0)
      });
    } else {
      setEditingItem(null);
      setFormData(initialItemForm);
    }
    setFormErrors(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (!isSubmitting) {
      setIsModalOpen(false);
      setEditingItem(null);
      setFormData(initialItemForm);
      setFormErrors(null);
    }
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleSubmitItem = async (e) => {
    e.preventDefault();
    setFormErrors(null);

    const nama = formData.nama.trim();
    if (!nama) {
      setFormErrors("Nama item perlengkapan wajib diisi.");
      return;
    }

    const qty = parseInt(formData.qty_per_set, 10);
    if (isNaN(qty) || qty < 0) {
      setFormErrors("Jumlah per Set harus berupa angka 0 atau lebih.");
      return;
    }

    setIsSubmitting(true);

    try {
      if (editingItem) {
        await updatePerlengkapanItem(editingItem.id, {
          nama,
          qty_per_set: qty
        });
      } else {
        await createPerlengkapanItem({
          nama,
          qty_per_set: qty
        });
      }

      handleCloseModal();
      fetchItems();
    } catch (error) {
      const msg = error.response?.data?.error || "Terjadi kesalahan saat menyimpan item.";
      setFormErrors(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDeleteItem = async () => {
    try {
      await deletePerlengkapanItem(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchItems();
    } catch (error) {
      setDeleteConfirmId(null);
      const msg = error.response?.data?.error || "Gagal menghapus item perlengkapan.";
      setErrorMessage(msg);
    }
  };

  const itemToDelete = items.find(it => it.id === deleteConfirmId);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Perlengkapan"
        actionLabel="+ Tambah Item"
        onAction={() => handleOpenModal()}
      />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-pure-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={[
            { header: 'Nama Item', key: 'nama' },
            { header: 'Set/Pax', key: 'qty_per_set' },
            { header: 'Aksi', key: 'aksi' },
          ]}
          data={items}
          itemsPerPage={10}
          emptyMessage='Belum ada item perlengkapan di katalog. Klik "+ Tambah Item" untuk menambahkan.'
          renderCell={(row, key) => {
            if (key === 'nama') {
              return (
                <span className="font-heading font-medium text-neutral-900 text-sm">
                  {row.nama}
                </span>
              );
            }
            if (key === 'qty_per_set') {
              if (!row.qty_per_set || row.qty_per_set === 0) {
                return <span className="text-xs text-neutral-400 font-body">Tidak termasuk set</span>;
              }
              return (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-primary-50 text-primary-700 border border-primary-200 font-body">
                  {row.qty_per_set} pcs / pax
                </span>
              );
            }
            if (key === 'aksi') {
              return (
                <div className="flex gap-3 items-center">
                  <button 
                    onClick={() => handleOpenModal(row)} 
                    title="Edit"
                    className="text-neutral-400 hover:text-neutral-700 transition-colors p-1 rounded hover:bg-neutral-100"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                    </svg>
                  </button>
                  <button 
                    onClick={() => setDeleteConfirmId(row.id)} 
                    title="Hapus"
                    className="text-neutral-400 hover:text-danger-600 transition-colors p-1 rounded hover:bg-danger-50"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              );
            }
            return row[key];
          }}
        />
      )}

      {/* Modal Tambah / Edit Item */}
      <Modal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        title={editingItem ? "Edit Item Perlengkapan" : "Tambah Item Perlengkapan"}
      >
        <form onSubmit={handleSubmitItem} className="space-y-4">
          {formErrors && (
            <Alert variant="error">{formErrors}</Alert>
          )}

          <FormField label="Nama Item" required>
            <Input
              type="text"
              name="nama"
              value={formData.nama}
              onChange={handleChange}
              placeholder="Contoh: Koper 24 inch, Kain Ihram, Buku Doa"
              autoFocus
              required
            />
          </FormField>

          <FormField
            label="Jumlah per Set"
            hint="Isi 0 jika item ini hanya katalog/tambahan dan tidak masuk ke set standar 1 pax jamaah."
          >
            <Input
              type="number"
              name="qty_per_set"
              min="0"
              step="1"
              value={formData.qty_per_set}
              onChange={handleChange}
              placeholder="0"
              required
            />
          </FormField>

          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="secondary"
              onClick={handleCloseModal}
              disabled={isSubmitting}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={isSubmitting}
            >
              {isSubmitting ? "Menyimpan..." : "Simpan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Konfirmasi Hapus */}
      <Modal
        isOpen={!!deleteConfirmId}
        onClose={() => setDeleteConfirmId(null)}
        title="Konfirmasi Hapus Item"
      >
        <div className="space-y-4">
          <p className="text-sm font-body text-neutral-700">
            Apakah Anda yakin ingin menghapus item <strong className="text-neutral-900 font-heading">{itemToDelete?.nama}</strong> dari katalog perlengkapan?
          </p>
          <p className="text-xs font-body text-neutral-500">
            Catatan: Item tidak dapat dihapus jika masih masuk dalam set perlengkapan atau masih memiliki stok di brand manapun.
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setDeleteConfirmId(null)}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={handleDeleteItem}
            >
              Hapus
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default InventoryPerlengkapanPage;

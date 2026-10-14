import React, { useState, useEffect, useMemo } from 'react';
import { Boxes, CheckCircle2, Package, Pencil, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import FormField from 'shared/ui/FormField';
import Input from 'shared/ui/Input';
import FileInput from 'shared/ui/FileInput';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import StatTile from 'shared/ui/StatTile';
import ActionMenu from 'shared/ui/ActionMenu';
import CustomDropdown from 'shared/ui/CustomDropdown';
import {
  listPerlengkapanItems,
  createPerlengkapanItem,
  updatePerlengkapanItem,
  deletePerlengkapanItem
} from '../api/perlengkapan';
import { uploadMedia } from '../api/media';
import ItemPhoto from '../components/ItemPhoto';


const initialItemForm = {
  nama: '',
  photo_url: '',
  qty_per_set: '0'
};

const InventoryPerlengkapanPage = () => {
  const [items, setItems] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [filter, setFilter] = useState('');

  // Modal state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [formData, setFormData] = useState(initialItemForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isUploadingPhoto, setIsUploadingPhoto] = useState(false);
  const [photoPreview, setPhotoPreview] = useState(null);

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
        photo_url: item.photo_url || '',
        qty_per_set: String(item.qty_per_set ?? 0)
      });
    } else {
      setEditingItem(null);
      setFormData(initialItemForm);
    }
    setPhotoPreview(null);
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

  const handlePhotoChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    setPhotoPreview(URL.createObjectURL(file));
    setIsUploadingPhoto(true);
    setFormErrors(null);
    try {
      const url = await uploadMedia(file, 'perlengkapan');
      setFormData(prev => ({ ...prev, photo_url: url }));
    } catch (error) {
      setFormErrors(error.response?.data?.error || 'Gagal mengunggah foto item.');
      setPhotoPreview(null);
    } finally {
      setIsUploadingPhoto(false);
    }
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
          photo_url: formData.photo_url || null,
          qty_per_set: qty
        });
      } else {
        await createPerlengkapanItem({
          nama,
          photo_url: formData.photo_url || null,
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

  const stats = useMemo(() => {
    const dalamSet = items.filter(i => (i.qty_per_set || 0) > 0);
    return {
      total: items.length,
      dalamSet: dalamSet.length,
      katalog: items.length - dalamSet.length,
      isiSet: dalamSet.reduce((sum, i) => sum + i.qty_per_set, 0),
    };
  }, [items]);

  const rows = useMemo(() => items.filter(i => {
    if (filter === 'set') return (i.qty_per_set || 0) > 0;
    if (filter === 'katalog') return !(i.qty_per_set > 0);
    return true;
  }), [items, filter]);

  const columns = [
    { header: 'Item', key: 'nama', sortable: true },
    { header: 'Isi Set Standar', key: 'qty_per_set', sortable: true, sortFn: (a, b) => (a.qty_per_set || 0) - (b.qty_per_set || 0) },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderCell = (row, key) => {
    if (key === 'nama') {
      return (
        <div className="flex items-center gap-3 min-w-0">
          <ItemPhoto url={row.photo_url} name={row.nama} />
          <button
            type="button"
            onClick={() => handleOpenModal(row)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
          >
            {row.nama}
          </button>
        </div>
      );
    }

    if (key === 'qty_per_set') {
      if (!row.qty_per_set || row.qty_per_set === 0) {
        return <span className="text-xs text-neutral-400 font-body">Hanya katalog</span>;
      }
      return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.qty_per_set} pcs / pax</span>;
    }

    if (key === 'aksi') {
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Item', onClick: () => handleOpenModal(row) },
        { label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus Item', onClick: () => setDeleteConfirmId(row.id) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Perlengkapan"
        actionLabel="+ Tambah Item"
        onAction={() => handleOpenModal()}
      />

      {errorMessage && (
        <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>
      )}

      {!isLoading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Boxes} variant="primary" value={stats.total} label="Item" />
          <StatTile icon={CheckCircle2} variant="success" value={stats.dalamSet} label="Dalam set" />
          <StatTile icon={Package} variant="info" value={stats.katalog} label="Hanya katalog" />
          <StatTile icon={Boxes} variant="violet" value={`${stats.isiSet} pcs`} label="Isi set / pax" />
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={10}
          emptyMessage='Belum ada item perlengkapan di katalog. Klik "+ Tambah Item" untuk menambahkan.'
          searchPlaceholder="Cari nama item perlengkapan..."
          toolbarActions={
            <CustomDropdown
              value={filter}
              onChange={(val) => setFilter(val)}
              className="!mb-0 w-44"
              placeholder="Semua Item"
              options={[
                { value: '', label: `Semua Item (${stats.total})` },
                { value: 'set', label: `Dalam set (${stats.dalamSet})` },
                { value: 'katalog', label: `Hanya katalog (${stats.katalog})` },
              ]}
            />
          }
          renderCell={renderCell}
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

          <FileInput
            label="Foto Item"
            accept="image/png,image/jpeg,image/webp"
            value={formData.photo_url}
            previewUrl={photoPreview}
            onChange={handlePhotoChange}
            onRemove={() => {
              setPhotoPreview(null);
              setFormData(prev => ({ ...prev, photo_url: '' }));
            }}
            isUploading={isUploadingPhoto}
            uploadingText="Mengunggah foto..."
            placeholder="Pilih foto item (PNG, JPG, WebP)..."
          />

          <FormField
            label="Jumlah per Set"
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
              disabled={isSubmitting || isUploadingPhoto}
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

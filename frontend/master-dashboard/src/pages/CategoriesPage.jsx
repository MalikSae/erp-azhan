import React, { useState, useEffect } from 'react';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import Input from '../components/ui/Input';
import Toggle from '../components/ui/Toggle';
import Badge from '../components/ui/Badge';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listCategories, createCategory, updateCategory, deleteCategory } from '../api/categories';
import { listBrands } from '../api/brands';

const initialForm = {
  name: '',
  slug: '',
  is_active: true,
  brand_ids: []
};

const CategoriesPage = () => {
  const [categories, setCategories] = useState([]);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState(null);
  const [formData, setFormData] = useState(initialForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [deleteConfirmId, setDeleteConfirmId] = useState(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchData = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const [catData, brandData] = await Promise.all([
        listCategories(),
        listBrands()
      ]);
      setCategories(catData || []);
      setBrands(brandData || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data kategori paket.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleOpenModal = (cat = null) => {
    if (cat) {
      setEditingCategory(cat);
      setFormData({
        name: cat.name || '',
        slug: cat.slug || '',
        is_active: cat.is_active !== undefined ? cat.is_active : true,
        brand_ids: cat.brands ? cat.brands.map(b => b.id) : []
      });
    } else {
      setEditingCategory(null);
      setFormData(initialForm);
    }
    setFormErrors(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (!isSubmitting) {
      setIsModalOpen(false);
      setEditingCategory(null);
      setFormData(initialForm);
    }
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const handleBrandCheckboxChange = (brandId, checked) => {
    setFormData(prev => {
      const current = [...prev.brand_ids];
      if (checked) {
        if (!current.includes(brandId)) current.push(brandId);
      } else {
        const idx = current.indexOf(brandId);
        if (idx > -1) current.splice(idx, 1);
      }
      return { ...prev, brand_ids: current };
    });
  };

  const handleSelectAllBrands = (select) => {
    setFormData(prev => ({
      ...prev,
      brand_ids: select ? brands.map(b => b.id) : []
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);
    setIsSubmitting(true);

    try {
      const payload = {
        name: formData.name.trim(),
        slug: formData.slug.trim(),
        is_active: formData.is_active,
        brand_ids: formData.brand_ids
      };

      if (editingCategory) {
        await updateCategory(editingCategory.id, payload);
      } else {
        await createCategory(payload);
      }

      handleCloseModal();
      fetchData();
    } catch (error) {
      const msg = error.response?.data?.error || "Terjadi kesalahan saat menyimpan kategori.";
      setFormErrors(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteConfirmId) return;
    setIsDeleting(true);
    try {
      await deleteCategory(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchData();
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal menghapus kategori.";
      setErrorMessage(msg);
      setDeleteConfirmId(null);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Kategori Paket"
        actionLabel="+ Tambah Kategori"
        onAction={() => handleOpenModal()}
      />

      {errorMessage && (
        <Alert variant="error" onClose={() => setErrorMessage(null)}>
          {errorMessage}
        </Alert>
      )}

      {isLoading ? (
        <div className="py-12 flex justify-center bg-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={[
            { header: 'Brand Berlaku', key: 'brands' },
            { header: 'Kategori', key: 'name', sortable: true },
            { header: 'Jumlah Paket', key: 'package_count', sortable: true },
            { header: 'Status', key: 'is_active', sortable: true },
            { header: 'Aksi', key: 'aksi' },
          ]}
          data={categories}
          emptyMessage='Belum ada data kategori paket. Klik "+ Tambah Kategori" untuk menambahkan.'
          searchPlaceholder="Cari nama atau slug kategori..."
          renderCell={(row, key) => {
            if (key === 'name') {
              return (
                <div>
                  <div className="font-semibold text-neutral-900 font-heading text-sm">{row.name}</div>
                  <div className="text-xs text-neutral-400 font-mono">/{row.slug}</div>
                </div>
              );
            }
            if (key === 'brands') {
              return (
                <div className="flex flex-wrap gap-1.5 max-w-sm">
                  {row.brands && row.brands.length > 0 ? (
                    row.brands.map(b => {
                      const brandColor = b.primary_color || (brands.find(item => item.id === b.id)?.primary_color);
                      return (
                        <span
                          key={b.id}
                          className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold border shadow-2xs"
                          style={{
                            backgroundColor: brandColor ? `${brandColor}15` : '#F4F4F5',
                            color: brandColor || '#27272A',
                            borderColor: brandColor ? `${brandColor}40` : '#E4E4E7'
                          }}
                        >
                          {brandColor && (
                            <span
                              className="w-2 h-2 rounded-full flex-shrink-0"
                              style={{ backgroundColor: brandColor }}
                            />
                          )}
                          <span>{b.name}</span>
                        </span>
                      );
                    })
                  ) : (
                    <span className="text-xs text-neutral-400 italic">Semua Brand</span>
                  )}
                </div>
              );
            }
            if (key === 'package_count') {
              return (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-neutral-100 text-neutral-700">
                  {row.package_count || 0} Paket
                </span>
              );
            }
            if (key === 'is_active') {
              return (
                <Badge variant={row.is_active ? 'published' : 'archived'}>
                  {row.is_active ? 'Aktif' : 'Nonaktif'}
                </Badge>
              );
            }
            if (key === 'aksi') {
              return (
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => handleOpenModal(row)}
                    title="Edit"
                    className="text-neutral-400 hover:text-neutral-700 transition-colors cursor-pointer"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                    </svg>
                  </button>
                  <button
                    onClick={() => setDeleteConfirmId(row.id)}
                    title="Hapus"
                    className="text-neutral-400 hover:text-danger-600 transition-colors cursor-pointer"
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

      {/* Modal Tambah / Edit */}
      <Modal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        title={editingCategory ? "Edit Kategori" : "Tambah Kategori"}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          {formErrors && <Alert variant="error">{formErrors}</Alert>}

          <Input
            label="Nama Kategori"
            name="name"
            value={formData.name}
            onChange={handleChange}
            placeholder="mis. Reguler"
            required
          />

          <Input
            label="Slug URL"
            name="slug"
            value={formData.slug}
            onChange={handleChange}
            placeholder="otomatis jika kosong"
          />

          {/* Checklist Brand */}
          <div className="space-y-2 pt-1 border-t border-neutral-100">
            <div className="flex items-center justify-between">
              <label className="text-xs font-semibold text-neutral-800 font-heading">
                Pilih Brand
              </label>
              <div className="flex items-center gap-2 text-xs">
                <button
                  type="button"
                  onClick={() => handleSelectAllBrands(true)}
                  className="text-primary-600 hover:text-primary-700 font-medium cursor-pointer"
                >
                  Pilih Semua
                </button>
                <span className="text-neutral-300">|</span>
                <button
                  type="button"
                  onClick={() => handleSelectAllBrands(false)}
                  className="text-neutral-500 hover:text-neutral-700 font-medium cursor-pointer"
                >
                  Kosongkan
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 p-3 bg-neutral-50 rounded-lg border border-neutral-200">
              {brands.map(b => (
                <label
                  key={b.id}
                  className="flex items-center gap-2 text-xs font-medium text-neutral-700 cursor-pointer p-1.5 rounded hover:bg-white transition-colors"
                >
                  <input
                    type="checkbox"
                    checked={formData.brand_ids.includes(b.id)}
                    onChange={(e) => handleBrandCheckboxChange(b.id, e.target.checked)}
                    className="rounded border-neutral-300 text-primary-600 focus:ring-primary-500 w-4 h-4 cursor-pointer"
                  />
                  <span>{b.name}</span>
                </label>
              ))}
            </div>
          </div>

          <div className="pt-2 border-t border-neutral-100 flex items-center justify-between">
            <div className="text-xs font-semibold text-neutral-800 font-heading">Status Aktif</div>
            <Toggle
              id="is_active"
              name="is_active"
              checked={formData.is_active}
              onChange={handleChange}
            />
          </div>

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
              {isSubmitting ? 'Menyimpan...' : (editingCategory ? 'Simpan Perubahan' : 'Simpan')}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Hapus */}
      <Modal
        isOpen={!!deleteConfirmId}
        onClose={() => setDeleteConfirmId(null)}
        title="Konfirmasi Hapus Kategori"
      >
        <div className="space-y-4">
          <p className="text-sm text-neutral-600">
            Apakah Anda yakin ingin menghapus kategori paket ini? Kategori tidak dapat dihapus jika masih digunakan oleh paket umroh yang sudah dibuat.
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              variant="secondary"
              onClick={() => setDeleteConfirmId(null)}
              disabled={isDeleting}
            >
              Batal
            </Button>
            <Button
              variant="danger"
              onClick={handleDelete}
              disabled={isDeleting}
            >
              {isDeleting ? 'Menghapus...' : 'Ya, Hapus'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default CategoriesPage;

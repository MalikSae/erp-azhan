import React, { useState, useEffect, useMemo } from 'react';
import BrandLabel from 'shared/ui/BrandLabel';
import { Pencil, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import SegmentedTabs from 'shared/ui/SegmentedTabs';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import Toggle from 'shared/ui/Toggle';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import ActionMenu from 'shared/ui/ActionMenu';
import { listCategories, createCategory, updateCategory, deleteCategory } from '../api/categories';
import { listBrands } from '../api/brands';
import { listPerlengkapanItems, getCategoryPerlengkapan, putCategoryPerlengkapan } from '../api/perlengkapan';
import PerlengkapanSetEditor from '../components/PerlengkapanSetEditor';
import { StatusChip } from '../components/penggunaUi';
import { isBadgeHex, badgeTextColor } from '../lib/badgeColors';
import BadgeColorField from '../components/BadgeColorField';

const initialForm = {
  name: '',
  slug: '',
  badge_color: '',
  is_active: true,
  brand_ids: []
};

// Brand: titik warna + nama (gaya sama dengan daftar Paket).
const BrandChip = ({ name, color }) => <BrandLabel name={name} color={color} />;

const CategoriesPage = () => {
  const [categories, setCategories] = useState([]);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [activeTab, setActiveTab] = useState('all');

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState(null);
  const [formData, setFormData] = useState(initialForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Isi perlengkapan bawaan kategori (dipakai semua paket kategori ini, kecuali ditimpa per paket).
  const [perlCatalog, setPerlCatalog] = useState([]);
  const [perlItems, setPerlItems] = useState([]);

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

  const counts = useMemo(() => ({
    all: categories.length,
    active: categories.filter(c => c.is_active).length,
    inactive: categories.filter(c => !c.is_active).length,
  }), [categories]);

  const rows = useMemo(() => categories
    .filter(c => activeTab === 'all' || (activeTab === 'active' ? c.is_active : !c.is_active))
    // Kolom bantu pencarian: nama brand ikut bisa dicari.
    .map(c => ({ ...c, _cari: (c.brands || []).map(b => b.name).join(' ') })), [categories, activeTab]);

  const tabs = [
    { id: 'all', label: 'Semua', count: counts.all },
    { id: 'active', label: 'Aktif', count: counts.active },
    { id: 'inactive', label: 'Nonaktif', count: counts.inactive },
  ];

  const toSetRows = (rows) => (rows || []).map((r) => ({ perlengkapan_item_id: r.perlengkapan_item_id, qty: r.qty, untuk: r.untuk || 'semua' }));

  const handleOpenModal = (cat = null) => {
    setPerlItems([]);
    listPerlengkapanItems().then((d) => setPerlCatalog(d || [])).catch(() => setPerlCatalog([]));
    if (cat) {
      getCategoryPerlengkapan(cat.id).then((d) => setPerlItems(toSetRows(d?.items))).catch(() => setPerlItems([]));
    }
    if (cat) {
      setEditingCategory(cat);
      setFormData({
        name: cat.name || '',
        slug: cat.slug || '',
        badge_color: cat.badge_color || '',
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
        badge_color: formData.badge_color || null,
        is_active: formData.is_active,
        brand_ids: formData.brand_ids
      };

      let categoryId = editingCategory?.id;
      if (editingCategory) {
        await updateCategory(editingCategory.id, payload);
      } else {
        const created = await createCategory(payload);
        categoryId = created?.id;
      }
      if (categoryId) {
        await putCategoryPerlengkapan(
          categoryId,
          perlItems.map((r) => ({ perlengkapan_item_id: r.perlengkapan_item_id, qty: Number(r.qty) || 1, untuk: r.untuk || 'semua' })),
        );
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

  const columns = [
    { header: 'Kategori', key: 'name', sortable: true },
    { header: 'Badge', key: 'badge_color' },
    { header: 'Brand Berlaku', key: 'brands' },
    { header: 'Jumlah Paket', key: 'package_count', sortable: true },
    { header: 'Status', key: 'is_active', sortable: true },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderCell = (row, key) => {
    if (key === 'name') {
      return (
        <div className="min-w-0">
          <button
            type="button"
            onClick={() => handleOpenModal(row)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
          >
            {row.name}
          </button>
          <div className="text-[11px] font-body text-neutral-400 truncate">/{row.slug}</div>
        </div>
      );
    }

    if (key === 'badge_color') {
      const hex = isBadgeHex(row.badge_color) ? row.badge_color : null;
      return hex ? (
        <span className="inline-block rounded px-2.5 py-0.5 text-[11px] font-bold" style={{ backgroundColor: hex, color: badgeTextColor(hex) }}>{row.name}</span>
      ) : (
        <span className="text-xs text-neutral-400 font-body">Otomatis</span>
      );
    }

    if (key === 'brands') {
      if (!row.brands || row.brands.length === 0) {
        return <span className="text-xs text-neutral-400 font-body">Semua brand</span>;
      }
      return (
        <div className="flex flex-wrap gap-1.5 max-w-sm">
          {row.brands.map(b => (
            <BrandChip
              key={b.id}
              name={b.name}
              color={b.primary_color || brands.find(item => item.id === b.id)?.primary_color}
            />
          ))}
        </div>
      );
    }

    if (key === 'package_count') {
      return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.package_count || 0} paket</span>;
    }

    if (key === 'is_active') return <StatusChip active={row.is_active} />;

    if (key === 'aksi') {
      const used = (row.package_count || 0) > 0;
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Kategori', onClick: () => handleOpenModal(row) },
        {
          label: 'Hapus',
          icon: Trash2,
          danger: true,
          tooltip: 'Hapus Kategori',
          disabled: used,
          disabledTooltip: `Tidak bisa dihapus, dipakai ${row.package_count} paket`,
          onClick: () => setDeleteConfirmId(row.id),
        },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Kategori"
        actionLabel="+ Tambah Kategori"
        onAction={() => handleOpenModal()}
      />

      {errorMessage && (
        <Alert variant="error" onClose={() => setErrorMessage(null)}>
          {errorMessage}
        </Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={15}
          emptyMessage='Belum ada data kategori paket. Klik "+ Tambah Kategori" untuk menambahkan.'
          searchPlaceholder="Cari nama, slug, atau brand..."
          toolbarLeading={<SegmentedTabs tabs={tabs} value={activeTab} onChange={setActiveTab} />}
          renderCell={renderCell}
        />
      )}

      {/* Modal Tambah / Edit */}
      <Modal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        title={editingCategory ? "Edit Kategori" : "Tambah Kategori"}
        size="lg"
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

          {/* Warna badge */}
          <div className="space-y-2 pt-1 border-t border-neutral-100">
            <div>
              <div className="text-xs font-semibold text-neutral-800 font-heading">Warna Badge</div>
              <p className="text-[11px] font-body text-neutral-500">Warna label kategori di situs booking. Otomatis = dipilihkan sistem.</p>
            </div>
            <BadgeColorField
              value={formData.badge_color}
              label={formData.name.trim()}
              onChange={(hex) => setFormData(prev => ({ ...prev, badge_color: hex }))}
            />
          </div>

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

          {/* Isi perlengkapan bawaan */}
          <div className="space-y-2 pt-3 border-t border-neutral-100">
            <div>
              <div className="text-xs font-semibold text-neutral-800 font-heading">Perlengkapan Bawaan</div>
              <p className="text-[11px] font-body text-neutral-500">
                Dibagikan ke tiap jamaah pada paket kategori ini. Bisa diubah khusus per paket di form Paket.
              </p>
            </div>
            <PerlengkapanSetEditor catalog={perlCatalog} value={perlItems} onChange={setPerlItems} />
          </div>

          <div className="pt-3 border-t border-neutral-100 flex items-center justify-between">
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

import React, { useState, useEffect, useMemo, useContext } from 'react';
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
import { AuthContext } from '../context/AuthContext';
import { StatusChip } from '../components/penggunaUi';
import { listDestinations, createDestination, updateDestination, deleteDestination } from '../api/destinations';

const initialForm = { name: '', slug: '', is_active: true };

const SLUG_RE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

const slugify = (text) =>
  text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');

const DestinationsPage = () => {
  const { hasPermission } = useContext(AuthContext);
  const canEdit = hasPermission('masterdata.edit');

  const [destinations, setDestinations] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [activeTab, setActiveTab] = useState('all');

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [formData, setFormData] = useState(initialForm);
  const [slugTouched, setSlugTouched] = useState(false);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [deleteConfirmId, setDeleteConfirmId] = useState(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchData = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listDestinations();
      setDestinations(data || []);
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal memuat data destinasi.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const counts = useMemo(() => ({
    all: destinations.length,
    active: destinations.filter((d) => d.is_active).length,
    inactive: destinations.filter((d) => !d.is_active).length,
  }), [destinations]);

  const rows = useMemo(
    () => destinations.filter((d) => activeTab === 'all' || (activeTab === 'active' ? d.is_active : !d.is_active)),
    [destinations, activeTab],
  );

  const tabs = [
    { id: 'all', label: 'Semua', count: counts.all },
    { id: 'active', label: 'Aktif', count: counts.active },
    { id: 'inactive', label: 'Nonaktif', count: counts.inactive },
  ];

  const handleOpenModal = (item = null) => {
    if (item) {
      setEditingItem(item);
      setFormData({ name: item.name || '', slug: item.slug || '', is_active: item.is_active !== undefined ? item.is_active : true });
      setSlugTouched(true);
    } else {
      setEditingItem(null);
      setFormData(initialForm);
      setSlugTouched(false);
    }
    setFormErrors(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (!isSubmitting) {
      setIsModalOpen(false);
      setEditingItem(null);
      setFormData(initialForm);
    }
  };

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    if (name === 'name') {
      setFormData((prev) => ({ ...prev, name: value, slug: slugTouched ? prev.slug : slugify(value) }));
      return;
    }
    if (name === 'slug') setSlugTouched(true);
    setFormData((prev) => ({ ...prev, [name]: type === 'checkbox' ? checked : value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);

    const name = formData.name.trim();
    const slug = formData.slug.trim();
    if (!name) {
      setFormErrors('Nama destinasi wajib diisi.');
      return;
    }
    if (!SLUG_RE.test(slug)) {
      setFormErrors('Slug hanya boleh berisi huruf kecil, angka, dan tanda hubung tunggal (mis. abu-dhabi).');
      return;
    }

    setIsSubmitting(true);
    try {
      const payload = { name, slug, is_active: formData.is_active };
      if (editingItem) {
        await updateDestination(editingItem.id, payload);
      } else {
        await createDestination(payload);
      }
      setIsModalOpen(false);
      setEditingItem(null);
      setFormData(initialForm);
      fetchData();
    } catch (error) {
      setFormErrors(error.response?.data?.error || 'Terjadi kesalahan saat menyimpan destinasi.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteConfirmId) return;
    setIsDeleting(true);
    try {
      await deleteDestination(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchData();
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal menghapus destinasi.');
      setDeleteConfirmId(null);
    } finally {
      setIsDeleting(false);
    }
  };

  const columns = [
    { header: 'Destinasi', key: 'name', sortable: true },
    { header: 'Slug', key: 'slug', sortable: true },
    { header: 'Jumlah Paket', key: 'package_count', sortable: true },
    { header: 'Status', key: 'is_active', sortable: true },
    ...(canEdit ? [{ header: 'Aksi', key: 'aksi' }] : []),
  ];

  const renderCell = (row, key) => {
    if (key === 'name') {
      return canEdit ? (
        <button
          type="button"
          onClick={() => handleOpenModal(row)}
          className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
        >
          {row.name}
        </button>
      ) : (
        <span className="font-heading font-semibold text-neutral-900">{row.name}</span>
      );
    }
    if (key === 'slug') return <span className="text-[13px] font-body text-neutral-600">{row.slug}</span>;
    if (key === 'package_count') {
      return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.package_count || 0} paket</span>;
    }
    if (key === 'is_active') return <StatusChip active={row.is_active} />;
    if (key === 'aksi') {
      const used = (row.package_count || 0) > 0;
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Destinasi', onClick: () => handleOpenModal(row) },
        {
          label: 'Hapus',
          icon: Trash2,
          danger: true,
          tooltip: 'Hapus Destinasi',
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
        title="Kelola Destinasi"
        {...(canEdit ? { actionLabel: '+ Tambah Destinasi', onAction: () => handleOpenModal() } : {})}
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
          emptyMessage={canEdit ? 'Belum ada data destinasi. Klik "+ Tambah Destinasi" untuk menambahkan.' : 'Belum ada data destinasi.'}
          searchPlaceholder="Cari nama atau slug..."
          toolbarLeading={<SegmentedTabs tabs={tabs} value={activeTab} onChange={setActiveTab} />}
          renderCell={renderCell}
        />
      )}

      <Modal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        title={editingItem ? 'Edit Destinasi' : 'Tambah Destinasi'}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          {formErrors && <Alert variant="error">{formErrors}</Alert>}

          <Input
            label="Nama Destinasi"
            name="name"
            value={formData.name}
            onChange={handleChange}
            placeholder="mis. Abu Dhabi"
            required
          />

          <div>
            <Input
              label="Slug URL"
              name="slug"
              value={formData.slug}
              onChange={handleChange}
              placeholder="mis. abu-dhabi"
              required
            />
            <p className="mt-1.5 text-[11px] font-body text-neutral-500">
              Huruf kecil, angka, dan tanda hubung. Dipakai di URL web publik, sebaiknya tidak diubah setelah dipakai.
            </p>
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
            <Button type="button" variant="secondary" onClick={handleCloseModal} disabled={isSubmitting}>
              Batal
            </Button>
            <Button type="submit" variant="primary" disabled={isSubmitting}>
              {isSubmitting ? 'Menyimpan...' : (editingItem ? 'Simpan Perubahan' : 'Simpan')}
            </Button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={!!deleteConfirmId}
        onClose={() => setDeleteConfirmId(null)}
        title="Konfirmasi Hapus Destinasi"
      >
        <div className="space-y-4">
          <p className="text-sm text-neutral-600">
            Apakah Anda yakin ingin menghapus destinasi ini? Destinasi tidak dapat dihapus jika masih digunakan oleh paket.
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button variant="secondary" onClick={() => setDeleteConfirmId(null)} disabled={isDeleting}>
              Batal
            </Button>
            <Button variant="danger" onClick={handleDelete} disabled={isDeleting}>
              {isDeleting ? 'Menghapus...' : 'Ya, Hapus'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default DestinationsPage;

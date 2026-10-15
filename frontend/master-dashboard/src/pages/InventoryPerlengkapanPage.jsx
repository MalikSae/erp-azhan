import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useContext } from 'react';
import { AuthContext } from '../context/AuthContext';
import { Pencil, Trash2, History, PackagePlus, SlidersHorizontal } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import FormField from 'shared/ui/FormField';
import Input from 'shared/ui/Input';
import FileInput from 'shared/ui/FileInput';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import ActionMenu from 'shared/ui/ActionMenu';
import CustomDropdown from 'shared/ui/CustomDropdown';
import {
  listPerlengkapanItems,
  createPerlengkapanItem,
  updatePerlengkapanItem,
  deletePerlengkapanItem,
  listPerlengkapanStok,
  createPembelianPerlengkapan,
  createKoreksiPerlengkapan,
} from '../api/perlengkapan';
import { uploadMedia } from '../api/media';
import ItemPhoto from '../components/ItemPhoto';

// Stok di bawah/sama dengan angka ini diberi tanda "menipis".
const LOW_STOCK = 10;

const initialItemForm = { nama: '', photo_url: '' };

// Angka stok (hanya baca). Perubahan stok dilakukan lewat aksi "Terima Barang" atau
// "Atur Stok" (dengan alasan), supaya tidak ada perubahan karena salah ketik.
const StokValue = ({ value }) => {
  const tone = value === 0
    ? 'border-danger-200 bg-danger-50 text-danger-700'
    : value <= LOW_STOCK
      ? 'border-warning-200 bg-warning-50 text-warning-800'
      : 'border-transparent text-neutral-900';
  return (
    <span className={`inline-flex min-w-12 items-center justify-center rounded-lg border px-2.5 py-1 text-[13px] font-semibold font-body ${tone}`}>
      {value}
    </span>
  );
};

// Katalog item perlengkapan + stok tiap brand dalam satu matriks.
// Isi set per jamaah diatur di Kategori Paket (bawaan) dan Form Paket (khusus paket).
const InventoryPerlengkapanPage = () => {
  const navigate = useNavigate();
  const { hasPermission } = useContext(AuthContext);
  // Katalog item (tambah/ubah/hapus) hanya untuk pengelola master data; stok brand mengikuti izin perlengkapan.
  const canEditCatalog = hasPermission('masterdata.edit');
  const canEditStok = hasPermission('perlengkapan.edit');
  const [items, setItems] = useState([]);
  const [stok, setStok] = useState([]);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);
  const [filter, setFilter] = useState('');

  // Modal item
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [formData, setFormData] = useState(initialItemForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isUploadingPhoto, setIsUploadingPhoto] = useState(false);
  const [photoPreview, setPhotoPreview] = useState(null);

  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  // Barang masuk ke stok brand dan riwayat pergerakan stok
  const [terimaOpen, setTerimaOpen] = useState(false);
  const [terimaForm, setTerimaForm] = useState({ item_id: '', brand_id: '', qty: '', harga: '', vendor: '', catatan: '' });
  const [stokModalError, setStokModalError] = useState(null);
  // Atur Stok: kurangi (rusak/hilang) atau setel ke angka hasil hitung fisik.
  const [aturOpen, setAturOpen] = useState(false);
  const [aturItem, setAturItem] = useState(null);
  const [aturForm, setAturForm] = useState({ brand_id: '', mode: 'kurangi', jumlah: '', catatan: '' });
  const [stokSubmitting, setStokSubmitting] = useState(false);

  const fetchData = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const [itemData, stokData] = await Promise.all([
        listPerlengkapanItems(),
        listPerlengkapanStok(),
      ]);
      setItems(itemData || []);
      setStok(stokData || []);
      // Brand diturunkan dari baris stok: API hanya mengembalikan brand yang boleh dilihat pengguna,
      // sehingga admin brand otomatis hanya mendapat kolom brand-nya.
      const seen = new Map();
      (stokData || []).forEach((r) => {
        if (!seen.has(r.brand_id)) seen.set(r.brand_id, { id: r.brand_id, name: r.brand_name });
      });
      setBrands([...seen.values()].sort((a, b) => String(a.name).localeCompare(String(b.name))));
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal memuat data perlengkapan.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  // item_id -> { brand_id -> stok }
  const stokMap = useMemo(() => {
    const map = {};
    stok.forEach((r) => {
      if (!map[r.perlengkapan_item_id]) map[r.perlengkapan_item_id] = {};
      map[r.perlengkapan_item_id][r.brand_id] = r.stok_tersedia;
    });
    return map;
  }, [stok]);

  const rowsAll = useMemo(() => items.map((it) => {
    const perBrand = stokMap[it.id] || {};
    const total = brands.reduce((sum, b) => sum + (perBrand[b.id] || 0), 0);
    const kosong = brands.some((b) => (perBrand[b.id] || 0) === 0);
    const menipis = brands.some((b) => { const v = perBrand[b.id] || 0; return v > 0 && v <= LOW_STOCK; });
    return { ...it, perBrand, total, kosong, menipis };
  }), [items, stokMap, brands]);

  const counts = useMemo(() => ({
    all: rowsAll.length,
    kosong: rowsAll.filter((r) => r.kosong).length,
    menipis: rowsAll.filter((r) => r.menipis).length,
  }), [rowsAll]);

  const rows = useMemo(() => rowsAll.filter((r) => {
    if (filter === 'kosong') return r.kosong;
    if (filter === 'menipis') return r.menipis;
    return true;
  }), [rowsAll, filter]);

  // ─── Modal item ─────────────────────────────────────────────────────────────
  const handleOpenModal = (item = null) => {
    if (item) {
      setEditingItem(item);
      setFormData({ nama: item.nama || '', photo_url: item.photo_url || '' });
    } else {
      setEditingItem(null);
      setFormData(initialItemForm);
    }
    setPhotoPreview(null);
    setFormErrors(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (isSubmitting) return;
    setIsModalOpen(false);
    setEditingItem(null);
    setFormData(initialItemForm);
    setFormErrors(null);
  };

  const handlePhotoChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    setPhotoPreview(URL.createObjectURL(file));
    setIsUploadingPhoto(true);
    setFormErrors(null);
    try {
      const url = await uploadMedia(file, 'perlengkapan');
      setFormData((prev) => ({ ...prev, photo_url: url }));
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
      setFormErrors('Nama item perlengkapan wajib diisi.');
      return;
    }
    setIsSubmitting(true);
    try {
      // qty_per_set sengaja tidak dikirim: isi set diatur per kategori/paket, bukan di item.
      const payload = { nama, photo_url: formData.photo_url || null };
      if (editingItem) {
        await updatePerlengkapanItem(editingItem.id, payload);
      } else {
        await createPerlengkapanItem(payload);
      }
      setIsSubmitting(false);
      setIsModalOpen(false);
      setEditingItem(null);
      setFormData(initialItemForm);
      setSuccessMessage(`Item "${nama}" berhasil ${editingItem ? 'diperbarui' : 'ditambahkan'}.`);
      fetchData();
    } catch (error) {
      setFormErrors(error.response?.data?.error || 'Terjadi kesalahan saat menyimpan item.');
      setIsSubmitting(false);
    }
  };

  const handleDeleteItem = async () => {
    try {
      await deletePerlengkapanItem(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchData();
    } catch (error) {
      setDeleteConfirmId(null);
      setErrorMessage(error.response?.data?.error || 'Gagal menghapus item perlengkapan.');
    }
  };

  const itemToDelete = items.find((it) => it.id === deleteConfirmId);

  // ─── Barang masuk & riwayat ─────────────────────────────────────────────────
  const itemOptions = items.map((it) => ({ value: String(it.id), label: it.nama }));
  const brandOptions = brands.map((b) => ({ value: String(b.id), label: b.name }));
  const valueOf = (v) => (v?.target ? v.target.value : v);

  const openTerima = () => {
    setTerimaForm({ item_id: '', brand_id: '', qty: '', harga: '', vendor: '', catatan: '' });
    setStokModalError(null);
    setTerimaOpen(true);
  };

  const submitTerima = async (e) => {
    e.preventDefault();
    const qty = parseInt(terimaForm.qty, 10);
    if (!terimaForm.item_id) return setStokModalError('Pilih item yang diterima.');
    if (!terimaForm.brand_id) return setStokModalError('Pilih brand penerima stok.');
    if (isNaN(qty) || qty < 1) return setStokModalError('Jumlah minimal 1.');
    setStokSubmitting(true);
    setStokModalError(null);
    try {
      await createPembelianPerlengkapan({
        perlengkapan_item_id: Number(terimaForm.item_id),
        brand_id: Number(terimaForm.brand_id),
        qty,
        harga_satuan: terimaForm.harga === '' ? null : Number(terimaForm.harga),
        vendor: terimaForm.vendor.trim() || null,
        catatan: terimaForm.catatan.trim() || null,
      });
      const nama = items.find((it) => String(it.id) === terimaForm.item_id)?.nama;
      const brand = brands.find((b) => String(b.id) === terimaForm.brand_id)?.name;
      setTerimaOpen(false);
      setSuccessMessage(`${qty} unit "${nama}" masuk ke stok ${brand}.`);
      fetchData();
    } catch (error) {
      setStokModalError(error.response?.data?.error || 'Gagal mencatat barang masuk.');
    } finally {
      setStokSubmitting(false);
    }
  };

  const openAtur = (item) => {
    setAturItem(item);
    setAturForm({ brand_id: brands.length === 1 ? String(brands[0].id) : '', mode: 'kurangi', jumlah: '', catatan: '' });
    setStokModalError(null);
    setAturOpen(true);
  };

  const aturCurrent = aturItem && aturForm.brand_id ? (aturItem.perBrand[Number(aturForm.brand_id)] || 0) : null;
  const aturJumlah = parseInt(aturForm.jumlah, 10);
  const aturNext = aturCurrent === null || isNaN(aturJumlah) || aturJumlah < 0
    ? null
    : aturForm.mode === 'kurangi' ? aturCurrent - aturJumlah : aturJumlah;

  const submitAtur = async (e) => {
    e.preventDefault();
    if (!aturForm.brand_id) return setStokModalError('Pilih brand yang stoknya diatur.');
    if (isNaN(aturJumlah) || aturJumlah < 0 || (aturForm.mode === 'kurangi' && aturJumlah < 1)) {
      return setStokModalError(aturForm.mode === 'kurangi' ? 'Jumlah pengurangan minimal 1.' : 'Isi hasil hitung stok (0 atau lebih).');
    }
    if (aturNext < 0) return setStokModalError(`Stok tidak cukup. Stok saat ini ${aturCurrent}.`);
    if (!aturForm.catatan.trim()) return setStokModalError('Alasan wajib diisi.');
    if (aturNext === aturCurrent) return setStokModalError('Stok tidak berubah.');
    setStokSubmitting(true);
    setStokModalError(null);
    try {
      await createKoreksiPerlengkapan({
        perlengkapan_item_id: aturItem.id,
        brand_id: Number(aturForm.brand_id),
        stok_baru: aturNext,
        catatan: aturForm.catatan.trim(),
      });
      const brand = brands.find((b) => String(b.id) === aturForm.brand_id)?.name;
      setAturOpen(false);
      setSuccessMessage(`Stok "${aturItem.nama}" (${brand}) diubah dari ${aturCurrent} menjadi ${aturNext}.`);
      fetchData();
    } catch (error) {
      setStokModalError(error.response?.data?.error || 'Gagal mengubah stok.');
    } finally {
      setStokSubmitting(false);
    }
  };

  // ─── Tabel ──────────────────────────────────────────────────────────────────
  const columns = useMemo(() => [
    { header: 'Item', key: 'nama', sortable: true },
    ...brands.map((b) => ({
      header: b.name,
      key: `brand_${b.id}`,
      sortable: true,
      sortFn: (a, c) => (a.perBrand[b.id] || 0) - (c.perBrand[b.id] || 0),
    })),
    { header: 'Total', key: 'total', sortable: true, sortFn: (a, c) => a.total - c.total },
    { header: 'Aksi', key: 'aksi' },
  ], [brands]);

  const renderCell = (row, key) => {
    if (key === 'nama') {
      return (
        <div className="flex items-center gap-3 min-w-0">
          <ItemPhoto url={row.photo_url} name={row.nama} />
          {canEditCatalog ? (
            <button
              type="button"
              onClick={() => handleOpenModal(row)}
              className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
            >
              {row.nama}
            </button>
          ) : (
            <span className="font-heading font-semibold text-neutral-900 leading-snug">{row.nama}</span>
          )}
        </div>
      );
    }

    if (key.startsWith('brand_')) {
      const brandId = Number(key.slice(6));
      return <StokValue value={row.perBrand[brandId] || 0} />;
    }

    if (key === 'total') {
      return <span className="text-[13px] font-body font-semibold text-neutral-900">{row.total}</span>;
    }

    if (key === 'aksi') {
      const menu = [];
      if (canEditStok) {
        menu.push({ label: 'Atur Stok', icon: SlidersHorizontal, tooltip: 'Kurangi stok (rusak/hilang) atau koreksi hasil hitung', onClick: () => openAtur(row) });
      }
      menu.push({ label: 'Riwayat Stok', icon: History, tooltip: 'Lihat pergerakan stok item ini', onClick: () => navigate(`/inventory/riwayat-stok?item=${row.id}`) });
      if (canEditCatalog) {
        menu.push({ label: 'Edit', icon: Pencil, tooltip: 'Edit item', onClick: () => handleOpenModal(row) });
        menu.push({ label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus item', onClick: () => setDeleteConfirmId(row.id) });
      }
      return <ActionMenu align="right" items={menu} />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Perlengkapan"
        {...(canEditCatalog ? { actionLabel: '+ Tambah Item', onAction: () => handleOpenModal() } : {})}
      >
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="secondary" onClick={() => navigate('/inventory/riwayat-stok')}>
            <History size={16} strokeWidth={1.75} />
            <span>Riwayat</span>
          </Button>
          {canEditStok && (
            <Button variant="secondary" onClick={openTerima}>
              <PackagePlus size={16} strokeWidth={1.75} />
              <span>Terima Barang</span>
            </Button>
          )}
        </div>
      </PageHeader>

      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={15}
          emptyMessage='Belum ada item perlengkapan. Klik "+ Tambah Item" untuk menambahkan.'
          searchPlaceholder="Cari nama item..."
          toolbarActions={
            <CustomDropdown
              value={filter}
              onChange={(val) => setFilter(val?.target ? val.target.value : val)}
              className="!mb-0 w-52"
              options={[
                { value: '', label: `Semua item (${counts.all})` },
                { value: 'kosong', label: `Ada stok kosong (${counts.kosong})` },
                { value: 'menipis', label: `Stok menipis ≤ ${LOW_STOCK} (${counts.menipis})` },
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
        title={editingItem ? 'Edit Item Perlengkapan' : 'Tambah Item Perlengkapan'}
      >
        <form onSubmit={handleSubmitItem} className="space-y-4">
          {formErrors && <Alert variant="error">{formErrors}</Alert>}

          <FormField label="Nama Item" required>
            <Input
              type="text"
              name="nama"
              value={formData.nama}
              onChange={(e) => setFormData((prev) => ({ ...prev, nama: e.target.value }))}
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
              setFormData((prev) => ({ ...prev, photo_url: '' }));
            }}
            isUploading={isUploadingPhoto}
            uploadingText="Mengunggah foto..."
            placeholder="Pilih foto item (PNG, JPG, WebP)..."
          />

          <p className="text-[11px] font-body text-neutral-500">
            Jumlah per jamaah diatur di Kelola Kategori (bawaan) atau di form Paket (khusus paket).
          </p>

          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={handleCloseModal} disabled={isSubmitting}>
              Batal
            </Button>
            <Button type="submit" variant="primary" disabled={isSubmitting || isUploadingPhoto}>
              {isSubmitting ? 'Menyimpan...' : 'Simpan'}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Terima Barang (pembelian -> stok brand) */}
      <Modal isOpen={terimaOpen} onClose={() => !stokSubmitting && setTerimaOpen(false)} title="Terima Barang">
        <form onSubmit={submitTerima} className="space-y-4">
          {stokModalError && <Alert variant="error">{stokModalError}</Alert>}
          <CustomDropdown
            label="Item"
            required
            value={terimaForm.item_id}
            onChange={(v) => setTerimaForm((p) => ({ ...p, item_id: valueOf(v) }))}
            options={[{ value: '', label: 'Pilih item' }, ...itemOptions]}
            className="!mb-0"
          />
          <CustomDropdown
            label="Masuk ke Stok Brand"
            required
            value={terimaForm.brand_id}
            onChange={(v) => setTerimaForm((p) => ({ ...p, brand_id: valueOf(v) }))}
            options={[{ value: '', label: 'Pilih brand' }, ...brandOptions]}
            className="!mb-0"
          />
          <div className="grid grid-cols-2 gap-4">
            <Input
              label="Jumlah"
              type="number"
              min="1"
              step="1"
              required
              value={terimaForm.qty}
              onChange={(e) => setTerimaForm((p) => ({ ...p, qty: e.target.value }))}
              className="!mb-0"
            />
            <Input
              label="Harga Satuan (Opsional)"
              type="number"
              min="0"
              step="1"
              value={terimaForm.harga}
              onChange={(e) => setTerimaForm((p) => ({ ...p, harga: e.target.value }))}
              placeholder="Rp"
              className="!mb-0"
            />
          </div>
          <Input
            label="Vendor (Opsional)"
            value={terimaForm.vendor}
            onChange={(e) => setTerimaForm((p) => ({ ...p, vendor: e.target.value }))}
            className="!mb-0"
          />
          <Input
            label="Catatan (Opsional)"
            value={terimaForm.catatan}
            onChange={(e) => setTerimaForm((p) => ({ ...p, catatan: e.target.value }))}
            placeholder="Mis. nomor PO / surat jalan"
            className="!mb-0"
          />
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setTerimaOpen(false)} disabled={stokSubmitting}>Batal</Button>
            <Button type="submit" variant="primary" disabled={stokSubmitting}>{stokSubmitting ? 'Menyimpan...' : 'Simpan'}</Button>
          </div>
        </form>
      </Modal>

      {/* Modal Atur Stok (kurangi / koreksi) */}
      <Modal isOpen={aturOpen} onClose={() => !stokSubmitting && setAturOpen(false)} title={`Atur Stok · ${aturItem?.nama || ''}`}>
        <form onSubmit={submitAtur} className="space-y-4">
          {stokModalError && <Alert variant="error">{stokModalError}</Alert>}
          <p className="text-xs font-body text-neutral-500">
            Untuk menambah stok dari pembelian, gunakan <strong>Terima Barang</strong>. Di sini untuk barang rusak/hilang
            atau menyamakan stok dengan hasil hitung fisik.
          </p>
          <CustomDropdown
            label="Brand"
            required
            value={aturForm.brand_id}
            onChange={(v) => setAturForm((p) => ({ ...p, brand_id: valueOf(v) }))}
            options={[
              { value: '', label: 'Pilih brand' },
              ...brands.map((b) => ({ value: String(b.id), label: `${b.name} (stok ${aturItem?.perBrand?.[b.id] || 0})` })),
            ]}
            className="!mb-0"
          />
          <CustomDropdown
            label="Tindakan"
            value={aturForm.mode}
            onChange={(v) => setAturForm((p) => ({ ...p, mode: valueOf(v), jumlah: '' }))}
            options={[
              { value: 'kurangi', label: 'Kurangi stok (rusak / hilang)' },
              { value: 'setel', label: 'Sesuaikan ke hasil hitung fisik' },
            ]}
            className="!mb-0"
          />
          <Input
            label={aturForm.mode === 'kurangi' ? 'Jumlah dikurangi' : 'Hasil hitung stok'}
            type="number"
            min={aturForm.mode === 'kurangi' ? '1' : '0'}
            step="1"
            required
            value={aturForm.jumlah}
            onChange={(e) => setAturForm((p) => ({ ...p, jumlah: e.target.value }))}
            className="!mb-0"
          />
          {aturCurrent !== null && aturNext !== null && (
            <p className={`text-xs font-body ${aturNext < 0 ? 'text-danger-700' : 'text-neutral-600'}`}>
              Stok: <strong>{aturCurrent}</strong> → <strong>{aturNext}</strong>
              {aturNext < 0 ? ' (melebihi stok yang ada)' : ''}
            </p>
          )}
          <Input
            label="Alasan"
            required
            value={aturForm.catatan}
            onChange={(e) => setAturForm((p) => ({ ...p, catatan: e.target.value }))}
            placeholder={aturForm.mode === 'kurangi' ? 'Mis. 2 pcs rusak saat dikirim' : 'Mis. stok opname Oktober'}
            className="!mb-0"
          />
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setAturOpen(false)} disabled={stokSubmitting}>Batal</Button>
            <Button type="submit" variant="primary" disabled={stokSubmitting}>{stokSubmitting ? 'Menyimpan...' : 'Simpan'}</Button>
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
            Catatan: item tidak dapat dihapus jika masih dipakai di isi perlengkapan kategori atau paket, atau masih memiliki stok di brand manapun.
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setDeleteConfirmId(null)}>Batal</Button>
            <Button type="button" variant="danger" onClick={handleDeleteItem}>Hapus</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default InventoryPerlengkapanPage;

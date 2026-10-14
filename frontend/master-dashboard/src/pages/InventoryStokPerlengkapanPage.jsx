import React, { useState, useEffect, useMemo } from 'react';
import { Boxes, Package, Building2, XCircle, SlidersHorizontal } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import FormField from 'shared/ui/FormField';
import Input from 'shared/ui/Input';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import StatTile from 'shared/ui/StatTile';
import ActionMenu from 'shared/ui/ActionMenu';
import { listPerlengkapanStok, updatePerlengkapanStok } from '../api/perlengkapan';
import { listBrands } from '../api/brands';
import ItemPhoto from '../components/ItemPhoto';

// Chip brand: warna brand hanya penanda (dot), bukan warna UI.
const BrandChip = ({ name, color }) => (
  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[12px] font-medium font-body bg-white border border-neutral-200 text-neutral-700 whitespace-nowrap">
    <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: color || '#9CA3AF' }} />
    {name}
  </span>
);

const STOK_OPTIONS = [
  { value: '', label: 'Semua Stok' },
  { value: 'habis', label: 'Stok habis' },
  { value: 'ada', label: 'Stok tersedia' },
];

const InventoryStokPerlengkapanPage = () => {
  const [stokList, setStokList] = useState([]);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);

  const [filterBrandId, setFilterBrandId] = useState('');
  const [filterStok, setFilterStok] = useState('');

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [activeRow, setActiveRow] = useState(null);
  const [amountInput, setAmountInput] = useState('1');
  const [modalError, setModalError] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const fetchData = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const [stokData, brandsData] = await Promise.all([
        listPerlengkapanStok(),
        listBrands()
      ]);

      setStokList(stokData || []);
      setBrands(brandsData || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data stok perlengkapan.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleOpenModal = (row) => {
    setActiveRow(row);
    setAmountInput('1');
    setModalError(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (!isSubmitting) {
      setIsModalOpen(false);
      setActiveRow(null);
      setModalError(null);
    }
  };

  const handleAdjustStock = async (type) => {
    if (!activeRow) return;
    setModalError(null);

    const amount = parseInt(amountInput, 10);
    if (isNaN(amount) || amount <= 0) {
      setModalError("Jumlah perubahan harus berupa angka positif minimal 1.");
      return;
    }

    const currentStock = activeRow.stok_tersedia;
    let newStock = currentStock;

    if (type === 'tambah') {
      newStock = currentStock + amount;
    } else if (type === 'kurang') {
      newStock = currentStock - amount;
      if (newStock < 0) {
        setModalError("Stok tidak bisa kurang dari 0.");
        return;
      }
    }

    setIsSubmitting(true);
    setErrorMessage(null);
    setSuccessMessage(null);

    try {
      await updatePerlengkapanStok(activeRow.perlengkapan_item_id, activeRow.brand_id, {
        stok_tersedia: newStock
      });

      // Update state lokal untuk baris ini secara langsung
      setStokList(prev => prev.map(r => {
        if (r.brand_id === activeRow.brand_id && r.perlengkapan_item_id === activeRow.perlengkapan_item_id) {
          return { ...r, stok_tersedia: newStock };
        }
        return r;
      }));

      const actionText = type === 'tambah' ? `ditambah ${amount}` : `dikurangi ${amount}`;
      setSuccessMessage(`Stok "${activeRow.nama_item}" (${activeRow.brand_name}) berhasil ${actionText} menjadi ${newStock}.`);

      setIsModalOpen(false);
      setActiveRow(null);
    } catch (error) {
      const msg = error.response?.data?.error || `Gagal mengubah stok "${activeRow.nama_item}".`;
      setModalError(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const brandColor = useMemo(() => {
    const map = {};
    brands.forEach(b => { map[b.id] = b.primary_color; });
    return map;
  }, [brands]);

  // Filter client-side. Ringkasan mengikuti filter brand supaya cocok dengan tabel.
  const brandRows = useMemo(
    () => stokList.filter(row => !filterBrandId || String(row.brand_id) === filterBrandId),
    [stokList, filterBrandId]
  );

  const rows = useMemo(() => brandRows.filter(row => {
    if (filterStok === 'habis') return row.stok_tersedia === 0;
    if (filterStok === 'ada') return row.stok_tersedia > 0;
    return true;
  }), [brandRows, filterStok]);

  const stats = useMemo(() => ({
    item: new Set(brandRows.map(r => r.perlengkapan_item_id)).size,
    brand: new Set(brandRows.map(r => r.brand_id)).size,
    totalStok: brandRows.reduce((sum, r) => sum + (r.stok_tersedia || 0), 0),
    habis: brandRows.filter(r => r.stok_tersedia === 0).length,
  }), [brandRows]);

  const columns = [
    { header: 'Item', key: 'nama_item', sortable: true },
    { header: 'Brand', key: 'brand_name', sortable: true },
    { header: 'Stok Tersedia', key: 'stok_tersedia', sortable: true, sortFn: (a, b) => (a.stok_tersedia || 0) - (b.stok_tersedia || 0) },
    { header: 'Aksi', key: 'aksi' }
  ];

  const renderCell = (row, key) => {
    if (key === 'nama_item') {
      return (
        <div className="flex items-center gap-3 min-w-0">
          <ItemPhoto url={row.photo_url} name={row.nama_item} />
          <button
            type="button"
            onClick={() => handleOpenModal(row)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
          >
            {row.nama_item}
          </button>
        </div>
      );
    }

    if (key === 'brand_name') {
      return <BrandChip name={row.brand_name} color={brandColor[row.brand_id]} />;
    }

    if (key === 'stok_tersedia') {
      return row.stok_tersedia === 0 ? (
        <span className="text-[13px] font-body whitespace-nowrap">
          <span className="font-semibold text-danger-600">0</span>
          <span className="ml-1.5 text-[11px] text-danger-600">Habis</span>
        </span>
      ) : (
        <span className="text-[13px] font-body font-semibold text-neutral-900">{row.stok_tersedia}</span>
      );
    }

    if (key === 'aksi') {
      const menuItems = [
        { label: 'Atur Stok', icon: SlidersHorizontal, tooltip: 'Tambah atau kurangi stok', onClick: () => handleOpenModal(row) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Stok per Brand" />

      {errorMessage && (
        <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>
      )}
      {successMessage && (
        <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>
      )}

      {!isLoading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Boxes} variant="primary" value={stats.item} label="Item" />
          <StatTile icon={Building2} variant="info" value={stats.brand} label="Brand" />
          <StatTile icon={Package} variant="success" value={stats.totalStok} label="Total stok" />
          <StatTile icon={XCircle} variant="danger" value={stats.habis} label="Stok habis" />
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={15}
          emptyMessage="Belum ada data stok perlengkapan."
          searchPlaceholder="Cari item perlengkapan atau brand..."
          toolbarActions={
            <div className="flex flex-wrap items-center gap-2">
              <CustomDropdown
                value={filterStok}
                onChange={(val) => setFilterStok(val)}
                className="!mb-0 w-40"
                placeholder="Semua Stok"
                options={STOK_OPTIONS}
              />
              <CustomDropdown
                value={filterBrandId}
                onChange={(val) => setFilterBrandId(val)}
                className="!mb-0 w-44"
                placeholder="Semua Brand"
                options={[
                  { value: '', label: 'Semua Brand' },
                  ...brands.map(b => ({ value: String(b.id), label: b.name }))
                ]}
              />
            </div>
          }
          renderCell={renderCell}
        />
      )}

      {/* Modal Atur Stok */}
      <Modal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        title={activeRow ? `Atur Stok — ${activeRow.nama_item} (${activeRow.brand_name})` : "Atur Stok"}
        size="md"
      >
        <div className="space-y-4">
          {modalError && (
            <Alert variant="error">{modalError}</Alert>
          )}

          {/* Info Stok Saat Ini */}
          <div className="p-3.5 bg-neutral-50 rounded-lg border border-neutral-200 flex items-center justify-between">
            <span className="text-sm font-body text-neutral-600">Stok Saat Ini:</span>
            <span className="text-base font-heading font-semibold text-neutral-900">
              {activeRow?.stok_tersedia ?? 0}
            </span>
          </div>

          <FormField label="Jumlah Perubahan" required>
            <Input
              type="number"
              min="1"
              step="1"
              value={amountInput}
              onChange={(e) => {
                setAmountInput(e.target.value);
                setModalError(null);
              }}
              placeholder="mis. 10"
              autoFocus
              required
            />
          </FormField>

          {/* Action Buttons */}
          <div className="flex flex-col sm:flex-row justify-between gap-3 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="secondary"
              onClick={handleCloseModal}
              disabled={isSubmitting}
            >
              Batal
            </Button>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="danger"
                onClick={() => handleAdjustStock('kurang')}
                disabled={isSubmitting}
                className="text-xs"
              >
                {isSubmitting ? "Memproses..." : "− Kurangi Stok"}
              </Button>
              <Button
                type="button"
                variant="primary"
                onClick={() => handleAdjustStock('tambah')}
                disabled={isSubmitting}
                className="text-xs"
              >
                {isSubmitting ? "Memproses..." : "+ Tambah Stok"}
              </Button>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default InventoryStokPerlengkapanPage;

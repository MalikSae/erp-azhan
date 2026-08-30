import React, { useState, useEffect, useMemo } from 'react';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import CustomDropdown from '../components/ui/CustomDropdown';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listPerlengkapanStok, updatePerlengkapanStok } from '../api/perlengkapan';
import { listBrands } from '../api/brands';

const BrandCell = ({ brandName, brandLogoUrl, brandIconUrl }) => {
  const [imageError, setImageError] = useState(false);

  const imageUrl = brandIconUrl || brandLogoUrl;
  const showInitial = !imageUrl || imageError;

  return (
    <div className="flex items-center gap-2">
      {showInitial ? (
        <div className="w-6 h-6 shrink-0 rounded-full bg-neutral-200 flex items-center justify-center text-xs font-bold text-neutral-600 uppercase font-heading">
          {brandName ? brandName.charAt(0) : '?'}
        </div>
      ) : (
        <img
          src={imageUrl.startsWith('http') ? imageUrl : `${import.meta.env.VITE_API_BASE_URL}${imageUrl.startsWith('/') ? '' : '/'}${imageUrl}`}
          alt={brandName}
          className="w-6 h-6 shrink-0 rounded-full object-cover bg-neutral-100"
          onError={() => setImageError(true)}
        />
      )}
      <span className="font-heading font-medium text-neutral-900 text-sm">{brandName}</span>
    </div>
  );
};

const InventoryStokPerlengkapanPage = () => {
  const [stokList, setStokList] = useState([]);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);

  const [filterBrandId, setFilterBrandId] = useState('');

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

  // Filter client-side berdasarkan dropdown Brand lokal
  const filteredData = useMemo(() => {
    if (!filterBrandId) return stokList;
    return stokList.filter(row => String(row.brand_id) === filterBrandId);
  }, [stokList, filterBrandId]);

  const columns = [
    { header: 'Brand', key: 'brand_name' },
    { header: 'Nama Item', key: 'nama_item' },
    { header: 'Stok Tersedia', key: 'stok_tersedia' },
    { header: 'Aksi', key: 'aksi' }
  ];

  return (
    <div className="space-y-6">
      <PageHeader title="Stok Perlengkapan per Brand" />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}
      {successMessage && (
        <Alert variant="success">{successMessage}</Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-pure-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={filteredData}
          itemsPerPage={15}
          emptyMessage="Belum ada data stok perlengkapan."
          searchPlaceholder="Cari nama atau kode perlengkapan..."
          toolbarActions={
            <div className="flex flex-col sm:flex-row gap-2 w-full sm:w-auto">
              <CustomDropdown
                value={filterBrandId}
                onChange={(val) => setFilterBrandId(val)}
                className="!mb-0 w-full sm:w-48"
                placeholder="Semua Brand"
                options={[
                  { value: '', label: 'Semua Brand' },
                  ...brands.map(b => ({ value: String(b.id), label: b.name }))
                ]}
              />
            </div>
          }
          renderCell={(row, key) => {
            if (key === 'brand_name') {
              return (
                <BrandCell
                  brandName={row.brand_name}
                  brandLogoUrl={row.brand_logo_url}
                  brandIconUrl={row.brand_icon_url}
                />
              );
            }
            if (key === 'nama_item') {
              return (
                <span className="font-heading font-medium text-neutral-900 text-sm">
                  {row.nama_item}
                </span>
              );
            }
            if (key === 'stok_tersedia') {
              return (
                <span className="font-heading font-medium text-neutral-900 text-sm">
                  {row.stok_tersedia}
                </span>
              );
            }
            if (key === 'aksi') {
              return (
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => handleOpenModal(row)}
                  className="text-xs"
                >
                  Atur Stok
                </Button>
              );
            }
            return row[key];
          }}
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

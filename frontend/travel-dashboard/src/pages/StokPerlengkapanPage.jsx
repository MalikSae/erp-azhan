import React, { useState, useEffect, useMemo } from 'react';
import { Boxes, Package, XCircle } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { getStokPerlengkapan } from 'shared';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import StatTile from 'shared/ui/StatTile';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';

const STOK_OPTIONS = [
  { value: '', label: 'Semua Stok' },
  { value: 'habis', label: 'Stok habis' },
  { value: 'ada', label: 'Stok tersedia' },
];

const StokPerlengkapanPage = () => {
  const { brandInfo } = useAuth();
  const [stokList, setStokList] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [filterStok, setFilterStok] = useState('');

  const fetchStok = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const res = await getStokPerlengkapan(brandInfo?.id);
      setStokList(res || []);
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal memuat data stok perlengkapan.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchStok();
  }, [brandInfo?.id]);

  const rows = useMemo(() => stokList.filter(row => {
    const stok = row.stok_tersedia || 0;
    if (filterStok === 'habis') return stok === 0;
    if (filterStok === 'ada') return stok > 0;
    return true;
  }), [stokList, filterStok]);

  const stats = useMemo(() => ({
    item: stokList.length,
    totalStok: stokList.reduce((sum, r) => sum + (r.stok_tersedia || 0), 0),
    habis: stokList.filter(r => (r.stok_tersedia || 0) === 0).length,
  }), [stokList]);

  const columns = [
    { header: 'Nama Item', key: 'nama', sortable: true },
    { 
      header: 'Set / Pax', 
      key: 'qty_per_set', 
      sortable: true,
      sortFn: (a, b) => (a.qty_per_set || 0) - (b.qty_per_set || 0)
    },
    { 
      header: 'Stok Tersedia', 
      key: 'stok_tersedia', 
      sortable: true,
      sortFn: (a, b) => (a.stok_tersedia || 0) - (b.stok_tersedia || 0)
    },
    { header: 'Status', key: 'status' },
  ];

  const renderCell = (row, key) => {
    switch (key) {
      case 'nama':
        return (
          <span className="font-heading font-semibold text-neutral-900 leading-snug">
            {row.nama}
          </span>
        );
      case 'qty_per_set':
        return (
          <span className="text-[13px] font-body text-neutral-600">
            {row.qty_per_set || 0} pcs / pax
          </span>
        );
      case 'stok_tersedia':
        return (
          <span className="text-[13px] font-body font-semibold text-neutral-900">
            {row.stok_tersedia || 0} pcs
          </span>
        );
      case 'status': {
        const stok = row.stok_tersedia || 0;
        if (stok > 0) {
          return (
            <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-neutral-700">
              <span className="w-1.5 h-1.5 rounded-full bg-success-500" />Tersedia
            </span>
          );
        }
        return (
          <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-danger-600">
            <span className="w-1.5 h-1.5 rounded-full bg-danger-500" />Stok Habis
          </span>
        );
      }
      default:
        return row[key];
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Stok Perlengkapan" />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}

      {!isLoading && (
        <div className="grid grid-cols-2 lg:grid-cols-3 gap-4">
          <StatTile icon={Boxes} variant="primary" value={stats.item} label="Item" />
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
          emptyMessage="Belum ada data stok perlengkapan untuk travel Anda."
          searchPlaceholder="Cari nama atau kode perlengkapan..."
          toolbarActions={
            <div className="flex flex-wrap items-center gap-2">
              <CustomDropdown
                value={filterStok}
                onChange={(val) => setFilterStok(val)}
                className="!mb-0 w-40"
                placeholder="Semua Stok"
                options={STOK_OPTIONS}
              />
            </div>
          }
          renderCell={renderCell}
        />
      )}
    </div>
  );
};

export default StokPerlengkapanPage;

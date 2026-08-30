import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { getStokPerlengkapan } from 'shared';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Badge from '../components/ui/Badge';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';

const StokPerlengkapanPage = () => {
  const { brandInfo } = useAuth();
  const [stokList, setStokList] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);

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
          <span className="font-heading font-medium text-neutral-900 text-sm">
            {row.nama}
          </span>
        );
      case 'qty_per_set':
        return (
          <span className="text-sm font-medium text-neutral-600">
            {row.qty_per_set || 0} pcs / pax
          </span>
        );
      case 'stok_tersedia':
        return (
          <span className="font-mono font-medium text-sm text-neutral-900">
            {row.stok_tersedia || 0} pcs
          </span>
        );
      case 'status': {
        const stok = row.stok_tersedia || 0;
        if (stok > 0) {
          return <Badge variant="published">Tersedia</Badge>;
        }
        return <Badge variant="draft">Stok Habis</Badge>;
      }
      default:
        return row[key];
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Stok Perlengkapan"
        subtitle={`Daftar ketersediaan stok fisik perlengkapan ibadah untuk ${brandInfo?.name || 'Travel'}.`}
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
          columns={columns}
          data={stokList}
          itemsPerPage={15}
          emptyMessage="Belum ada data stok perlengkapan untuk travel Anda."
          searchPlaceholder="Cari nama atau kode perlengkapan..."
          renderCell={renderCell}
        />
      )}
    </div>
  );
};

export default StokPerlengkapanPage;

import React, { useState, useEffect, useMemo, useCallback, useContext } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import CustomDropdown from 'shared/ui/CustomDropdown';
import { listPerlengkapanItems, listMutasiPerlengkapan } from '../api/perlengkapan';
import { listBrands } from '../api/brands';

const PAGE_SIZE = 100;

const JENIS_OPTIONS = [
  { value: '', label: 'Semua jenis' },
  { value: 'pembelian', label: 'Pembelian' },
  { value: 'distribusi', label: 'Diserahkan ke jamaah' },
  { value: 'retur', label: 'Penyerahan dibatalkan' },
  { value: 'koreksi', label: 'Koreksi stok' },
];
const JENIS_LABEL = Object.fromEntries(JENIS_OPTIONS.filter((o) => o.value).map((o) => [o.value, o.label]));

const formatWaktu = (value) => {
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '-';
  return new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(d);
};

const formatRupiah = (v) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(v) || 0);

const valueOf = (v) => (v?.target ? v.target.value : v);

// Riwayat pergerakan stok perlengkapan (buku besar): pembelian, penyerahan ke jamaah,
// pembatalan penyerahan, dan koreksi. Dimuat per 100 baris; yang lebih lama bisa dimuat bertahap.
const InventoryRiwayatStokPage = () => {
  const { isHolding } = useContext(AuthContext);
  const [searchParams, setSearchParams] = useSearchParams();
  const itemId = searchParams.get('item') || '';
  const brandId = searchParams.get('brand') || '';
  const jenis = searchParams.get('jenis') || '';

  const [items, setItems] = useState([]);
  const [brands, setBrands] = useState([]);
  const [rows, setRows] = useState([]);
  const [hasMore, setHasMore] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [errorMessage, setErrorMessage] = useState(null);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set(key, value); else next.delete(key);
    setSearchParams(next, { replace: true });
  };

  useEffect(() => {
    listPerlengkapanItems().then((i) => setItems(i || [])).catch(() => {});
    // Daftar brand hanya relevan (dan hanya boleh dimuat) untuk akun holding.
    if (isHolding) listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, [isHolding]);

  const params = useMemo(() => ({
    ...(itemId ? { item_id: itemId } : {}),
    ...(brandId ? { brand_id: brandId } : {}),
    ...(jenis ? { jenis } : {}),
    limit: PAGE_SIZE,
  }), [itemId, brandId, jenis]);

  const load = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listMutasiPerlengkapan(params);
      setRows(data || []);
      setHasMore((data || []).length === PAGE_SIZE);
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal memuat riwayat stok.');
    } finally {
      setIsLoading(false);
    }
  }, [params]);

  useEffect(() => { load(); }, [load]);

  const loadMore = async () => {
    const last = rows[rows.length - 1];
    if (!last) return;
    setIsLoadingMore(true);
    try {
      const data = await listMutasiPerlengkapan({ ...params, before_id: last.id });
      setRows((prev) => [...prev, ...(data || [])]);
      setHasMore((data || []).length === PAGE_SIZE);
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal memuat riwayat stok.');
    } finally {
      setIsLoadingMore(false);
    }
  };

  const columns = [
    { header: 'Waktu', key: 'created_at' },
    { header: 'Item', key: 'item_nama' },
    { header: 'Brand', key: 'lokasi' },
    { header: 'Jenis', key: 'jenis' },
    { header: 'Jumlah', key: 'qty' },
    { header: 'Stok', key: 'stok_setelah' },
    { header: 'Keterangan', key: 'ket' },
  ];

  const renderCell = (m, key) => {
    if (key === 'created_at') return <span className="whitespace-nowrap">{formatWaktu(m.created_at)}</span>;
    if (key === 'item_nama') return <span className="font-medium text-neutral-900">{m.item_nama}</span>;
    if (key === 'jenis') return JENIS_LABEL[m.jenis] || m.jenis;
    if (key === 'qty') {
      return (
        <span className={`font-semibold ${m.qty < 0 ? 'text-danger-600' : 'text-success-700'}`}>
          {m.qty > 0 ? `+${m.qty}` : m.qty}
        </span>
      );
    }
    if (key === 'ket') {
      const parts = [
        m.vendor,
        m.harga_satuan != null ? `@ ${formatRupiah(m.harga_satuan)}` : null,
        m.referensi,
        m.catatan,
        m.oleh ? `oleh ${m.oleh}` : null,
      ].filter(Boolean);
      return <span className="text-xs text-neutral-600">{parts.length ? parts.join(' · ') : '-'}</span>;
    }
    return m[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Riwayat Stok" />

      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <div className="space-y-4">
          <DataTable
            columns={columns}
            data={rows}
            itemsPerPage={15}
            searchPlaceholder="Cari item, brand, jamaah, atau keterangan..."
            emptyMessage="Belum ada pergerakan stok untuk filter ini."
            toolbarActions={
              <div className="flex flex-wrap items-center gap-2">
                <CustomDropdown
                  value={itemId}
                  onChange={(v) => setFilter('item', valueOf(v))}
                  className="!mb-0 w-48"
                  options={[{ value: '', label: 'Semua item' }, ...items.map((i) => ({ value: String(i.id), label: i.nama }))]}
                />
                {isHolding && (
                  <CustomDropdown
                    value={brandId}
                    onChange={(v) => setFilter('brand', valueOf(v))}
                    className="!mb-0 w-40"
                    options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
                  />
                )}
                <CustomDropdown
                  value={jenis}
                  onChange={(v) => setFilter('jenis', valueOf(v))}
                  className="!mb-0 w-52"
                  options={JENIS_OPTIONS}
                />
              </div>
            }
            renderCell={renderCell}
          />
          {hasMore && (
            <div className="flex justify-center">
              <Button variant="secondary" onClick={loadMore} disabled={isLoadingMore}>
                {isLoadingMore ? 'Memuat...' : 'Muat riwayat lebih lama'}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default InventoryRiwayatStokPage;

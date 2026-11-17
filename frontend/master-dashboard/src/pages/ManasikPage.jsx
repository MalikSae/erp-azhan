import React, { useCallback, useContext, useEffect, useState } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import SegmentedTabs from 'shared/ui/SegmentedTabs';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Alert from 'shared/ui/Alert';
import { AuthContext } from '../context/AuthContext';
import { tanggalPendek } from '../components/PilihPaketManasik';
import { listManasik } from '../api/manasik';
import { listBrandOptions } from '../api/brands';

const TABS = [
  { id: 'akan_datang', label: 'Akan datang' },
  { id: 'selesai', label: 'Selesai' },
];

const jam = (hhmm) => (hhmm || '').replace(':', '.');
const valueOf = (v) => (v?.target ? v.target.value : v);

// Daftar manasik. Tab Akan datang: hari ini dan sesudahnya; Selesai: sebelum hari ini, dengan jumlah yang benar-benar hadir.
const ManasikPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { isHolding, hasPermission } = useContext(AuthContext);
  const bisaUbah = hasPermission('manasik.edit');
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = searchParams.get('tab') === 'selesai' ? 'selesai' : 'akan_datang';
  const brandId = searchParams.get('brand') || '';
  const [rows, setRows] = useState([]);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);

  useEffect(() => {
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, '');
    }
  }, [location.state]);

  useEffect(() => {
    if (isHolding) listBrandOptions().then((b) => setBrands(b || [])).catch(() => {});
  }, [isHolding]);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set(key, value); else next.delete(key);
    setSearchParams(next, { replace: true });
  };

  const load = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      setRows((await listManasik({ tab, ...(brandId ? { brand_id: brandId } : {}) })) || []);
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal memuat manasik.');
    } finally {
      setIsLoading(false);
    }
  }, [tab, brandId]);

  useEffect(() => { load(); }, [load]);

  const columns = [
    { header: 'Manasik', key: 'judul' },
    { header: 'Waktu', key: 'tanggal' },
    { header: 'Paket', key: 'jumlah_paket', align: 'right' },
    { header: 'Jamaah', key: 'jumlah_jamaah', align: 'right' },
    { header: 'Konfirmasi hadir', key: 'berniat_hadir', align: 'right' },
    ...(tab === 'selesai' ? [{ header: 'Hadir', key: 'hadir', align: 'right' }] : []),
  ];

  const renderCell = (r, key) => {
    if (key === 'judul') {
      return (
        <div className="min-w-0">
          <div className="font-medium text-neutral-900">{r.judul}</div>
          <div className="text-xs text-neutral-500">{r.tempat}{isHolding ? ` · ${r.brand_nama}` : ''}</div>
        </div>
      );
    }
    if (key === 'tanggal') {
      return (
        <div className="whitespace-nowrap">
          <div>{tanggalPendek(r.tanggal)}</div>
          <div className="text-xs text-neutral-500">{jam(r.jam_mulai)}–{jam(r.jam_selesai)} WIB</div>
        </div>
      );
    }
    if (['jumlah_paket', 'jumlah_jamaah', 'berniat_hadir', 'hadir'].includes(key)) {
      return <span className="tabular-nums">{r[key]}</span>;
    }
    return r[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Manasik"
        actionLabel={bisaUbah ? '+ Buat Manasik' : undefined}
        onAction={bisaUbah ? () => navigate('/manasik/baru') : undefined}
      />
      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}
      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      <DataTable
        columns={columns}
        data={rows}
        loading={isLoading}
        itemsPerPage={20}
        searchPlaceholder="Cari manasik"
        emptyMessage="Belum ada manasik."
        toolbarLeading={<SegmentedTabs tabs={TABS} value={tab} onChange={(v) => setFilter('tab', v === 'akan_datang' ? '' : v)} />}
        toolbarActions={isHolding ? (
          <CustomDropdown
            value={brandId}
            onChange={(v) => setFilter('brand', valueOf(v))}
            className="!mb-0 w-40"
            options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
          />
        ) : null}
        renderCell={renderCell}
        onRowClick={(r) => navigate(`/manasik/${r.id}`)}
      />
    </div>
  );
};

export default ManasikPage;

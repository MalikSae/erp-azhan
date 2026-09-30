import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listAgen } from '../api/agen';
import { listBrands } from '../api/brands';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';
import BrandCell from '../components/BrandCell';
import { dateLabel, money, STATUS_AGEN } from '../utils/agen';

// Screen B2 — daftar agen (aktif & nonaktif). Admin Travel: brand sendiri.
// Admin Master (showBrandColumn): lintas brand dengan filter brand. Tidak ada
// aksi di baris; klik baris membuka Detail Agen (B2a).
export default function DaftarAgenPage({ showBrandColumn = false }) {
  const navigate = useNavigate();
  const [items, setItems] = useState([]);
  const [brands, setBrands] = useState([]);
  const [filterBrandId, setFilterBrandId] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listAgen()
      .then(setItems)
      .catch(() => setError('Daftar agen gagal dimuat. Silakan coba kembali.'))
      .finally(() => setLoading(false));
    if (showBrandColumn) listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, [showBrandColumn]);

  const brandsMap = useMemo(() => Object.fromEntries(brands.map((b) => [b.id, b])), [brands]);
  const data = useMemo(
    () => (filterBrandId ? items.filter((a) => String(a.brand_id) === filterBrandId) : items),
    [items, filterBrandId]
  );

  const columns = [
    { header: 'Nama Agen', key: 'nama_lengkap', sortable: true },
    ...(showBrandColumn ? [{ header: 'Brand', key: 'brand_name', sortable: true }] : []),
    { header: 'Kode Referral', key: 'kode_referral', sortable: true },
    { header: 'Status', key: 'status_agen', sortable: true },
    { header: 'Closing', key: 'jumlah_closing', sortable: true },
    { header: 'Total Komisi', key: 'total_komisi', sortable: true },
    { header: 'Disetujui', key: 'disetujui_agen_at', sortable: true },
  ];

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') return <span className="font-semibold text-neutral-900">{row.nama_lengkap}</span>;
    if (key === 'brand_name') return <BrandCell brand={brandsMap[row.brand_id]} brandName={row.brand_name} showText />;
    if (key === 'kode_referral') return <span className="font-mono text-neutral-800">{row.kode_referral || '-'}</span>;
    if (key === 'status_agen') {
      const meta = STATUS_AGEN[row.status_agen] || { label: row.status_agen, variant: 'neutral' };
      return <Badge variant={meta.variant}>{meta.label}</Badge>;
    }
    if (key === 'jumlah_closing') return <span className="font-semibold text-neutral-900">{row.jumlah_closing} jamaah</span>;
    if (key === 'total_komisi') return <span className="font-semibold text-neutral-900">{money(row.total_komisi)}</span>;
    if (key === 'disetujui_agen_at') return dateLabel(row.disetujui_agen_at);
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-5">
      <PageHeader
        title="Daftar Agen"
        subtitle={showBrandColumn
          ? 'Seluruh agen Syiar lintas brand. Total komisi tanpa cashback. Pilih agen untuk melihat detail dan kinerjanya.'
          : 'Seluruh agen Syiar brand ini. Total komisi tanpa cashback. Pilih agen untuk melihat detail dan kinerjanya.'}
      />
      {error && <Alert variant="error" message={error} />}
      <DataTable
        columns={columns}
        data={data}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/agen/${row.jamaah_id}`)}
        itemsPerPage={15}
        searchPlaceholder="Cari nama atau kode referral..."
        emptyMessage={loading ? 'Memuat agen...' : 'Belum ada agen'}
        toolbarActions={showBrandColumn && brands.length > 0 && (
          <CustomDropdown
            value={filterBrandId}
            onChange={(val) => setFilterBrandId(val?.target ? val.target.value : val)}
            className="!mb-0 w-40"
            placeholder="Semua Brand"
            options={[{ value: '', label: 'Semua Brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
          />
        )}
      />
    </div>
  );
}

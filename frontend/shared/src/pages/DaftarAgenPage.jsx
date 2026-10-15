import { useEffect, useMemo, useState } from 'react';
import BrandLabel from '../components/ui/BrandLabel';
import { useNavigate } from 'react-router-dom';
import { Network, CheckCircle2, Users, Banknote, Eye } from 'lucide-react';
import { listAgen } from '../api/agen';
import { listBrands } from '../api/brands';
import Alert from '../components/ui/Alert';
import ActionMenu from '../components/ui/ActionMenu';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';
import StatTile from '../components/ui/StatTile';
import { dateLabel, money, STATUS_AGEN } from '../utils/agen';

const STATUS_OPTIONS = [
  { value: '', label: 'Semua Status' },
  { value: 'aktif', label: 'Aktif' },
  { value: 'nonaktif', label: 'Nonaktif' },
];

// Brand: titik warna + nama (gaya sama dengan daftar Paket).
const BrandChip = ({ name, color }) => <BrandLabel name={name} color={color} />;

const StatusDot = ({ status }) => {
  const label = STATUS_AGEN[status]?.label || status || '-';
  return status === 'aktif' ? (
    <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-success-700 whitespace-nowrap">
      <span className="w-2 h-2 rounded-full bg-success-500" />{label}
    </span>
  ) : (
    <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-neutral-500 whitespace-nowrap">
      <span className="w-2 h-2 rounded-full bg-neutral-300" />{label}
    </span>
  );
};

// Screen B2 — daftar agen (aktif & nonaktif). Admin Travel: brand sendiri.
// Admin Master (showBrandColumn): lintas brand dengan filter brand. Klik baris
// atau menu "Lihat Detail" membuka Detail Agen (B2a).
export default function DaftarAgenPage({ showBrandColumn = false }) {
  const navigate = useNavigate();
  const [items, setItems] = useState([]);
  const [brands, setBrands] = useState([]);
  const [filterBrandId, setFilterBrandId] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listAgen()
      .then((data) => setItems(data || []))
      .catch(() => setError('Daftar agen gagal dimuat. Silakan coba kembali.'))
      .finally(() => setLoading(false));
    if (showBrandColumn) listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, [showBrandColumn]);

  const brandsMap = useMemo(() => Object.fromEntries(brands.map((b) => [b.id, b])), [brands]);

  // Ringkasan mengikuti filter brand supaya cocok dengan tabel.
  const brandItems = useMemo(
    () => (filterBrandId ? items.filter((a) => String(a.brand_id) === filterBrandId) : items),
    [items, filterBrandId]
  );
  const data = useMemo(
    () => (filterStatus ? brandItems.filter((a) => a.status_agen === filterStatus) : brandItems)
      // Kolom bantu pencarian: nama brand ikut bisa dicari.
      .map((a) => ({ ...a, _cari: brandsMap[a.brand_id]?.name || a.brand_name || '' })),
    [brandItems, filterStatus, brandsMap]
  );

  const stats = useMemo(() => ({
    total: brandItems.length,
    aktif: brandItems.filter((a) => a.status_agen === 'aktif').length,
    closing: brandItems.reduce((sum, a) => sum + (Number(a.jumlah_closing) || 0), 0),
    komisi: brandItems.reduce((sum, a) => sum + (Number(a.total_komisi) || 0), 0),
  }), [brandItems]);

  const columns = [
    { header: 'Agen', key: 'nama_lengkap', sortable: true },
    ...(showBrandColumn ? [{ header: 'Brand', key: 'brand_name', sortable: true }] : []),
    { header: 'Status', key: 'status_agen', sortable: true },
    { header: 'Closing', key: 'jumlah_closing', sortable: true, sortFn: (a, b) => (a.jumlah_closing || 0) - (b.jumlah_closing || 0) },
    { header: 'Total Komisi', key: 'total_komisi', sortable: true, sortFn: (a, b) => (Number(a.total_komisi) || 0) - (Number(b.total_komisi) || 0) },
    { header: 'Disetujui', key: 'disetujui_agen_at', sortable: true },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') {
      return (
        <div className="min-w-0">
          <div className="font-heading font-semibold text-neutral-900 leading-snug">{row.nama_lengkap}</div>
          <div className="text-[11px] font-body text-neutral-400">
            Kode <span className="font-mono">{row.kode_referral || '-'}</span>
          </div>
        </div>
      );
    }
    if (key === 'brand_name') {
      return <BrandChip name={brandsMap[row.brand_id]?.name || row.brand_name || '-'} color={brandsMap[row.brand_id]?.primary_color} />;
    }
    if (key === 'status_agen') return <StatusDot status={row.status_agen} />;
    if (key === 'jumlah_closing') return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.jumlah_closing || 0} jamaah</span>;
    if (key === 'total_komisi') return <span className="text-[13px] font-body font-semibold text-neutral-900 whitespace-nowrap">{money(row.total_komisi)}</span>;
    if (key === 'disetujui_agen_at') return <span className="text-[13px] font-body text-neutral-500 whitespace-nowrap">{dateLabel(row.disetujui_agen_at)}</span>;
    if (key === 'aksi') {
      return (
        <ActionMenu
          align="right"
          items={[{ label: 'Lihat Detail', icon: Eye, tooltip: 'Lihat detail dan kinerja agen', onClick: () => navigate(`/agen/${row.jamaah_id}`) }]}
        />
      );
    }
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Daftar Agen" />
      {error && <Alert variant="error" message={error} />}

      {!loading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Network} variant="primary" value={stats.total} label="Agen" />
          <StatTile icon={CheckCircle2} variant="success" value={stats.aktif} label="Aktif" />
          <StatTile icon={Users} variant="info" value={stats.closing} label="Closing" hint="Jamaah" />
          <StatTile icon={Banknote} variant="violet" value={money(stats.komisi)} label="Total komisi" hint="Tanpa cashback" />
        </div>
      )}

      <DataTable
        columns={columns}
        data={data}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/agen/${row.jamaah_id}`)}
        itemsPerPage={15}
        searchPlaceholder="Cari nama atau kode referral..."
        emptyMessage={loading ? 'Memuat agen...' : 'Belum ada agen'}
        toolbarActions={(
          <div className="flex flex-wrap items-center gap-2">
            <CustomDropdown
              value={filterStatus}
              onChange={(val) => setFilterStatus(val?.target ? val.target.value : val)}
              className="!mb-0 w-40"
              placeholder="Semua Status"
              options={STATUS_OPTIONS}
            />
            {showBrandColumn && brands.length > 0 && (
              <CustomDropdown
                value={filterBrandId}
                onChange={(val) => setFilterBrandId(val?.target ? val.target.value : val)}
                className="!mb-0 w-40"
                placeholder="Semua Brand"
                options={[{ value: '', label: 'Semua Brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
              />
            )}
          </div>
        )}
      />
    </div>
  );
}

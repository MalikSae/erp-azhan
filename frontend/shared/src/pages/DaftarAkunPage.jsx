import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Eye } from 'lucide-react';
import { listAkunPelanggan } from '../api/akun';
import { listBrands } from '../api/brands';
import Alert from '../components/ui/Alert';
import ActionMenu from '../components/ui/ActionMenu';
import BrandLabel from '../components/ui/BrandLabel';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';
import { dateLabel } from '../utils/agen';

const PER_PAGE = 20;

export const STATUS_AKUN = {
  tidak_aktif: { label: 'Akun biasa', dot: 'bg-neutral-300', text: 'text-neutral-500' },
  pengajuan: { label: 'Mengajukan agen', dot: 'bg-warning-500', text: 'text-warning-700' },
  aktif: { label: 'Agen aktif', dot: 'bg-success-500', text: 'text-success-700' },
  nonaktif: { label: 'Agen nonaktif', dot: 'bg-neutral-300', text: 'text-neutral-500' },
};

const STATUS_OPTIONS = [{ value: '', label: 'Semua Status' }, ...Object.entries(STATUS_AKUN).map(([value, s]) => ({ value, label: s.label }))];
const UPLINE_OPTIONS = [
  { value: '', label: 'Semua Upline' },
  { value: 'ada', label: 'Punya upline' },
  { value: 'tanpa', label: 'Tanpa upline' },
];

export const StatusAkun = ({ status }) => {
  const s = STATUS_AKUN[status] || STATUS_AKUN.tidak_aktif;
  return (
    <span className={`inline-flex items-center gap-1.5 text-[12px] font-body whitespace-nowrap ${s.text}`}>
      <span className={`w-2 h-2 rounded-full ${s.dot}`} />{s.label}
    </span>
  );
};

const COLUMNS_BASE = [
  { header: 'Akun', key: 'nama_lengkap' },
  { header: 'Status', key: 'status_agen' },
  { header: 'Upline', key: 'upline_nama' },
  { header: 'Terdaftar', key: 'created_at' },
  { header: 'Aksi', key: 'aksi' },
];

// Daftar akun pelanggan (baca saja). Paging dan pencarian dari server. Admin brand hanya melihat brand-nya;
// Admin Master (showBrandColumn) lintas brand dengan filter brand.
export default function DaftarAkunPage({ showBrandColumn = false }) {
  const navigate = useNavigate();
  const [brands, setBrands] = useState([]);
  const [brandId, setBrandId] = useState('');
  const [status, setStatus] = useState('');
  const [upline, setUpline] = useState('');
  const [input, setInput] = useState('');
  const [q, setQ] = useState('');
  const [page, setPage] = useState(1);
  const [hasil, setHasil] = useState({ items: [], total: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (showBrandColumn) listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, [showBrandColumn]);

  // Pencarian ditunda supaya API tidak dipanggil di tiap ketikan.
  useEffect(() => {
    const t = setTimeout(() => { setQ(input.trim()); setPage(1); }, 300);
    return () => clearTimeout(t);
  }, [input]);

  useEffect(() => {
    let batal = false;
    setLoading(true);
    listAkunPelanggan({ q, brandId, statusAgen: status, upline, page, perPage: PER_PAGE })
      .then((data) => { if (!batal) { setHasil(data); setError(''); } })
      .catch(() => { if (!batal) setError('Daftar akun gagal dimuat. Silakan coba kembali.'); })
      .finally(() => { if (!batal) setLoading(false); });
    return () => { batal = true; };
  }, [q, brandId, status, upline, page]);

  const columns = useMemo(
    () => (showBrandColumn ? [COLUMNS_BASE[0], { header: 'Brand', key: 'brand_name' }, ...COLUMNS_BASE.slice(1)] : COLUMNS_BASE),
    [showBrandColumn]
  );

  const filter = (setter) => (val) => { setter(val?.target ? val.target.value : val); setPage(1); };

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') {
      return (
        <div className="min-w-0">
          <div className="font-heading font-semibold text-neutral-900 leading-snug">{row.nama_lengkap}</div>
          <div className="text-[11px] font-body text-neutral-400 break-all">{row.email} · {row.no_hp}</div>
        </div>
      );
    }
    if (key === 'brand_name') return <BrandLabel name={row.brand_name} />;
    if (key === 'status_agen') return <StatusAkun status={row.status_agen} />;
    if (key === 'upline_nama') return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.upline_nama || '-'}</span>;
    if (key === 'created_at') return <span className="text-[13px] font-body text-neutral-500 whitespace-nowrap">{dateLabel(row.created_at)}</span>;
    if (key === 'aksi') {
      return <ActionMenu align="right" items={[{ label: 'Lihat Detail', icon: Eye, onClick: () => navigate(`/akun/${row.akun_id}`) }]} />;
    }
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Akun Pelanggan" subtitle="Akun yang bisa booking di situs. Ubah upline lewat menu Ganti Kaitan Agen." />
      {error && <Alert variant="error" message={error} />}
      <DataTable
        columns={columns}
        data={hasil.items}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/akun/${row.akun_id}`)}
        server={{ total: hasil.total, page, perPage: PER_PAGE, onPage: setPage, query: input, onQuery: setInput }}
        searchPlaceholder="Cari nama, email, WhatsApp, atau kode referral..."
        emptyMessage={loading ? 'Memuat akun...' : 'Belum ada akun'}
        toolbarActions={(
          <div className="flex flex-wrap items-center gap-2">
            <CustomDropdown value={status} onChange={filter(setStatus)} className="!mb-0 w-44" placeholder="Semua Status" options={STATUS_OPTIONS} />
            <CustomDropdown value={upline} onChange={filter(setUpline)} className="!mb-0 w-40" placeholder="Semua Upline" options={UPLINE_OPTIONS} />
            {showBrandColumn && brands.length > 0 && (
              <CustomDropdown
                value={brandId}
                onChange={filter(setBrandId)}
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

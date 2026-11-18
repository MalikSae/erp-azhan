import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { listAkunPelanggan } from '../api/akun';
import { listBrandOptions } from '../api/brands';
import { ResetHitungModal } from '../components/AktivasiAkunModal';
import Alert from '../components/ui/Alert';
import Badge, { toneFromClass } from '../components/ui/Badge';
import BrandLabel from '../components/ui/BrandLabel';
import Button from '../components/ui/Button';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';

const PER_PAGE = 20;

export const STATUS_AKUN = {
  tidak_aktif: { label: 'Akun biasa', dot: 'bg-neutral-300', text: 'text-neutral-500' },
  pengajuan: { label: 'Mengajukan agen', dot: 'bg-warning-500', text: 'text-warning-700' },
  aktif: { label: 'Agen aktif', dot: 'bg-success-500', text: 'text-success-700' },
  nonaktif: { label: 'Agen nonaktif', dot: 'bg-neutral-300', text: 'text-neutral-500' },
};

const AGEN_OPTIONS = [
  { value: '', label: 'Semua Akun' },
  { value: 'agen', label: 'Agen' },
  { value: 'bukan', label: 'Bukan agen' },
];
const UPLINE_OPTIONS = [
  { value: '', label: 'Semua Upline' },
  { value: 'ada', label: 'Punya upline' },
  { value: 'tanpa', label: 'Tanpa upline' },
];

export const StatusAkun = ({ status }) => {
  const s = STATUS_AKUN[status] || STATUS_AKUN.tidak_aktif;
  return <Badge variant={toneFromClass(s.dot)}>{s.label}</Badge>;
};

// Kolom hanya memuat data milik akun. Closing, komisi, dan kode referral ada di menu Agen; di sini
// cukup status keagenannya. Tanpa kolom Aksi: klik baris (atau nama) membuka detail.
const COLUMNS_BASE = [
  { header: 'Akun', key: 'nama_lengkap' },
  { header: 'Upline', key: 'upline_nama' },
  { header: 'Jamaah Tertaut', key: 'jamaah_nama' },
  { header: 'Booking', key: 'jumlah_booking' },
  { header: 'Agen', key: 'agen' },
];

// Sel kolom Agen: status keagenan akun itu, dengan kata yang sama dengan menu Agen. "-" untuk akun biasa.
// Kode referral dan rincian lain ada di halaman agen.
const STATUS_KEAGENAN = {
  aktif: { label: 'Aktif', dot: 'bg-success-500', text: 'text-success-700' },
  nonaktif: { label: 'Nonaktif', dot: 'bg-neutral-300', text: 'text-neutral-500' },
  pengajuan: { label: 'Mengajukan', dot: 'bg-warning-500', text: 'text-warning-700' },
};
const TandaAgen = ({ row }) => {
  const s = STATUS_KEAGENAN[row.status_agen];
  if (!s) return <span className="text-sm text-neutral-400">-</span>;
  return <Badge variant={toneFromClass(s.dot)}>{s.label}</Badge>;
};

// Daftar akun (baca saja). Paging dan pencarian dari server. Admin brand hanya melihat brand-nya;
// Admin Master (showBrandColumn) lintas brand dengan filter brand. bolehResetHitung (akun.edit) menampilkan
// Reset Batas Aktivasi; Admin Master harus memilih brand dulu.
export default function DaftarAkunPage({ showBrandColumn = false, bolehResetHitung = false, bolehLihatJamaah = false }) {
  const navigate = useNavigate();
  const [brands, setBrands] = useState([]);
  const [brandId, setBrandId] = useState('');
  const [agen, setAgen] = useState('');
  const [upline, setUpline] = useState('');
  const [input, setInput] = useState('');
  const [q, setQ] = useState('');
  const [page, setPage] = useState(1);
  const [hasil, setHasil] = useState({ items: [], total: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [resetBuka, setResetBuka] = useState(false);
  const [pesan, setPesan] = useState('');

  useEffect(() => {
    if (showBrandColumn) listBrandOptions().then((b) => setBrands(b || [])).catch(() => {});
  }, [showBrandColumn]);

  // Pencarian ditunda supaya API tidak dipanggil di tiap ketikan.
  useEffect(() => {
    const t = setTimeout(() => { setQ(input.trim()); setPage(1); }, 300);
    return () => clearTimeout(t);
  }, [input]);

  useEffect(() => {
    let batal = false;
    setLoading(true);
    listAkunPelanggan({ q, brandId, agen, upline, page, perPage: PER_PAGE })
      .then((data) => { if (!batal) { setHasil(data); setError(''); } })
      .catch(() => { if (!batal) setError('Daftar akun gagal dimuat. Silakan coba kembali.'); })
      .finally(() => { if (!batal) setLoading(false); });
    return () => { batal = true; };
  }, [q, brandId, agen, upline, page]);

  const columns = useMemo(
    () => (showBrandColumn ? [COLUMNS_BASE[0], { header: 'Brand', key: 'brand_name' }, ...COLUMNS_BASE.slice(1)] : COLUMNS_BASE),
    [showBrandColumn]
  );

  const filter = (setter) => (val) => { setter(val?.target ? val.target.value : val); setPage(1); };

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') {
      return (
        <div className="min-w-0">
          <Link to={`/akun/${row.akun_id}`} onClick={(e) => e.stopPropagation()} className="font-body font-medium text-neutral-900 leading-snug hover:text-primary-600 transition-colors">{row.nama_lengkap}</Link>
          <div className="text-xs font-body text-neutral-400">{row.no_hp || '-'}</div>
        </div>
      );
    }
    if (key === 'brand_name') return <BrandLabel name={row.brand_name} />;
    if (key === 'upline_nama') return <span className="text-sm font-body text-neutral-700 whitespace-nowrap">{row.upline_nama || '-'}</span>;
    if (key === 'jamaah_nama') {
      if (!row.jamaah_id) return <span className="text-sm text-neutral-400">-</span>;
      return bolehLihatJamaah
        ? <Link to={`/jamaah/${row.jamaah_id}`} onClick={(e) => e.stopPropagation()} className="text-sm font-body text-neutral-700 whitespace-nowrap hover:text-primary-600 transition-colors">{row.jamaah_nama}</Link>
        : <span className="text-sm font-body text-neutral-700 whitespace-nowrap">{row.jamaah_nama}</span>;
    }
    if (key === 'jumlah_booking') return <span className="text-sm font-body text-neutral-700 whitespace-nowrap">{row.jumlah_booking > 0 ? `${row.jumlah_booking} booking` : '-'}</span>;
    if (key === 'agen') return <TandaAgen row={row} />;
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Akun">
        {bolehResetHitung && (
          <Button variant="secondary" onClick={() => setResetBuka(true)} disabled={showBrandColumn && !brandId} title={showBrandColumn && !brandId ? 'Pilih brand dulu' : undefined}>Reset Batas Aktivasi</Button>
        )}
      </PageHeader>
      {pesan && <Alert variant="success" message={pesan} onClose={() => setPesan('')} />}
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
            <CustomDropdown value={upline} onChange={filter(setUpline)} className="!mb-0 w-44" placeholder="Semua Upline" options={UPLINE_OPTIONS} />
            <CustomDropdown value={agen} onChange={filter(setAgen)} className="!mb-0 w-44" placeholder="Semua Akun" options={AGEN_OPTIONS} />
            {showBrandColumn && brands.length > 0 && (
              <CustomDropdown
                value={brandId}
                onChange={filter(setBrandId)}
                className="!mb-0 w-44"
                placeholder="Semua Brand"
                options={[{ value: '', label: 'Semua Brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
              />
            )}
          </div>
        )}
      />
      <ResetHitungModal
        brandId={showBrandColumn ? brandId : undefined}
        isOpen={resetBuka}
        onClose={() => setResetBuka(false)}
        onSaved={() => { setResetBuka(false); setPesan('Hitungan direset.'); }}
      />
    </div>
  );
}

import { useContext, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { TriangleAlert } from 'lucide-react';
import { dateLabel, listAgen, listBrandOptions, money } from 'shared';
import Alert from 'shared/ui/Alert';
import BrandLabel from 'shared/ui/BrandLabel';
import CustomDropdown from 'shared/ui/CustomDropdown';
import DataTable from 'shared/ui/DataTable';
import PageHeader from 'shared/ui/PageHeader';
import PeriodePicker from 'shared/ui/PeriodePicker';
import SegmentedTabs from 'shared/ui/SegmentedTabs';
import { AuthContext } from '../context/AuthContext';
import { BAYAR_META, muatPengajuanAgen, useJumlahPengajuanAgen } from '../utils/pengajuanAgen';

const ALAMAT_TAB = { semua: '/agen', pengajuan: '/agen/pengajuan' };

const STATUS = {
  aktif: { label: 'Aktif', dot: 'bg-success-500', text: 'text-success-700' },
  nonaktif: { label: 'Nonaktif', dot: 'bg-neutral-300', text: 'text-neutral-500' },
  pengajuan: { label: 'Mengajukan', dot: 'bg-warning-500', text: 'text-warning-700' },
};
// Urutan saat kolom Status diklik: yang butuh perhatian dulu.
const URUT_STATUS = { pengajuan: 0, aktif: 1, nonaktif: 2 };

const kosong = <span className="text-[13px] text-neutral-400">-</span>;

// Keadaan biaya pendaftaran satu pengajuan, untuk kolom Biaya Pendaftaran.
function keadaanBiaya(p) {
  if (!p) return null;
  if (p.status === 'menunggu_verifikasi' && !p.bukti_transfer_url) return { label: 'Belum ada bukti', dot: 'bg-neutral-300' };
  return BAYAR_META[p.status] || { label: p.status, dot: 'bg-neutral-300' };
}

// Menu Agen: semua agen dan antrean pengajuan dalam satu tabel. Empat kolom pertama (Agen, Brand, Upline,
// Status) sama di kedua tab dan tidak bergeser. Dua kolom terakhir mengikuti isi tab: Closing dan Komisi
// untuk agen; Diajukan dan Biaya Pendaftaran untuk pengajuan, karena calon agen belum punya closing.
// tab "semua" = /agen (agen aktif dan nonaktif), tab "pengajuan" = /agen/pengajuan (calon agen, dan agen
// yang tagihan pendaftarannya belum selesai). Tab Pengajuan hanya untuk pemegang izin agen.approve.
// Rentang waktu menggantikan halaman Peringkat Agen: Closing dan Komisi dihitung untuk rentang itu, dan
// urutan bawaan adalah closing terbanyak.
export default function KelolaAgenPage({ tab = 'semua', showBrandColumn = false }) {
  const navigate = useNavigate();
  const { hasPermission } = useContext(AuthContext);
  const bolehSetujui = hasPermission('agen.approve');
  const bolehLihat = hasPermission('agen.view');
  const jumlahPengajuan = useJumlahPengajuanAgen(bolehSetujui);

  const [brands, setBrands] = useState([]);
  const [brandId, setBrandId] = useState('');
  // Rentang waktu untuk Closing dan Komisi. preset kosong = sepanjang waktu.
  const [periode, setPeriode] = useState({ preset: '', dari: '', sampai: '' });
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listBrandOptions().then((b) => setBrands(b || [])).catch(() => {});
  }, []);
  const brandsMap = useMemo(() => Object.fromEntries(brands.map((b) => [b.id, b])), [brands]);

  useEffect(() => {
    let batal = false;
    setError('');
    if (tab === 'pengajuan') {
      setLoading(true);
      muatPengajuanAgen()
        .then((data) => {
          if (batal) return;
          // Pengajuan terlama di atas.
          const urut = [...data].sort((a, b) => new Date(a.diajukan_agen_at || 0) - new Date(b.diajukan_agen_at || 0));
          setRows(urut.map((it) => ({
            akun_id: it.akun_id,
            nama_lengkap: it.nama_lengkap,
            keterangan: it.no_hp || '-',
            brand_id: it.brand_id,
            upline_nama: it.direkrut_oleh_nama,
            status_agen: it.status_agen,
            periksa_nik: it.peringatan_nik?.length > 0,
            diajukan_agen_at: it.diajukan_agen_at,
            biaya: Number(it.pembayaran?.nominal_tagihan || 0),
            biaya_keadaan: keadaanBiaya(it.pembayaran),
          })));
        })
        .catch(() => { if (!batal) { setRows([]); setError('Pengajuan agen gagal dimuat. Silakan coba kembali.'); } })
        .finally(() => { if (!batal) setLoading(false); });
      return () => { batal = true; };
    }
    setLoading(true);
    listAgen({ brandId, dari: periode.dari, sampai: periode.sampai })
      .then((data) => {
        if (batal) return;
        // Closing terbanyak di atas, lalu komisi terbesar.
        const urut = [...data].sort((a, b) => (b.pax_closing - a.pax_closing) || (Number(b.total_komisi) - Number(a.total_komisi)) || a.nama_lengkap.localeCompare(b.nama_lengkap));
        setRows(urut.map((it) => ({ ...it, keterangan: it.kode_referral || '-' })));
      })
      .catch(() => { if (!batal) { setRows([]); setError('Daftar agen gagal dimuat. Silakan coba kembali.'); } })
      .finally(() => { if (!batal) setLoading(false); });
    return () => { batal = true; };
  }, [tab, brandId, periode]);

  // Tab Pengajuan dimuat tanpa filter brand dari server, jadi disaring di sini.
  const data = useMemo(
    () => (tab === 'pengajuan' && brandId ? rows.filter((r) => String(r.brand_id) === brandId) : rows),
    [rows, tab, brandId]
  );

  const columns = useMemo(() => [
    // Lebar tetap: posisi kolom sama persis di kedua tab.
    { header: 'Agen', key: 'nama_lengkap', sortable: true, width: '24%' },
    ...(showBrandColumn ? [{ header: 'Brand', key: 'brand_name', width: '20%' }] : []),
    { header: 'Upline', key: 'upline_nama', width: '16%' },
    { header: 'Status', key: 'status_agen', width: '14%', sortable: true, sortFn: (a, b) => (URUT_STATUS[a.status_agen] ?? 9) - (URUT_STATUS[b.status_agen] ?? 9) },
    ...(tab === 'pengajuan' ? [
      { header: 'Diajukan', key: 'diajukan_agen_at', width: '12%', sortable: true, sortFn: (a, b) => new Date(a.diajukan_agen_at || 0) - new Date(b.diajukan_agen_at || 0) },
      { header: 'Biaya Pendaftaran', key: 'biaya', width: '14%', sortable: true, sortFn: (a, b) => a.biaya - b.biaya },
    ] : [
      { header: 'Closing', key: 'pax_closing', width: '12%', sortable: true, sortFn: (a, b) => (a.pax_closing ?? 0) - (b.pax_closing ?? 0) },
      { header: 'Komisi', key: 'total_komisi', width: '14%', sortable: true, sortFn: (a, b) => Number(a.total_komisi ?? 0) - Number(b.total_komisi ?? 0) },
    ]),
  ], [showBrandColumn, tab]);

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') {
      return (
        <div className="min-w-0">
          <Link to={`/agen/${row.akun_id}`} onClick={(e) => e.stopPropagation()} className="font-heading font-semibold text-neutral-900 leading-snug hover:text-primary-600 transition-colors">{row.nama_lengkap}</Link>
          <div className="text-[11px] font-mono text-neutral-400">{row.keterangan}</div>
        </div>
      );
    }
    if (key === 'brand_name') return <BrandLabel name={brandsMap[row.brand_id]?.name || row.brand_name || '-'} color={brandsMap[row.brand_id]?.primary_color} />;
    if (key === 'upline_nama') return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.upline_nama || '-'}</span>;
    if (key === 'status_agen') {
      const s = STATUS[row.status_agen] || { label: row.status_agen || '-', dot: 'bg-neutral-300', text: 'text-neutral-500' };
      return (
        <div>
          <span className={`inline-flex items-center gap-1.5 text-[12px] font-body whitespace-nowrap ${s.text}`}>
            <span className={`w-2 h-2 rounded-full ${s.dot}`} />{s.label}
          </span>
          {row.periksa_nik && <div className="mt-0.5 flex items-center gap-1 text-[11px] font-semibold text-warning-700 whitespace-nowrap"><TriangleAlert size={12} />Periksa NIK</div>}
        </div>
      );
    }
    if (key === 'diajukan_agen_at') return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{dateLabel(row.diajukan_agen_at)}</span>;
    if (key === 'biaya') {
      if (!row.biaya) return <span className="text-[13px] font-body text-neutral-500 whitespace-nowrap">Tanpa biaya</span>;
      return (
        <div>
          <div className="text-[13px] font-body font-semibold text-neutral-900 whitespace-nowrap">{money(row.biaya)}</div>
          {row.biaya_keadaan && (
            <span className="mt-0.5 inline-flex items-center gap-1.5 text-[11px] font-body text-neutral-500 whitespace-nowrap">
              <span className={`w-1.5 h-1.5 rounded-full ${row.biaya_keadaan.dot}`} />{row.biaya_keadaan.label}
            </span>
          )}
        </div>
      );
    }
    if (key === 'pax_closing') return row.pax_closing == null ? kosong : <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.pax_closing > 0 ? `${row.pax_closing} pax` : '-'}</span>;
    if (key === 'total_komisi') return row.total_komisi == null ? kosong : <span className="text-[13px] font-body font-semibold text-neutral-900 whitespace-nowrap">{Number(row.total_komisi) > 0 ? money(row.total_komisi) : '-'}</span>;
    return row[key] ?? '-';
  };

  const tabs = [
    bolehLihat && { id: 'semua', label: 'Semua Agen' },
    bolehSetujui && { id: 'pengajuan', label: 'Pengajuan', count: jumlahPengajuan ?? undefined },
  ].filter(Boolean);
  // Satu tab saja tidak perlu pemilih tab.
  const pemilihTab = tabs.length > 1
    ? <SegmentedTabs tabs={tabs} value={tab} onChange={(id) => navigate(ALAMAT_TAB[id])} />
    : undefined;

  const pilih = (setter) => (val) => setter(val?.target ? val.target.value : val);

  return (
    <div className="space-y-6">
      <PageHeader title="Agen" />
      {pemilihTab}
      {error && <Alert variant="error" message={error} />}
      <DataTable
        columns={columns}
        data={data}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/agen/${row.akun_id}`)}
        itemsPerPage={20}
        searchPlaceholder="Cari nama, kode referral, atau upline..."
        emptyMessage={loading ? 'Memuat...' : tab === 'pengajuan' ? 'Tidak ada pengajuan agen dalam antrean' : 'Belum ada agen'}
        toolbarActions={(
          <div className="flex flex-wrap items-center gap-2">
            {/* Calon agen belum punya closing dan komisi, jadi rentang waktu tidak berlaku di tab Pengajuan. */}
            {tab !== 'pengajuan' && (
              <PeriodePicker value={periode} onChange={setPeriode} labelSemua="Sepanjang waktu" className="w-48" />
            )}
            {showBrandColumn && brands.length > 0 && (
              <CustomDropdown
                value={brandId}
                onChange={pilih(setBrandId)}
                className="!mb-0 w-44"
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

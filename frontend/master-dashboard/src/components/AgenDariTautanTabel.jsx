import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { TriangleAlert } from 'lucide-react';
import { dateLabel, listAgenDariTautan } from 'shared';
import Alert from 'shared/ui/Alert';
import Badge from 'shared/ui/Badge';
import BrandLabel from 'shared/ui/BrandLabel';
import DataTable from 'shared/ui/DataTable';
import { STATUS_AGEN_ADMIN as STATUS } from '../utils/statusAgen';

const PER_PAGE = 20;


const tautan = 'font-body font-medium text-neutral-900 leading-snug hover:text-primary-600 transition-colors';
const peringatan = 'mt-0.5 flex items-center gap-1 text-xs font-semibold text-warning-700 whitespace-nowrap';

// Audit akun yang menjadi agen lewat jamaah tertaut (ADR 0003). Dua penanda perlu diperiksa admin:
// upline akun beda dari pemilik kunci jamaah, dan booking pendukung (dp/lunas) sudah tidak aktif.
// Baris membuka detail akun, tempat Lepas Tautan dan Ganti Upline. Paging dari server. Pemanggil memberi
// `key={brandId}` supaya ganti brand mengulang dari halaman 1.
export default function AgenDariTautanTabel({ brandId, brandsMap, showBrandColumn, toolbarActions }) {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [hasil, setHasil] = useState({ items: [], total: 0 });
  const [dimuat, setDimuat] = useState(null); // halaman yang sudah selesai dimuat
  const [error, setError] = useState('');
  const loading = dimuat !== page;

  useEffect(() => {
    let batal = false;
    listAgenDariTautan({ brandId, page, perPage: PER_PAGE })
      .then((data) => { if (!batal) { setHasil(data); setError(''); } })
      .catch(() => { if (!batal) setError('Daftar agen dari tautan gagal dimuat. Coba lagi.'); })
      .finally(() => { if (!batal) setDimuat(page); });
    return () => { batal = true; };
  }, [brandId, page]);

  const columns = useMemo(() => [
    { header: 'Agen', key: 'nama_lengkap', width: '17%' },
    ...(showBrandColumn ? [{ header: 'Brand', key: 'brand_name', width: '19%' }] : []),
    { header: 'Jamaah Tertaut', key: 'jamaah_nama', width: '20%' },
    { header: 'Upline', key: 'upline_nama', width: '12%' },
    { header: 'Status', key: 'status_agen', width: '15%' },
    { header: 'Disetujui', key: 'disetujui_agen_at', width: '17%' },
  ], [showBrandColumn]);

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') {
      return <Link to={`/akun/${row.akun_id}`} onClick={(e) => e.stopPropagation()} className={tautan}>{row.nama_lengkap}</Link>;
    }
    if (key === 'brand_name') return <BrandLabel name={brandsMap[row.brand_id]?.name || '-'} color={brandsMap[row.brand_id]?.primary_color} />;
    if (key === 'jamaah_nama') {
      return (
        <div className="min-w-0">
          <div className="text-sm font-body text-neutral-900">{row.jamaah_nama}</div>
          <div className="text-xs font-body text-neutral-500">Pemilik kunci: {row.pemilik_kunci_nama || '-'}</div>
        </div>
      );
    }
    if (key === 'upline_nama') return <span className="text-sm font-body text-neutral-700 whitespace-nowrap">{row.upline_nama || '-'}</span>;
    if (key === 'status_agen') {
      const s = STATUS[row.status_agen] || { label: row.status_agen, variant: 'neutral' };
      return (
        <div>
          <Badge variant={s.variant}>{s.label}</Badge>
          {row.pemilik_kunci_akun_id && !row.upline_sama_kunci && <div className={peringatan}><TriangleAlert size={12} />Upline beda</div>}
          {!row.booking_pendukung_aktif && <div className={peringatan}><TriangleAlert size={12} />Booking batal</div>}
        </div>
      );
    }
    if (key === 'disetujui_agen_at') return <span className="text-sm font-body text-neutral-700 whitespace-nowrap">{row.disetujui_agen_at ? dateLabel(row.disetujui_agen_at) : '-'}</span>;
    return row[key] ?? '-';
  };

  return (
    <>
      {error && <Alert variant="error" message={error} />}
      <DataTable
        columns={columns}
        data={hasil.items}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/akun/${row.akun_id}`)}
        server={{ total: hasil.total, page, perPage: PER_PAGE, onPage: setPage, query: '', onQuery: () => {} }}
        searchable={false}
        loading={loading}
        emptyMessage="Belum ada agen dari tautan"
        toolbarActions={toolbarActions}
      />
    </>
  );
}

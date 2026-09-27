import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listAgen } from 'shared';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';
import { dateLabel, STATUS_AGEN } from '../utils/agen';

// Screen B2 — daftar agen brand ini (aktif & nonaktif). Tidak ada aksi di
// baris; klik baris membuka Detail Agen (B2a).
export default function DaftarAgenPage() {
  const navigate = useNavigate();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listAgen()
      .then(setItems)
      .catch(() => setError('Daftar agen gagal dimuat. Silakan coba kembali.'))
      .finally(() => setLoading(false));
  }, []);

  const columns = [
    { header: 'Nama Agen', key: 'nama_lengkap', sortable: true },
    { header: 'Kode Referral', key: 'kode_referral', sortable: true },
    { header: 'Status', key: 'status_agen', sortable: true },
    { header: 'Disetujui', key: 'disetujui_agen_at', sortable: true },
  ];

  const renderCell = (row, key) => {
    if (key === 'nama_lengkap') return <span className="font-semibold text-neutral-900">{row.nama_lengkap}</span>;
    if (key === 'kode_referral') return <span className="font-mono text-neutral-800">{row.kode_referral || '-'}</span>;
    if (key === 'status_agen') {
      const meta = STATUS_AGEN[row.status_agen] || { label: row.status_agen, variant: 'neutral' };
      return <Badge variant={meta.variant}>{meta.label}</Badge>;
    }
    if (key === 'disetujui_agen_at') return dateLabel(row.disetujui_agen_at);
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-5">
      <PageHeader title="Daftar Agen" subtitle="Seluruh agen Syiar brand ini. Pilih agen untuk melihat detail dan kinerjanya." />
      {error && <Alert variant="error" message={error} />}
      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/agen/${row.jamaah_id}`)}
        itemsPerPage={15}
        searchPlaceholder="Cari nama atau kode referral..."
        emptyMessage={loading ? 'Memuat agen...' : 'Belum ada agen'}
      />
    </div>
  );
}

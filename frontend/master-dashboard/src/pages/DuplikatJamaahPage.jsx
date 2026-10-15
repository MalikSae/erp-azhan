import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CopyCheck, Users } from 'lucide-react';
import { listDuplikatLintasBrand } from 'shared';
import Alert from 'shared/ui/Alert';
import Badge from 'shared/ui/Badge';
import DataTable from 'shared/ui/DataTable';
import PageHeader from 'shared/ui/PageHeader';
import StatTile from 'shared/ui/StatTile';

const JENIS = {
  nik: { label: 'NIK sama', variant: 'danger' },
  no_hp: { label: 'No HP sama', variant: 'warning' },
};

// Jamaah dengan NIK atau nomor HP yang sama di lebih dari satu brand. Hanya penanda untuk
// holding: data tiap brand tetap terpisah dan tidak digabung otomatis.
export default function DuplikatJamaahPage() {
  const navigate = useNavigate();
  const [groups, setGroups] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listDuplikatLintasBrand()
      .then((d) => setGroups(d || []))
      .catch((err) => setError(err.response?.data?.error || 'Daftar duplikat gagal dimuat. Silakan coba kembali.'))
      .finally(() => setLoading(false));
  }, []);

  // Satu baris per jamaah; kelompok ditandai kunci yang sama.
  const rows = useMemo(
    () => groups.flatMap((g) => g.anggota.map((a) => ({
      ...a,
      rowKey: `${g.jenis}-${g.kunci}-${a.id}`,
      jenis: g.jenis,
      kunci: g.kunci,
      ukuran: g.anggota.length,
    }))),
    [groups],
  );

  const ringkasan = useMemo(() => ({
    nik: groups.filter((g) => g.jenis === 'nik').length,
    hp: groups.filter((g) => g.jenis === 'no_hp').length,
  }), [groups]);

  const columns = [
    { header: 'Kecocokan', key: 'jenis', sortable: true },
    { header: 'Kunci', key: 'kunci', sortable: true },
    { header: 'Nama', key: 'nama_lengkap', sortable: true },
    { header: 'Brand', key: 'brand_name', sortable: true },
    { header: 'ID Jamaah', key: 'id_jamaah', sortable: true },
    { header: 'No HP', key: 'no_hp', sortable: true },
    { header: 'Total Booking', key: 'total_booking', sortable: true },
  ];

  const renderCell = (row, key) => {
    if (key === 'jenis') {
      const j = JENIS[row.jenis] || { label: row.jenis, variant: 'neutral' };
      return <Badge variant={j.variant}>{j.label}</Badge>;
    }
    if (key === 'kunci') return <span className="font-mono text-[12px] text-neutral-700">{row.kunci}</span>;
    if (key === 'nama_lengkap') return <span className="font-heading font-semibold text-neutral-900">{row.nama_lengkap}</span>;
    if (key === 'id_jamaah') return <span className="font-mono text-[12px] text-neutral-600">{row.id_jamaah || '-'}</span>;
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Duplikat Lintas Brand" />

      {error && <Alert variant="error" message={error} />}

      {!loading && !error && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={CopyCheck} variant="danger" value={ringkasan.nik} label="Kelompok NIK sama" />
          <StatTile icon={Users} variant="warning" value={ringkasan.hp} label="Kelompok No HP sama" hint="Bisa keluarga berbagi nomor" />
        </div>
      )}

      <DataTable
        columns={columns}
        data={rows}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/jamaah/${row.id}`)}
        itemsPerPage={20}
        searchPlaceholder="Cari nama, NIK, atau nomor HP..."
        emptyMessage={loading ? 'Memuat daftar...' : 'Tidak ada jamaah dengan identitas sama di brand berbeda'}
      />
    </div>
  );
}

import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Banknote, Clock3, Receipt, Undo2, Wallet } from 'lucide-react';
import { money } from 'shared';
import Alert from 'shared/ui/Alert';
import DataTable from 'shared/ui/DataTable';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import PageHeader from 'shared/ui/PageHeader';
import StatTile from 'shared/ui/StatTile';
import { getFinanceSummary } from '../api/finance';

const waktu = (v) => (v ? new Date(v).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : '-');

// Ringkasan keuangan: kas masuk, antrean verifikasi, piutang, dan kewajiban refund.
// Admin brand yang punya izin finance otomatis hanya melihat angka brand-nya.
export default function FinanceDashboardPage() {
  const navigate = useNavigate();
  const [data, setData] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    getFinanceSummary()
      .then(setData)
      .catch((err) => setError(err.response?.data?.error || 'Ringkasan keuangan gagal dimuat. Silakan coba kembali.'));
  }, []);

  if (error) return <Alert variant="error" message={error} />;
  if (!data) return <LoadingSpinner />;

  const { kas, menunggu_verifikasi: pending, piutang, perlu_refund: refund } = data;
  const naik = kas.bulan_lalu > 0 ? Math.round(((kas.bulan_ini - kas.bulan_lalu) / kas.bulan_lalu) * 100) : null;

  const brandColumns = [
    { header: 'Brand', key: 'brand_name', sortable: true },
    { header: 'Kas Masuk Bulan Ini', key: 'kas_bulan_ini', sortable: true },
    { header: 'Piutang', key: 'piutang', sortable: true },
    { header: 'Menunggu Verifikasi', key: 'menunggu_verifikasi', sortable: true },
  ];
  const brandCell = (row, key) => {
    if (key === 'kas_bulan_ini' || key === 'piutang') return <span className="whitespace-nowrap">{money(row[key])}</span>;
    if (key === 'brand_name') return <span className="font-heading font-semibold text-neutral-900">{row.brand_name}</span>;
    return row[key] ?? '-';
  };

  const pendingColumns = [
    { header: 'Booking', key: 'id_booking' },
    { header: 'Jamaah', key: 'jamaah_name' },
    { header: 'Brand', key: 'brand_name' },
    { header: 'Jumlah', key: 'jumlah' },
    { header: 'Masuk', key: 'created_at' },
  ];
  const pendingCell = (row, key) => {
    if (key === 'jumlah') return <span className="whitespace-nowrap font-semibold">{money(row.jumlah)}</span>;
    if (key === 'created_at') return <span className="text-neutral-500">{waktu(row.created_at)}</span>;
    if (key === 'id_booking') return <span className="font-mono text-[12px]">{row.id_booking || `#${row.booking_id}`}</span>;
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Dashboard Keuangan" />

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatTile icon={Banknote} variant="success" value={money(kas.bulan_ini)} label="Kas masuk bulan ini"
          hint={naik === null ? `Hari ini ${money(kas.hari_ini)}` : `${naik >= 0 ? '+' : ''}${naik}% dari bulan lalu · hari ini ${money(kas.hari_ini)}`} />
        <StatTile icon={Clock3} variant="warning" value={pending.jumlah} label="Menunggu verifikasi" hint={money(pending.nominal)} />
        <StatTile icon={Receipt} variant="primary" value={money(piutang.nominal)} label="Piutang (sisa tagihan)" hint={`${piutang.jumlah_booking} booking belum lunas`} />
        <StatTile icon={Undo2} variant="danger" value={money(refund.nominal)} label="Perlu dikembalikan" hint={`${refund.jumlah_booking} booking batal`} />
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatTile icon={Wallet} variant="violet" value={money(kas.net_bulan_ini)} label="Kas bersih bulan ini" hint={`Refund ${money(kas.refund_bulan_ini)}`} />
      </div>

      {data.per_brand.length > 0 && (
        <DataTable
          columns={brandColumns}
          data={data.per_brand}
          renderCell={brandCell}
          itemsPerPage={10}
          searchPlaceholder="Cari brand..."
          emptyMessage="Belum ada data brand"
        />
      )}

      <div className="flex items-center justify-between">
        <h2 className="text-base font-heading font-bold text-neutral-900">Menunggu verifikasi terbaru</h2>
        <button type="button" onClick={() => navigate('/payments')} className="text-xs font-semibold text-primary-600 hover:text-primary-700">
          Buka semua pembayaran
        </button>
      </div>
      <DataTable
        columns={pendingColumns}
        data={data.pending_terbaru}
        renderCell={pendingCell}
        onRowClick={(row) => navigate(`/bookings/${row.booking_id}`)}
        itemsPerPage={8}
        searchPlaceholder="Cari pembayaran..."
        emptyMessage="Tidak ada pembayaran yang menunggu verifikasi"
      />
    </div>
  );
}

import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { History, IdCard, Network, UserRound } from 'lucide-react';
import { getAkunPelanggan } from '../api/akun';
import Alert from '../components/ui/Alert';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import MetaBox from '../components/ui/MetaBox';
import PageHeader from '../components/ui/PageHeader';
import { dateLabel } from '../utils/agen';
import { StatusAkun } from './DaftarAkunPage';

const dateTime = (v) => (!v ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(v)));

const Row = ({ label, children }) => (
  <div className="flex justify-between gap-3 border-b border-neutral-100 pb-2 last:border-b-0 last:pb-0 text-sm">
    <span className="text-[13px] font-body text-neutral-500">{label}</span>
    <span className="text-neutral-900 text-right font-heading font-semibold break-all">{children}</span>
  </div>
);

// Detail satu akun pelanggan (baca saja). showBrandColumn: Admin Master, boleh membuka layar Ganti Kaitan Agen.
export default function DetailAkunPage({ showBrandColumn = false }) {
  const { id } = useParams();
  const navigate = useNavigate();
  const [akun, setAkun] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    setAkun(null);
    getAkunPelanggan(id)
      .then(setAkun)
      .catch((e) => setError(e.response?.status === 404 ? 'Akun tidak ditemukan.' : 'Detail akun gagal dimuat. Silakan coba kembali.'));
  }, [id]);

  if (error) return <div className="space-y-6"><PageHeader title="Detail Akun" onBack={() => navigate('/akun')} /><Alert variant="error" message={error} /></div>;
  if (!akun) return <LoadingSpinner />;

  const jadiAgen = akun.status_agen === 'aktif' || akun.status_agen === 'nonaktif';
  const gantiUpline = `/komisi/ganti-kaitan?akun=${akun.akun_id}&q=${encodeURIComponent(akun.nama_lengkap)}`;

  return (
    <div className="space-y-6">
      <PageHeader title={akun.nama_lengkap} subtitle={`Akun #${akun.akun_id}`} onBack={() => navigate('/akun')} breadcrumb={false} />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <MetaBox title="Profil" icon={<UserRound size={18} />} contentClassName="p-4 sm:p-5 space-y-2.5">
          <Row label="Nama">{akun.nama_lengkap}</Row>
          <Row label="Email">{akun.email}{akun.email_verified_at ? '' : ' (belum diverifikasi)'}</Row>
          <Row label="WhatsApp">{akun.no_hp}</Row>
          <Row label="Domisili">{akun.domisili || '-'}</Row>
          {showBrandColumn && <Row label="Brand">{akun.brand_name}</Row>}
          <Row label="Terdaftar">{dateTime(akun.created_at)}</Row>
          <Row label="Login terakhir">{dateTime(akun.last_login_at)}</Row>
          <Row label="Booking">{akun.jumlah_booking}</Row>
          <Row label="Jamaah akun ini">
            {akun.jamaah ? <Link to={`/jamaah/${akun.jamaah.id}`} className="text-primary-600 hover:underline">{akun.jamaah.nama_lengkap}</Link> : '-'}
          </Row>
        </MetaBox>

        <div className="space-y-6">
          <MetaBox
            title="Upline"
            subtitle="Agen yang mengajak akun ini dan menerima bonus pembinaan darinya."
            icon={<Network size={18} />}
            contentClassName="p-4 sm:p-5 space-y-3"
          >
            <p className="text-sm font-heading font-semibold text-neutral-900">
              {akun.upline_akun_id ? <Link to={`/agen/${akun.upline_akun_id}`} className="text-primary-600 hover:underline">{akun.upline_nama}</Link> : 'Tanpa upline'}
            </p>
            {showBrandColumn && <Button variant="secondary" size="sm" onClick={() => navigate(gantiUpline)}>Ganti upline</Button>}
          </MetaBox>

          <MetaBox title="Agen" icon={<IdCard size={18} />} contentClassName="p-4 sm:p-5 space-y-2.5">
            <Row label="Status"><StatusAkun status={akun.status_agen} /></Row>
            {akun.status_agen !== 'tidak_aktif' && (
              <>
                <Row label="Kode referral"><span className="font-mono">{akun.kode_referral || '-'}</span></Row>
                <Row label="Diajukan">{dateLabel(akun.diajukan_agen_at)}</Row>
                <Row label="Disetujui">{dateLabel(akun.disetujui_agen_at)}</Row>
                <Row label="NIK agen">{akun.nik_agen || '-'}</Row>
              </>
            )}
            {jadiAgen && (
              <>
                <Row label="Jamaah agen ini">{akun.jumlah_jamaah_agen}</Row>
                <Row label="Akun yang diajak">{akun.jumlah_diajak}</Row>
                <div className="pt-1"><Button variant="secondary" size="sm" onClick={() => navigate(`/agen/${akun.akun_id}`)}>Lihat kinerja agen</Button></div>
              </>
            )}
          </MetaBox>
        </div>
      </div>

      <MetaBox title="Riwayat Perubahan Upline" icon={<History size={18} />} contentClassName="p-4 sm:p-5">
        {akun.riwayat_upline.length === 0 ? (
          <p className="text-sm text-neutral-500">Upline belum pernah diubah admin.</p>
        ) : (
          <ol className="space-y-4">
            {akun.riwayat_upline.map((l) => (
              <li key={l.id} className="text-sm">
                <p className="font-heading font-semibold text-neutral-900">{l.agen_lama_nama || 'Tanpa upline'} → {l.agen_baru_nama || 'Tanpa upline'}</p>
                <p className="text-neutral-700">{l.alasan}</p>
                <p className="text-[11px] text-neutral-400">{l.diganti_oleh} · {dateTime(l.diganti_at)}</p>
              </li>
            ))}
          </ol>
        )}
      </MetaBox>
    </div>
  );
}

import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { getDetailAgen, ubahStatusAgen } from 'shared';
import { Banknote, History, IdCard, Network, TrendingUp, UserCheck, Wallet } from 'lucide-react';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import MetaBox from '../components/ui/MetaBox';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';
import useProtectedImage from '../utils/useProtectedImage';
import { dateLabel, money, STATUS_AGEN } from '../utils/agen';

const Row = ({ label, children }) => (
  <div className="flex justify-between gap-3 border-b border-neutral-100 pb-2 last:border-b-0 last:pb-0 text-sm">
    <span className="text-neutral-500">{label}</span>
    <span className="text-neutral-900 text-right font-medium">{children}</span>
  </div>
);

const bayarMeta = {
  menunggu_verifikasi: { label: 'Menunggu verifikasi', variant: 'pending' },
  terverifikasi: { label: 'Terverifikasi', variant: 'success' },
  ditolak: { label: 'Ditolak', variant: 'danger' },
};
const keputusanMeta = {
  menunggu: { label: 'Menunggu', variant: 'pending' },
  disetujui: { label: 'Disetujui', variant: 'success' },
  ditolak: { label: 'Ditolak', variant: 'danger' },
};
const pencairanMeta = {
  pending: { label: 'Pending', variant: 'pending' },
  disetujui: { label: 'Disetujui', variant: 'success' },
  ditolak: { label: 'Ditolak', variant: 'danger' },
};

// Screen B2a — detail satu agen, 8 kelompok data (screen-agen.md). Satu-satunya
// aksi: toggle aktif <-> nonaktif.
export default function DetailAgenPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [agen, setAgen] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');
  const fotoSrc = useProtectedImage(agen?.foto_agen_url);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setAgen(await getDetailAgen(id));
    } catch (err) {
      setError(err.response?.status === 404 ? 'Agen tidak ditemukan.' : 'Detail agen gagal dimuat.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  const targetStatus = agen?.status_agen === 'aktif' ? 'nonaktif' : 'aktif';
  const toggle = async () => {
    setSaving(true);
    setActionError('');
    try {
      await ubahStatusAgen(agen.jamaah_id, targetStatus);
      setConfirmOpen(false);
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Status agen gagal diubah.');
    } finally {
      setSaving(false);
    }
  };

  if (loading && !agen) return <p className="text-sm text-neutral-500">Memuat detail agen...</p>;
  if (error) return <Alert variant="error" message={error} />;
  if (!agen) return null;

  const statusMeta = STATUS_AGEN[agen.status_agen] || { label: agen.status_agen, variant: 'neutral' };
  const siklusAktifId = agen.siklus?.find((s) => s.keputusan_agen === 'disetujui')?.id;

  return (
    <div className="space-y-5">
      <PageHeader title={agen.nama_lengkap} subtitle="Detail agen Syiar" onBack={() => navigate('/agen')}>
        <Button variant={agen.status_agen === 'aktif' ? 'danger-light' : 'primary'} onClick={() => setConfirmOpen(true)}>
          {agen.status_agen === 'aktif' ? 'Nonaktifkan Agen' : 'Aktifkan Kembali'}
        </Button>
      </PageHeader>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5 items-start">
        <div className="space-y-5 lg:col-span-2">
          {/* e. Ringkasan komisi & saldo */}
          <MetaBox title="Komisi & Saldo" subtitle="Saldo tersedia hanya dari booking yang sudah berangkat" icon={<Wallet size={18} className="text-neutral-700" />}>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div className="rounded-xl border border-success-200 bg-success-50 p-3">
                <p className="text-xs text-success-700">Saldo tersedia</p>
                <p className="mt-1 text-lg font-bold text-neutral-900">{money(agen.saldo.saldo_tersedia)}</p>
              </div>
              <div className="rounded-xl border border-warning-200 bg-warning-50 p-3">
                <p className="text-xs text-warning-800">Saldo tertahan</p>
                <p className="mt-1 text-lg font-bold text-neutral-900">{money(agen.saldo.saldo_tertahan)}</p>
              </div>
              <div className="rounded-xl border border-neutral-200 bg-neutral-50 p-3">
                <p className="text-xs text-neutral-600">Kredit cashback</p>
                <p className="mt-1 text-lg font-bold text-neutral-900">{money(agen.saldo.kredit_cashback)}</p>
              </div>
            </div>
            <div className="space-y-2">
              <Row label="Total komisi langsung">{money(agen.total_per_jenis.langsung)}</Row>
              <Row label="Total bonus pembinaan">{money(agen.total_per_jenis.pembinaan)}</Row>
              <Row label="Total repeat order">{money(agen.total_per_jenis.repeat_order)}</Row>
              <Row label="Total cashback">{money(agen.total_per_jenis.cashback)}</Row>
            </div>
            <Link to={`/agen/komisi?agen_id=${agen.jamaah_id}`} className="inline-block text-sm font-semibold text-neutral-900 underline underline-offset-2">
              Lihat riwayat lengkap
            </Link>
          </MetaBox>

          {/* f. Riwayat pencairan (read-only) */}
          <MetaBox title="Riwayat Pencairan" subtitle="Persetujuan pencairan dilakukan Admin Master" icon={<Banknote size={18} className="text-neutral-700" />}>
            {agen.pencairan.length === 0 ? (
              <p className="text-sm text-neutral-500">Belum ada pengajuan pencairan.</p>
            ) : (
              <div className="space-y-2">
                {agen.pencairan.map((p) => {
                  const meta = pencairanMeta[p.status] || { label: p.status, variant: 'neutral' };
                  return (
                    <div key={p.id} className="flex items-center justify-between gap-3 border-b border-neutral-100 pb-2 last:border-b-0 text-sm">
                      <div>
                        <p className="font-semibold text-neutral-900">{money(p.nominal_diajukan)}</p>
                        <p className="text-xs text-neutral-500">{dateLabel(p.diajukan_at)}</p>
                      </div>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </div>
                  );
                })}
              </div>
            )}
          </MetaBox>

          {/* g. Riwayat siklus pengajuan */}
          <MetaBox title="Riwayat Siklus Pengajuan" subtitle="Setiap pengajuan keagenan dan pembayarannya" icon={<History size={18} className="text-neutral-700" />}>
            {agen.siklus.length === 0 ? (
              <p className="text-sm text-neutral-500">Tidak ada riwayat pengajuan.</p>
            ) : (
              <div className="space-y-3">
                {agen.siklus.map((s) => {
                  const kep = keputusanMeta[s.keputusan_agen] || { label: s.keputusan_agen, variant: 'neutral' };
                  const bayar = bayarMeta[s.status] || { label: s.status, variant: 'neutral' };
                  const aktif = s.id === siklusAktifId;
                  return (
                    <div key={s.id} className={`rounded-xl border p-3 space-y-2 ${aktif ? 'border-success-200 bg-success-50' : 'border-neutral-200'}`}>
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <span className="text-sm font-semibold text-neutral-900">Diajukan {dateLabel(s.created_at)}</span>
                        <div className="flex flex-wrap gap-1.5">
                          {aktif && <Badge variant="primary">Siklus aktif</Badge>}
                          <Badge variant={kep.variant}>Keagenan: {kep.label}</Badge>
                        </div>
                      </div>
                      {s.alasan_ditolak_agen && <p className="text-xs text-danger-700">Alasan ditolak: {s.alasan_ditolak_agen}</p>}
                      <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-neutral-600">
                        <span>{Number(s.nominal_tagihan) > 0 ? `Biaya ${money(s.nominal_tagihan)}` : 'Tanpa biaya pendaftaran'}</span>
                        {Number(s.nominal_tagihan) > 0 && <Badge variant={bayar.variant}>{bayar.label}</Badge>}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </MetaBox>
        </div>

        <div className="space-y-5">
          {/* a. Identitas & data profil */}
          <MetaBox title="Identitas" icon={<IdCard size={18} className="text-neutral-700" />}>
            <div className="flex items-center gap-3">
              <div className="h-16 w-16 shrink-0 overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50">
                {fotoSrc && <img src={fotoSrc} alt={`Foto ${agen.nama_lengkap}`} className="h-full w-full object-cover" />}
              </div>
              <div className="min-w-0">
                <p className="font-semibold text-neutral-900 truncate">{agen.nama_lengkap}</p>
                <p className="text-xs text-neutral-500 font-mono">{agen.no_hp || '-'}</p>
              </div>
            </div>
            <div className="space-y-2">
              <Row label="Domisili">{agen.domisili || '-'}</Row>
              <Row label="Setuju S&K">{dateLabel(agen.menyetujui_syarat_ketentuan_agen_at)}</Row>
            </div>
          </MetaBox>

          {/* b. Status keagenan */}
          <MetaBox title="Status Keagenan" icon={<UserCheck size={18} className="text-neutral-700" />} badge={<Badge variant={statusMeta.variant}>{statusMeta.label}</Badge>}>
            <div className="space-y-2">
              <Row label="Kode referral"><span className="font-mono">{agen.kode_referral || '-'}</span></Row>
              <Row label="Diajukan">{dateLabel(agen.diajukan_agen_at)}</Row>
              <Row label="Disetujui">{dateLabel(agen.disetujui_agen_at)}</Row>
              <Row label="Disetujui oleh">{agen.disetujui_agen_oleh_nama || '-'}</Row>
            </div>
          </MetaBox>

          {/* c. Jaringan & d. Performa closing */}
          <MetaBox title="Jaringan & Closing" icon={<Network size={18} className="text-neutral-700" />}>
            <div className="space-y-2">
              <Row label="Upline">
                {agen.upline_jamaah_id ? (
                  <Link to={`/agen/${agen.upline_jamaah_id}`} className="underline underline-offset-2">{agen.upline_nama}</Link>
                ) : 'Tidak ada'}
              </Row>
              <Row label="Jumlah downline (agen)">{agen.jumlah_downline}</Row>
            </div>
            <div className="flex items-center gap-2 rounded-xl bg-neutral-50 border border-neutral-200 p-3">
              <TrendingUp size={16} className="text-neutral-600" />
              <span className="text-sm text-neutral-700">
                <span className="font-bold text-neutral-900">{agen.jumlah_closing}</span> jamaah di-closing
              </span>
            </div>
          </MetaBox>
        </div>
      </div>

      <Modal
        isOpen={confirmOpen}
        onClose={() => !saving && setConfirmOpen(false)}
        title={targetStatus === 'nonaktif' ? 'Nonaktifkan Agen' : 'Aktifkan Kembali Agen'}
        footer={
          <div className="flex justify-end gap-2 w-full">
            <Button variant="secondary" onClick={() => setConfirmOpen(false)} disabled={saving}>Batal</Button>
            <Button variant={targetStatus === 'nonaktif' ? 'danger' : 'primary'} onClick={toggle} isLoading={saving}>
              {targetStatus === 'nonaktif' ? 'Nonaktifkan' : 'Aktifkan'}
            </Button>
          </div>
        }
      >
        <div className="space-y-3 text-sm text-neutral-700">
          {actionError && <Alert variant="error" message={actionError} />}
          <p>
            {targetStatus === 'nonaktif'
              ? `${agen.nama_lengkap} tidak bisa membuat booking dan tidak menerima komisi baru (termasuk sebagai upline) selama nonaktif.`
              : `${agen.nama_lengkap} kembali bisa membuat booking dan menerima komisi.`}
          </p>
        </div>
      </Modal>
    </div>
  );
}

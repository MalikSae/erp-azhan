import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { getDetailAgen, ubahStatusAgen } from '../api/agen';
import { ubahNikAgen } from '../api/akun';
import { Banknote, Clock3, History, Network, Pencil, TrendingUp, UserCheck, Wallet } from 'lucide-react';
import Alert from '../components/ui/Alert';
import EmptyState from '../components/ui/EmptyState';
import Badge, { toneFromClass } from '../components/ui/Badge';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import MetaBox from '../components/ui/MetaBox';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';
import StatTile from '../components/ui/StatTile';
import Textarea from '../components/ui/Textarea';
import useProtectedImage from '../utils/useProtectedImage';
import { dateLabel, HARI_TAHAN_KOMISI, money, STATUS_AGEN } from '../utils/agen';

const Row = ({ label, children }) => (
  <div className="flex justify-between gap-3 border-b border-neutral-100 pb-2 last:border-b-0 last:pb-0 text-sm">
    <span className="text-sm font-body text-neutral-500">{label}</span>
    <span className="text-neutral-900 text-right font-heading font-semibold">{children}</span>
  </div>
);

// Status = titik berwarna kecil + teks (bukan pil berwarna).
const DOT_STYLE = {
  success: { text: 'text-success-700', dot: 'bg-success-500' },
  danger: { text: 'text-danger-700', dot: 'bg-danger-500' },
  pending: { text: 'text-warning-700', dot: 'bg-warning-500' },
  primary: { text: 'text-primary-700', dot: 'bg-primary-500' },
  neutral: { text: 'text-neutral-500', dot: 'bg-neutral-300' },
};

const StatusDot = ({ variant = 'neutral', children }) => {
  const s = DOT_STYLE[variant] || DOT_STYLE.neutral;
  return <Badge variant={toneFromClass(s.dot)}>{children}</Badge>;
};

const bayarMeta = {
  menunggu_verifikasi: { label: 'Menunggu verifikasi', variant: 'pending' },
  terverifikasi: { label: 'Lunas', variant: 'success' },
  dibebaskan: { label: 'Dibebaskan', variant: 'success' },
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

// Halaman satu agen (screen B2a). Hanya memuat data keagenan: status, kinerja, komisi, dan riwayatnya.
// Profil (nama, WhatsApp, domisili) dan upline milik akun, diubah di Detail Akun; di sini hanya tautan.
//
// atas: isi tambahan di bawah judul (dashboard menaruh kartu keputusan pengajuan di situ).
// calon: akun yang masih mengajukan dan belum jadi agen ({ nama_lengkap, no_hp }); halaman lalu hanya
// menampilkan judul dan `atas`. undefined berarti belum diketahui (antrean masih dimuat).
// izin: { lihatAkun, ubahNik } menentukan tautan dan tombol; API tetap memeriksa izinnya sendiri.
// muatUlang: angka yang dinaikkan pemanggil untuk memuat ulang data (mis. setelah pengajuan disetujui).
export default function DetailAgenPage({ showBrandColumn = false, atas = null, calon, izin = {}, muatUlang = 0 }) {
  const { id } = useParams();
  const navigate = useNavigate();
  const [agen, setAgen] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [bukanAgen, setBukanAgen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');
  const [alasan, setAlasan] = useState('');
  const [nikOpen, setNikOpen] = useState(false);
  const [nikForm, setNikForm] = useState({ nik: '', alasan: '' });
  const [nikGalat, setNikGalat] = useState('');
  const fotoSrc = useProtectedImage(agen?.foto_url);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    setBukanAgen(false);
    try {
      setAgen(await getDetailAgen(id));
    } catch (err) {
      setAgen(null);
      const tidakAda = err.response?.status === 404;
      setBukanAgen(tidakAda);
      setError(tidakAda ? 'Agen tidak ditemukan.' : 'Detail agen gagal dimuat.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load, muatUlang]);

  const simpanNik = async () => {
    setSaving(true);
    setNikGalat('');
    try {
      await ubahNikAgen(agen.akun_id, nikForm);
      setNikOpen(false);
      await load();
    } catch (err) {
      setNikGalat(err.response?.data?.error || 'NIK gagal disimpan.');
    } finally {
      setSaving(false);
    }
  };

  const targetStatus = agen?.status_agen === 'aktif' ? 'nonaktif' : 'aktif';
  const alasanValid = alasan.trim().length >= 5 && alasan.trim().length <= 500;
  const toggle = async () => {
    if (!alasanValid) {
      setActionError('Alasan wajib diisi (5 sampai 500 karakter).');
      return;
    }
    setSaving(true);
    setActionError('');
    try {
      await ubahStatusAgen(agen.akun_id, targetStatus, alasan.trim());
      setConfirmOpen(false);
      setAlasan('');
      await load();
    } catch (err) {
      setActionError(err.response?.data?.error || 'Status agen gagal diubah.');
    } finally {
      setSaving(false);
    }
  };

  const tautanAkun = izin.lihatAkun
    ? <Button variant="secondary" onClick={() => navigate(`/akun/${id}`)}>Lihat akun</Button>
    : null;

  // Belum ada data agen: sedang memuat, akun yang masih mengajukan, atau galat. Susunan luarnya sama
  // dengan halaman penuh supaya `atas` tidak dipasang ulang saat keadaan berganti.
  if (!agen) {
    const masihCalon = bukanAgen && calon;
    const menunggu = loading || (bukanAgen && calon === undefined && atas);
    return (
      <div className="space-y-6">
        <PageHeader
          title={masihCalon ? calon.nama_lengkap : 'Agen'}
          meta={masihCalon ? [calon.no_hp || '-', calon.diajukan_agen_at ? `Diajukan ${dateLabel(calon.diajukan_agen_at)}` : null] : undefined}
          badges={masihCalon ? <Badge variant="warning">Mengajukan agen</Badge> : undefined}
          onBack={() => navigate(masihCalon ? '/agen/pengajuan' : '/agen')}
        >
          {masihCalon && tautanAkun}
        </PageHeader>
        {atas}
        {menunggu && <p className="text-sm text-neutral-500">Memuat data agen...</p>}
        {!menunggu && !masihCalon && error && <Alert variant="error" message={error} />}
      </div>
    );
  }

  const statusMeta = STATUS_AGEN[agen.status_agen] || { label: agen.status_agen, variant: 'neutral' };
  const pencairanTerbaru = agen.pencairan.slice(0, 3);

  return (
    <div className="space-y-6">
      <PageHeader
        title={agen.nama_lengkap}
        meta={[agen.no_hp || '-', `Kode ${agen.kode_referral || '-'}`]}
        badges={<Badge variant={toneFromClass((DOT_STYLE[statusMeta.variant] || DOT_STYLE.neutral).dot)}>{statusMeta.label}</Badge>}
        onBack={() => navigate('/agen')}
      >
        {tautanAkun}
        <Button variant={agen.status_agen === 'aktif' ? 'danger-light' : 'primary'} onClick={() => { setAlasan(''); setActionError(''); setConfirmOpen(true); }}>
          {agen.status_agen === 'aktif' ? 'Nonaktifkan Agen' : 'Aktifkan Kembali'}
        </Button>
      </PageHeader>

      {atas}

      {/* e. Ringkasan komisi & saldo */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <StatTile icon={Wallet} variant="success" value={money(agen.saldo.saldo_tersedia)} label="Dapat Dicairkan" hint="Setelah dikurangi pencairan" />
        <StatTile icon={Clock3} variant="warning" value={money(agen.saldo.saldo_tertahan)} label="Tertahan" hint={`Hingga H+${HARI_TAHAN_KOMISI} keberangkatan`} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        <div className="space-y-6 lg:col-span-2">
          <MetaBox title="Komisi & Saldo" subtitle="Total komisi per jenis" icon={<Wallet size={16} strokeWidth={1.75} className="text-neutral-700" />}>
            <div className="space-y-2">
              <Row label="Total komisi langsung">{money(agen.total_per_jenis.langsung)}</Row>
              <Row label="Total bonus pembinaan">{money(agen.total_per_jenis.pembinaan)}</Row>
            </div>
            <Link to={`/agen/komisi?agen_id=${agen.akun_id}`} className="inline-block text-sm font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900">
              Lihat riwayat lengkap
            </Link>
          </MetaBox>

          {/* f. Riwayat pencairan (read-only) */}
          <MetaBox title="Pencairan Terbaru" subtitle="Tiga pengajuan terakhir. Persetujuan dan riwayat lengkap ada di menu Komisi." icon={<Banknote size={16} strokeWidth={1.75} className="text-neutral-700" />}>
            {pencairanTerbaru.length === 0 ? (
              <EmptyState variant="inline" message="Belum ada pengajuan pencairan." />
            ) : (
              <div className="space-y-2">
                {pencairanTerbaru.map((p) => {
                  const meta = pencairanMeta[p.status] || { label: p.status, variant: 'neutral' };
                  return (
                    <div key={p.id} className="flex items-center justify-between gap-3 border-b border-neutral-100 pb-2 last:border-b-0 text-sm">
                      <div>
                        <p className="font-heading font-semibold text-neutral-900">{money(p.nominal_diajukan)}</p>
                        <p className="text-xs text-neutral-400">{dateLabel(p.diajukan_at)}</p>
                      </div>
                      <StatusDot variant={meta.variant}>{meta.label}</StatusDot>
                    </div>
                  );
                })}
              </div>
            )}
          </MetaBox>

          {/* g. Riwayat pengajuan: satu baris per pengajuan (tanggal, keputusan, biaya). */}
          <MetaBox title="Riwayat Pengajuan" icon={<History size={16} strokeWidth={1.75} className="text-neutral-700" />}>
            {agen.siklus.length === 0 ? (
              <EmptyState variant="inline" message="Tidak ada riwayat pengajuan." />
            ) : (
              <div className="divide-y divide-neutral-100">
                {agen.siklus.map((s) => {
                  const kep = keputusanMeta[s.keputusan_agen] || { label: s.keputusan_agen, variant: 'neutral' };
                  const bayar = bayarMeta[s.status] || { label: s.status, variant: 'neutral' };
                  const berbayar = Number(s.nominal_tagihan) > 0;
                  return (
                    <div key={s.id} className="py-2.5 first:pt-0 last:pb-0">
                      <div className="grid grid-cols-[7rem_7rem_1fr] items-center gap-3 text-sm">
                        <span className="text-neutral-700 whitespace-nowrap">{dateLabel(s.created_at)}</span>
                        <StatusDot variant={kep.variant}>{kep.label}</StatusDot>
                        <span className="text-right text-neutral-700 whitespace-nowrap">
                          {berbayar ? <><span className="font-heading font-semibold text-neutral-900">{money(s.nominal_tagihan)}</span> · {bayar.label}</> : 'Tanpa biaya'}
                        </span>
                      </div>
                      {s.alasan_ditolak_agen && <p className="mt-1 text-xs text-danger-700">{s.alasan_ditolak_agen}</p>}
                    </div>
                  );
                })}
              </div>
            )}
          </MetaBox>
        </div>

        <div className="space-y-6">
          {/* b. Status keagenan */}
          <MetaBox
            title="Status Keagenan"
            icon={<UserCheck size={16} strokeWidth={1.75} className="text-neutral-700" />}
            headerAction={izin.ubahNik ? (
              <Button variant="secondary" size="sm" onClick={() => { setNikForm({ nik: '', alasan: '' }); setNikGalat(''); setNikOpen(true); }}>
                <Pencil size={14} className="mr-1.5" />Ubah NIK
              </Button>
            ) : null}
          >
            {agen.foto_url && (
              <div className="flex justify-center">
                <div className="h-20 w-20 overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50">
                  {fotoSrc && <img src={fotoSrc} alt={`Foto ${agen.nama_lengkap}`} className="h-full w-full object-cover" />}
                </div>
              </div>
            )}
            <div className="space-y-2">
              <Row label="Status"><StatusDot variant={statusMeta.variant}>{statusMeta.label}</StatusDot></Row>
              <Row label="Kode referral"><span className="font-mono">{agen.kode_referral || '-'}</span></Row>
              <Row label="NIK agen"><span className="font-mono">{agen.nik_masked || '-'}</span></Row>
              {showBrandColumn && <Row label="Brand">{agen.brand_name || '-'}</Row>}
              <Row label="Setuju S&K">{dateLabel(agen.menyetujui_syarat_ketentuan_agen_at)}</Row>
              <Row label="Diajukan">{dateLabel(agen.diajukan_agen_at)}</Row>
              <Row label="Disetujui">{dateLabel(agen.disetujui_agen_at)}</Row>
              <Row label="Disetujui oleh">{agen.disetujui_agen_oleh_nama || '-'}</Row>
            </div>
          </MetaBox>

          {/* h. Riwayat perubahan status aktif dan nonaktif */}
          {agen.riwayat_status?.length > 0 && (
            <MetaBox title="Riwayat Status" icon={<History size={16} strokeWidth={1.75} className="text-neutral-700" />}>
              <div className="space-y-3">
                {agen.riwayat_status.map((r, i) => (
                  <div key={i} className="text-xs text-neutral-700 border-b border-neutral-100 pb-2 last:border-b-0 last:pb-0">
                    <p className="font-heading font-semibold text-neutral-900">
                      {STATUS_AGEN[r.status_lama]?.label || r.status_lama} ke {STATUS_AGEN[r.status_baru]?.label || r.status_baru}
                    </p>
                    <p className="text-xs text-neutral-400">{dateLabel(r.diubah_at)} oleh {r.oleh_nama || 'admin'}</p>
                    <p className="mt-1">{r.alasan}</p>
                  </div>
                ))}
              </div>
            </MetaBox>
          )}

          {/* c. Jaringan & d. Performa closing */}
          <MetaBox title="Jaringan" icon={<Network size={16} strokeWidth={1.75} className="text-neutral-700" />}>
            <div className="space-y-2">
              <Row label="Upline">
                {agen.upline_akun_id ? (
                  <Link to={`/agen/${agen.upline_akun_id}`} className="font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900">{agen.upline_nama}</Link>
                ) : 'Tidak ada'}
              </Row>
              <Row label="Jumlah downline (agen)">{agen.jumlah_downline}</Row>
            </div>
            <div className="flex items-center gap-2 rounded-xl bg-neutral-50 border border-neutral-200 p-3">
              <TrendingUp size={16} strokeWidth={1.75} className="text-neutral-600" />
              <span className="text-sm text-neutral-700">
                <span className="font-heading font-semibold text-neutral-900">{agen.jumlah_closing}</span> jamaah terikat ke agen ini
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
            <Button variant={targetStatus === 'nonaktif' ? 'danger' : 'primary'} onClick={toggle} isLoading={saving} disabled={!alasanValid}>
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
          <div className="space-y-1.5">
            <label htmlFor="alasan-status-agen" className="block text-xs font-semibold text-neutral-700">Alasan <span className="text-danger-600">*</span></label>
            <textarea
              id="alasan-status-agen"
              value={alasan}
              onChange={(e) => setAlasan(e.target.value)}
              rows={3}
              maxLength={500}
              disabled={saving}
              placeholder={targetStatus === 'nonaktif' ? 'Mis. melanggar ketentuan program, atas permintaan agen' : 'Mis. sudah diklarifikasi, agen kembali aktif'}
              className="w-full rounded-lg border border-neutral-300 bg-white p-2.5 text-xs text-neutral-800 placeholder-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
            <p className="text-xs text-neutral-400">Minimal 5 karakter. Alasan, nama Anda, dan waktunya tercatat di riwayat status agen.</p>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={nikOpen}
        onClose={() => !saving && setNikOpen(false)}
        title="Ubah NIK Agen"
        footer={(
          <div className="flex justify-end gap-2 w-full">
            <Button variant="secondary" onClick={() => setNikOpen(false)} disabled={saving}>Batal</Button>
            <Button variant="primary" onClick={simpanNik} isLoading={saving}>Simpan</Button>
          </div>
        )}
      >
        <div className="space-y-4">
          {nikGalat && <Alert variant="error" message={nikGalat} />}
          <Input
            label="NIK baru (16 digit)"
            value={nikForm.nik}
            onChange={(e) => setNikForm((f) => ({ ...f, nik: e.target.value.replace(/\D/g, '').slice(0, 16) }))}
            helpText={agen.nik_masked ? `NIK sekarang ${agen.nik_masked}.` : 'Belum ada NIK tercatat.'}
          />
          <Textarea label="Alasan perubahan *" value={nikForm.alasan} onChange={(e) => setNikForm((f) => ({ ...f, alasan: e.target.value }))} rows={3} placeholder="Contoh: salah ketik saat mendaftar, diminta pemilik akun" />
        </div>
      </Modal>
    </div>
  );
}

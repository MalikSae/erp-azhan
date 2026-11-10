import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { History, IdCard, Network, Pencil, UserRound } from 'lucide-react';
import { getAkunPelanggan, ubahNikAgen, ubahProfilAkun } from '../api/akun';
import Alert from '../components/ui/Alert';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import MetaBox from '../components/ui/MetaBox';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';
import Textarea from '../components/ui/Textarea';
import { dateLabel } from '../utils/agen';
import useProtectedImage from '../utils/useProtectedImage';
import { StatusAkun } from './DaftarAkunPage';

const dateTime = (v) => (!v ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(v)));

const KOLOM = { nama_lengkap: 'Nama', no_hp: 'WhatsApp', domisili: 'Domisili', nik_agen: 'NIK agen' };

const Row = ({ label, children }) => (
  <div className="flex justify-between gap-3 border-b border-neutral-100 pb-2 last:border-b-0 last:pb-0 text-sm">
    <span className="text-[13px] font-body text-neutral-500">{label}</span>
    <span className="text-neutral-900 text-right font-heading font-semibold break-all">{children}</span>
  </div>
);

const TombolUbah = ({ onClick, children }) => (
  <Button variant="secondary" size="sm" onClick={onClick}><Pencil size={14} className="mr-1.5" />{children}</Button>
);

// Detail satu akun pelanggan. showBrandColumn: Admin Master, boleh membuka layar Ganti Kaitan Agen.
// izin: { ubahProfil, nikPengajuan, nikAktif } menentukan tombol Ubah (API tetap memeriksa izin sendiri).
export default function DetailAkunPage({ showBrandColumn = false, izin = {} }) {
  const { id } = useParams();
  const navigate = useNavigate();
  const [akun, setAkun] = useState(null);
  const [error, setError] = useState('');
  const [dialog, setDialog] = useState(null); // 'profil' | 'nik'
  const [form, setForm] = useState({});
  const [saving, setSaving] = useState(false);
  const [galat, setGalat] = useState(null);

  const muat = useCallback(() => getAkunPelanggan(id)
    .then(setAkun)
    .catch((e) => setError(e.response?.status === 404 ? 'Akun tidak ditemukan.' : 'Detail akun gagal dimuat. Silakan coba kembali.')), [id]);

  useEffect(() => {
    setAkun(null);
    muat();
  }, [muat]);

  const fotoSrc = useProtectedImage(akun?.foto_url);

  if (error) return <div className="space-y-6"><PageHeader title="Detail Akun" onBack={() => navigate('/akun')} /><Alert variant="error" message={error} /></div>;
  if (!akun) return <LoadingSpinner />;

  const jadiAgen = akun.status_agen === 'aktif' || akun.status_agen === 'nonaktif';
  const gantiUpline = `/komisi/ganti-kaitan?akun=${akun.akun_id}&q=${encodeURIComponent(akun.nama_lengkap)}`;
  const bolehNik = akun.status_agen !== 'tidak_aktif' && (jadiAgen ? izin.nikAktif : izin.nikPengajuan);

  const buka = (jenis) => {
    setGalat(null);
    setForm(jenis === 'profil'
      ? { nama_lengkap: akun.nama_lengkap, no_hp: akun.no_hp, domisili: akun.domisili || '', alasan: '' }
      : { nik: '', alasan: '' });
    setDialog(jenis);
  };
  const isi = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const simpan = async () => {
    setSaving(true);
    setGalat(null);
    try {
      if (dialog === 'profil') await ubahProfilAkun(akun.akun_id, form);
      else await ubahNikAgen(akun.akun_id, form);
      setDialog(null);
      await muat();
    } catch (e) {
      const d = e.response?.data;
      setGalat({ pesan: d?.error || 'Perubahan gagal disimpan.', akunId: d?.akun_id || null });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader title={akun.nama_lengkap} subtitle={`Akun #${akun.akun_id}`} onBack={() => navigate('/akun')} breadcrumb={false} />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <MetaBox
          title="Profil"
          icon={<UserRound size={18} />}
          contentClassName="p-4 sm:p-5 space-y-2.5"
          headerAction={izin.ubahProfil ? <TombolUbah onClick={() => buka('profil')}>Ubah</TombolUbah> : null}
        >
          {akun.foto_url && (
            <div className="flex justify-center pb-2">
              <div className="size-24 overflow-hidden rounded-full border border-neutral-200 bg-neutral-50">
                {fotoSrc && <img src={fotoSrc} alt={`Foto ${akun.nama_lengkap}`} className="h-full w-full object-cover" />}
              </div>
            </div>
          )}
          <Row label="Nama">{akun.nama_lengkap}</Row>
          <Row label="Email">{akun.email}</Row>
          <Row label="WhatsApp">{akun.no_hp}</Row>
          <Row label="Domisili">{akun.domisili || '-'}</Row>
          <Row label="Jenis kelamin">{akun.jenis_kelamin === 'L' ? 'Laki-laki' : akun.jenis_kelamin === 'P' ? 'Perempuan' : '-'}</Row>
          <Row label="Tanggal lahir">{akun.tanggal_lahir ? dateLabel(akun.tanggal_lahir) : '-'}</Row>
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
            icon={<Network size={18} />}
            contentClassName="p-4 sm:p-5 space-y-3"
          >
            <p className="text-xs text-neutral-500">Agen yang mengajak akun ini dan menerima bonus pembinaan darinya.</p>
            <p className="text-sm font-heading font-semibold text-neutral-900">
              {akun.upline_akun_id ? <Link to={`/agen/${akun.upline_akun_id}`} className="text-primary-600 hover:underline">{akun.upline_nama}</Link> : 'Tanpa upline'}
            </p>
            {izin.gantiUpline && <Button variant="secondary" size="sm" onClick={() => navigate(gantiUpline)}>Ganti upline</Button>}
          </MetaBox>

          <MetaBox
            title="Agen"
            icon={<IdCard size={18} />}
            contentClassName="p-4 sm:p-5 space-y-2.5"
            headerAction={bolehNik ? <TombolUbah onClick={() => buka('nik')}>Ubah NIK</TombolUbah> : null}
          >
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

      <MetaBox title="Riwayat Perubahan Data" icon={<History size={18} />} contentClassName="p-4 sm:p-5">
        {akun.riwayat_data.length === 0 ? (
          <p className="text-sm text-neutral-500">Data akun belum pernah diubah admin.</p>
        ) : (
          <ol className="space-y-4">
            {akun.riwayat_data.map((l, i) => (
              <li key={i} className="text-sm">
                <p className="font-heading font-semibold text-neutral-900">{KOLOM[l.kolom] || l.kolom}: {l.nilai_lama || '-'} → {l.nilai_baru || '-'}</p>
                <p className="text-neutral-700">{l.alasan}</p>
                <p className="text-[11px] text-neutral-400">{l.diubah_oleh} · {dateTime(l.diubah_at)}</p>
              </li>
            ))}
          </ol>
        )}
      </MetaBox>

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

      <Modal
        isOpen={dialog !== null}
        onClose={() => !saving && setDialog(null)}
        title={dialog === 'nik' ? 'Ubah NIK Agen' : 'Ubah Data Akun'}
        footer={(
          <div className="flex justify-end gap-2 w-full">
            <Button variant="secondary" onClick={() => setDialog(null)} disabled={saving}>Batal</Button>
            <Button variant="primary" onClick={simpan} isLoading={saving}>Simpan</Button>
          </div>
        )}
      >
        <div className="space-y-4">
          {galat && (
            <Alert variant="error" message={galat.pesan} />
          )}
          {galat?.akunId && <p className="text-sm"><Link to={`/akun/${galat.akunId}`} onClick={() => setDialog(null)} className="text-primary-600 hover:underline">Buka akun pemakai</Link></p>}
          {dialog === 'profil' && (
            <>
              <Input label="Nama lengkap" value={form.nama_lengkap || ''} onChange={isi('nama_lengkap')} />
              <Input label="Nomor WhatsApp" value={form.no_hp || ''} onChange={isi('no_hp')} />
              <Input label="Domisili" value={form.domisili || ''} onChange={isi('domisili')} />
            </>
          )}
          {dialog === 'nik' && (
            <Input
              label="NIK baru (16 digit)"
              value={form.nik || ''}
              onChange={(e) => setForm((f) => ({ ...f, nik: e.target.value.replace(/\D/g, '').slice(0, 16) }))}
              helpText={akun.nik_agen ? `NIK sekarang ${akun.nik_agen}.` : 'Belum ada NIK tercatat.'}
            />
          )}
          <Textarea label="Alasan perubahan *" value={form.alasan || ''} onChange={isi('alasan')} rows={3} placeholder="Contoh: salah ketik saat mendaftar, diminta pemilik akun" />
        </div>
      </Modal>
    </div>
  );
}

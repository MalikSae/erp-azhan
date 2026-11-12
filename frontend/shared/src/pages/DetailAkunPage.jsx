import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ChevronDown, ChevronRight, Pencil } from 'lucide-react';
import { getAkunPelanggan, ubahProfilAkun } from '../api/akun';
import Alert from '../components/ui/Alert';
import { GantiUplineModal } from '../components/GantiKaitanModal';
import EmptyState from '../components/ui/EmptyState';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import Modal from '../components/ui/Modal';
import PageHeader from '../components/ui/PageHeader';
import Textarea from '../components/ui/Textarea';
import { dateLabel } from '../utils/agen';
import useProtectedImage from '../utils/useProtectedImage';
import { StatusAkun } from './DaftarAkunPage';

const dateTime = (v) => (!v ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(v)));

const KOLOM = { nama_lengkap: 'Nama', no_hp: 'WhatsApp', domisili: 'Domisili', nik_agen: 'NIK agen' };


const BULAN = ['JAN', 'FEB', 'MAR', 'APR', 'MEI', 'JUN', 'JUL', 'AGU', 'SEP', 'OKT', 'NOV', 'DES'];

// Nada Badge per status booking, sama dengan daftar Booking (satu gaya status, lihat Badge.jsx).
const STATUS_BOOKING = {
  draft: { label: 'Draft', variant: 'warning' },
  baru: { label: 'Baru', variant: 'info' },
  dp: { label: 'DP', variant: 'warning' },
  lunas: { label: 'Lunas', variant: 'success' },
  batal: { label: 'Batal', variant: 'danger' },
};
const StatusBooking = ({ status }) => {
  const m = STATUS_BOOKING[status] || { label: status, variant: 'neutral' };
  return <Badge variant={m.variant}>{m.label}</Badge>;
};

const TombolUbah = ({ onClick, children }) => (
  <Button variant="secondary" size="sm" onClick={onClick}><Pencil size={14} className="mr-1.5" />{children}</Button>
);

// Detail satu akun pelanggan. showBrandColumn: Admin Master, boleh membuka layar Ganti Kaitan Agen.
// izin: { ubahProfil, gantiUpline, lihatAgen, setujuiAgen, lihatBooking } menentukan tombol dan tautan (API tetap memeriksa izin sendiri).
// Data keagenan (NIK agen, kinerja, komisi) tidak ditampilkan di sini; rumahnya di halaman agen.
export default function DetailAkunPage({ showBrandColumn = false, izin = {} }) {
  const { id } = useParams();
  const navigate = useNavigate();
  const [akun, setAkun] = useState(null);
  const [error, setError] = useState('');
  const [dialog, setDialog] = useState(null); // 'profil'
  const [form, setForm] = useState({});
  const [saving, setSaving] = useState(false);
  const [galat, setGalat] = useState(null);
  const [gantiUplineBuka, setGantiUplineBuka] = useState(false);

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
  // Halaman agen: untuk agen butuh agen.view; untuk yang masih mengajukan butuh agen.approve.
  const bukaAgen = jadiAgen ? izin.lihatAgen : (akun.status_agen === 'pengajuan' && izin.setujuiAgen);

  const buka = (jenis) => {
    setGalat(null);
    setForm({ nama_lengkap: akun.nama_lengkap, no_hp: akun.no_hp, domisili: akun.domisili || '', alasan: '' });
    setDialog(jenis);
  };
  const isi = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const simpan = async () => {
    setSaving(true);
    setGalat(null);
    try {
      await ubahProfilAkun(akun.akun_id, form);
      setDialog(null);
      await muat();
    } catch (e) {
      const d = e.response?.data;
      setGalat({ pesan: d?.error || 'Perubahan gagal disimpan.', akunId: d?.akun_id || null });
    } finally {
      setSaving(false);
    }
  };

  // Riwayat perubahan data dan upline digabung, terbaru dulu.
  const riwayat = [
    ...akun.riwayat_data.map((l, i) => ({
      key: `d${i}`, waktu: l.diubah_at, oleh: l.diubah_oleh, alasan: l.alasan,
      judul: `${KOLOM[l.kolom] || l.kolom}: ${l.nilai_lama || '-'} → ${l.nilai_baru || '-'}`,
    })),
    ...akun.riwayat_upline.map((l) => ({
      key: `u${l.id}`, waktu: l.diganti_at, oleh: l.diganti_oleh, alasan: l.alasan,
      judul: `Upline: ${l.agen_lama_nama || 'Tanpa upline'} → ${l.agen_baru_nama || 'Tanpa upline'}`,
    })),
  ].sort((x, y) => new Date(y.waktu) - new Date(x.waktu));

  // Data diri bertumpuk (label di atas nilai); hanya yang terisi.
  const detail = [
    ['Domisili', akun.domisili],
    ['Jenis kelamin', akun.jenis_kelamin === 'L' ? 'Laki-laki' : akun.jenis_kelamin === 'P' ? 'Perempuan' : null],
    ['Tanggal lahir', akun.tanggal_lahir ? dateLabel(akun.tanggal_lahir) : null],
    ['Terdaftar', dateLabel(akun.created_at)],
    ['Login terakhir', akun.last_login_at ? dateTime(akun.last_login_at) : null],
    ['Jamaah', akun.jamaah ? <Link to={`/jamaah/${akun.jamaah.id}`} className="font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900">{akun.jamaah.nama_lengkap}</Link> : null],
  ].filter(([, v]) => v);

  const inisial = akun.nama_lengkap.split(/\s+/).filter(Boolean).slice(0, 2).map((k) => k[0]).join('').toUpperCase();
  const kartu = 'bg-white rounded-2xl border border-neutral-200';
  const tautan = 'font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900 cursor-pointer';

  return (
    <div className="space-y-6">
      {/* Header detail standar (design-system.md §5.4): nama, baris meta, baris badge, aksi di kanan. */}
      <PageHeader
        title={akun.nama_lengkap}
        onBack={() => navigate('/akun')}
        meta={[akun.no_hp, akun.email, showBrandColumn ? akun.brand_name : null]}
        badges={(
          <>
            {akun.status_agen !== 'tidak_aktif' && (
              <span className="inline-flex items-center gap-2 text-xs font-body">
                <StatusAkun status={akun.status_agen} />
                {bukaAgen && <Link to={`/agen/${akun.akun_id}`} className={tautan}>Buka</Link>}
              </span>
            )}
            <span className="inline-flex items-center gap-2 text-xs font-body">
              <Badge variant="neutral">
                Upline: {akun.upline_akun_id
                  ? <Link to={`/agen/${akun.upline_akun_id}`} className={tautan}>{akun.upline_nama}</Link>
                  : 'Tanpa upline'}
              </Badge>
              {izin.gantiUpline && <button type="button" onClick={() => setGantiUplineBuka(true)} className={tautan}>Ganti</button>}
            </span>
          </>
        )}
      >
        {izin.ubahProfil && <TombolUbah onClick={() => buka('profil')}>Ubah</TombolUbah>}
      </PageHeader>

      <div className="max-w-3xl space-y-6">
      {/* Profil: foto dan data diri (label di atas nilai). */}
      <div className={kartu}>
        <div className="flex items-start gap-5 p-5">
          <div className="h-14 w-14 shrink-0 overflow-hidden rounded-2xl bg-neutral-100 border border-neutral-200 flex items-center justify-center">
            {fotoSrc
              ? <img src={fotoSrc} alt={`Foto ${akun.nama_lengkap}`} className="h-full w-full object-cover" />
              : <span className="font-heading text-base font-semibold text-neutral-600">{inisial}</span>}
          </div>
          {detail.length > 0 && (
            <dl className="grid min-w-0 flex-1 grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-3">
              {detail.map(([label, nilai]) => (
                <div key={label} className="min-w-0">
                  <dt className="text-xs font-body text-neutral-500">{label}</dt>
                  <dd className="mt-0.5 truncate text-sm font-body font-medium text-neutral-900">{nilai}</dd>
                </div>
              ))}
            </dl>
          )}
        </div>
      </div>

      {/* Booking: tile tanggal berangkat di kiri, tiap baris membuka detail bookingnya. */}
      <div className={kartu}>
        <div className="flex items-baseline justify-between px-5 pt-5 pb-2">
          <h2 className="font-heading text-base md:text-lg font-semibold tracking-tight text-neutral-900">Booking</h2>
          <span className="text-xs tabular-nums text-neutral-500">{akun.booking.length}</span>
        </div>
        {akun.booking.length === 0 ? (
          <EmptyState variant="inline" message="Belum ada booking." className="px-5 pb-5" />
        ) : (
          <ul className="px-2 pb-2">
            {akun.booking.map((bk) => {
              const tgl = bk.berangkat_tanggal ? new Date(`${bk.berangkat_tanggal}T00:00:00`) : null;
              const isi = (
                <div className="flex items-center gap-4 rounded-xl px-3 py-2.5 transition-colors duration-150 group-hover:bg-neutral-50">
                  <div className="w-12 shrink-0 overflow-hidden rounded-lg border border-neutral-200 text-center">
                    <div className="bg-neutral-50 py-0.5 text-xs font-semibold text-neutral-500">{tgl ? BULAN[tgl.getMonth()] : '-'}</div>
                    <div className="py-0.5 font-heading text-base font-semibold tabular-nums text-neutral-900">{tgl ? tgl.getDate() : '-'}</div>
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-body font-medium text-neutral-900">{bk.jadwal_nama}</p>
                    <p className="text-xs text-neutral-500"><span className="font-mono">{bk.id_booking || `#${bk.id}`}</span> · {bk.jumlah_pax} pax{tgl ? ` · ${tgl.getFullYear()}` : ''}</p>
                  </div>
                  <StatusBooking status={bk.status} />
                  {izin.lihatBooking && <ChevronRight size={16} className="text-neutral-300 transition-colors group-hover:text-neutral-500" />}
                </div>
              );
              return (
                <li key={bk.id}>
                  {izin.lihatBooking ? <Link to={`/bookings/${bk.id}`} className="group block rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500">{isi}</Link> : isi}
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {/* Riwayat perubahan jarang dibutuhkan: tertutup sampai diklik. */}
      <details className={`${kartu} group`}>
        <summary className="flex cursor-pointer list-none items-center justify-between p-5 [&::-webkit-details-marker]:hidden">
          <span className="font-heading text-base md:text-lg font-semibold tracking-tight text-neutral-900">Riwayat Perubahan</span>
          <span className="flex items-center gap-2 text-xs tabular-nums text-neutral-500">
            {riwayat.length}
            <ChevronDown size={16} className="text-neutral-400 transition-transform duration-200 group-open:rotate-180" />
          </span>
        </summary>
        <div className="border-t border-neutral-200 p-5">
          {riwayat.length === 0 ? (
            <p className="text-sm text-neutral-500">Belum pernah diubah admin.</p>
          ) : (
            <ol className="relative space-y-4 border-l border-neutral-200 pl-4">
              {riwayat.map((r) => (
                <li key={r.key} className="relative text-sm">
                  <span className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-neutral-300 ring-4 ring-white" />
                  <p className="font-body font-medium text-neutral-900">{r.judul}</p>
                  <p className="text-neutral-600">{r.alasan}</p>
                  <p className="mt-0.5 text-xs text-neutral-400">{r.oleh} · {dateTime(r.waktu)}</p>
                </li>
              ))}
            </ol>
          )}
        </div>
      </details>
      </div>

      <Modal
        isOpen={dialog !== null}
        onClose={() => !saving && setDialog(null)}
        title="Ubah Data Akun"
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
          {galat?.akunId && <p className="text-sm"><Link to={`/akun/${galat.akunId}`} onClick={() => setDialog(null)} className="font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900">Buka akun pemakai</Link></p>}
          {dialog === 'profil' && (
            <>
              <Input label="Nama lengkap" value={form.nama_lengkap || ''} onChange={isi('nama_lengkap')} />
              <Input label="Nomor WhatsApp" value={form.no_hp || ''} onChange={isi('no_hp')} />
              <Input label="Domisili" value={form.domisili || ''} onChange={isi('domisili')} />
            </>
          )}
          <Textarea label="Alasan perubahan *" value={form.alasan || ''} onChange={isi('alasan')} rows={3} placeholder="Contoh: salah ketik saat mendaftar, diminta pemilik akun" />
        </div>
      </Modal>

      {izin.gantiUpline && (
        <GantiUplineModal
          akun={akun}
          isOpen={gantiUplineBuka}
          onClose={() => setGantiUplineBuka(false)}
          onSaved={muat}
        />
      )}
    </div>
  );
}

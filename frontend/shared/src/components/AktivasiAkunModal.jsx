import { useEffect, useState } from 'react';
import { kosongkanNikAgen, lepasTautanAkun, resetHitungAktivasi } from '../api/akun';
import Alert from './ui/Alert';
import Button from './ui/Button';
import Input from './ui/Input';
import Modal from './ui/Modal';
import Textarea from './ui/Textarea';

// Aksi admin untuk aktivasi akun jamaah (ADR 0003), semuanya dalam modal konfirmasi:
//   LepasTautanModal      → akun lepas dari jamaah tertaut (akun.edit)
//   KosongkanNikAgenModal → NIK agen akun yang bentrok dengan jamaah tertaut (izin menurut status akun)
//   ResetHitungModal      → buka kunci percobaan aktivasi satu kode booking atau NIK (akun.edit)
// Backend tetap penentu izin dan aturan; pesan galatnya dipakai apa adanya.

const ALASAN_MIN = 5;

const pesanGagal = (err, fallback) => err?.response?.data?.error || fallback;

const Ringkasan = ({ children }) => (
  <div className="space-y-1 rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-sm font-body text-neutral-800">{children}</div>
);

// Dasar kedua aksi berbasis alasan: ringkasan akibat, kolom alasan wajib, tombol Batal lalu aksi merah.
function AlasanModal({ isOpen, onClose, onSaved, title, aksiLabel, ringkasan, kirim, gagalPesan, placeholder }) {
  const [alasan, setAlasan] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen) { setAlasan(''); setError(''); }
  }, [isOpen]);

  const simpan = async () => {
    if (alasan.trim().length < ALASAN_MIN) { setError(`Alasan wajib diisi, minimal ${ALASAN_MIN} karakter.`); return; }
    setSaving(true);
    setError('');
    try {
      await kirim(alasan.trim());
      onSaved();
    } catch (e) {
      setError(pesanGagal(e, gagalPesan));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={() => !saving && onClose()}
      title={title}
      size="sm"
      footer={(
        <div className="flex w-full justify-end gap-2">
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button variant="danger" onClick={simpan} isLoading={saving}>{aksiLabel}</Button>
        </div>
      )}
    >
      <div className="space-y-4">
        {error && <Alert variant="error" message={error} />}
        {ringkasan && <Ringkasan>{ringkasan}</Ringkasan>}
        <Textarea label="Alasan *" value={alasan} onChange={(e) => setAlasan(e.target.value)} rows={3} placeholder={placeholder} />
      </div>
    </Modal>
  );
}

// akun: detail akun (nama_lengkap, jamaah). Pemanggil menyembunyikan tombolnya untuk akun agen; backend menolak 409.
export function LepasTautanModal({ akun, isOpen, onClose, onSaved }) {
  if (!akun?.jamaah) return null;
  return (
    <AlasanModal
      isOpen={isOpen}
      onClose={onClose}
      onSaved={onSaved}
      title="Lepas Tautan"
      aksiLabel="Lepas Tautan"
      placeholder="Tertaut ke jamaah yang salah"
      gagalPesan="Tautan gagal dilepas. Coba lagi."
      ringkasan={(
        <>
          <p><span className="font-medium">{akun.nama_lengkap}</span> lepas dari jamaah <span className="font-medium">{akun.jamaah.nama_lengkap}</span>.</p>
          <p className="text-neutral-600">Semua sesi akun dicabut. Upline tidak berubah.</p>
        </>
      )}
      kirim={(alasan) => lepasTautanAkun(akun.akun_id, { alasan })}
    />
  );
}

export function KosongkanNikAgenModal({ akun, isOpen, onClose, onSaved }) {
  if (!akun) return null;
  return (
    <AlasanModal
      isOpen={isOpen}
      onClose={onClose}
      onSaved={onSaved}
      title="Kosongkan NIK Agen"
      aksiLabel="Kosongkan"
      placeholder="NIK dipakai akun jamaah yang sah"
      gagalPesan="NIK agen gagal dikosongkan. Coba lagi."
      ringkasan={(
        <>
          <p>NIK agen <span className="font-mono">{akun.nik_agen}</span> milik {akun.nama_lengkap} juga milik jamaah yang tertaut ke akun lain.</p>
          <p className="text-neutral-600">NIK agen dikosongkan. Pemilik akun bisa mengisinya lagi.</p>
        </>
      )}
      kirim={(alasan) => kosongkanNikAgen(akun.akun_id, { alasan })}
    />
  );
}

// brandId: wajib untuk Admin Master (brand yang sedang dipilih); admin brand memakai brand-nya sendiri.
export function ResetHitungModal({ brandId, isOpen, onClose, onSaved }) {
  const [kode, setKode] = useState('');
  const [nik, setNik] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen) { setKode(''); setNik(''); setError(''); }
  }, [isOpen]);

  const simpan = async () => {
    const k = kode.trim();
    const n = nik.trim();
    if (!k && !n) { setError('Isi kode booking atau NIK.'); return; }
    if (k && n) { setError('Isi salah satu: kode booking atau NIK.'); return; }
    if (n && !/^d{16}$/.test(n)) { setError('NIK harus 16 digit.'); return; }
    setSaving(true);
    setError('');
    try {
      await resetHitungAktivasi({ kodeBooking: k, nik: n, brandId });
      onSaved();
    } catch (e) {
      setError(e.response?.status === 404 ? 'Aktivasi akun jamaah belum aktif.' : pesanGagal(e, 'Batas gagal direset. Coba lagi.'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={() => !saving && onClose()}
      title="Reset Batas Aktivasi"
      size="sm"
      footer={(
        <div className="flex w-full justify-end gap-2">
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button variant="primary" onClick={simpan} isLoading={saving}>Reset</Button>
        </div>
      )}
    >
      <div className="space-y-4">
        {error && <Alert variant="error" message={error} />}
        <Input label="Kode booking" value={kode} onChange={(e) => setKode(e.target.value)} placeholder="HN-AB12" />
        <Input label="NIK" value={nik} onChange={(e) => setNik(e.target.value)} placeholder="3201234567890001" inputMode="numeric" />
      </div>
    </Modal>
  );
}

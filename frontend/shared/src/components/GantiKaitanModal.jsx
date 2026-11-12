import { useEffect, useState } from 'react';
import { cariJamaahKaitan, gantiAgenAkun, gantiKaitanAgen, jumlahJamaahIkutUpline } from '../api/agen';
import Alert from './ui/Alert';
import Button from './ui/Button';
import Modal from './ui/Modal';
import Textarea from './ui/Textarea';
import KaitanAgenPicker, { EMPTY_KAITAN, validateKaitan } from './KaitanAgenPicker';

// Override admin (spek keagenan 4.4, izin agen.kaitan.edit) langsung dari halaman
// detail, dalam modal: tidak perlu pindah halaman dan mencari ulang orang yang sama.
//   GantiUplineModal     → upline sebuah akun (penerima bonus pembinaan bila akun jadi agen)
//   GantiAgenJamaahModal → agen tempat jamaah terikat (penerima komisi langsung)
// Riwayat perubahan tampil di halaman detail masing-masing.

const ALASAN_MIN = 5;

const Baris = ({ label, children }) => (
  <div className="flex items-start justify-between gap-3 text-sm font-body">
    <span className="text-neutral-500">{label}</span>
    <span className="text-right font-medium text-neutral-900">{children}</span>
  </div>
);

const Ringkasan = ({ children }) => (
  <div className="space-y-2 rounded-xl border border-neutral-200 bg-neutral-50 p-3">{children}</div>
);

const pesanGagal = (err, fallback) => err?.response?.data?.error || fallback;

// ─── Upline akun ────────────────────────────────────────────────────────────
// akun: { akun_id, brand_id, upline_akun_id, upline_nama, status_agen }
export const GantiUplineModal = ({ akun, isOpen, onClose, onSaved }) => {
  const [kaitan, setKaitan] = useState(EMPTY_KAITAN);
  const [alasan, setAlasan] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  // Jamaah yang ikut pindah (keputusan 12 November 2026): booking pertamanya dari akun ini dan
  // masih terikat ke upline lama. null = sedang dimuat atau gagal dimuat.
  const [jamaahIkut, setJamaahIkut] = useState(null);

  useEffect(() => {
    if (!isOpen || !akun) return;
    setKaitan(EMPTY_KAITAN);
    setAlasan('');
    setError('');
    setJamaahIkut(null);
    // -1 = gagal dimuat (pratinjau tidak menghalangi simpan; jumlah pasti dihitung ulang di server).
    jumlahJamaahIkutUpline(akun.akun_id).then(setJamaahIkut).catch(() => setJamaahIkut(-1));
  }, [isOpen, akun]);

  if (!akun) return null;

  const simpan = async () => {
    const kErr = validateKaitan(kaitan);
    if (kErr) { setError(kErr); return; }
    if (alasan.trim().length < ALASAN_MIN) { setError(`Alasan wajib diisi (minimal ${ALASAN_MIN} karakter).`); return; }
    setSaving(true);
    setError('');
    try {
      await gantiAgenAkun(akun.akun_id, {
        agenAkunId: kaitan.mode === 'agen' ? kaitan.agen.id : null,
        alasan: alasan.trim(),
      });
      onSaved?.();
      onClose();
    } catch (err) {
      setError(pesanGagal(err, 'Upline gagal diganti.'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={() => !saving && onClose()}
      title="Ganti upline"
      footer={(
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button variant="primary" onClick={simpan} isLoading={saving}>Simpan</Button>
        </>
      )}
    >
      <div className="space-y-4">
        <Ringkasan>
          <Baris label="Upline sekarang">{akun.upline_nama || 'Tanpa upline'}</Baris>
          {akun.upline_akun_id && (
            <Baris label="Jamaah ikut pindah">{jamaahIkut === null ? '…' : jamaahIkut < 0 ? '-' : jamaahIkut}</Baris>
          )}
        </Ringkasan>
        {error && <Alert variant="error" message={error} />}
        <KaitanAgenPicker
          name="ganti_upline"
          value={kaitan}
          onChange={(v) => { setKaitan(v); setError(''); }}
          brandId={akun.brand_id}
          brandRequired
          excludeAgenId={akun.upline_akun_id ?? undefined}
          tanpaAgenLabel="Tanpa upline"
          label="Upline baru"
          pilihAgenLabel="Pilih agen"
        />
        <Textarea
          label="Alasan *"
          value={alasan}
          onChange={(e) => setAlasan(e.target.value)}
          rows={3}
          placeholder="Contoh: akun diajak agen lain, salah pasang upline"
        />
      </div>
    </Modal>
  );
};

// ─── Agen tempat jamaah terikat ─────────────────────────────────────────────
// jamaah: { id, id_jamaah, brand_id }. Status kaitan (agen sekarang, komisi tercatat,
// booking belum lunas) dimuat saat modal dibuka.
export const GantiAgenJamaahModal = ({ jamaah, isOpen, onClose, onSaved }) => {
  const [info, setInfo] = useState(null);
  const [memuat, setMemuat] = useState(false);
  const [kaitan, setKaitan] = useState(EMPTY_KAITAN);
  const [alasan, setAlasan] = useState('');
  const [pindahkanKomisi, setPindahkanKomisi] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isOpen || !jamaah) return;
    setKaitan(EMPTY_KAITAN);
    setAlasan('');
    setPindahkanKomisi(false);
    setError('');
    setInfo(null);
    setMemuat(true);
    cariJamaahKaitan(jamaah.id_jamaah || String(jamaah.id), jamaah.brand_id)
      .then((items) => {
        const row = (items || []).find((it) => Number(it.jamaah_id) === Number(jamaah.id));
        if (row) setInfo(row);
        else setError('Data kaitan jamaah tidak ditemukan.');
      })
      .catch(() => setError('Data kaitan jamaah gagal dimuat.'))
      .finally(() => setMemuat(false));
  }, [isOpen, jamaah]);

  if (!jamaah) return null;
  const punyaKomisi = Boolean(info?.punya_komisi);

  const simpan = async () => {
    const kErr = validateKaitan(kaitan);
    if (kErr) { setError(kErr); return; }
    if (alasan.trim().length < ALASAN_MIN) { setError(`Alasan wajib diisi (minimal ${ALASAN_MIN} karakter).`); return; }
    if (punyaKomisi && kaitan.mode === 'agen' && !pindahkanKomisi) {
      setError('Jamaah ini sudah punya komisi tercatat. Centang "Pindahkan komisi" agar komisi ikut ke agen baru.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await gantiKaitanAgen(jamaah.id, {
        mode: kaitan.mode,
        agenAkunId: kaitan.agen?.id,
        alasan: alasan.trim(),
        pindahkanKomisi,
      });
      onSaved?.();
      onClose();
    } catch (err) {
      setError(pesanGagal(err, 'Agen gagal diganti.'));
    } finally {
      setSaving(false);
    }
  };

  const agenSekarang = info?.kaitan_status === 'terikat_agen'
    ? (info.agen_nama || `Agen #${info.agen_id}`)
    : info?.kaitan_status === 'tanpa_agen' ? 'Tanpa agen' : 'Belum ditentukan';

  return (
    <Modal
      isOpen={isOpen}
      onClose={() => !saving && onClose()}
      title="Ganti agen"
      footer={(
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button variant="primary" onClick={simpan} isLoading={saving} disabled={!info}>Simpan</Button>
        </>
      )}
    >
      <div className="space-y-4">
        {memuat && <p className="text-sm font-body text-neutral-500">Memuat data kaitan…</p>}
        {info && (
          <Ringkasan>
            <Baris label="Terikat ke">{agenSekarang}</Baris>
            <Baris label="Komisi tercatat">{punyaKomisi ? 'Ada' : 'Belum ada'}</Baris>
            <Baris label="Booking belum lunas">{info.booking_belum_lunas}</Baris>
          </Ringkasan>
        )}
        {info?.booking_belum_lunas > 0 && (
          <Alert
            variant="warning"
            message={`Komisi ${info.booking_belum_lunas} booking yang belum lunas dicatat saat lunas, jadi akan diberikan ke agen baru (atau tidak ada komisi bila tanpa agen).`}
          />
        )}
        {error && <Alert variant="error" message={error} />}
        {info && (
          <>
            <KaitanAgenPicker
              name="ganti_agen_jamaah"
              value={kaitan}
              onChange={(v) => { setKaitan(v); setError(''); }}
              brandId={info.brand_id}
              brandRequired
              excludeAgenId={info.agen_id ?? undefined}
              tanpaAgenLabel="Tanpa agen"
              label="Agen baru"
              pilihAgenLabel="Pilih agen"
              hint={punyaKomisi
                ? 'Jamaah yang sudah punya komisi tercatat tidak bisa dijadikan tanpa agen.'
                : 'Jamaah lain di booking yang sama tidak ikut berubah.'}
            />
            {punyaKomisi && kaitan.mode === 'agen' && (
              <label className="flex items-start gap-2.5 rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-sm font-body text-neutral-700 cursor-pointer">
                <input
                  type="checkbox"
                  checked={pindahkanKomisi}
                  onChange={(e) => setPindahkanKomisi(e.target.checked)}
                  className="mt-0.5 h-4 w-4 rounded border-neutral-300 text-primary-500 focus:ring-primary-500"
                />
                <span>
                  <span className="block font-medium text-neutral-900">Pindahkan komisi</span>
                  Komisi langsung pindah ke agen baru, bonus pembinaan ke upline-nya. Ditolak bila saldo agen lama sudah dicairkan.
                </span>
              </label>
            )}
            <Textarea
              label="Alasan *"
              value={alasan}
              onChange={(e) => setAlasan(e.target.value)}
              rows={3}
              placeholder="Contoh: jamaah dibawa agen lain secara langsung"
            />
          </>
        )}
      </div>
    </Modal>
  );
};

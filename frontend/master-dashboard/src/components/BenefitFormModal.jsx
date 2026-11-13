import { useEffect, useState } from 'react';
import Alert from 'shared/ui/Alert';
import BenefitIconPicker from 'shared/ui/BenefitIconPicker';
import Button from 'shared/ui/Button';
import Input from 'shared/ui/Input';
import Modal from 'shared/ui/Modal';

// Batas teks sama dengan backend (benefit.MaksTeks) dan lebar kotak benefit di app.
export const MAKS_TEKS_BENEFIT = 30;

// Jendela tambah/ubah satu highlight benefit: teks dan ikon. Dipakai halaman master Benefit Paket dan bagian
// Highlight Benefit di form paket.
//   awal    : { teks, ikon } untuk ubah; null/undefined untuk tambah
//   ikon    : daftar ikon dari server [{ key, label }]
//   onSimpan: async ({ teks, ikon }) => void; lempar error ber-response untuk menampilkan pesan server
export default function BenefitFormModal({ isOpen, onClose, awal, ikon = [], onSimpan, judul }) {
  const [teks, setTeks] = useState('');
  const [kunci, setKunci] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setTeks(awal?.teks || '');
    setKunci(awal?.ikon || '');
    setError('');
  }, [isOpen, awal]);

  const sisa = MAKS_TEKS_BENEFIT - [...teks.trim()].length;

  const simpan = async () => {
    if (!teks.trim()) return setError('Isi teks benefit.');
    if (sisa < 0) return setError(`Teks benefit maksimal ${MAKS_TEKS_BENEFIT} karakter.`);
    if (!kunci) return setError('Pilih satu ikon.');
    setSaving(true);
    setError('');
    try {
      await onSimpan({ teks: teks.trim(), ikon: kunci });
    } catch (err) {
      setError(err.response?.data?.error || 'Benefit gagal disimpan.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={() => !saving && onClose()}
      title={judul || (awal ? 'Ubah Benefit' : 'Tambah Benefit')}
      size="md"
      footer={
        <div className="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 w-full">
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button onClick={simpan} isLoading={saving}>Simpan</Button>
        </div>
      }
    >
      <div className="space-y-1">
        {error && <Alert variant="error" message={error} />}
        <Input
          label="Teks benefit"
          required
          value={teks}
          // Batas dihitung per karakter (code point) seperti server, bukan per unit UTF-16 seperti maxLength.
          onChange={(e) => setTeks([...e.target.value.trimStart()].slice(0, MAKS_TEKS_BENEFIT).join(''))}
          // Enter menyimpan benefit, bukan form lain yang mungkin membungkus jendela ini.
          onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); simpan(); } }}
          placeholder="Mis. Hotel dekat Haram"
          helpText={`Tampil di app dan web. Sisa ${Math.max(sisa, 0)} dari ${MAKS_TEKS_BENEFIT} karakter.`}
        />
        <BenefitIconPicker required value={kunci} onChange={setKunci} options={ikon} />
      </div>
    </Modal>
  );
}

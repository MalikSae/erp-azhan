import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { ArrowDown, ArrowUp, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { getBenefitOptions, createBenefitOption } from 'shared';
import Badge from 'shared/ui/Badge';
import BenefitIcon from 'shared/ui/BenefitIcon';
import Button from 'shared/ui/Button';
import CustomDropdown from 'shared/ui/CustomDropdown';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import BenefitFormModal from './BenefitFormModal';

// Jumlah benefit per paket, sama dengan backend (benefit.JumlahPerPaket).
export const JUMLAH_BENEFIT_PAKET = 4;

// Bagian "Highlight Benefit" di form paket: admin memilih tepat empat benefit dari master brand, mengatur
// urutannya, dan boleh menambah benefit baru ke master tanpa meninggalkan form. Mengubah atau menonaktifkan
// benefit hanya lewat menu Benefit Paket.
//   brandId : brand paket; kosong = belum bisa memilih
//   value   : benefit terpilih, berurutan [{ id, teks, ikon, is_active }]
//   onChange: (daftarBaru) => void
//   error   : pesan galat bagian ini (mis. kurang dari empat saat paket dijual)
export default function HighlightBenefitSection({ brandId, value = [], onChange, error }) {
  const [opsi, setOpsi] = useState([]);
  const [ikon, setIkon] = useState([]);
  const [memuat, setMemuat] = useState(false);
  const [gagalMuat, setGagalMuat] = useState(false);
  const [modalBuka, setModalBuka] = useState(false);
  // Dinaikkan untuk memuat ulang pilihan tanpa memuat ulang halaman (isian form tidak hilang).
  const [percobaan, setPercobaan] = useState(0);

  useEffect(() => {
    // Pilihan brand sebelumnya dibuang dulu, supaya benefit brand lain tidak pernah ditawarkan.
    setOpsi([]);
    if (!brandId) return undefined;
    let batal = false;
    setMemuat(true);
    setGagalMuat(false);
    getBenefitOptions(brandId)
      .then((d) => {
        if (batal) return;
        setOpsi(d.benefits);
        setIkon(d.ikon);
      })
      .catch(() => { if (!batal) setGagalMuat(true); })
      .finally(() => { if (!batal) setMemuat(false); });
    return () => { batal = true; };
  }, [brandId, percobaan]);

  const siapPilih = !memuat && !gagalMuat;
  const penuh = value.length >= JUMLAH_BENEFIT_PAKET;
  const terpilih = new Set(value.map((b) => b.id));
  const sisaOpsi = opsi.filter((b) => !terpilih.has(b.id));

  const tambah = (id) => {
    const b = opsi.find((o) => String(o.id) === String(id));
    if (!b || penuh || terpilih.has(b.id)) return;
    onChange([...value, { id: b.id, teks: b.teks, ikon: b.ikon, is_active: b.is_active }]);
  };

  const hapus = (id) => onChange(value.filter((b) => b.id !== id));

  const geser = (indeks, arah) => {
    const tujuan = indeks + arah;
    if (tujuan < 0 || tujuan >= value.length) return;
    const baru = [...value];
    [baru[indeks], baru[tujuan]] = [baru[tujuan], baru[indeks]];
    onChange(baru);
  };

  const pilih = (b) => {
    if (!penuh && !terpilih.has(b.id)) onChange([...value, { id: b.id, teks: b.teks, ikon: b.ikon, is_active: b.is_active !== false }]);
  };

  // Benefit baru masuk ke master brand, lalu langsung dipilih bila masih ada tempat. Teks yang sama dengan
  // benefit yang sudah ada (tanpa beda huruf besar/kecil) memilih benefit itu, bukan membuat yang baru.
  const simpanBaru = async ({ teks, ikon: kunci }) => {
    const kunciTeks = teks.trim().toLowerCase();
    const ada = opsi.find((o) => (o.teks || '').trim().toLowerCase() === kunciTeks);
    if (ada) {
      pilih(ada);
      setModalBuka(false);
      return;
    }
    // Server mengembalikan benefit lama bila teksnya sudah ada.
    const b = await createBenefitOption({ brand_id: Number(brandId), teks, ikon: kunci });
    setOpsi((lama) => (lama.some((o) => o.id === b.id) ? lama : [...lama, b]));
    pilih(b);
    setModalBuka(false);
  };

  if (!brandId) {
    return <p className="text-sm font-body text-neutral-500">Pilih brand dulu.</p>;
  }

  return (
    <div id="benefit_ids" tabIndex={-1} className="space-y-4 focus:outline-none">
      {value.length > 0 && (
        <ol className="divide-y divide-neutral-200 rounded-xl border border-neutral-200">
          {value.map((b, i) => (
            <li key={b.id} className="flex items-center gap-3 px-3 py-2">
              <span className="w-4 text-xs font-body tabular-nums text-neutral-500">{i + 1}</span>
              <BenefitIcon name={b.ikon} className="shrink-0 text-neutral-700" />
              <span className="min-w-0 flex-1 truncate text-sm font-body font-medium text-neutral-900">{b.teks}</span>
              {b.is_active === false && <Badge variant="neutral">Nonaktif</Badge>}
              <Button variant="ghost" size="sm" onClick={() => geser(i, -1)} disabled={i === 0} aria-label={`Naikkan ${b.teks}`}>
                <ArrowUp className="h-4 w-4" aria-hidden="true" />
              </Button>
              <Button variant="ghost" size="sm" onClick={() => geser(i, 1)} disabled={i === value.length - 1} aria-label={`Turunkan ${b.teks}`}>
                <ArrowDown className="h-4 w-4" aria-hidden="true" />
              </Button>
              <Button variant="danger-outline" size="sm" onClick={() => hapus(b.id)} aria-label={`Lepas ${b.teks}`}>
                <Trash2 className="h-4 w-4" aria-hidden="true" />
              </Button>
            </li>
          ))}
        </ol>
      )}

      {error && (
        <p className="text-xs font-body text-danger-600" role="alert">
          {error}
        </p>
      )}

      {gagalMuat && (
        <div className="flex flex-wrap items-center gap-3">
          <p className="text-xs font-body text-danger-600" role="alert">Daftar benefit gagal dimuat.</p>
          <Button variant="secondary" size="sm" onClick={() => setPercobaan((n) => n + 1)}>
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            <span>Coba lagi</span>
          </Button>
        </div>
      )}

      {memuat && <LoadingSpinner />}

      {!penuh && siapPilih && (sisaOpsi.length > 0 || ikon.length > 0) && (
        <div className={`grid grid-cols-1 gap-3 ${sisaOpsi.length > 0 && ikon.length > 0 ? 'md:grid-cols-2' : ''}`}>
          {sisaOpsi.length > 0 && (
            <CustomDropdown
              label="Tambah dari daftar benefit"
              className="!mb-0"
              value=""
              onChange={(v) => tambah(v?.target ? v.target.value : v)}
              placeholder="Pilih benefit"
              options={sisaOpsi.map((b) => ({ value: b.id, label: b.teks }))}
            />
          )}
          {ikon.length > 0 && (
            <div className="flex items-end">
              <Button variant="dashed" className="w-full" onClick={() => setModalBuka(true)}>
                <Plus className="h-4 w-4" aria-hidden="true" />
                <span>Buat benefit baru</span>
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Di luar <form> paket: tombol dan Enter di jendela ini tidak boleh ikut menyimpan paket. */}
      {createPortal(
        <BenefitFormModal
          isOpen={modalBuka}
          onClose={() => setModalBuka(false)}
          ikon={ikon}
          onSimpan={simpanBaru}
          judul="Buat Benefit Baru"
        />,
        document.body,
      )}
    </div>
  );
}

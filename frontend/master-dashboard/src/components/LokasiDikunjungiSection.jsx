import { useEffect, useState } from 'react';
import { ArrowDown, ArrowUp, MapPin, RefreshCw, Trash2 } from 'lucide-react';
import AutocompleteInput from 'shared/ui/AutocompleteInput';
import Badge from 'shared/ui/Badge';
import Button from 'shared/ui/Button';
import { getScheduleLokasiOptions } from '../api/schedules';
import { KOTA_HARAMAIN, mediaSrc } from '../pages/HaramainPage';

// Jumlah lokasi maksimal per paket, sama dengan backend.
export const MAKS_LOKASI_PAKET = 12;

const labelKota = (kota) => KOTA_HARAMAIN.find((k) => k.value === kota)?.label || kota || '';

// Memuat tempat aktif (pilihan). Tempat nonaktif yang sudah terpilih dilengkapi dari
// `tersimpan` (field "lokasi" detail paket: [{ id, nama, kota, foto_url, aktif }]).
export function useLokasiTempat(tersimpan = []) {
  const [opsi, setOpsi] = useState([]);
  const [memuat, setMemuat] = useState(true);
  const [gagalMuat, setGagalMuat] = useState(false);
  const [percobaan, setPercobaan] = useState(0);

  useEffect(() => {
    let batal = false;
    setMemuat(true);
    setGagalMuat(false);
    getScheduleLokasiOptions()
      .then((d) => { if (!batal) setOpsi(Array.isArray(d) ? d : []); })
      .catch(() => { if (!batal) setGagalMuat(true); })
      .finally(() => { if (!batal) setMemuat(false); });
    return () => { batal = true; };
  }, [percobaan]);

  const cari = (id) => {
    const o = opsi.find((x) => x.id === id);
    if (o) return { ...o, aktif: true };
    const s = tersimpan.find((x) => Number(x.id) === id);
    // Pilihan termuat: yang tidak ada di pilihan berarti nonaktif. Gagal dimuat: ikut data tersimpan.
    if (s) return { ...s, id, aktif: gagalMuat || memuat ? Boolean(s.aktif) : false };
    return { id, nama: `Tempat #${id}`, kota: '', foto_url: '', aktif: gagalMuat ? null : false };
  };

  return { opsi, cari, memuat, gagalMuat, muatUlang: () => setPercobaan((n) => n + 1) };
}

function Thumb({ url }) {
  const src = mediaSrc(url);
  if (!src) {
    return (
      <span className="flex h-10 w-14 shrink-0 items-center justify-center rounded-md border border-neutral-200 bg-neutral-50 text-neutral-400">
        <MapPin className="h-4 w-4" aria-hidden="true" />
      </span>
    );
  }
  return <img src={src} alt="" className="h-10 w-14 shrink-0 rounded-md border border-neutral-200 object-cover" />;
}

function BarisLokasi({ t, nomor, children }) {
  return (
    <li className="flex items-center gap-3 px-3 py-2">
      <span className="w-5 text-xs font-body tabular-nums text-neutral-500">{nomor}</span>
      <Thumb url={t.foto_url} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-body font-medium text-neutral-900">{t.nama}</p>
        {t.kota && <p className="text-xs font-body text-neutral-500">{labelKota(t.kota)}</p>}
      </div>
      {t.aktif === false && <Badge variant="neutral">Nonaktif</Badge>}
      {children}
    </li>
  );
}

// Daftar lokasi baca-saja untuk halaman detail paket (field "lokasi" detail paket).
export function LokasiTerpilihList({ lokasi = [] }) {
  return (
    <ol className="divide-y divide-neutral-200 rounded-xl border border-neutral-200">
      {lokasi.map((t, i) => <BarisLokasi key={t.id} t={{ ...t, aktif: Boolean(t.aktif) }} nomor={i + 1} />)}
    </ol>
  );
}

// Bagian "Lokasi Dikunjungi" di form paket: pilih tempat dari master Jelajah Haramain, atur urutan.
//   value   : id tempat terpilih, berurutan
//   tersimpan: field "lokasi" detail paket, untuk nama/foto tempat nonaktif
//   onChange: (idsBaru) => void
//   error   : pesan galat bagian ini
export default function LokasiDikunjungiSection({ value = [], tersimpan = [], onChange, error }) {
  const { opsi, cari, memuat, gagalMuat, muatUlang } = useLokasiTempat(tersimpan);

  const penuh = value.length >= MAKS_LOKASI_PAKET;
  const terpilih = new Set(value);
  const sisaOpsi = opsi.filter((t) => !terpilih.has(t.id));

  const tambah = (opt) => {
    const idBaru = Number(opt?.value);
    if (!idBaru || penuh || terpilih.has(idBaru)) return;
    onChange([...value, idBaru]);
  };

  const hapus = (idLepas) => onChange(value.filter((x) => x !== idLepas));

  const geser = (indeks, arah) => {
    const tujuan = indeks + arah;
    if (tujuan < 0 || tujuan >= value.length) return;
    const baru = [...value];
    [baru[indeks], baru[tujuan]] = [baru[tujuan], baru[indeks]];
    onChange(baru);
  };

  return (
    <div id="lokasi_ids" tabIndex={-1} className="space-y-4 focus:outline-none">
      {value.length > 0 && (
        <ol className="divide-y divide-neutral-200 rounded-xl border border-neutral-200">
          {value.map((idTempat, i) => {
            const t = cari(idTempat);
            return (
              <BarisLokasi key={idTempat} t={t} nomor={i + 1}>
                <Button variant="ghost" size="sm" onClick={() => geser(i, -1)} disabled={i === 0} aria-label={`Naikkan ${t.nama}`}>
                  <ArrowUp className="h-4 w-4" aria-hidden="true" />
                </Button>
                <Button variant="ghost" size="sm" onClick={() => geser(i, 1)} disabled={i === value.length - 1} aria-label={`Turunkan ${t.nama}`}>
                  <ArrowDown className="h-4 w-4" aria-hidden="true" />
                </Button>
                <Button variant="danger-outline" size="sm" onClick={() => hapus(idTempat)} aria-label={`Lepas ${t.nama}`}>
                  <Trash2 className="h-4 w-4" aria-hidden="true" />
                </Button>
              </BarisLokasi>
            );
          })}
        </ol>
      )}

      {error && (
        <p className="text-xs font-body text-danger-600" role="alert">
          {error}
        </p>
      )}

      {gagalMuat && (
        <div className="flex flex-wrap items-center gap-3">
          <p className="text-xs font-body text-danger-600" role="alert">Daftar tempat gagal dimuat.</p>
          <Button variant="secondary" size="sm" onClick={muatUlang}>
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            <span>Coba lagi</span>
          </Button>
        </div>
      )}

      {!penuh && !gagalMuat && !memuat && sisaOpsi.length > 0 && (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <AutocompleteInput
            label="Tambah lokasi"
            className="!mb-0"
            value=""
            placeholder="Cari nama atau kota"
            options={sisaOpsi.map((t) => ({ value: t.id, label: `${t.nama} · ${labelKota(t.kota)}` }))}
            onSelect={tambah}
          />
        </div>
      )}
    </div>
  );
}

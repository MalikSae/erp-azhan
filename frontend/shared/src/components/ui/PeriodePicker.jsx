import { useEffect, useRef, useState } from 'react';

const ymd = (d) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

const tampil = (v) => (!v ? '' : new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(`${v}T00:00:00`)));

// Pilihan periode siap pakai, dihitung dari tanggal hari ini (zona waktu browser).
const PRESET = {
  bulan_ini: { label: 'Bulan ini', range: (t) => [new Date(t.getFullYear(), t.getMonth(), 1), t] },
  bulan_lalu: { label: 'Bulan lalu', range: (t) => [new Date(t.getFullYear(), t.getMonth() - 1, 1), new Date(t.getFullYear(), t.getMonth(), 0)] },
  tahun_ini: { label: 'Tahun ini', range: (t) => [new Date(t.getFullYear(), 0, 1), t] },
};

// Pemilih periode dalam satu kontrol: tombol seukuran dropdown lain, menunya berisi pilihan siap pakai dan
// isian rentang tanggal. Rentang kustom diisi di dalam menu, jadi tidak ada baris baru yang mendorong isi
// halaman ke bawah.
//
//   value: { preset, dari, sampai }   preset: '' (semua) | bulan_ini | bulan_lalu | tahun_ini | kustom
//   onChange({ preset, dari, sampai }) dari dan sampai berformat YYYY-MM-DD, kosong untuk semua periode
//   labelSemua: teks pilihan tanpa batas waktu, mis. "Semua periode" atau "Sepanjang waktu"
export default function PeriodePicker({ value = {}, onChange, labelSemua = 'Semua periode', className = '' }) {
  const { preset = '', dari = '', sampai = '' } = value;
  const [buka, setBuka] = useState(false);
  const [dDari, setDDari] = useState(dari);
  const [dSampai, setDSampai] = useState(sampai);
  const ref = useRef(null);

  useEffect(() => {
    if (!buka) return undefined;
    const klikLuar = (e) => { if (ref.current && !ref.current.contains(e.target)) setBuka(false); };
    const tekan = (e) => { if (e.key === 'Escape') setBuka(false); };
    document.addEventListener('mousedown', klikLuar);
    document.addEventListener('keydown', tekan);
    return () => {
      document.removeEventListener('mousedown', klikLuar);
      document.removeEventListener('keydown', tekan);
    };
  }, [buka]);

  const bukaTutup = () => {
    if (!buka) {
      setDDari(dari);
      setDSampai(sampai);
    }
    setBuka(!buka);
  };

  const pilih = (key) => {
    if (!key) onChange?.({ preset: '', dari: '', sampai: '' });
    else {
      const [d, s] = PRESET[key].range(new Date()).map(ymd);
      onChange?.({ preset: key, dari: d, sampai: s });
    }
    setBuka(false);
  };

  const rentangSah = Boolean(dDari && dSampai && dDari <= dSampai);
  const terapkan = () => {
    if (!rentangSah) return;
    onChange?.({ preset: 'kustom', dari: dDari, sampai: dSampai });
    setBuka(false);
  };

  const label = preset === 'kustom' && dari && sampai
    ? `${tampil(dari)} - ${tampil(sampai)}`
    : PRESET[preset]?.label || labelSemua;

  const kelasPilihan = (aktif) => `w-full px-3.5 py-2.5 text-left text-sm font-body cursor-pointer transition-colors ${aktif ? 'bg-primary-50 text-primary-700 font-semibold' : 'text-neutral-700 hover:bg-neutral-50'}`;
  const kelasTanggal = 'h-9 w-full rounded-lg border border-neutral-200 bg-white px-2.5 text-sm font-body text-neutral-900 focus:outline-none focus:border-primary-400';

  return (
    <div className={`relative ${className}`} ref={ref}>
      <button
        type="button"
        aria-haspopup="dialog"
        aria-expanded={buka}
        onClick={bukaTutup}
        className="h-9 w-full min-w-0 flex items-center justify-between rounded-lg border border-neutral-300 bg-white px-3 text-sm font-body text-left text-neutral-900 cursor-pointer transition-all focus:outline-none"
      >
        <span className="truncate mr-2">{label}</span>
        <svg className={`w-4 h-4 transition-transform duration-200 shrink-0 text-neutral-400 ${buka ? 'transform rotate-180 text-primary-500' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {buka && (
        <div role="dialog" aria-label="Pilih periode" className="absolute left-0 top-full z-50 mt-1.5 w-72 max-w-[calc(100vw-2rem)] rounded-lg border border-neutral-200 bg-white shadow-popover overflow-hidden animate-in fade-in duration-150">
          <ul className="py-1">
            <li><button type="button" className={kelasPilihan(preset === '')} onClick={() => pilih('')}>{labelSemua}</button></li>
            {Object.entries(PRESET).map(([key, p]) => (
              <li key={key}><button type="button" className={kelasPilihan(preset === key)} onClick={() => pilih(key)}>{p.label}</button></li>
            ))}
          </ul>
          <div className={`border-t border-neutral-200 p-3 space-y-2 ${preset === 'kustom' ? 'bg-primary-50/40' : 'bg-neutral-50/60'}`}>
            <p className="text-xs font-semibold text-neutral-500">Rentang tanggal</p>
            <div className="grid grid-cols-2 gap-2">
              <label className="block">
                <span className="mb-1 block text-xs text-neutral-500">Dari</span>
                <input type="date" value={dDari} max={dSampai || undefined} onChange={(e) => setDDari(e.target.value)} className={kelasTanggal} />
              </label>
              <label className="block">
                <span className="mb-1 block text-xs text-neutral-500">Sampai</span>
                <input type="date" value={dSampai} min={dDari || undefined} onChange={(e) => setDSampai(e.target.value)} className={kelasTanggal} />
              </label>
            </div>
            <button
              type="button"
              onClick={terapkan}
              disabled={!rentangSah}
              className={`h-9 w-full rounded-lg text-sm font-heading font-semibold transition-colors ${rentangSah ? 'bg-accent-500 bg-btn-primary hover:bg-btn-primary-hover text-white border border-accent-600 shadow-btn-primary cursor-pointer' : 'bg-neutral-100 text-neutral-400 cursor-not-allowed'}`}
            >
              Terapkan
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

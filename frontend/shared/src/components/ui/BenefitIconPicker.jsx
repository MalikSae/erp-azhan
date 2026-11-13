import React from 'react';
import BenefitIcon from './BenefitIcon';
import Label from './Label';

// Pemilih ikon highlight benefit: kisi tombol ikon dari daftar tetap (options: [{ key, label }] dari server).
// Satu ikon terpilih; nama ikon terbaca lewat title dan aria-label.
export default function BenefitIconPicker({ label = 'Ikon', required = false, value, onChange, options = [], error }) {
  const terpilih = options.find((o) => o.key === value);
  // Pola radio group: satu tab stop (ikon terpilih, atau ikon pertama), panah memindahkan fokus dan pilihan.
  const KOLOM = 7;
  const indeksFokus = Math.max(0, options.findIndex((o) => o.key === value));
  const padaTombol = (e, i) => {
    const langkah = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: KOLOM, ArrowUp: -KOLOM }[e.key];
    let tujuan;
    if (langkah) tujuan = i + langkah;
    else if (e.key === 'Home') tujuan = 0;
    else if (e.key === 'End') tujuan = options.length - 1;
    else return;
    e.preventDefault();
    if (tujuan < 0 || tujuan >= options.length) return;
    onChange(options[tujuan].key);
    e.currentTarget.parentElement?.children[tujuan]?.focus();
  };
  return (
    <div className="mb-4">
      <Label required={required}>{label}</Label>
      <div role="radiogroup" aria-label={label} className="grid grid-cols-7 gap-2 max-h-64 overflow-y-auto rounded-xl border border-neutral-200 p-2">
        {options.map((o, i) => {
          const aktif = o.key === value;
          return (
            <button
              key={o.key}
              type="button"
              role="radio"
              aria-checked={aktif}
              aria-label={o.label}
              title={o.label}
              tabIndex={i === indeksFokus ? 0 : -1}
              onKeyDown={(e) => padaTombol(e, i)}
              onClick={() => onChange(o.key)}
              className={`flex h-10 items-center justify-center rounded-lg border transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 ${
                aktif ? 'border-primary-600 bg-primary-50 text-neutral-900' : 'border-transparent text-neutral-600 hover:bg-neutral-100'
              }`}
            >
              <BenefitIcon name={o.key} size={20} />
            </button>
          );
        })}
      </div>
      <p className="mt-1 text-xs text-neutral-500">{terpilih ? `Ikon terpilih: ${terpilih.label}` : 'Pilih satu ikon.'}</p>
      {error && <p className="mt-1 text-xs text-danger-600">{error}</p>}
    </div>
  );
}

import React, { useEffect, useRef, useState } from 'react';
import { Check } from 'lucide-react';
import { BADGE_FALLBACK, BADGE_PRESETS, isBadgeHex, badgeTextColor } from '../lib/badgeColors';

// Pemilih warna badge kategori: kotak warna (color picker bawaan browser), kolom kode hex, pilihan cepat,
// dan "Otomatis" (kosong = situs booking memilihkan warna). `value` berupa hex #rrggbb atau ''.
const BadgeColorField = ({ value, onChange, label }) => {
  const [text, setText] = useState(value || '');
  const pickerRef = useRef(null);
  const hasColor = isBadgeHex(value);

  // Sinkronkan kolom hex bila nilai berubah dari luar (pilih warna, pilihan cepat, buka kategori lain).
  useEffect(() => { setText(value || ''); }, [value]);

  const commitText = () => {
    const raw = text.trim().toLowerCase();
    if (raw === '') { onChange(''); return; }
    const hex = raw.startsWith('#') ? raw : `#${raw}`;
    if (isBadgeHex(hex)) onChange(hex);
    else setText(value || ''); // tidak valid: kembalikan ke nilai terakhir
  };

  const shown = hasColor ? value : BADGE_FALLBACK;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-3">
        {/* Kotak warna: <input type="color"> transparan di atas swatch agar tampilannya seragam */}
        <div className="relative h-10 w-10 shrink-0 overflow-hidden rounded-lg border border-neutral-300 shadow-sm focus-within:ring-2 focus-within:ring-primary-500">
          <div
            aria-hidden="true"
            className="absolute inset-0"
            style={hasColor
              ? { backgroundColor: value }
              : { backgroundImage: 'repeating-linear-gradient(45deg, #e5e7eb 0 6px, #f9fafb 6px 12px)' }}
          />
          <input
            ref={pickerRef}
            type="color"
            aria-label="Pilih warna badge"
            value={shown}
            onChange={(e) => onChange(e.target.value.toLowerCase())}
            className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
          />
        </div>

        <input
          type="text"
          inputMode="text"
          aria-label="Kode warna hex"
          placeholder="Otomatis"
          maxLength={7}
          value={text.toUpperCase()}
          onChange={(e) => setText(e.target.value)}
          onBlur={commitText}
          onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); commitText(); } }}
          className="h-10 w-28 rounded-lg border border-neutral-300 bg-white px-3 text-sm font-mono tracking-wide text-neutral-800 placeholder:font-body placeholder:tracking-normal placeholder:text-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30"
        />

        <button
          type="button"
          onClick={() => onChange('')}
          aria-pressed={!hasColor}
          className={`h-10 rounded-lg border px-3 text-xs font-semibold cursor-pointer transition-colors ${!hasColor ? 'border-neutral-900 bg-neutral-900 text-white' : 'border-neutral-300 text-neutral-600 hover:bg-neutral-50'}`}
        >
          Otomatis
        </button>
      </div>

      {/* Pilihan cepat */}
      <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Pilihan cepat warna">
        <span className="text-[11px] font-body text-neutral-500 mr-1">Cepat:</span>
        {BADGE_PRESETS.map((c) => {
          const active = hasColor && value === c.hex;
          return (
            <button
              key={c.hex}
              type="button"
              title={c.label}
              aria-label={c.label}
              aria-pressed={active}
              onClick={() => onChange(c.hex)}
              style={{ backgroundColor: c.hex, color: badgeTextColor(c.hex) }}
              className={`flex h-6 w-6 items-center justify-center rounded-full cursor-pointer ring-offset-1 transition-shadow ${active ? 'ring-2 ring-neutral-900' : 'hover:ring-2 hover:ring-neutral-300'}`}
            >
              {active && <Check size={13} strokeWidth={3} />}
            </button>
          );
        })}
      </div>

      {/* Pratinjau */}
      <div className="flex items-center gap-2 rounded-lg bg-neutral-50 px-3 py-2">
        <span className="text-[11px] font-body text-neutral-500">Pratinjau:</span>
        <span
          className="inline-block rounded px-2.5 py-0.5 text-[11px] font-bold"
          style={{ backgroundColor: shown, color: badgeTextColor(shown) }}
        >
          {label || 'Kategori'}
        </span>
        {!hasColor && <span className="text-[11px] font-body text-neutral-400">warna final ditentukan situs booking</span>}
      </div>
    </div>
  );
};

export default BadgeColorField;

import React, { useEffect, useState } from 'react';
import { Search, X, UserCheck } from 'lucide-react';
import { listAgenAktif } from '../api/agen';

// Pilihan kaitan agen untuk jamaah baru yang dibuat admin (Jalur 3,
// agen-azhan.md 3.5). Admin wajib memilih eksplisit: kaitkan ke agen aktif
// atau tanpa agen. Nilai: { mode: '' | 'agen' | 'tanpa_agen', agen: {id, nama_lengkap, kode_referral} | null }.

export const EMPTY_KAITAN = { mode: '', agen: null };

// Validasi sisi klien; mengembalikan pesan error atau null.
export function validateKaitan(value) {
  if (!value || (value.mode !== 'agen' && value.mode !== 'tanpa_agen')) {
    return 'Pilih kaitan agen: kaitkan ke agen atau tanpa agen';
  }
  if (value.mode === 'agen' && !value.agen?.id) {
    return 'Pilih agen terlebih dahulu';
  }
  return null;
}

export function toKaitanPayload(value) {
  return value.mode === 'agen'
    ? { mode: 'agen', agen_jamaah_id: value.agen.id }
    : { mode: 'tanpa_agen' };
}

const agenLabel = (a) => (a.kode_referral ? `${a.nama_lengkap} · ${a.kode_referral}` : a.nama_lengkap);

const KaitanAgenPicker = ({ value = EMPTY_KAITAN, onChange, brandId, brandRequired = false, error, hint, name = 'kaitan_agen' }) => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(null);

  const brandMissing = brandRequired && !brandId;
  const searching = value.mode === 'agen' && !value.agen && !brandMissing;

  useEffect(() => {
    if (!searching) return undefined;
    let cancelled = false;
    const timer = setTimeout(async () => {
      setLoading(true);
      setLoadError(null);
      try {
        const items = await listAgenAktif({ brandId: brandRequired ? brandId : undefined, q: query.trim() });
        if (!cancelled) setResults(items);
      } catch (err) {
        if (!cancelled) setLoadError(err.response?.data?.error || 'Gagal memuat daftar agen');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }, 300);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [searching, query, brandId, brandRequired]);

  const setMode = (mode) => onChange({ mode, agen: mode === 'agen' ? value.agen : null });

  return (
    <div className="flex flex-col gap-2">
      <label className="text-sm font-semibold text-neutral-900 font-heading">
        Kaitan Agen <span className="text-danger-500">*</span>
      </label>
      <div className="flex flex-wrap gap-x-6 gap-y-2">
        {[
          { mode: 'agen', label: 'Kaitkan ke Agen' },
          { mode: 'tanpa_agen', label: 'Tanpa Agen' },
        ].map((opt) => (
          <label key={opt.mode} className="flex items-center gap-2 text-sm font-body cursor-pointer text-neutral-700 hover:text-neutral-900">
            <input
              type="radio"
              name={name}
              value={opt.mode}
              checked={value.mode === opt.mode}
              onChange={() => setMode(opt.mode)}
              className="w-4 h-4 text-primary-500 border-neutral-300 focus:ring-primary-500"
            />
            {opt.label}
          </label>
        ))}
      </div>
      {hint && <p className="text-[11px] text-neutral-500 font-body">{hint}</p>}

      {value.mode === 'agen' && brandMissing && (
        <p className="text-xs text-warning-700 font-body">Pilih biro travel / brand terlebih dahulu untuk mencari agen.</p>
      )}

      {value.mode === 'agen' && value.agen && (
        <div className="flex items-center justify-between gap-2 rounded-xl border border-primary-200 bg-primary-50 px-3 py-2">
          <div className="flex items-center gap-2 min-w-0">
            <UserCheck size={16} className="text-neutral-700 shrink-0" />
            <span className="text-sm text-neutral-900 font-body truncate">{agenLabel(value.agen)}</span>
          </div>
          <button
            type="button"
            onClick={() => onChange({ mode: 'agen', agen: null })}
            className="text-neutral-500 hover:text-neutral-800 shrink-0"
            aria-label="Ganti agen"
          >
            <X size={16} />
          </button>
        </div>
      )}

      {searching && (
        <div className="flex flex-col gap-1.5">
          <div className="relative">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Cari nama, no HP, atau kode referral agen..."
              className="h-11 w-full min-w-0 rounded-xl border border-neutral-200/90 bg-white pl-9 pr-3.5 text-xs md:text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs font-body"
            />
          </div>
          <div className="max-h-52 overflow-y-auto rounded-xl border border-neutral-200/90 bg-white">
            {loading && <p className="px-3 py-2 text-xs text-neutral-500 font-body">Memuat agen...</p>}
            {!loading && loadError && <p className="px-3 py-2 text-xs text-danger-600 font-body">{loadError}</p>}
            {!loading && !loadError && results.length === 0 && (
              <p className="px-3 py-2 text-xs text-neutral-500 font-body">Tidak ada agen aktif yang cocok.</p>
            )}
            {!loading && !loadError && results.map((a) => (
              <button
                key={a.id}
                type="button"
                onClick={() => onChange({ mode: 'agen', agen: a })}
                className="flex w-full flex-col items-start px-3 py-2 text-left hover:bg-neutral-50 border-b border-neutral-100 last:border-b-0"
              >
                <span className="text-sm text-neutral-900 font-body">{a.nama_lengkap}</span>
                <span className="text-[11px] text-neutral-500 font-body">
                  {[a.kode_referral, a.no_hp].filter(Boolean).join(' · ')}
                </span>
              </button>
            ))}
          </div>
        </div>
      )}

      {error && <p className="text-xs text-danger-600 font-body">{error}</p>}
    </div>
  );
};

export default KaitanAgenPicker;

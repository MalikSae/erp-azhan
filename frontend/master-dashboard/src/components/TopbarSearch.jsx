import React, { useContext, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Loader2, Plane, CalendarDays, Users } from 'lucide-react';
import { cariGlobal, getStatusBadgeConfig } from 'shared';
import { AuthContext } from '../context/AuthContext';

const PANJANG_MINIMAL = 2;
const JEDA_KETIK_MS = 250;

const STATUS_PAKET = { published: 'Published', draft: 'Draft', archived: 'Archived' };
// Status yang perlu menonjol supaya data lama atau batal tidak salah dipilih.
const STATUS_REDUP = new Set(['archived', 'draft', 'batal']);

const tanggal = (iso) => {
  if (!iso) return '';
  const d = new Date(`${iso}T00:00:00`);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
};

const isMac = typeof navigator !== 'undefined' && /mac|iphone|ipad/i.test(navigator.platform || navigator.userAgent || '');

// Pencarian global topbar: paket, booking, jamaah. Server menyaring per izin dan brand, jadi
// kelompok yang tidak boleh dilihat peran datang kosong dan tidak ditampilkan.
const TopbarSearch = () => {
  const { isHolding } = useContext(AuthContext);
  const navigate = useNavigate();
  const [kata, setKata] = useState('');
  const [hasil, setHasil] = useState(null); // null = belum ada pencarian selesai untuk kata ini
  const [memuat, setMemuat] = useState(false);
  const [gagal, setGagal] = useState(false);
  const [terbuka, setTerbuka] = useState(false);
  const [aktif, setAktif] = useState(0);
  const wadahRef = useRef(null);
  const inputRef = useRef(null);

  const kataBersih = kata.trim();
  const cukupPanjang = kataBersih.length >= PANJANG_MINIMAL;

  // Ctrl+K / ⌘K memfokuskan kotak pencarian dari halaman mana pun.
  useEffect(() => {
    const onKey = (e) => {
      if ((e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
        inputRef.current?.select();
        setTerbuka(true);
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);

  useEffect(() => {
    const onDown = (e) => {
      if (wadahRef.current && !wadahRef.current.contains(e.target)) setTerbuka(false);
    };
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, []);

  // Cari setelah jeda ketik; permintaan yang tertinggal ketikan dibatalkan.
  useEffect(() => {
    setHasil(null);
    setGagal(false);
    setAktif(0);
    if (!cukupPanjang) {
      setMemuat(false);
      return undefined;
    }
    setMemuat(true);
    const pembatal = new AbortController();
    const timer = setTimeout(() => {
      cariGlobal(kataBersih, { signal: pembatal.signal })
        .then((d) => { setHasil(d); setMemuat(false); })
        .catch(() => {
          if (pembatal.signal.aborted) return;
          setGagal(true);
          setMemuat(false);
        });
    }, JEDA_KETIK_MS);
    return () => {
      clearTimeout(timer);
      pembatal.abort();
    };
  }, [kataBersih, cukupPanjang]);

  const kelompok = useMemo(() => {
    if (!hasil) return [];
    const brand = (b) => (isHolding && b ? b : '');
    return [
      {
        judul: 'Paket',
        ikon: Plane,
        butir: hasil.paket.map((p) => ({
          kunci: `paket-${p.id}`,
          ke: `/schedules/${p.id}`,
          utama: p.nama,
          rinci: [tanggal(p.berangkat_tanggal), brand(p.brand)],
          status: STATUS_PAKET[p.status] || p.status,
          redup: STATUS_REDUP.has(p.status),
        })),
      },
      {
        judul: 'Booking',
        ikon: CalendarDays,
        butir: hasil.booking.map((b) => ({
          kunci: `booking-${b.id}`,
          ke: `/bookings/${b.id}`,
          utama: b.id_booking || `Booking #${b.id}`,
          rinci: [b.nama_pemesan, b.jadwal_nama, brand(b.brand)],
          status: getStatusBadgeConfig(b.status)[1],
          redup: STATUS_REDUP.has(b.status),
        })),
      },
      {
        judul: 'Jamaah',
        ikon: Users,
        butir: hasil.jamaah.map((j) => ({
          kunci: `jamaah-${j.id}`,
          ke: `/jamaah/${j.id}`,
          utama: j.nama_lengkap,
          rinci: [j.id_jamaah, j.no_hp, brand(j.brand)],
        })),
      },
    ].filter((k) => k.butir.length > 0);
  }, [hasil, isHolding]);

  const datar = useMemo(() => kelompok.flatMap((k) => k.butir), [kelompok]);

  const buka = (butir) => {
    setTerbuka(false);
    setKata('');
    inputRef.current?.blur();
    navigate(butir.ke);
  };

  const onKeyDown = (e) => {
    if (e.key === 'Escape') {
      setTerbuka(false);
      inputRef.current?.blur();
      return;
    }
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      setTerbuka(true);
      if (datar.length === 0) return;
      const arah = e.key === 'ArrowDown' ? 1 : -1;
      setAktif((i) => (i + arah + datar.length) % datar.length);
      return;
    }
    if (e.key === 'Enter' && terbuka && datar[aktif]) {
      e.preventDefault();
      buka(datar[aktif]);
    }
  };

  // Butir terpilih ikut tergulir ke dalam panel saat dipilih lewat papan ketik.
  useEffect(() => {
    if (!terbuka || !datar[aktif]) return;
    document.getElementById(`cari-${datar[aktif].kunci}`)?.scrollIntoView({ block: 'nearest' });
  }, [aktif, terbuka, datar]);

  const panelTampil = terbuka && kataBersih.length > 0;
  const idAktif = panelTampil && datar[aktif] ? `cari-${datar[aktif].kunci}` : undefined;

  return (
    <div ref={wadahRef} className="hidden md:block relative w-72 lg:w-96">
      <label className="flex items-center gap-2 bg-neutral-50 border border-neutral-200/80 rounded-xl px-3.5 py-1.5 text-sm hover:border-neutral-300 focus-within:border-primary-400 focus-within:bg-white focus-within:ring-2 focus-within:ring-primary-100 transition-colors cursor-text">
        {memuat
          ? <Loader2 className="w-4 h-4 text-neutral-400 shrink-0 animate-spin" />
          : <Search className="w-4 h-4 text-neutral-400 shrink-0" />}
        <input
          ref={inputRef}
          type="text"
          value={kata}
          onChange={(e) => { setKata(e.target.value); setTerbuka(true); }}
          onFocus={() => setTerbuka(true)}
          onKeyDown={onKeyDown}
          placeholder="Cari paket, booking, jamaah..."
          aria-label="Cari paket, booking, jamaah"
          role="combobox"
          aria-expanded={panelTampil}
          aria-controls="cari-hasil"
          aria-activedescendant={idAktif}
          aria-autocomplete="list"
          autoComplete="off"
          spellCheck={false}
          maxLength={80}
          className="flex-1 min-w-0 bg-transparent text-xs text-neutral-800 placeholder:text-neutral-500 font-body focus:outline-none"
        />
        <kbd className="ml-auto shrink-0 text-[10px] font-mono bg-white px-1.5 py-0.5 rounded border border-neutral-200 text-neutral-400 shadow-2xs">
          {isMac ? '⌘K' : 'Ctrl K'}
        </kbd>
      </label>

      {panelTampil && (
        <div
          id="cari-hasil"
          role="listbox"
          aria-label="Hasil pencarian"
          className="absolute left-0 right-0 mt-2 bg-white rounded-xl shadow-lg border border-neutral-200/90 z-50 py-1.5 max-h-[70vh] overflow-y-auto"
        >
          {!cukupPanjang && (
            <p className="px-4 py-3 text-xs text-neutral-500 font-body">Ketik minimal {PANJANG_MINIMAL} karakter.</p>
          )}
          {cukupPanjang && gagal && (
            <p className="px-4 py-3 text-xs text-danger-600 font-body">Pencarian gagal. Coba lagi.</p>
          )}
          {cukupPanjang && !gagal && !hasil && (
            <p className="px-4 py-3 text-xs text-neutral-500 font-body">Mencari...</p>
          )}
          {cukupPanjang && hasil && datar.length === 0 && (
            <p className="px-4 py-3 text-xs text-neutral-500 font-body">Tidak ada hasil untuk “{kataBersih}”.</p>
          )}
          {kelompok.map((k) => (
            <div key={k.judul} role="group" aria-label={k.judul} className="py-1">
              <p className="px-4 pt-1.5 pb-1 text-[10px] font-bold uppercase tracking-wider text-neutral-400 font-heading flex items-center gap-1.5">
                <k.ikon className="w-3 h-3" />
                {k.judul}
              </p>
              {k.butir.map((b) => {
                const terpilih = datar[aktif]?.kunci === b.kunci;
                const rinci = b.rinci.filter(Boolean).join(' · ');
                return (
                  <div
                    key={b.kunci}
                    id={`cari-${b.kunci}`}
                    role="option"
                    aria-selected={terpilih}
                    // mousedown, bukan click: input tidak kehilangan fokus lebih dulu lalu menutup panel.
                    onMouseDown={(e) => { e.preventDefault(); buka(b); }}
                    onMouseEnter={() => setAktif(datar.findIndex((d) => d.kunci === b.kunci))}
                    className={`px-4 py-2 flex items-center gap-3 cursor-pointer ${terpilih ? 'bg-primary-50' : ''}`}
                  >
                    <div className="min-w-0 flex-1">
                      <p className="text-xs font-semibold text-neutral-900 font-body truncate">{b.utama}</p>
                      {rinci && <p className="text-[11px] text-neutral-500 font-body truncate">{rinci}</p>}
                    </div>
                    {b.status && (
                      <span className={`shrink-0 text-[10px] font-semibold px-1.5 py-0.5 rounded-md border ${b.redup ? 'bg-neutral-100 border-neutral-300 text-neutral-600' : 'bg-white border-neutral-200 text-neutral-700'}`}>
                        {b.status}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default TopbarSearch;

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Bell, ChevronRight, CircleCheckBig } from 'lucide-react';
import { getNotifikasi, EVENT_PERUBAHAN_JAMAAH, EVENT_PENCAIRAN_KOMISI } from 'shared';

const SELANG_MUAT_MS = 60_000;

// Jenis antrean dari server -> kalimat dan halaman tempat antrean itu diproses.
const ANTREAN = {
  pembayaran: { ke: '/payments', teks: 'pembayaran menunggu verifikasi' },
  perubahan_jamaah: { ke: '/jamaah', teks: 'perubahan data jamaah menunggu persetujuan' },
  pengajuan_agen: { ke: '/agen/pengajuan', teks: 'pengajuan agen menunggu keputusan' },
  pencairan_komisi: { ke: '/komisi', teks: 'pencairan komisi menunggu keputusan' },
};

// Lonceng topbar: pekerjaan yang menunggu keputusan, dihitung server sesuai izin dan brand.
// Tidak ada status dibaca: titik merah hilang sendiri saat antrean kosong.
const TopbarNotifikasi = () => {
  const location = useLocation();
  const [antrean, setAntrean] = useState([]);
  const [gagal, setGagal] = useState(false);
  const [terbuka, setTerbuka] = useState(false);
  const wadahRef = useRef(null);

  const muat = useCallback(() => {
    getNotifikasi()
      .then((d) => { setAntrean(d.antrean); setGagal(false); })
      .catch(() => setGagal(true));
  }, []);

  // Dimuat saat dashboard dibuka dan tiap pindah halaman, begitu ada keputusan atas antrean,
  // dan tiap menit selama tab terlihat.
  useEffect(() => {
    muat();
  }, [muat, location.pathname]);

  useEffect(() => {
    window.addEventListener(EVENT_PERUBAHAN_JAMAAH, muat);
    window.addEventListener(EVENT_PENCAIRAN_KOMISI, muat);
    const muatBilaTerlihat = () => { if (document.visibilityState === 'visible') muat(); };
    const timer = setInterval(muatBilaTerlihat, SELANG_MUAT_MS);
    document.addEventListener('visibilitychange', muatBilaTerlihat);
    return () => {
      window.removeEventListener(EVENT_PERUBAHAN_JAMAAH, muat);
      window.removeEventListener(EVENT_PENCAIRAN_KOMISI, muat);
      clearInterval(timer);
      document.removeEventListener('visibilitychange', muatBilaTerlihat);
    };
  }, [muat]);

  useEffect(() => {
    if (!terbuka) return undefined;
    const onDown = (e) => {
      if (wadahRef.current && !wadahRef.current.contains(e.target)) setTerbuka(false);
    };
    const onKey = (e) => { if (e.key === 'Escape') setTerbuka(false); };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [terbuka]);

  const menunggu = antrean.filter((a) => a.jumlah > 0 && ANTREAN[a.jenis]);
  const total = menunggu.reduce((n, a) => n + a.jumlah, 0);
  const judul = total > 0 ? `Notifikasi: ${total} pekerjaan menunggu` : 'Notifikasi';

  return (
    <div className="relative" ref={wadahRef}>
      <button
        type="button"
        onClick={() => { if (!terbuka) muat(); setTerbuka(!terbuka); }}
        className="w-9 h-9 flex items-center justify-center text-neutral-500 hover:text-neutral-800 bg-white hover:bg-neutral-50 border border-neutral-200 rounded-lg transition-colors relative"
        title={judul}
        aria-label={judul}
        aria-haspopup="true"
        aria-expanded={terbuka}
      >
        <Bell className="w-[18px] h-[18px]" />
        {total > 0 && (
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-danger-500 rounded-full ring-2 ring-white" />
        )}
      </button>

      {terbuka && (
        <div className="absolute right-0 mt-2 w-80 max-w-[calc(100vw-2rem)] bg-white rounded-xl shadow-lg border border-neutral-200/90 z-50 py-1.5">
          <p className="px-4 py-2 border-b border-neutral-100 text-xs font-bold text-neutral-900 font-heading">
            Menunggu keputusan
          </p>
          {gagal && menunggu.length === 0 ? (
            <p className="px-4 py-4 text-xs text-danger-600 font-body">Notifikasi gagal dimuat. Coba lagi.</p>
          ) : menunggu.length === 0 ? (
            <p className="px-4 py-4 text-xs text-neutral-500 font-body flex items-center gap-2">
              <CircleCheckBig className="w-4 h-4 text-success-600 shrink-0" />
              Tidak ada yang menunggu.
            </p>
          ) : (
            <ul className="py-1">
              {menunggu.map((a) => (
                <li key={a.jenis}>
                  <Link
                    to={ANTREAN[a.jenis].ke}
                    onClick={() => setTerbuka(false)}
                    className="px-4 py-2.5 flex items-center gap-3 hover:bg-neutral-50 transition-colors"
                  >
                    <span className="shrink-0 min-w-7 h-7 px-1.5 rounded-lg bg-danger-50 border border-danger-200 text-danger-700 text-xs font-bold font-heading flex items-center justify-center tabular-nums">
                      {a.jumlah}
                    </span>
                    <span className="flex-1 text-xs text-neutral-800 font-body">{ANTREAN[a.jenis].teks}</span>
                    <ChevronRight className="w-3.5 h-3.5 text-neutral-400 shrink-0" />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
};

export default TopbarNotifikasi;

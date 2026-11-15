import React, { useMemo, useState } from 'react';
import { Search } from 'lucide-react';
import Input from 'shared/ui/Input';
import EmptyState from 'shared/ui/EmptyState';

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
export const tanggalPendek = (iso) => {
  if (!iso) return '-';
  const [y, m, d] = iso.slice(0, 10).split('-').map(Number);
  return `${d} ${BULAN[m - 1]} ${y}`;
};

// Daftar centang paket untuk manasik. Paket datang dari /manasik/paket-tersedia (belum berangkat, belum punya manasik),
// urut tanggal berangkat terdekat. Satu manasik hanya untuk paket satu brand: setelah satu paket dicentang, paket brand
// lain tidak bisa dipilih. Paket yang berangkat pada atau sebelum tanggal manasik juga tidak bisa dipilih.
const PilihPaketManasik = ({ paket, value, onChange, tanggalManasik, brandNama = {}, tampilBrand = false }) => {
  const [cari, setCari] = useState('');
  const terpilih = useMemo(() => new Set(value), [value]);
  const brandTerkunci = useMemo(() => paket.find((p) => terpilih.has(p.schedule_id))?.brand_id ?? null, [paket, terpilih]);
  const tampil = useMemo(() => {
    const q = cari.trim().toLowerCase();
    return q ? paket.filter((p) => p.nama.toLowerCase().includes(q)) : paket;
  }, [paket, cari]);

  const alasanNonaktif = (p) => {
    if (brandTerkunci !== null && p.brand_id !== brandTerkunci) return 'Brand lain';
    if (tanggalManasik && p.berangkat_tanggal <= tanggalManasik) return 'Berangkat sebelum manasik';
    return null;
  };

  const ubah = (id) => onChange(terpilih.has(id) ? value.filter((x) => x !== id) : [...value, id]);

  return (
    <div className="space-y-3">
      <Input
        name="cari_paket"
        value={cari}
        onChange={(e) => setCari(e.target.value)}
        placeholder="Cari nama paket"
        prefixIcon={<Search size={16} />}
      />
      {tampil.length === 0 ? (
        <EmptyState
          variant="inline"
          message={paket.length === 0 ? 'Tidak ada paket yang bisa dipilih. Paket harus belum berangkat dan belum punya manasik.' : 'Tidak ada paket yang cocok.'}
        />
      ) : (
        <ul className="max-h-80 divide-y divide-neutral-100 overflow-y-auto rounded-xl border border-neutral-200">
          {tampil.map((p) => {
            const dipilih = terpilih.has(p.schedule_id);
            const alasan = dipilih ? null : alasanNonaktif(p);
            return (
              <li key={p.schedule_id}>
                <label className={`flex items-center gap-3 px-4 py-3 font-body ${alasan ? 'cursor-not-allowed text-neutral-400' : 'cursor-pointer hover:bg-neutral-50'}`}>
                  <input
                    type="checkbox"
                    className="h-4 w-4 shrink-0 rounded border-neutral-300 accent-primary-500"
                    checked={dipilih}
                    disabled={Boolean(alasan)}
                    onChange={() => ubah(p.schedule_id)}
                  />
                  <span className="min-w-0 flex-1">
                    <span className={`block truncate text-sm font-medium ${alasan ? '' : 'text-neutral-900'}`}>{p.nama}</span>
                    <span className="block text-xs text-neutral-500">
                      Berangkat {tanggalPendek(p.berangkat_tanggal)} · {p.jumlah_jamaah} jamaah
                      {tampilBrand && brandNama[p.brand_id] ? ` · ${brandNama[p.brand_id]}` : ''}
                    </span>
                  </span>
                  {alasan && <span className="shrink-0 text-xs text-neutral-500">{alasan}</span>}
                </label>
              </li>
            );
          })}
        </ul>
      )}
      <p className="text-xs text-neutral-500 font-body">{value.length} paket dipilih</p>
    </div>
  );
};

export default PilihPaketManasik;

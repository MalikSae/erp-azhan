import React, { useMemo, useState } from 'react';
import Input from 'shared/ui/Input';
import EmptyState from 'shared/ui/EmptyState';

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
export const tanggalPendek = (iso) => {
  if (!iso) return '-';
  const [y, m, d] = iso.slice(0, 10).split('-').map(Number);
  return `${d} ${BULAN[m - 1]} ${y}`;
};

// Daftar centang paket untuk manasik (dari /manasik/paket-tersedia, urut berangkat terdekat). Hanya paket yang bisa
// dipilih yang tampil: setelah satu paket dicentang, paket brand lain disembunyikan; paket yang berangkat pada atau
// sebelum tanggal manasik juga disembunyikan.
const PilihPaketManasik = ({ paket, value, onChange, tanggalManasik, brandNama = {}, tampilBrand = false }) => {
  const [cari, setCari] = useState('');
  const terpilih = useMemo(() => new Set(value), [value]);
  const brandTerkunci = useMemo(() => paket.find((p) => terpilih.has(p.schedule_id))?.brand_id ?? null, [paket, terpilih]);
  const tampil = useMemo(() => {
    const q = cari.trim().toLowerCase();
    return paket.filter((p) => {
      if (terpilih.has(p.schedule_id)) return !q || p.nama.toLowerCase().includes(q);
      if (brandTerkunci !== null && p.brand_id !== brandTerkunci) return false;
      if (tanggalManasik && p.berangkat_tanggal <= tanggalManasik) return false;
      return !q || p.nama.toLowerCase().includes(q);
    });
  }, [paket, cari, terpilih, brandTerkunci, tanggalManasik]);

  const ubah = (id) => onChange(terpilih.has(id) ? value.filter((x) => x !== id) : [...value, id]);

  return (
    <div className="space-y-3">
      <Input
        name="cari_paket"
        className="!mb-0"
        value={cari}
        onChange={(e) => setCari(e.target.value)}
        placeholder="Cari paket"
       
      />
      {tampil.length === 0 ? (
        <EmptyState variant="inline" message={cari.trim() ? 'Tidak ditemukan.' : 'Tidak ada paket.'} />
      ) : (
        <ul className="max-h-80 divide-y divide-neutral-100 overflow-y-auto rounded-xl border border-neutral-200">
          {tampil.map((p) => (
            <li key={p.schedule_id}>
              <label className="flex cursor-pointer items-center gap-3 px-4 py-3 font-body hover:bg-neutral-50">
                <input
                  type="checkbox"
                  className="h-4 w-4 shrink-0 rounded border-neutral-300 accent-primary-500"
                  checked={terpilih.has(p.schedule_id)}
                  onChange={() => ubah(p.schedule_id)}
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium text-neutral-900">{p.nama}</span>
                  <span className="block text-xs text-neutral-500">
                    {tanggalPendek(p.berangkat_tanggal)} · {p.jumlah_jamaah} jamaah
                    {tampilBrand && brandNama[p.brand_id] ? ` · ${brandNama[p.brand_id]}` : ''}
                  </span>
                </span>
              </label>
            </li>
          ))}
        </ul>
      )}
      {value.length > 0 && <p className="text-xs text-neutral-500 font-body">{value.length} dipilih</p>}
    </div>
  );
};

export default PilihPaketManasik;

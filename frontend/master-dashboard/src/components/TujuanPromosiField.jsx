import React from 'react';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Input from 'shared/ui/Input';

export const JENIS_TUJUAN = [
  { value: 'tidak_ada', label: 'Tanpa tujuan' },
  { value: 'paket', label: 'Paket' },
  { value: 'jenis_paket', label: 'Jenis paket' },
  { value: 'whatsapp', label: 'WhatsApp admin' },
  { value: 'tautan', label: 'Tautan web' },
];

// Filter tab Paket di app Hana.
export const JENIS_PAKET = [
  { value: 'umroh', label: 'Umroh' },
  { value: 'low-cost', label: 'Low Cost' },
  { value: 'private', label: 'Private' },
  { value: 'wisata', label: 'Wisata' },
  { value: 'haji', label: 'Haji' },
];

const valueOf = (v) => (v?.target ? v.target.value : v);

// label tujuan untuk daftar: "Paket: Umroh Reguler", "Jenis paket: Low Cost", dst.
export function labelTujuan(tujuan, nilai, paket = []) {
  switch (tujuan) {
    case 'paket':
      return `Paket: ${paket.find((p) => String(p.id) === String(nilai))?.jadwal_nama || nilai}`;
    case 'jenis_paket':
      return `Jenis paket: ${JENIS_PAKET.find((j) => j.value === nilai)?.label || nilai}`;
    case 'whatsapp':
      return 'WhatsApp admin';
    case 'tautan':
      return nilai || 'Tautan web';
    default:
      return '-';
  }
}

// Pilihan tujuan saat banner atau strip ditekan, beserta isian nilainya.
const TujuanPromosiField = ({ tujuan, nilai, onChange, paket }) => {
  const ubahJenis = (v) => onChange({ tujuan: valueOf(v), tujuan_nilai: '' });
  const ubahNilai = (v) => onChange({ tujuan, tujuan_nilai: valueOf(v) });
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <CustomDropdown label="Tujuan saat ditekan" value={tujuan} onChange={ubahJenis} options={JENIS_TUJUAN} className="!mb-0" />
      {tujuan === 'paket' && (
        <CustomDropdown
          label="Paket"
          value={nilai}
          onChange={ubahNilai}
          options={[{ value: '', label: 'Pilih paket' }, ...paket.map((p) => ({ value: String(p.id), label: p.jadwal_nama }))]}
          className="!mb-0"
        />
      )}
      {tujuan === 'jenis_paket' && (
        <CustomDropdown label="Jenis paket" value={nilai} onChange={ubahNilai} options={[{ value: '', label: 'Pilih jenis' }, ...JENIS_PAKET]} className="!mb-0" />
      )}
      {tujuan === 'whatsapp' && (
        <Input label="Pesan awal" name="pesan_wa" value={nilai} onChange={ubahNilai} maxLength={200} className="!mb-0" />
      )}
      {tujuan === 'tautan' && (
        <Input label="Tautan" name="tautan" value={nilai} onChange={ubahNilai} placeholder="https://" maxLength={500} className="!mb-0" />
      )}
    </div>
  );
};

export default TujuanPromosiField;

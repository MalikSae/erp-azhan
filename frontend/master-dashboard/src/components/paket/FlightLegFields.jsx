import { Plus, Trash2 } from 'lucide-react';
import Input from 'shared/ui/Input';
import AutocompleteInput from 'shared/ui/AutocompleteInput';
import Button from 'shared/ui/Button';
import { transitKosong } from '../../utils/paketTransit';

const TEKS = {
  berangkat: { judul: 'Berangkat', tanggal: 'Tanggal berangkat', jam: 'Jam berangkat', jamContoh: '09:00', nomorContoh: '822', durasiContoh: '2' },
  pulang: { judul: 'Pulang', tanggal: 'Tanggal pulang', jam: 'Jam pulang', jamContoh: '15:30', nomorContoh: '823', durasiContoh: '3' },
};

// Nomor penerbangan dari isian berawalan kode maskapai ("SV822" → "822"), maksimal 5 digit.
const nomorDariIsian = (isian, kode) => {
  const teks = isian.toUpperCase().replace(/\s+/g, '');
  const sisa = teks.startsWith(kode) ? teks.slice(kode.length) : teks;
  return sisa.replace(/\D/g, '').slice(0, 5);
};

// Satu arah penerbangan (berangkat/pulang) di form paket: tanggal, jam, kode penerbangan, bandara, transit.
//   leg            : 'berangkat' | 'pulang'
//   data           : formData paket (field `${leg}_*` dan `transit_items_${leg}`)
//   kodeMaskapai   : kode maskapai terpilih; bila ada dan kode tidak khusus, kode penerbangan = kode + nomor
//   nomor          : nomor penerbangan tanpa kode maskapai
//   onNomorChange  : (nomor) => void
//   children       : peringatan di bawah judul (mis. alasan ubah tanggal)
export default function FlightLegFields({
  leg,
  data,
  errors = {},
  onFieldChange,
  onTimeChange,
  kodeMaskapai,
  kodeKhusus,
  nomor,
  onNomorChange,
  airportOptions,
  onTransitChange,
  children,
}) {
  const teks = TEKS[leg];
  const f = (nama) => `${leg}_${nama}`;
  const ubah = (name, value) => onFieldChange({ target: { name, value } });
  const transit = data[`transit_items_${leg}`] || [transitKosong()];

  const ubahTransit = (idx, field, value) => {
    const baru = [...transit];
    baru[idx] = { ...(baru[idx] || transitKosong()), [field]: value };
    onTransitChange(baru);
  };
  const hapusTransit = (idx) => {
    if (transit.length <= 1) return;
    onTransitChange(transit.filter((_, i) => i !== idx));
  };

  const pakaiAwalan = !kodeKhusus && kodeMaskapai;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 text-neutral-600 text-sm font-semibold font-heading">
        <span>{teks.judul}</span>
      </div>

      {children}

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Input
          label={teks.tanggal}
          className="!mb-0"
          type="date"
          name={f('tanggal')}
          error={errors[f('tanggal')]}
          value={data[f('tanggal')]}
          onChange={onFieldChange}
          required
        />
        <Input
          label={teks.jam}
          className="!mb-0"
          type="text"
          name={f('jam')}
          error={errors[f('jam')]}
          value={data[f('jam')]}
          onChange={onTimeChange}
          placeholder={teks.jamContoh}
        />
        {pakaiAwalan ? (
          <Input
            label="Kode penerbangan"
            className="!mb-0"
            name={f('kode_penerbangan')}
            error={errors[f('kode_penerbangan')]}
            value={`${kodeMaskapai}${nomor}`}
            onChange={(e) => onNomorChange(nomorDariIsian(e.target.value, kodeMaskapai))}
            placeholder={`${kodeMaskapai}${teks.nomorContoh}`}
            inputMode="numeric"
            autoComplete="off"
          />
        ) : (
          <Input
            label="Kode penerbangan"
            className="!mb-0"
            name={f('kode_penerbangan')}
            error={errors[f('kode_penerbangan')]}
            value={data[f('kode_penerbangan')]}
            onChange={onFieldChange}
            placeholder={`SV ${teks.nomorContoh}`}
          />
        )}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <AutocompleteInput
          label="Bandara asal"
          className="!mb-0"
          name={f('bandara_asal')}
          value={data[f('bandara_asal')]}
          onChange={(e) => ubah(f('bandara_asal'), e.target.value)}
          onSelect={(opt) => ubah(f('bandara_asal'), opt?.value || '')}
          placeholder="Cari bandara"
          options={airportOptions}
        />
        <AutocompleteInput
          label="Bandara tujuan akhir"
          className="!mb-0"
          name={f('bandara_tujuan')}
          value={data[f('bandara_tujuan')]}
          onChange={(e) => ubah(f('bandara_tujuan'), e.target.value)}
          onSelect={(opt) => ubah(f('bandara_tujuan'), opt?.value || '')}
          placeholder="Cari bandara"
          options={airportOptions}
        />
      </div>

      {!data.is_direct_flight && (
        <div className="pt-2 space-y-3">
          <span className="block text-xs font-semibold text-neutral-700 font-heading">Transit</span>

          <div className="space-y-3">
            {transit.map((item, idx) => (
              <div key={idx} className="flex flex-col sm:flex-row items-start sm:items-end gap-3 pb-2.5 border-b border-neutral-100 last:border-b-0">
                <div className="flex-1 w-full grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div className="sm:col-span-2">
                    <AutocompleteInput
                      label={`Bandara transit ${transit.length > 1 ? `#${idx + 1}` : ''}`}
                      className="!mb-0"
                      name={`transit_${leg}_${idx}`}
                      value={item.bandara}
                      onChange={(e) => ubahTransit(idx, 'bandara', e.target.value)}
                      onSelect={(opt) => ubahTransit(idx, 'bandara', opt?.value || '')}
                      placeholder="Cari bandara"
                      options={airportOptions}
                    />
                  </div>
                  <Input
                    label="Durasi (jam)"
                    className="!mb-0"
                    value={item.durasi}
                    onChange={(e) => ubahTransit(idx, 'durasi', e.target.value.replace(/\D/g, '').slice(0, 3))}
                    inputMode="numeric"
                    placeholder={teks.durasiContoh}
                  />
                </div>

                {transit.length > 1 && (
                  <Button variant="danger-outline" className="shrink-0" onClick={() => hapusTransit(idx)} title="Hapus" aria-label="Hapus transit">
                    <Trash2 size={16} aria-hidden="true" />
                    <span className="sm:hidden">Hapus</span>
                  </Button>
                )}
              </div>
            ))}

            <Button variant="dashed" className="w-full" onClick={() => onTransitChange([...transit, transitKosong()])}>
              <Plus size={16} aria-hidden="true" />
              <span>Tambah transit</span>
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

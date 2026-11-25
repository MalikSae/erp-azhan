// Kolom schedules.transit_bandara: satu teks untuk transit berangkat dan pulang.
// Format simpan: "Berangkat: DXB, 2 Jam; DOH | Pulang: MCT, 3 Jam".
// Data lama tanpa label arah ("DXB, 2 Jam") dibaca sebagai transit berangkat.

export const transitKosong = () => ({ bandara: '', durasi: '' });

const parseDaftar = (str) => {
  if (!str || !str.trim()) return [transitKosong()];
  const items = str.split(';').map((seg) => {
    const trimmed = seg.trim();
    if (trimmed.includes(',')) {
      const p = trimmed.split(',');
      return { bandara: p[0].trim(), durasi: (p.slice(1).join(',').match(/\d+/) || [''])[0] };
    }
    return { bandara: trimmed, durasi: '' };
  }).filter((i) => i.bandara || i.durasi);
  return items.length > 0 ? items : [transitKosong()];
};

// { berangkat: [{ bandara, durasi }], pulang: [...], tanpaArah }. Daftar kosong berisi satu baris kosong.
export const parseTransit = (raw) => {
  const str = typeof raw === 'string' ? raw.trim() : '';
  if (!str) return { berangkat: [transitKosong()], pulang: [transitKosong()], tanpaArah: false };

  const lower = str.toLowerCase();
  if (!lower.includes('berangkat:') && !lower.includes('pulang:')) {
    return { berangkat: parseDaftar(str), pulang: [transitKosong()], tanpaArah: true };
  }

  let rawBerangkat = '';
  let rawPulang = '';
  str.split('|').forEach((part) => {
    const trimmed = part.trim();
    const l = trimmed.toLowerCase();
    if (l.startsWith('berangkat:')) rawBerangkat = trimmed.substring(trimmed.indexOf(':') + 1).trim();
    else if (l.startsWith('pulang:')) rawPulang = trimmed.substring(trimmed.indexOf(':') + 1).trim();
  });
  return { berangkat: parseDaftar(rawBerangkat), pulang: parseDaftar(rawPulang), tanpaArah: false };
};

const bersihkan = (items) => (items || [])
  .map((t) => ({ bandara: t.bandara?.trim() || '', durasi: String(t.durasi ?? '').trim() }))
  .filter((t) => t.bandara || t.durasi);

const tulisDaftar = (items) => items.map((t) => (t.durasi ? `${t.bandara}, ${t.durasi} Jam` : t.bandara)).join('; ');

export const buildTransit = ({ berangkat, pulang }) => {
  const b = bersihkan(berangkat);
  const p = bersihkan(pulang);
  const parts = [];
  if (b.length > 0) parts.push(`Berangkat: ${tulisDaftar(b)}`);
  if (p.length > 0) parts.push(`Pulang: ${tulisDaftar(p)}`);
  return parts.join(' | ');
};

const kodeBandara = (items) => {
  const bandara = items?.[0]?.bandara;
  if (!bandara) return null;
  const cleaned = bandara.split('(')[0].trim().toUpperCase();
  const match = cleaned.match(/[A-Z]{3}/);
  return match ? match[0] : (cleaned || null);
};

// Kode bandara transit pertama per arah untuk ringkasan rute. Data lama tanpa label berlaku dua arah.
export const transitUtama = (raw) => {
  const t = parseTransit(raw);
  const berangkat = kodeBandara(t.berangkat);
  return { berangkat, pulang: t.tanpaArah ? berangkat : kodeBandara(t.pulang) };
};

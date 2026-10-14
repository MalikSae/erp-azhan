// Cetak / simpan detail paket sebagai PDF (lembar informasi paket).
// Semua nilai sudah diformat oleh pemanggil; modul ini hanya menyusun dokumen.

import {
  escapeHtml,
  richToHtml,
  normalizeHex,
  hexToRgba,
  readableOn,
  printHtmlDocument,
} from './printItinerary';

const row = (label, value) => (value
  ? `<div class="kv"><span class="k">${escapeHtml(label)}</span><span class="v">${escapeHtml(value)}</span></div>`
  : '');

const flightCard = (heading, leg) => `
  <div class="card">
    <div class="card-h">${escapeHtml(heading)}</div>
    ${row('Tanggal', leg.tanggal)}
    ${row('Jam', leg.jam)}
    ${row('Kode penerbangan', leg.kode)}
    ${row('Rute', leg.rute)}
  </div>`;

const hotelCard = (heading, hotel) => `
  <div class="card">
    <div class="card-h">${escapeHtml(heading)}</div>
    <div class="hotel-name">${escapeHtml(hotel.nama || '-')}</div>
    ${row('Bintang', hotel.bintang)}
    ${row('Jarak ke masjid', hotel.jarak)}
  </div>`;

const list = (items, mark) => (items && items.length
  ? `<ul class="check">${items.map(item => `<li data-mark="${mark}">${richToHtml(item)}</li>`).join('')}</ul>`
  : '<p class="muted">Tidak ada item tercatat.</p>');

const buildDocument = (data) => {
  const accent = normalizeHex(data.accentColor);
  const tint = hexToRgba(accent, 0.08);
  const onAccent = readableOn(accent);

  const brand = data.logoUrl
    ? `<img id="brand-logo" src="${escapeHtml(data.logoUrl)}" alt="${escapeHtml(data.brandName || 'Logo')}">`
    : (data.brandName ? `<span class="brand-name">${escapeHtml(data.brandName)}</span>` : '');

  const prices = (data.prices || [])
    .map(p => `<tr><td>${escapeHtml(p.label)}</td><td class="num">${escapeHtml(p.value)}</td></tr>`)
    .join('');

  return `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<title>${escapeHtml(data.title)}</title>
<style>
  @page { size: A4; margin: 12mm 12mm 14mm; }
  * { box-sizing: border-box; }
  html { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  body { font-family: 'Segoe UI', 'Inter', Calibri, Arial, sans-serif; color: #1f2937; font-size: 11.5px; line-height: 1.55; margin: 0; }
  .topbar { height: 6px; border-radius: 3px; background: ${accent}; margin-bottom: 14px; }
  .head { display: flex; align-items: center; gap: 16px; margin-bottom: 14px; }
  .brand { min-width: 70px; max-width: 170px; }
  .brand img { display: block; max-height: 54px; max-width: 170px; object-fit: contain; }
  .brand-name { font-weight: 800; font-size: 15px; color: ${accent}; }
  .titles { flex: 1; text-align: right; }
  h1 { font-size: 18px; margin: 0; line-height: 1.25; color: #111827; }
  .summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-bottom: 16px; }
  .sum { border-radius: 10px; background: ${tint}; border: 1px solid ${hexToRgba(accent, 0.25)}; padding: 9px 12px; }
  .sum .k { display: block; font-size: 9px; text-transform: uppercase; letter-spacing: .8px; color: #6b7280; font-weight: 700; }
  .sum .v { display: block; font-size: 13.5px; font-weight: 800; color: #111827; margin-top: 2px; }
  .sum .s { display: block; font-size: 10.5px; color: #4b5563; }
  h2 { font-size: 13px; margin: 18px 0 8px; padding-left: 9px; border-left: 4px solid ${accent}; color: #111827; page-break-after: avoid; }
  .grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
  .card { border: 1px solid #e5e7eb; border-radius: 10px; padding: 9px 12px; page-break-inside: avoid; break-inside: avoid; }
  .card-h { font-size: 10px; text-transform: uppercase; letter-spacing: .8px; color: ${accent}; font-weight: 800; margin-bottom: 4px; }
  .hotel-name { font-weight: 700; font-size: 12.5px; margin-bottom: 3px; }
  .kv { display: flex; justify-content: space-between; gap: 10px; padding: 2px 0; border-top: 1px dashed #eceff3; }
  .kv:first-of-type { border-top: 0; }
  .kv .k { color: #6b7280; }
  .kv .v { font-weight: 600; color: #111827; text-align: right; }
  table.price { width: 100%; border-collapse: collapse; }
  table.price td { padding: 7px 10px; border-bottom: 1px solid #e5e7eb; }
  table.price tr:last-child td { border-bottom: 0; }
  table.price td.num { text-align: right; font-weight: 800; font-size: 13px; }
  .price-wrap { border: 1px solid #e5e7eb; border-radius: 10px; overflow: hidden; page-break-inside: avoid; }
  .note { margin: 6px 2px 0; color: #6b7280; font-size: 10.5px; }
  ul.check { list-style: none; padding: 0; margin: 0; }
  ul.check li { position: relative; padding-left: 16px; margin: 3px 0; }
  ul.check li::before { content: attr(data-mark); position: absolute; left: 0; font-weight: 800; color: ${accent}; }
  ul.check li[data-mark="✕"]::before { color: #9ca3af; }
  .muted { color: #9ca3af; margin: 0; }
  strong { font-weight: 700; color: #111827; }
  .foot { margin-top: 18px; padding-top: 8px; border-top: 1px solid #e5e7eb; color: #9ca3af; font-size: 9.5px; display: flex; justify-content: space-between; }
</style>
</head>
<body>
  <div class="topbar"></div>
  <div class="head">
    <div class="brand">${brand}</div>
    <div class="titles">
      <h1>${escapeHtml(data.title)}</h1>
    </div>
  </div>

  <div class="summary">
    <div class="sum"><span class="k">Keberangkatan</span><span class="v">${escapeHtml(data.berangkat || '-')}</span><span class="s">${escapeHtml(data.durasi || '')}</span></div>
    <div class="sum"><span class="k">Maskapai</span><span class="v">${escapeHtml(data.maskapai || '-')}</span><span class="s">${escapeHtml(data.jenisPenerbangan || '')}</span></div>
    <div class="sum"><span class="k">Harga mulai</span><span class="v">${escapeHtml(data.hargaMulai || '-')}</span><span class="s">per orang</span></div>
  </div>

  <h2>Penerbangan</h2>
  <div class="grid2">
    ${flightCard('Berangkat', data.flights.berangkat)}
    ${flightCard('Pulang', data.flights.pulang)}
  </div>

  <h2>Akomodasi Hotel</h2>
  <div class="grid2">
    ${hotelCard('Makkah', data.hotels.makkah)}
    ${hotelCard('Madinah', data.hotels.madinah)}
  </div>
  ${(data.hotels.transit || []).length
    ? `<p class="note">Hotel transit: ${escapeHtml(data.hotels.transit.join(', '))}</p>` : ''}

  <h2>Harga Paket</h2>
  <div class="price-wrap"><table class="price">${prices}</table></div>
  ${data.priceNote ? `<p class="note">${escapeHtml(data.priceNote)}</p>` : ''}

  <h2>Fasilitas Paket</h2>
  <div class="grid2">
    <div class="card"><div class="card-h">Termasuk</div>${list(data.include, '✓')}</div>
    <div class="card"><div class="card-h">Tidak termasuk</div>${list(data.exclude, '✕')}</div>
  </div>

  <div class="foot"><span>${escapeHtml(data.brandName || '')}</span><span>Dicetak ${escapeHtml(data.printedAt || '')}</span></div>
</body>
</html>`;
};

export const printPackage = (data) => {
  printHtmlDocument(buildDocument(data));
};

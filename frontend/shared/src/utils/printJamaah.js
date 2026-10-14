// Cetak / simpan profil jamaah sebagai PDF. Semua nilai sudah diformat oleh pemanggil;
// modul ini hanya menyusun dokumen.

import { escapeHtml, normalizeHex, hexToRgba, printHtmlDocument } from './printDocument';

const kv = (label, value) => `
  <div class="kv">
    <span class="k">${escapeHtml(label)}</span>
    <span class="v">${escapeHtml(value || '-')}</span>
  </div>`;

const section = (title, rows) => `
  <h2>${escapeHtml(title)}</h2>
  <div class="card"><div class="grid2">${rows.map(([l, v]) => kv(l, v)).join('')}</div></div>`;

const buildDocument = (data) => {
  const accent = normalizeHex(data.accentColor);
  const tint = hexToRgba(accent, 0.08);

  const brand = data.logoUrl
    ? `<img id="brand-logo" src="${escapeHtml(data.logoUrl)}" alt="${escapeHtml(data.brandName || 'Logo')}">`
    : (data.brandName ? `<span class="brand-name">${escapeHtml(data.brandName)}</span>` : '');

  return `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<title>${escapeHtml(data.nama)} - Data Jamaah</title>
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
  .doc-label { font-size: 9px; text-transform: uppercase; letter-spacing: 1px; color: #6b7280; font-weight: 700; }
  h1 { font-size: 18px; margin: 2px 0 0; line-height: 1.25; color: #111827; }
  .summary { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; margin-bottom: 16px; }
  .sum { border-radius: 10px; background: ${tint}; border: 1px solid ${hexToRgba(accent, 0.25)}; padding: 9px 12px; }
  .sum .k { display: block; font-size: 9px; text-transform: uppercase; letter-spacing: .8px; color: #6b7280; font-weight: 700; }
  .sum .v { display: block; font-size: 13px; font-weight: 800; color: #111827; margin-top: 2px; word-break: break-word; }
  h2 { font-size: 13px; margin: 16px 0 8px; padding-left: 9px; border-left: 4px solid ${accent}; color: #111827; page-break-after: avoid; }
  .card { border: 1px solid #e5e7eb; border-radius: 10px; padding: 8px 12px; page-break-inside: avoid; break-inside: avoid; }
  .grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0 22px; }
  .kv { padding: 4px 0; border-top: 1px dashed #eceff3; }
  .grid2 .kv:nth-child(-n+2) { border-top: 0; }
  .kv .k { display: block; font-size: 9.5px; color: #6b7280; }
  .kv .v { display: block; font-weight: 600; color: #111827; word-break: break-word; }
  .foot { margin-top: 18px; padding-top: 8px; border-top: 1px solid #e5e7eb; color: #9ca3af; font-size: 9.5px; display: flex; justify-content: space-between; }
</style>
</head>
<body>
  <div class="topbar"></div>
  <div class="head">
    <div class="brand">${brand}</div>
    <div class="titles">
      <div class="doc-label">Data Jamaah</div>
      <h1>${escapeHtml(data.nama)}</h1>
    </div>
  </div>

  <div class="summary">
    <div class="sum"><span class="k">ID Jamaah</span><span class="v">${escapeHtml(data.idJamaah || '-')}</span></div>
    <div class="sum"><span class="k">No. HP</span><span class="v">${escapeHtml(data.noHp || '-')}</span></div>
  </div>

  ${section('Identitas Pribadi', data.identitas || [])}
  ${section('Pendidikan & Pekerjaan', data.pekerjaan || [])}
  ${section('Dokumen Paspor', data.paspor || [])}
  ${section('Kontak Darurat', data.darurat || [])}

  <div class="foot"><span>${escapeHtml(data.brandName || '')}</span><span>Dicetak ${escapeHtml(data.printedAt || '')}</span></div>
</body>
</html>`;
};

export const printJamaah = (data) => {
  printHtmlDocument(buildDocument(data));
};

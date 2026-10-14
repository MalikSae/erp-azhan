// Cetak / simpan itinerary sebagai PDF.
// Membuat dokumen cetak di iframe tersembunyi lalu memanggil dialog cetak browser;
// pilih "Simpan sebagai PDF" untuk mengunduh. Warna aksen mengikuti warna brand.

const DEFAULT_ACCENT = '#F26522';

export const escapeHtml = (value) =>
  String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');

// **tebal** -> <strong>; sisanya di-escape.
export const richToHtml = (text) =>
  String(text ?? '')
    .split('**')
    .map((part, index) => (index % 2 === 1 ? `<strong>${escapeHtml(part)}</strong>` : escapeHtml(part)))
    .join('');

// Terima #RGB / #RRGGBB; selain itu pakai warna bawaan.
export const normalizeHex = (value) => {
  const raw = String(value || '').trim();
  const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(raw);
  if (!m) return DEFAULT_ACCENT;
  const hex = m[1].length === 3 ? m[1].split('').map(c => c + c).join('') : m[1];
  return `#${hex.toUpperCase()}`;
};

export const hexToRgba = (hex, alpha) => {
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
};

// Teks di atas warna aksen: putih atau gelap, mana yang lebih terbaca.
export const readableOn = (hex) => {
  const n = parseInt(hex.slice(1), 16);
  const luminance = (0.299 * ((n >> 16) & 255) + 0.587 * ((n >> 8) & 255) + 0.114 * (n & 255)) / 255;
  return luminance > 0.65 ? '#111827' : '#FFFFFF';
};

const buildDocument = ({ title, subtitle, days, footerItems, logoUrl, brandName, accentColor }) => {
  const accent = normalizeHex(accentColor);
  const tint = hexToRgba(accent, 0.08);
  const onAccent = readableOn(accent);

  const dayBlocks = days
    .map((day, index) => {
      const number = day.day_number || index + 1;
      const place = day.location || day.title || '';
      const points = (day.activities || [])
        .map(act => `<li>${richToHtml([act.time, act.text].filter(Boolean).join(' '))}</li>`)
        .join('');
      return `
      <section class="day">
        <div class="badge"><span class="lbl">HARI</span><span class="num">${number}</span></div>
        <div class="body">
          ${place ? `<h2>${escapeHtml(place)}</h2>` : ''}
          <ul>${points || '<li>-</li>'}</ul>
        </div>
      </section>`;
    })
    .join('');

  const footer = footerItems
    .filter(item => item && item.value)
    .map(item => `<div class="f-item"><span class="f-lbl">${escapeHtml(item.label)}</span><span class="f-val">${escapeHtml(item.value)}</span></div>`)
    .join('');

  const brand = logoUrl
    ? `<img id="brand-logo" src="${escapeHtml(logoUrl)}" alt="${escapeHtml(brandName || 'Logo')}">`
    : (brandName ? `<span class="brand-name">${escapeHtml(brandName)}</span>` : '');

  return `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<title>${escapeHtml(title)}</title>
<style>
  @page { size: A4; margin: 12mm 12mm 14mm; }
  * { box-sizing: border-box; }
  html { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  body { font-family: 'Segoe UI', 'Inter', Calibri, Arial, sans-serif; color: #1f2937; font-size: 11.5px; line-height: 1.55; margin: 0; }
  .topbar { height: 6px; border-radius: 3px; background: ${accent}; margin-bottom: 14px; }
  .head { display: flex; align-items: center; gap: 16px; margin-bottom: 16px; }
  .brand { min-width: 70px; max-width: 170px; }
  .brand img { display: block; max-height: 54px; max-width: 170px; object-fit: contain; }
  .brand-name { font-weight: 800; font-size: 15px; color: ${accent}; }
  .titles { flex: 1; text-align: right; }
  h1 { font-size: 17px; margin: 0; line-height: 1.25; letter-spacing: .2px; color: #111827; }
  .sub { margin: 3px 0 0; color: #6b7280; font-size: 11px; }
  .day { display: flex; gap: 14px; padding: 12px 0; border-top: 1px solid #e5e7eb; page-break-inside: avoid; break-inside: avoid; }
  .day:first-of-type { border-top: 0; }
  .badge { flex: 0 0 52px; height: 52px; border-radius: 12px; background: ${accent}; color: ${onAccent};
           display: flex; flex-direction: column; align-items: center; justify-content: center; line-height: 1; }
  .badge .lbl { font-size: 8.5px; letter-spacing: 1.2px; font-weight: 700; opacity: .9; }
  .badge .num { font-size: 21px; font-weight: 800; margin-top: 3px; }
  .body { flex: 1; min-width: 0; }
  .body h2 { margin: 2px 0 5px; font-size: 13px; color: ${accent}; font-weight: 700; }
  .body ul { margin: 0; padding-left: 15px; }
  .body li { margin: 3px 0; padding-left: 2px; }
  .body li::marker { color: ${accent}; }
  .footer { margin-top: 14px; padding: 10px 14px; border-radius: 10px; background: ${tint}; border: 1px solid ${hexToRgba(accent, 0.25)};
            display: flex; flex-wrap: wrap; gap: 6px 26px; page-break-inside: avoid; break-inside: avoid; }
  .f-item { display: flex; flex-direction: column; }
  .f-lbl { font-size: 9px; text-transform: uppercase; letter-spacing: .8px; color: #6b7280; font-weight: 700; }
  .f-val { font-size: 11.5px; font-weight: 700; color: #111827; }
  strong { font-weight: 700; color: #111827; }
</style>
</head>
<body>
  <div class="topbar"></div>
  <div class="head">
    <div class="brand">${brand}</div>
    <div class="titles">
      <h1>${escapeHtml(title)}</h1>
      ${subtitle ? `<p class="sub">${escapeHtml(subtitle)}</p>` : ''}
    </div>
  </div>
  ${dayBlocks}
  ${footer ? `<div class="footer">${footer}</div>` : ''}
</body>
</html>`;
};

// Menulis dokumen HTML ke iframe tersembunyi lalu membuka dialog cetak (setelah logo termuat).
export const printHtmlDocument = (html) => {
  const iframe = document.createElement('iframe');
  iframe.setAttribute('aria-hidden', 'true');
  iframe.style.cssText = 'position:fixed;right:0;bottom:0;width:0;height:0;border:0;';
  document.body.appendChild(iframe);

  const doc = iframe.contentWindow.document;
  doc.open();
  doc.write(html);
  doc.close();

  iframe.contentWindow.onafterprint = () => setTimeout(() => iframe.remove(), 1000);

  const doPrint = () => {
    iframe.contentWindow.focus();
    iframe.contentWindow.print();
    // Cadangan jika onafterprint tidak terpanggil.
    setTimeout(() => iframe.remove(), 60000);
  };

  // Tunggu semua gambar (logo brand, logo maskapai) selesai dimuat atau gagal sebelum cetak.
  const pending = Array.from(doc.images).filter(img => !img.complete);
  if (pending.length) {
    let done = false;
    let remaining = pending.length;
    const go = () => { if (!done) { done = true; setTimeout(doPrint, 150); } };
    const settle = () => { remaining -= 1; if (remaining <= 0) go(); };
    pending.forEach(img => {
      img.addEventListener('load', settle);
      img.addEventListener('error', settle);
    });
    setTimeout(go, 4000);
  } else {
    setTimeout(doPrint, 250);
  }
};

export const printItinerary = ({
  title,
  subtitle = '',
  days = [],
  footerItems = [],
  logoUrl = '',
  brandName = '',
  accentColor = '',
}) => {
  printHtmlDocument(buildDocument({ title, subtitle, days, footerItems, logoUrl, brandName, accentColor }));
};

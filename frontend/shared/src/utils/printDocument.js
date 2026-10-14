// Util cetak dokumen HTML ke PDF (lewat dialog cetak browser; pilih "Simpan sebagai PDF").
// Dokumen dibuat di iframe tersembunyi; warna aksen mengikuti warna brand.

const DEFAULT_ACCENT = '#F26522';

export const escapeHtml = (value) =>
  String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');

// Terima #RGB / #RRGGBB; selain itu pakai warna bawaan.
export const normalizeHex = (value) => {
  const raw = String(value || '').trim();
  const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(raw);
  if (!m) return DEFAULT_ACCENT;
  const hex = m[1].length === 3 ? m[1].split('').map((c) => c + c).join('') : m[1];
  return `#${hex.toUpperCase()}`;
};

export const hexToRgba = (hex, alpha) => {
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
};

// Menulis dokumen HTML ke iframe tersembunyi lalu membuka dialog cetak (setelah gambar termuat).
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
    setTimeout(() => iframe.remove(), 60000);
  };

  const pending = Array.from(doc.images).filter((img) => !img.complete);
  if (pending.length) {
    let done = false;
    let remaining = pending.length;
    const go = () => { if (!done) { done = true; setTimeout(doPrint, 150); } };
    const settle = () => { remaining -= 1; if (remaining <= 0) go(); };
    pending.forEach((img) => {
      img.addEventListener('load', settle);
      img.addEventListener('error', settle);
    });
    setTimeout(go, 4000);
  } else {
    setTimeout(doPrint, 250);
  }
};

// Cetak / simpan invoice booking sebagai PDF. Semua nilai uang & tanggal sudah diformat
// oleh pemanggil; modul ini hanya menyusun dokumen (kop surat, tagihan, instruksi bayar).

import { escapeHtml, printHtmlDocument } from './printDocument';

const STATUS_COLOR = {
  lunas: '#15803d',
  sebagian: '#b45309',
  belum: '#b91c1c',
  batal: '#6b7280',
};

export const buildInvoiceDocument = (data) => {
  const statusColor = STATUS_COLOR[data.statusTone] || STATUS_COLOR.belum;

  const logo = data.logoUrl
    ? `<img id="brand-logo" src="${escapeHtml(data.logoUrl)}" alt="${escapeHtml(data.brandName || 'Logo')}">`
    : '';
  const kop = data.kop || {};
  const kopLines = [
    kop.alamat,
    [kop.kota, kop.provinsi].filter(Boolean).join(', '),
    [kop.telepon && `Telp. ${kop.telepon}`, kop.whatsapp && `WA ${kop.whatsapp}`].filter(Boolean).join('  •  '),
    [kop.email, kop.website].filter(Boolean).join('  •  '),
    kop.legalitas,
  ].filter(Boolean);

  const paxRows = (data.pax || []).map((p, i) => `
      <tr>
        <td class="c">${i + 1}</td>
        <td><b>${escapeHtml(p.nama)}</b><div class="sub">${escapeHtml(p.tipe)}</div></td>
        <td class="r">${escapeHtml(p.harga)}</td>
      </tr>`).join('');

  const diskonRows = (data.diskon || []).map((d) => `
      <tr><td>Diskon ${escapeHtml(d.nama)}</td><td class="r pos">- ${escapeHtml(d.nominal)}</td></tr>`).join('');

  const paymentRows = (data.pembayaran || []).map((p) => `
      <tr><td>${escapeHtml(p.tanggal)}</td><td>${escapeHtml(p.metode)}</td><td class="r">${escapeHtml(p.jumlah)}</td></tr>`).join('');

  const bankRows = (data.rekening || []).map((r) => `
      <tr>
        <td><b>${escapeHtml(r.bank)}</b>${r.catatan ? `<div class="sub">${escapeHtml(r.catatan)}</div>` : ''}</td>
        <td class="mono">${escapeHtml(r.nomor)}</td>
        <td>${escapeHtml(r.pemilik)}</td>
      </tr>`).join('');

  // Pelunasan dan pembatalan satu daftar; minimal DP masuk ke instruksi pembayaran.
  const sk = data.sk || {};
  const skLines = [sk.pelunasan, ...(sk.pembatalan || [])].filter(Boolean);

  return `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<title>${escapeHtml(data.kode)} - Invoice</title>
<style>
  @page { size: A4; margin: 12mm 15mm 12mm; }
  * { box-sizing: border-box; }
  html { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  body { font-family: 'Segoe UI', 'Inter', Calibri, Arial, sans-serif; color: #374151; font-size: 11px; line-height: 1.5; margin: 0; min-height: 270mm; display: flex; flex-direction: column; }
  .bottom { margin-top: auto; padding-top: 20px; }
  b { color: #111827; font-weight: 600; }
  table { width: 100%; border-collapse: collapse; }
  tr { page-break-inside: avoid; break-inside: avoid; }
  .sub { font-size: 9.5px; color: #9ca3af; font-weight: 400; }
  .r { text-align: right; white-space: nowrap; }
  .c { text-align: center; width: 34px; }
  .pos { color: #15803d; }
  .mono { font-family: Consolas, 'Courier New', monospace; font-size: 12px; color: #111827; letter-spacing: .4px; }

  /* Kop surat: identitas brand (kiri) + judul dokumen (kanan) */
  .letterhead { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding-bottom: 14px; border-bottom: 2px solid #111827; }
  .id { display: flex; align-items: center; gap: 14px; }
  .brand img { display: block; max-height: 56px; max-width: 96px; object-fit: contain; }
  .brand-name { font-weight: 700; font-size: 15px; color: #111827; line-height: 1.25; }
  .kop-line { font-size: 9.5px; color: #6b7280; line-height: 1.45; }
  .doc { text-align: right; flex-shrink: 0; }
  .doc-title { font-size: 24px; font-weight: 700; letter-spacing: 2px; color: #111827; line-height: 1.1; }
  .doc-no { font-size: 11.5px; color: #374151; margin-top: 2px; }
  .status { display: inline-block; margin-top: 7px; padding: 2px 10px; border: 1px solid ${statusColor}; border-radius: 4px; font-size: 9px; font-weight: 700; letter-spacing: 1px; text-transform: uppercase; color: ${statusColor}; }

  /* Strip info */
  .info { display: grid; grid-template-columns: 1.25fr 1.5fr 1fr 1fr; gap: 18px; background: #f8f9fb; border: 1px solid #eceff3; border-radius: 6px; padding: 12px 14px; margin: 16px 0 18px; }
  .lbl { font-size: 8.5px; text-transform: uppercase; letter-spacing: 1px; color: #9ca3af; margin-bottom: 2px; }
  .val { font-size: 11.5px; font-weight: 600; color: #111827; line-height: 1.35; }
  .val-sub { font-size: 10px; color: #6b7280; font-weight: 400; }

  /* Item */
  table.items thead th { background: #f3f4f6; text-align: left; font-size: 8.5px; text-transform: uppercase; letter-spacing: 1px; color: #6b7280; font-weight: 600; padding: 7px 10px; }
  table.items thead th.r { text-align: right; }
  table.items thead th.c { text-align: center; }
  table.items td { padding: 9px 10px; border-bottom: 1px solid #eceff3; vertical-align: top; }

  /* Total */
  .totals-wrap { display: flex; justify-content: space-between; gap: 24px; margin-top: 14px; margin-bottom: 36px; align-items: flex-start; }
  .note-box { flex: 1; }
  table.totals { width: 268px; }
  table.totals td { padding: 4px 10px; }
  table.totals td:first-child { color: #6b7280; }
  table.totals tr.grand td { border-top: 1px solid #d1d5db; padding-top: 8px; font-weight: 700; font-size: 12.5px; color: #111827; }
  table.totals tr.sisa td { background: #111827; color: #fff; font-weight: 700; font-size: 12.5px; padding-top: 7px; padding-bottom: 7px; }

  h2 { font-size: 9px; text-transform: uppercase; letter-spacing: 1.2px; margin: 20px 0 7px; color: #111827; font-weight: 700; page-break-after: avoid; }
  table.plain th { text-align: left; font-size: 8.5px; text-transform: uppercase; letter-spacing: .8px; color: #9ca3af; font-weight: 600; padding: 5px 10px; border-bottom: 1px solid #d1d5db; }
  table.plain th.r { text-align: right; }
  table.plain td { padding: 8px 10px; border-bottom: 1px solid #eceff3; vertical-align: middle; }
  .pay-box { font-size: 10.5px; color: #4b5563; margin-bottom: 8px; page-break-inside: avoid; break-inside: avoid; }
  ol.terms { margin: 0; padding-left: 18px; font-size: 10px; color: #6b7280; }
  ol.terms li { margin-bottom: 2px; }

  .thanks { margin-top: 20px; text-align: center; font-size: 10px; color: #6b7280; }
  .foot { margin-top: 10px; padding-top: 7px; border-top: 1px solid #e5e7eb; color: #9ca3af; font-size: 8.5px; display: flex; justify-content: space-between; }
</style>
</head>
<body>
  <div class="letterhead">
    <div class="id">
      <div class="brand">${logo}</div>
      <div>
        <div class="brand-name">${escapeHtml(data.brandName || '')}</div>
        ${kopLines.map((l) => `<div class="kop-line">${escapeHtml(l)}</div>`).join('')}
      </div>
    </div>
    <div class="doc">
      <div class="doc-title">INVOICE</div>
      <div class="doc-no">${escapeHtml(data.kode)}</div>
      <span class="status">${escapeHtml(data.statusLabel || '')}</span>
    </div>
  </div>

  <div class="info">
    <div>
      <div class="lbl">Ditagihkan Kepada</div>
      <div class="val">${escapeHtml(data.kontak || '-')}</div>
      ${data.kontakHp ? `<div class="val-sub">${escapeHtml(data.kontakHp)}</div>` : ''}
    </div>
    <div>
      <div class="lbl">Paket Perjalanan</div>
      <div class="val">${escapeHtml(data.paket || '-')}</div>
    </div>
    <div>
      <div class="lbl">Tanggal Booking</div>
      <div class="val">${escapeHtml(data.tanggalBooking || '-')}</div>
    </div>
    <div>
      <div class="lbl">Keberangkatan</div>
      <div class="val">${escapeHtml(data.berangkat || '-')}</div>
    </div>
  </div>

  <table class="items">
    <thead><tr><th class="c">No</th><th>Jamaah</th><th class="r">Harga</th></tr></thead>
    <tbody>${paxRows || '<tr><td colspan="3">-</td></tr>'}</tbody>
  </table>

  <div class="totals-wrap">
    <div class="note-box">
      ${paymentRows ? `
      <h2 style="margin-top:0">Pembayaran Diterima</h2>
      <table class="plain">
        <thead><tr><th>Tanggal</th><th>Metode</th><th class="r">Jumlah</th></tr></thead>
        <tbody>${paymentRows}</tbody>
      </table>` : ''}
    </div>
    <table class="totals">
      <tr><td>Subtotal</td><td class="r">${escapeHtml(data.hargaDasar)}</td></tr>
      ${diskonRows}
      <tr class="grand"><td>Total Tagihan</td><td class="r">${escapeHtml(data.totalTagihan)}</td></tr>
      <tr><td>Sudah Dibayar</td><td class="r pos">${escapeHtml(data.totalDibayar)}</td></tr>
      <tr class="sisa"><td>Sisa Tagihan</td><td class="r">${escapeHtml(data.sisaTagihan)}</td></tr>
    </table>
  </div>

  ${bankRows ? `
  <h2>Instruksi Pembayaran</h2>
  <div class="pay-box">Pembayaran dilakukan melalui transfer ke rekening berikut. Cantumkan kode booking <b>${escapeHtml(data.kode)}</b> pada berita transfer dan kirim bukti pembayaran kepada admin.${sk.minimalDp ? ` Pembayaran awal (DP) minimal <b>${escapeHtml(sk.minimalDp)}</b> per jamaah.` : ''}</div>
  <table class="plain">
    <thead><tr><th>Bank</th><th>No. Rekening</th><th>Atas Nama</th></tr></thead>
    <tbody>${bankRows}</tbody>
  </table>` : ''}

  <div class="bottom">
  ${skLines.length ? `
  <h2>Syarat &amp; Ketentuan</h2>
  <ol class="terms">${skLines.map((t) => `<li>${escapeHtml(t)}</li>`).join('')}</ol>` : ''}

  <div class="thanks">Terima kasih atas kepercayaan Anda. Semoga ibadah Anda lancar dan mabrur.</div>
  <div class="foot"><span>${escapeHtml(data.brandName || '')} — ${escapeHtml(data.kode)}</span><span>Dicetak ${escapeHtml(data.printedAt || '')}</span></div>
  </div>
</body>
</html>`;
};

export const printInvoice = (data) => {
  printHtmlDocument(buildInvoiceDocument(data));
};

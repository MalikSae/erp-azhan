import client from './client';
import { compressImage } from '../utils/imageCompressor';

export async function uploadMedia(file, category) {
  // Auto-compress client-side jika file berupa foto/gambar dari HP/kamera
  const processedFile = await compressImage(file);

  const formData = new FormData();
  formData.append('file', processedFile);
  formData.append('category', category);
  
  const response = await client.post('/api/admin/media/upload', formData);
  return response.data.url;
}

export async function uploadMediaWithOptions(file, category, options = {}) {
  const processedFile = await compressImage(file);

  const formData = new FormData();
  formData.append('file', processedFile);
  formData.append('category', category);
  
  if (options.maxWidth) {
    formData.append('max_width', options.maxWidth);
  }
  if (options.generateThumbnail) {
    formData.append('generate_thumbnail', options.generateThumbnail ? 'true' : 'false');
  }
  
  const response = await client.post('/api/admin/media/upload', formData);
  return response.data;
}

// Tampilkan berkas media terlindungi (bukti transfer, paspor, KTP, dst.) lewat klien API
// berautentikasi, di lapisan pratinjau dalam halaman. Tidak memakai window.open karena pop-up
// sering diblokir (browser maupun panel tertanam), sehingga tab kosong / tidak terjadi apa-apa.
// Berkas tidak pernah diakses lewat URL publik.
export async function openProtectedMedia(url, title_) {
  const normalizedUrl = String(url || '').replace(
    /^\/uploads\/(dokumen-jamaah|payment-proofs)\/(.+)$/,
    '/api/admin/media/$1/$2',
  );
  const response = await client.get(normalizedUrl, { responseType: 'blob' });
  const blob = response.data;
  const objectUrl = URL.createObjectURL(blob);
  const type = blob.type || '';

  const css = (el, text) => { el.style.cssText = text; return el; };
  const make = (tag, text) => document.createElement(tag);

  // Lapisan gelap + kartu modal ala Modal aplikasi: judul + tombol X, isi, footer aksi.
  const overlay = css(make('div'),
    'position:fixed;inset:0;z-index:9999;background:rgba(17,24,39,.6);display:flex;align-items:center;justify-content:center;padding:24px;');
  overlay.setAttribute('role', 'dialog');
  overlay.setAttribute('aria-modal', 'true');

  const card = css(make('div'),
    'background:#fff;border-radius:10px;box-shadow:0 20px 25px -5px rgba(0,0,0,.2);width:100%;max-width:880px;max-height:100%;display:flex;flex-direction:column;overflow:hidden;font-family:inherit;');

  // Header
  const header = css(make('div'),
    'display:flex;align-items:center;justify-content:space-between;gap:12px;padding:14px 20px;border-bottom:1px solid #e5e7eb;');
  const title = css(make('h3'), 'margin:0;font-size:15px;font-weight:700;color:#111827;');
  title.textContent = title_ || 'Pratinjau Berkas';
  const xBtn = css(make('button'),
    'cursor:pointer;width:32px;height:32px;display:inline-flex;align-items:center;justify-content:center;background:#fff;border:1px solid #e5e7eb;border-radius:8px;color:#6b7280;font-size:16px;line-height:1;');
  xBtn.type = 'button';
  xBtn.setAttribute('aria-label', 'Tutup');
  xBtn.textContent = '✕';
  header.append(title, xBtn);

  // Isi
  const body = css(make('div'),
    'flex:1;min-height:0;overflow:auto;padding:16px 20px;background:#f9fafb;display:flex;align-items:center;justify-content:center;');
  if (type.startsWith('image/')) {
    const img = css(make('img'), 'max-width:100%;max-height:70vh;object-fit:contain;border-radius:6px;background:#fff;');
    img.src = objectUrl;
    img.alt = 'Berkas';
    body.append(img);
  } else if (type === 'application/pdf') {
    const frame = css(make('iframe'), 'width:100%;height:70vh;border:0;border-radius:6px;background:#fff;');
    frame.src = objectUrl;
    body.append(frame);
  } else {
    const note = css(make('p'), 'margin:0;color:#6b7280;font-size:13px;');
    note.textContent = 'Pratinjau tidak tersedia untuk jenis berkas ini. Gunakan tombol Download.';
    body.append(note);
  }

  // Footer: aksi unduh (menutup lewat tombol ✕, klik area gelap, atau Esc)
  const footer = css(make('div'),
    'display:flex;align-items:center;justify-content:flex-end;gap:8px;padding:12px 20px;border-top:1px solid #e5e7eb;');
  const btnStyle = 'cursor:pointer;display:inline-flex;align-items:center;justify-content:center;height:36px;padding:0 16px;border-radius:12px;font-size:13px;font-weight:600;text-decoration:none;';
  const downloadBtn = css(make('a'), btnStyle + 'background:#fff;color:#1f2937;border:1px solid #e5e7eb;');
  downloadBtn.href = objectUrl;
  downloadBtn.download = String(normalizedUrl).split('/').pop() || 'berkas';
  // Ikon unduh (gaya lucide "download"); markup statis, bukan dari data pengguna.
  downloadBtn.innerHTML =
    '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" style="margin-right:8px"><path d="M12 15V3"/><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/></svg><span>Download</span>';
  footer.append(downloadBtn);

  card.append(header, body, footer);
  overlay.append(card);

  const close = () => {
    document.removeEventListener('keydown', onKey);
    overlay.remove();
    URL.revokeObjectURL(objectUrl);
  };
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  xBtn.addEventListener('click', close);
  overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });
  document.addEventListener('keydown', onKey);
  document.body.append(overlay);
}

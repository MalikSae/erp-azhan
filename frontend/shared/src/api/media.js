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

// Open a protected media response using the authenticated API client. This
// keeps passport/KTP/payment proof files out of public URL navigation.
export async function openProtectedMedia(url) {
  const popup = window.open('', '_blank', 'noopener,noreferrer');
  try {
    const normalizedUrl = String(url || '').replace(
      /^\/uploads\/(dokumen-jamaah|payment-proofs)\/(.+)$/,
      '/api/admin/media/$1/$2',
    );
    const response = await client.get(normalizedUrl, { responseType: 'blob' });
    const objectUrl = URL.createObjectURL(response.data);
    if (popup) popup.location.href = objectUrl;
    else window.location.href = objectUrl;
    window.setTimeout(() => URL.revokeObjectURL(objectUrl), 60_000);
  } catch (error) {
    if (popup) popup.close();
    throw error;
  }
}

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

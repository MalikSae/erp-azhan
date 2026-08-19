import client from './client';

export async function uploadMedia(file, category) {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('category', category);
  
  const response = await client.post('/api/admin/media/upload', formData);
  return response.data.url;
}

export async function uploadMediaWithOptions(file, category, options = {}) {
  const formData = new FormData();
  formData.append('file', file);
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

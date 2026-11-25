import client from './client';

// Manifest paket: PNR, detail penerbangan, pemeriksaan data pax, dan unduh .xlsx.

export async function getManifest(scheduleId) {
  const response = await client.get(`/api/admin/schedules/${scheduleId}/manifest`);
  return response.data;
}

export async function simpanManifest(scheduleId, payload) {
  const response = await client.put(`/api/admin/schedules/${scheduleId}/manifest`, payload);
  return response.data;
}

// Mengunduh file manifest dan menyimpannya lewat browser. Nama file dari header server.
export async function unduhManifest(scheduleId) {
  const response = await client.get(`/api/admin/schedules/${scheduleId}/manifest/unduh`, { responseType: 'blob' });
  const header = response.headers?.['content-disposition'] || '';
  const utf8 = /filename\*=UTF-8''([^;]+)/i.exec(header);
  const biasa = /filename="([^"]+)"/i.exec(header);
  const nama = utf8 ? decodeURIComponent(utf8[1]) : (biasa ? biasa[1] : 'manifest.xlsx');
  const url = URL.createObjectURL(response.data);
  const a = document.createElement('a');
  a.href = url;
  a.download = nama;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

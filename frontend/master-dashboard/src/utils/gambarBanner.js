// Gambar Banner Promosi: hasil akhir selalu 1200 × 600 (2:1), di bawah 1 MB. Server memeriksa ulang.

export const LEBAR = 1200;
export const TINGGI = 600;
const MAKS_BYTE = 1024 * 1024;

/** Memuat gambar dari berkas. URL sementara (img.src) tetap hidup untuk pratinjau pemotong; lepas dengan lepasGambar. */
export function bacaGambar(file) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);
    img.onload = () => resolve(img);
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Gambar harus JPG, PNG, atau WebP'));
    };
    img.src = url;
  });
}

export const lepasGambar = (img) => img?.src?.startsWith('blob:') && URL.revokeObjectURL(img.src);

/** Gambar sudah tepat 2:1 dan cukup lebar: tidak perlu dipotong. */
export const pas21 = (img) => img.naturalWidth === img.naturalHeight * 2 && img.naturalWidth >= LEBAR;

/** Gambar cukup besar untuk dipotong menjadi 1200 × 600. */
export const cukupBesar = (img) => img.naturalWidth >= LEBAR && img.naturalHeight >= TINGGI;

const keBlob = (canvas, tipe, mutu) => new Promise((resolve) => canvas.toBlob(resolve, tipe, mutu));

// Kompres kanvas 1200 × 600 ke WebP (mutu turun bertahap) sampai di bawah 1 MB; JPEG bila browser tanpa encoder WebP.
async function kompres(canvas, namaDasar) {
  for (const mutu of [0.92, 0.85, 0.75, 0.65, 0.5]) {
    const blob = await keBlob(canvas, 'image/webp', mutu);
    if (blob && blob.type === 'image/webp' && blob.size <= MAKS_BYTE) return new File([blob], `${namaDasar}.webp`, { type: 'image/webp' });
  }
  for (const mutu of [0.9, 0.8, 0.7, 0.6]) {
    const blob = await keBlob(canvas, 'image/jpeg', mutu);
    if (blob && blob.size <= MAKS_BYTE) return new File([blob], `${namaDasar}.jpg`, { type: 'image/jpeg' });
  }
  throw new Error('Ukuran gambar maksimal 1 MB');
}

/**
 * Potong area { x, y, w } (koordinat gambar asli; tinggi = w / 2) menjadi 1200 × 600 lalu kompres.
 * Tanpa area: seluruh gambar (dipakai untuk gambar yang sudah 2:1).
 */
export async function potongGambar(img, namaFile, area) {
  const { x = 0, y = 0, w = img.naturalWidth } = area || {};
  const canvas = document.createElement('canvas');
  canvas.width = LEBAR;
  canvas.height = TINGGI;
  const ctx = canvas.getContext('2d');
  ctx.imageSmoothingQuality = 'high';
  ctx.drawImage(img, x, y, w, w / 2, 0, 0, LEBAR, TINGGI);
  return kompres(canvas, namaFile.replace(/\.[^.]+$/, '') || 'banner');
}

/** Gambar 2:1 yang sudah 1200 px dan di bawah 1 MB diunggah apa adanya; selebihnya dikecilkan dan dikompres. */
export async function siapkanGambar21(file, img) {
  if (img.naturalWidth === LEBAR && file.size <= MAKS_BYTE) return file;
  return potongGambar(img, file.name);
}

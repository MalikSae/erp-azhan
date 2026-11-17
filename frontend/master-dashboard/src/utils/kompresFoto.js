// Kecilkan foto ke lebar maksimal lalu kompres ke WebP (JPEG bila browser tanpa encoder WebP) sampai di bawah 1 MB.
// Rasio dipertahankan. Foto yang sudah kecil dan ringan diunggah apa adanya.

const MAKS_BYTE = 1024 * 1024;

function bacaGambar(file) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Foto harus JPG, PNG, atau WebP'));
    };
    img.src = url;
  });
}

const keBlob = (canvas, tipe, mutu) => new Promise((resolve) => canvas.toBlob(resolve, tipe, mutu));

export async function kompresFoto(file, lebarMaks = 1600) {
  const img = await bacaGambar(file);
  if (img.naturalWidth <= lebarMaks && file.size <= MAKS_BYTE) return file;
  const skala = Math.min(1, lebarMaks / img.naturalWidth);
  const canvas = document.createElement('canvas');
  canvas.width = Math.round(img.naturalWidth * skala);
  canvas.height = Math.round(img.naturalHeight * skala);
  const ctx = canvas.getContext('2d');
  ctx.imageSmoothingQuality = 'high';
  ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
  const dasar = file.name.replace(/\.[^.]+$/, '') || 'foto';
  for (const mutu of [0.9, 0.82, 0.72, 0.6]) {
    const blob = await keBlob(canvas, 'image/webp', mutu);
    if (blob && blob.type === 'image/webp' && blob.size <= MAKS_BYTE) return new File([blob], `${dasar}.webp`, { type: 'image/webp' });
  }
  for (const mutu of [0.85, 0.7, 0.6]) {
    const blob = await keBlob(canvas, 'image/jpeg', mutu);
    if (blob && blob.size <= MAKS_BYTE) return new File([blob], `${dasar}.jpg`, { type: 'image/jpeg' });
  }
  throw new Error('Ukuran foto terlalu besar');
}

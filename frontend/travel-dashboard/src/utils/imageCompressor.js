/**
 * Kompresi gambar client-side menggunakan Canvas API.
 * Mengubah foto kamera HP berukuran besar (5-20MB) menjadi foto berukuran ringkas (~200KB-600KB)
 * dengan resolusi optimal (max 1920px) dan teks/detail tetap tajam tanpa membebani server.
 */
export async function compressImage(file, { maxWidth = 1920, maxHeight = 1920, quality = 0.82 } = {}) {
  // Jika bukan gambar (misal PDF), kembalikan file aslinya
  if (!file || !file.type.startsWith('image/')) {
    return file;
  }

  // Jika format SVG atau GIF animasi, jangan diubah
  if (file.type === 'image/svg+xml' || file.type === 'image/gif') {
    return file;
  }

  return new Promise((resolve) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);

    reader.onload = (event) => {
      const img = new Image();
      img.src = event.target.result;

      img.onload = () => {
        let { width, height } = img;

        // Hitung scaling dimensi proporsional
        if (width > maxWidth || height > maxHeight) {
          if (width / height > maxWidth / maxHeight) {
            height = Math.round((height * maxWidth) / width);
            width = maxWidth;
          } else {
            width = Math.round((width * maxHeight) / height);
            height = maxHeight;
          }
        }

        const canvas = document.createElement('canvas');
        canvas.width = width;
        canvas.height = height;

        const ctx = canvas.getContext('2d');
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = 'high';
        ctx.drawImage(img, 0, 0, width, height);

        // Convert ke format JPEG
        canvas.toBlob(
          (blob) => {
            if (!blob) {
              resolve(file); // Fallback ke file asli jika gagal
              return;
            }

            const fileName = file.name.replace(/\.[^/.]+$/, "") + ".jpg";
            const compressedFile = new File([blob], fileName, {
              type: 'image/jpeg',
              lastModified: Date.now(),
            });

            // Jika hasil kompresi ternyata lebih besar dari file asli, pakai file asli
            if (compressedFile.size >= file.size) {
              resolve(file);
            } else {
              resolve(compressedFile);
            }
          },
          'image/jpeg',
          quality
        );
      };

      img.onerror = () => {
        resolve(file); // Fallback
      };
    };

    reader.onerror = () => {
      resolve(file); // Fallback
    };
  });
}

import { useEffect, useState } from 'react';
import client from '../api/client';
import { protectedMediaPath } from '../api/media';

// Memuat file dari endpoint media terproteksi sebagai blob URL (foto agen).
export default function useProtectedImage(url) {
  const [src, setSrc] = useState(null);
  useEffect(() => {
    let alive = true;
    let objectUrl = null;
    setSrc(null);
    const path = protectedMediaPath(url);
    if (!path) return undefined;
    client.get(path, { responseType: 'blob' })
      .then((res) => {
        if (!alive) return;
        objectUrl = URL.createObjectURL(res.data);
        setSrc(objectUrl);
      })
      .catch(() => alive && setSrc(null));
    return () => {
      alive = false;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [url]);
  return src;
}

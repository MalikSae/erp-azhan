import { useEffect, useState } from 'react';
import client from '../api/client';

// Memuat file dari endpoint media terproteksi sebagai blob URL (foto agen).
export default function useProtectedImage(url) {
  const [src, setSrc] = useState(null);
  useEffect(() => {
    let alive = true;
    let objectUrl = null;
    setSrc(null);
    if (!url) return undefined;
    client.get(url, { responseType: 'blob' })
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

import React, { useState } from 'react';
import { ImageIcon } from 'lucide-react';

const photoSrcOf = (url) => {
  if (!url) return null;
  if (url.startsWith('http')) return url;
  return `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${url.startsWith('/') ? '' : '/'}${url}`;
};

// Thumbnail foto item perlengkapan; placeholder ikon bila belum ada foto atau gagal dimuat.
const ItemPhoto = ({ url, name }) => {
  const [failed, setFailed] = useState(false);
  const src = photoSrcOf(url);
  if (!src || failed) {
    return (
      <div className="w-10 h-10 shrink-0 rounded-lg border border-neutral-200 bg-neutral-100 flex items-center justify-center text-neutral-400">
        <ImageIcon className="w-4 h-4" strokeWidth={1.75} />
      </div>
    );
  }
  return (
    <img
      src={src}
      alt={name}
      className="w-10 h-10 shrink-0 object-cover rounded-lg border border-neutral-200 bg-white"
      onError={() => setFailed(true)}
    />
  );
};

export default ItemPhoto;

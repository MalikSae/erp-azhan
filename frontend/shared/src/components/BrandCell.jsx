import { useState } from 'react';

export const BrandCell = ({ brand, brandId, brandName, brandIconUrl, showText = false }) => {
  const [imageError, setImageError] = useState(false);

  const name = brand?.name || brandName || (brandId ? `Brand #${brandId}` : '-');
  const iconUrl = brand?.icon_url || brandIconUrl;

  if (!name || name === '-') return <span>-</span>;

  const showInitial = !iconUrl || imageError;
  const initial = name.charAt(0).toUpperCase();

  return (
    <div className="flex items-center gap-2" title={name}>
      {showInitial ? (
        <div className="w-7 h-7 shrink-0 rounded-full bg-neutral-200 border border-neutral-300 flex items-center justify-center text-xs font-bold text-neutral-700 uppercase font-heading">
          {initial}
        </div>
      ) : (
        <img
          src={iconUrl.startsWith('http') ? iconUrl : `${import.meta.env.VITE_API_BASE_URL}${iconUrl.startsWith('/') ? '' : '/'}${iconUrl}`}
          alt={name}
          className="w-7 h-7 shrink-0 rounded-full object-cover bg-neutral-100 border border-neutral-200"
          onError={() => setImageError(true)}
        />
      )}
      {showText && (
        <span className="font-medium text-neutral-900 font-body text-sm truncate max-w-36">
          {name}
        </span>
      )}
    </div>
  );
};

export default BrandCell;

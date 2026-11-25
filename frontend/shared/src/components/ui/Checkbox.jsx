import React from 'react';

// Kotak centang standar. Dengan `label`, seluruh baris bisa diklik; tanpa `label`
// (mis. di dalam sel matriks) wajib diberi `ariaLabel`.
const Checkbox = ({ id, checked, onChange, disabled = false, label, ariaLabel, className = '' }) => {
  const box = (
    <input
      id={id}
      type="checkbox"
      checked={checked}
      onChange={(e) => onChange?.(e.target.checked)}
      disabled={disabled}
      aria-label={label ? undefined : ariaLabel}
      className="h-4 w-4 shrink-0 cursor-pointer rounded-sm border-neutral-300 accent-primary-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-1 disabled:cursor-not-allowed"
    />
  );
  if (!label) return box;
  return (
    <label className={`flex items-center gap-2.5 font-body text-sm text-neutral-800 ${disabled ? 'cursor-not-allowed' : 'cursor-pointer'} ${className}`}>
      {box}
      <span>{label}</span>
    </label>
  );
};

export default Checkbox;

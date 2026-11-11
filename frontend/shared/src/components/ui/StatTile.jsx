import React from 'react';

// StatTile — kartu statistik gaya dash.cloudflare.com (DESIGN.md): kartu datar,
// label kecil di atas, angka besar di bawah. Ikon kecil netral di samping
// label; `variant` mewarnai ikon saja, tanpa kotak ber-tint.
const VARIANTS = {
  primary: 'text-neutral-500',
  success: 'text-success-600',
  info: 'text-info-600',
  violet: 'text-violet-600',
  warning: 'text-warning-600',
  danger: 'text-danger-600',
};

const StatTile = ({ icon: Icon, label, value, hint, variant = 'primary' }) => (
  <div className="bg-white rounded-2xl border border-neutral-200 p-4 min-w-0">
    <div className="flex items-center gap-1.5 text-xs font-body font-medium text-neutral-600">
      {Icon && <Icon className={`w-3.5 h-3.5 shrink-0 ${VARIANTS[variant] || VARIANTS.primary}`} strokeWidth={2} />}
      <span className="truncate">{label}</span>
    </div>
    <div className="mt-1.5 text-xl md:text-2xl font-heading font-semibold tracking-tight tabular-nums text-neutral-900 leading-tight">{value}</div>
    {hint && <div className="mt-0.5 text-xs font-body text-neutral-500 truncate">{hint}</div>}
  </div>
);

export default StatTile;

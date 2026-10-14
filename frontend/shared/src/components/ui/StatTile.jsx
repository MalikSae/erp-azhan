import React from 'react';

// StatTile — kartu statistik gaya SmartHR: ikon kotak ber-tint di kiri,
// angka besar + label. Varian menentukan warna tint ikon.
const VARIANTS = {
  primary: 'bg-primary-100 text-primary-600',
  success: 'bg-success-100 text-success-600',
  info: 'bg-info-100 text-info-600',
  violet: 'bg-violet-100 text-violet-600',
  warning: 'bg-warning-100 text-warning-600',
  danger: 'bg-danger-100 text-danger-600',
};

const StatTile = ({ icon: Icon, label, value, hint, variant = 'primary' }) => (
  <div className="bg-white rounded-2xl border border-neutral-200 shadow-card p-4 flex items-center gap-4">
    {Icon && (
      <div className={`w-11 h-11 rounded-xl flex items-center justify-center shrink-0 ${VARIANTS[variant] || VARIANTS.primary}`}>
        <Icon className="w-5 h-5" strokeWidth={2} />
      </div>
    )}
    <div className="min-w-0">
      <div className="text-2xl font-heading font-bold text-neutral-900 leading-tight">{value}</div>
      <div className="text-xs font-body text-neutral-500 truncate">{label}</div>
      {hint && <div className="text-[11px] font-body text-neutral-400 truncate">{hint}</div>}
    </div>
  </div>
);

export default StatTile;

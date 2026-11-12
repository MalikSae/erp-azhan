import React from 'react';

// MetaBox — kartu berjudul. Tampilannya kini sama dengan Surface + Section di
// DetailLayout (design-system.md §5.3): kartu datar berbingkai, judul di atas isi,
// tanpa pita header abu dan tanpa kotak ikon. Untuk halaman baru pakai
// Surface/Section; MetaBox dipertahankan agar pemanggil lama tetap jalan.
//
// `icon` sengaja diabaikan (aturan: tanpa ikon dekoratif di judul).
// eslint-disable-next-line no-unused-vars
const MetaBox = ({
  title,
  subtitle,
  icon,
  badge,
  action,
  headerAction,
  headerActions,
  children,
  className = '',
  contentClassName = 'px-5 pb-5 space-y-4',
}) => {
  const actionsSlot = headerActions || headerAction || action;

  return (
    <section className={`bg-white border border-neutral-200 rounded-2xl ${className}`}>
      <div className="flex items-start justify-between gap-3 px-5 pt-5 pb-4">
        <div className="min-w-0">
          <h3 className="font-heading text-base font-semibold text-neutral-900 leading-tight">{title}</h3>
          {subtitle && <p className="mt-1 text-sm font-body text-neutral-500">{subtitle}</p>}
        </div>
        {(badge || actionsSlot) && (
          <div className="flex items-center gap-2 shrink-0">
            {badge && (
              <span className="text-xs font-medium tabular-nums text-neutral-500">{badge}</span>
            )}
            {actionsSlot}
          </div>
        )}
      </div>
      <div className={contentClassName}>{children}</div>
    </section>
  );
};

export default MetaBox;

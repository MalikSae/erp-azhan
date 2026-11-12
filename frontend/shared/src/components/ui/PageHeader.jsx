import React from 'react';
import Button from './Button';
import { ArrowLeft, Home, ChevronRight } from 'lucide-react';

// PageHeader — satu-satunya header halaman (design-system.md §5.4).
//
// Halaman daftar:  <PageHeader title="Kelola Booking" actionLabel="+ Buat Booking" onAction={…} />
// Halaman detail:  <PageHeader title={nama} onBack={…} meta={['HN-123', '0812…', brand]}
//                              badges={<><Badge variant="success">Agen aktif</Badge>…</>}>
//                    <Button …>Ubah</Button>
//                  </PageHeader>
//
// Badge sebaris dengan judul (jarak 12px), meta di bawahnya (mt-1, 4px): header detail dua baris. Header tanpa margin luar;
// jarak ke blok berikutnya diatur `space-y-6` di akar halaman. Detail memakai onBack tanpa breadcrumb.
const PageHeader = ({
  title,
  actionLabel,
  onAction,
  secondaryActionLabel,
  onSecondaryAction,
  onBack,
  subtitle,
  meta,
  badges,
  breadcrumb,
  children,
}) => {
  // Halaman detail (punya tombol kembali) tidak menampilkan breadcrumb.
  const tampilBreadcrumb = breadcrumb ?? !onBack;
  const metaList = (Array.isArray(meta) ? meta : meta ? [meta] : []).filter((m) => m !== null && m !== undefined && m !== false && m !== '');
  const punyaBaris = metaList.length > 0 || badges;
  const punyaAksi = actionLabel || secondaryActionLabel || children;

  return (
    <div className={`flex flex-col gap-3 md:flex-row md:justify-between ${punyaBaris ? 'md:items-start' : 'md:items-center'}`}>
      <div className="flex min-w-0 items-start gap-3">
        {onBack && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onBack}
            className="!p-1.5 shrink-0 text-neutral-600 hover:text-neutral-900 hover:bg-neutral-100 rounded-lg"
            aria-label="Kembali"
          >
            <ArrowLeft size={20} />
          </Button>
        )}
        <div className="min-w-0">
          {/* Badge status/atribut sebaris dengan judul supaya header detail cukup dua baris (judul, meta). */}
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <h1 className="text-lg md:text-xl font-heading font-semibold text-neutral-900 tracking-tight break-words">{title}</h1>
            {badges && <div className="flex flex-wrap items-center gap-2">{badges}</div>}
          </div>
          {tampilBreadcrumb && (
            <div className="mt-1 flex items-center gap-1.5 text-xs font-body text-neutral-500">
              <Home className="w-3.5 h-3.5" />
              <span>Dashboard</span>
              <ChevronRight className="w-3 h-3 text-neutral-300" />
              <span className="text-neutral-700 font-medium">{title}</span>
            </div>
          )}
          {metaList.length > 0 && (
            <p className="mt-1 text-sm font-body text-neutral-500 break-words">
              {metaList.map((m, i) => (
                <React.Fragment key={i}>
                  {i > 0 && <span className="mx-1.5 text-neutral-300">·</span>}
                  <span className="tabular-nums">{m}</span>
                </React.Fragment>
              ))}
            </p>
          )}
          {subtitle && <p className="mt-1 text-sm text-neutral-500 font-body">{subtitle}</p>}
        </div>
      </div>
      {punyaAksi && (
        <div className="flex w-full flex-wrap items-center gap-2 md:w-auto md:shrink-0">
          {children}
          {secondaryActionLabel && <Button variant="secondary" onClick={onSecondaryAction} className="flex-1 md:flex-none">{secondaryActionLabel}</Button>}
          {actionLabel && <Button variant="primary" onClick={onAction} className="flex-1 md:flex-none">{actionLabel}</Button>}
        </div>
      )}
    </div>
  );
};

export default PageHeader;

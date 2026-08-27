import React from 'react';

const MetaBox = ({ 
  title, 
  subtitle,
  icon,
  badge,
  headerActions,
  children, 
  className = '',
  contentClassName = 'p-5 space-y-4'
}) => {
  return (
    <div className={`bg-white border border-neutral-200/90 rounded-xl shadow-2xs overflow-visible ${className}`}>
      <div className="bg-neutral-50/80 border-b border-neutral-200/80 px-4 sm:px-5 py-3 flex items-center justify-between gap-3 rounded-t-xl">
        <div className="flex items-center gap-2.5 min-w-0">
          {icon && (
            <div className="w-7 h-7 rounded-lg bg-white border border-neutral-200/90 flex items-center justify-center text-primary-600 shrink-0 shadow-2xs">
              {icon}
            </div>
          )}
          <div className="min-w-0">
            <h3 className="font-heading text-sm font-bold text-neutral-900 leading-tight truncate">
              {title}
            </h3>
            {subtitle && (
              <p className="text-xs text-neutral-500 font-normal leading-normal truncate">
                {subtitle}
              </p>
            )}
          </div>
        </div>
        
        <div className="flex items-center gap-2 shrink-0">
          {badge && (
            <span className="text-[10px] font-bold uppercase tracking-wider bg-neutral-100 text-neutral-600 px-2 py-0.5 rounded border border-neutral-200">
              {badge}
            </span>
          )}
          {headerActions}
        </div>
      </div>
      <div className={contentClassName}>
        {children}
      </div>
    </div>
  );
};

export default MetaBox;


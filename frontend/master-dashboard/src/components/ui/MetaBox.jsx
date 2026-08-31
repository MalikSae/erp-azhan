import React from 'react';

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
  contentClassName = 'p-5 space-y-4'
}) => {
  const actionsSlot = headerActions || headerAction || action;

  return (
    <div className={`bg-white border border-neutral-200/80 rounded-2xl shadow-card relative transition-all ${className}`}>
      <div className="bg-neutral-50/70 border-b border-neutral-200/80 px-4 sm:px-5 py-3.5 flex items-center justify-between gap-3 rounded-t-2xl">
        <div className="flex items-center gap-2.5 min-w-0">
          {icon && (
            <div className="w-8 h-8 rounded-xl bg-white border border-neutral-200/90 flex items-center justify-center text-neutral-800 shrink-0 shadow-2xs">
              {icon}
            </div>
          )}
          <div className="min-w-0">
            <h3 className="font-heading text-sm font-bold text-neutral-900 leading-tight truncate">
              {title}
            </h3>
            {subtitle && (
              <p className="text-xs text-neutral-500 font-normal leading-normal truncate mt-0.5">
                {subtitle}
              </p>
            )}
          </div>
        </div>
        
        <div className="flex items-center gap-2 shrink-0">
          {badge && (
            <span className="text-[10px] font-bold uppercase tracking-wider bg-neutral-100 text-neutral-700 px-2 py-0.5 rounded-full border border-neutral-200">
              {badge}
            </span>
          )}
          {actionsSlot}
        </div>
      </div>
      <div className={contentClassName}>
        {children}
      </div>
    </div>
  );
};

export default MetaBox;

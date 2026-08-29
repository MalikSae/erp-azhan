import React from 'react';
import Button from './Button';
import { ArrowLeft } from 'lucide-react';

const PageHeader = ({ title, actionLabel, onAction, secondaryActionLabel, onSecondaryAction, onBack, subtitle, children }) => {
  return (
    <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-6 gap-4">
      <div className="flex items-center gap-3">
        {onBack && (
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={onBack} 
            className="!p-2 text-neutral-600 hover:text-neutral-900 hover:bg-neutral-100 rounded-xl" 
            aria-label="Kembali"
          >
            <ArrowLeft size={18} />
          </Button>
        )}
        <div>
          <h1 className="text-xl md:text-2xl font-heading font-extrabold text-neutral-900 tracking-tight">{title}</h1>
          {subtitle && <p className="mt-0.5 text-xs md:text-sm text-neutral-500 font-body">{subtitle}</p>}
        </div>
      </div>
      {(actionLabel || secondaryActionLabel || children) && (
        <div className="flex w-full gap-2.5 md:w-auto items-center flex-wrap">
          {children}
          {secondaryActionLabel && (
            <Button variant="secondary" onClick={onSecondaryAction} className="flex-1 md:flex-none">
              {secondaryActionLabel}
            </Button>
          )}
          {actionLabel && (
            <Button variant="primary" onClick={onAction} className="flex-1 md:flex-none">
              {actionLabel}
            </Button>
          )}
        </div>
      )}
    </div>
  );
};

export default PageHeader;

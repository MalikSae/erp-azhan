import React from 'react';
import Button from './Button';
import { ArrowLeft } from 'lucide-react';

const PageHeader = ({ title, actionLabel, onAction, secondaryActionLabel, onSecondaryAction, onBack, subtitle }) => {
  return (
    <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-6 gap-3">
      <div className="flex items-center gap-3">
        {onBack && (
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={onBack} 
            className="!p-1.5 text-neutral-600 hover:text-neutral-900 hover:bg-neutral-100 rounded-lg" 
            aria-label="Kembali"
          >
            <ArrowLeft size={20} />
          </Button>
        )}
        <div>
          <h1 className="text-xl md:text-2xl font-heading font-semibold text-neutral-900">{title}</h1>
          {subtitle && <p className="mt-1 text-sm text-neutral-500 font-body">{subtitle}</p>}
        </div>
      </div>
      {(actionLabel || secondaryActionLabel) && <div className="flex w-full gap-2 md:w-auto">
        {secondaryActionLabel && <Button variant="secondary" onClick={onSecondaryAction} className="flex-1 md:flex-none">{secondaryActionLabel}</Button>}
        {actionLabel && <Button variant="primary" onClick={onAction} className="flex-1 md:flex-none">{actionLabel}</Button>}
      </div>}
    </div>
  );
};

export default PageHeader;

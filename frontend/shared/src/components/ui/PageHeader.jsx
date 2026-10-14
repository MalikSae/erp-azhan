import React from 'react';
import Button from './Button';
import { ArrowLeft, Home, ChevronRight } from 'lucide-react';

const PageHeader = ({ title, actionLabel, onAction, secondaryActionLabel, onSecondaryAction, onBack, subtitle, breadcrumb = true, children }) => {
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
          <h1 className="text-lg md:text-xl font-heading font-bold text-neutral-900 tracking-tight">{title}</h1>
          {breadcrumb && (
            <div className="mt-1 flex items-center gap-1.5 text-xs font-body text-neutral-500">
              <Home className="w-3.5 h-3.5" />
              <span>Dashboard</span>
              <ChevronRight className="w-3 h-3 text-neutral-300" />
              <span className="text-neutral-700 font-medium">{title}</span>
            </div>
          )}
          {subtitle && <p className="mt-1 text-sm text-neutral-500 font-body">{subtitle}</p>}
        </div>
      </div>
      {(actionLabel || secondaryActionLabel || children) && (
        <div className="flex w-full gap-2 md:w-auto items-center flex-wrap">
          {children}
          {secondaryActionLabel && <Button variant="secondary" onClick={onSecondaryAction} className="flex-1 md:flex-none">{secondaryActionLabel}</Button>}
          {actionLabel && <Button variant="primary" onClick={onAction} className="flex-1 md:flex-none">{actionLabel}</Button>}
        </div>
      )}
    </div>
  );
};

export default PageHeader;

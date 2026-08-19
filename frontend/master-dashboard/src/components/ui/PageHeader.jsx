import React from 'react';
import Button from './Button';

const PageHeader = ({ title, actionLabel, onAction, onBack }) => {
  return (
    <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-6 gap-3">
      <div className="flex items-center gap-3">
        {onBack && (
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={onBack} 
            className="!px-2" 
            aria-label="Kembali"
          >
            <svg className="w-5 h-5 text-neutral-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
          </Button>
        )}
        <h1 className="text-xl md:text-2xl font-heading font-semibold text-neutral-900">{title}</h1>
      </div>
      {actionLabel && (
        <Button variant="primary" onClick={onAction} className="w-full md:w-auto">
          {actionLabel}
        </Button>
      )}
    </div>
  );
};

export default PageHeader;

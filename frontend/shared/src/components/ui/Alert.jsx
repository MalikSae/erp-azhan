import React from 'react';

const Alert = ({ variant = 'error', message, children, className = '', onClose }) => {
  const variants = {
    error: 'bg-danger-50 border-danger-200 text-danger-700',
    success: 'bg-success-50 border-success-200 text-success-700',
    warning: 'bg-warning-50 border-warning-200 text-warning-700',
    info: 'bg-neutral-50 border-neutral-200 text-neutral-700'
  };

  const selectedVariant = variants[variant] || variants.error;

  return (
    <div className={`p-4 rounded-xl border font-body text-xs md:text-sm flex items-center justify-between shadow-2xs ${selectedVariant} ${className}`}>
      <div>{children || message}</div>
      {onClose && (
        <button 
          type="button" 
          onClick={onClose} 
          className="ml-3 inline-flex text-current hover:opacity-75 focus:outline-none"
        >
          <span className="sr-only">Tutup</span>
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  );
};

export default Alert;

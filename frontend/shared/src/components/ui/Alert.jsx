import React from 'react';

const Alert = ({ variant = 'error', message, children, className = '', onClose }) => {
  // Gaya dash.cloudflare.com (DESIGN.md): latar status lembut, garis kiri 2px warna status.
  const variants = {
    error: 'bg-danger-50 border-danger-200 border-l-danger-600 text-danger-800',
    success: 'bg-success-50 border-success-200 border-l-success-600 text-success-800',
    warning: 'bg-warning-50 border-warning-200 border-l-warning-600 text-warning-800',
    info: 'bg-neutral-50 border-neutral-200 border-l-neutral-500 text-neutral-800'
  };

  const selectedVariant = variants[variant] || variants.error;

  return (
    <div role={variant === 'error' ? 'alert' : 'status'} className={`px-4 py-3 rounded-md border border-l-2 font-body text-sm flex items-center justify-between ${selectedVariant} ${className}`}>
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

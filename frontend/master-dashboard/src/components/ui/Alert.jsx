import React from 'react';

const Alert = ({ variant = 'error', children, className = '' }) => {
  const variants = {
    error: 'bg-danger-50 border-danger-200 text-danger-700',
    success: 'bg-success-50 border-success-200 text-success-700',
    warning: 'bg-warning-50 border-warning-200 text-warning-700',
    info: 'bg-neutral-50 border-neutral-200 text-neutral-700'
  };

  const selectedVariant = variants[variant] || variants.error;

  return (
    <div className={`p-4 rounded-xl border font-body text-xs md:text-sm shadow-2xs ${selectedVariant} ${className}`}>
      {children}
    </div>
  );
};

export default Alert;

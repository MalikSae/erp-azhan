import React from 'react';

const Pill = ({ label, variant = 'default', onRemove, className = '' }) => {
  const baseClasses = 'inline-flex items-center px-3 py-1 rounded-full text-sm font-medium transition-colors border';
  
  const variantClasses = {
    default: 'bg-white border-neutral-200 text-neutral-700 hover:bg-neutral-50',
    primary: 'bg-primary-50 border-primary-200 text-primary-700 hover:bg-primary-100',
    success: 'bg-success-50 border-success-200 text-success-700 hover:bg-success-100',
  };

  const selectedVariant = variantClasses[variant] || variantClasses.default;

  return (
    <div className={`${baseClasses} ${selectedVariant} ${className}`}>
      <span>{label}</span>
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          className="ml-1.5 flex-shrink-0 inline-flex items-center justify-center h-4 w-4 rounded-full text-current hover:bg-black/10 focus:outline-none focus:bg-black/10 transition-colors"
          aria-label={`Remove ${label}`}
        >
          <svg className="h-2 w-2" stroke="currentColor" fill="none" viewBox="0 0 8 8">
            <path strokeLinecap="round" strokeWidth="1.5" d="M1 1l6 6m0-6L1 7" />
          </svg>
        </button>
      )}
    </div>
  );
};

export default Pill;

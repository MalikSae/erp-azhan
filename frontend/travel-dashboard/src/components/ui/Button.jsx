import React from 'react';

const Button = ({ variant = 'primary', size = 'md', disabled, isLoading, onClick, type = 'button', children, className = '' }) => {
  const baseClasses = 'inline-flex items-center justify-center font-medium focus:outline-none rounded-full transition-all duration-200 ease-in-out';
  
  const variantClasses = {
    primary: 'bg-brand hover:opacity-90 text-white shadow-lg shadow-primary-500/30 hover:shadow-primary-600/40 hover:-translate-y-0.5',
    secondary: 'bg-white hover:bg-neutral-50 text-neutral-800 border border-neutral-200 shadow-sm hover:shadow hover:-translate-y-0.5',
    danger: 'bg-gradient-to-r from-danger-500 to-danger-600 hover:from-danger-600 hover:to-danger-700 text-white shadow-lg shadow-danger-500/30 hover:shadow-danger-600/40 hover:-translate-y-0.5',
    ghost: 'bg-transparent hover:bg-neutral-100 text-neutral-700 hover:-translate-y-0.5'
  };

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-sm'
  };

  const isDisabled = disabled || isLoading;
  const disabledClasses = isDisabled ? 'opacity-60 cursor-not-allowed pointer-events-none' : 'cursor-pointer';

  return (
    <button
      type={type}
      onClick={isDisabled ? undefined : onClick}
      disabled={isDisabled}
      className={`${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${disabledClasses} ${className}`}
    >
      {isLoading && (
        <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-current" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      )}
      {children}
    </button>
  );
};

export default Button;

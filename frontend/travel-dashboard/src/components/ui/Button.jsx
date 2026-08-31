import React from 'react';

const Button = ({
  variant = 'primary',
  size = 'md',
  disabled,
  isLoading,
  onClick,
  type = 'button',
  children,
  className = '',
  icon
}) => {
  const baseClasses = 'inline-flex items-center justify-center font-heading font-semibold focus:outline-none focus:ring-2 focus:ring-offset-1 transition-all duration-150 ease-in-out select-none';
  
  const variantClasses = {
    primary: 'bg-primary-500 hover:bg-primary-600 text-brand-dark border border-amber-300/60 shadow-2xs hover:shadow-xs focus:ring-primary-500/60 active:scale-[0.98]',
    dark: 'bg-sidebar-bg hover:bg-sidebar-surface text-white shadow-2xs hover:shadow-xs focus:ring-neutral-700 active:scale-[0.98]',
    secondary: 'bg-white hover:bg-neutral-50 text-neutral-800 border border-neutral-200 shadow-2xs hover:border-neutral-300 focus:ring-neutral-200 active:scale-[0.98]',
    danger: 'bg-danger-600 hover:bg-danger-700 text-white shadow-2xs hover:shadow-xs focus:ring-danger-500 active:scale-[0.98]',
    'danger-light': 'bg-danger-50 hover:bg-danger-100 text-danger-700 border border-danger-200 focus:ring-danger-200 active:scale-[0.98]',
    ghost: 'bg-transparent hover:bg-neutral-100 text-neutral-700 focus:ring-neutral-200 active:scale-[0.98]'
  };

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-xs rounded-lg gap-1.5',
    md: 'px-4 py-2 text-xs md:text-sm rounded-xl gap-2',
    lg: 'px-5 py-2.5 text-sm md:text-base rounded-xl gap-2.5 font-bold'
  };

  const isDisabled = disabled || isLoading;
  const disabledClasses = isDisabled ? 'opacity-50 cursor-not-allowed pointer-events-none' : 'cursor-pointer';

  const primaryStyle = variant === 'primary' ? {
    backgroundColor: 'var(--brand-primary, #FED853)',
    color: 'var(--brand-primary-text, #14171A)',
  } : undefined;

  return (
    <button
      type={type}
      onClick={isDisabled ? undefined : onClick}
      disabled={isDisabled}
      style={primaryStyle}
      className={`${baseClasses} ${variantClasses[variant] || variantClasses.primary} ${sizeClasses[size] || sizeClasses.md} ${disabledClasses} ${className}`}
    >
      {isLoading ? (
        <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-current" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      ) : icon ? (
        <span className="shrink-0">{icon}</span>
      ) : null}
      {children}
    </button>
  );
};

export default Button;

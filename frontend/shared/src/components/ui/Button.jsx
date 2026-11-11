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
  icon,
  ...props
}) => {
  // Gaya dash.cloudflare.com (DESIGN.md): radius 8px; primer oranye bergradien
  // dengan border lebih tua, kilap tepi atas, bayangan lembut. Fokus cincin 2px primary-500.
  const baseClasses = 'inline-flex items-center justify-center font-heading font-medium rounded-2xl border focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 transition-colors duration-100 select-none';

  const variantClasses = {
    primary: 'bg-accent-500 bg-btn-primary hover:bg-btn-primary-hover text-white border-accent-600 shadow-btn-primary',
    dark: 'bg-neutral-800 hover:bg-neutral-700 text-white border-neutral-800',
    secondary: 'bg-white hover:bg-neutral-50 text-neutral-800 border-neutral-200 hover:border-neutral-300 shadow-btn-secondary',
    danger: 'bg-danger-600 hover:bg-danger-700 text-white border-danger-600',
    'danger-light': 'bg-danger-50 hover:bg-danger-100 text-danger-700 border-danger-200',
    // Hapus baris di form (ikon sampah di kanan baris): putih, teks merah.
    'danger-outline': 'bg-white hover:bg-danger-50 text-danger-600 hover:text-danger-700 border-neutral-200 shadow-btn-secondary',
    // "+ Tambah …" selebar penuh di bawah daftar baris (lihat design-system.md).
    dashed: 'bg-white hover:bg-neutral-50 text-neutral-600 hover:text-neutral-900 border-dashed border-neutral-300 hover:border-neutral-400',
    ghost: 'bg-transparent hover:bg-neutral-100 text-neutral-600 hover:text-neutral-900 border-transparent'
  };

  const sizeClasses = {
    sm: 'h-8 px-2.5 text-xs gap-1.5',
    md: 'h-9 px-3 text-sm gap-2',
    lg: 'h-10 px-4 text-sm gap-2 font-semibold'
  };

  const isDisabled = disabled || isLoading;
  const disabledClasses = isDisabled ? 'opacity-50 cursor-not-allowed pointer-events-none' : 'cursor-pointer';

  return (
    <button
      {...props}
      type={type}
      onClick={isDisabled ? undefined : onClick}
      disabled={isDisabled}
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

import React, { useState } from 'react';
import FormField from './FormField';

const Input = ({ 
  label, 
  type = 'text', 
  value, 
  onChange, 
  error, 
  helperText,
  required, 
  placeholder, 
  className = '', 
  name, 
  disabled, 
  readOnly, 
  icon,
  prefixIcon,
  list,
  ...props 
}) => {
  const [showPassword, setShowPassword] = useState(false);
  const isPasswordField = type === 'password';
  const inputType = isPasswordField && showPassword ? 'text' : type;

  const activeIcon = prefixIcon || icon;
  const inputBaseClasses = 'w-full rounded-md border border-neutral-300 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body transition-colors';
  const paddingLeftClass = activeIcon ? 'pl-9 pr-3' : 'px-3';
  const disabledClasses = disabled || readOnly ? 'bg-neutral-50 text-neutral-500 cursor-not-allowed pointer-events-none' : '';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500' : '';
  const paddingRightClass = isPasswordField ? 'pr-10' : '';

  return (
    <FormField label={label} error={error} helperText={helperText} required={required} className={className}>
      <div className="relative">
        {activeIcon && (
          <div className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400 pointer-events-none flex items-center justify-center">
            {activeIcon}
          </div>
        )}
        <input
          type={inputType}
          name={name}
          value={value}
          onChange={onChange}
          required={required}
          placeholder={placeholder}
          disabled={disabled}
          readOnly={readOnly}
          list={list}
          className={`${inputBaseClasses} ${paddingLeftClass} ${errorClasses} ${paddingRightClass} ${disabledClasses}`}
          {...props}
        />
        {isPasswordField && (
          <button
            type="button"
            onClick={() => setShowPassword(!showPassword)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600 focus:outline-none transition-colors"
            tabIndex="-1"
            aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
          >
            {showPassword ? (
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
            )}
          </button>
        )}
      </div>
    </FormField>
  );
};

export default Input;

import React from 'react';
import FormField from './FormField';

const Select = ({ 
  label, 
  value, 
  onChange, 
  options = [], 
  error, 
  required, 
  placeholder = 'Pilih...', 
  className = '', 
  name,
  children,
  disabled,
  ...props 
}) => {
  const selectBaseClasses = 'w-full rounded-md border border-neutral-300 px-3 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body bg-white transition-colors cursor-pointer';
  const disabledClasses = disabled ? 'bg-neutral-50 text-neutral-500 cursor-not-allowed pointer-events-none' : '';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500' : '';

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <select
        name={name}
        value={value}
        onChange={onChange}
        required={required}
        disabled={disabled}
        className={`${selectBaseClasses} ${errorClasses} ${disabledClasses}`}
        {...props}
      >
        {children ? (
          children
        ) : (
          <>
            {placeholder && <option value="">{placeholder}</option>}
            {options.map((opt, idx) => (
              <option key={idx} value={opt.value !== undefined ? opt.value : opt}>
                {opt.label !== undefined ? opt.label : opt}
              </option>
            ))}
          </>
        )}
      </select>
    </FormField>
  );
};

export default Select;

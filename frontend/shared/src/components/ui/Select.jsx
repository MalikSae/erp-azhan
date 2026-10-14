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
  const selectBaseClasses = 'h-10 w-full min-w-0 rounded-lg border border-neutral-200 bg-white px-3 text-[13px] text-neutral-900 focus:outline-none font-body transition-colors cursor-pointer';
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

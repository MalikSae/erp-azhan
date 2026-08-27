import React from 'react';
import FormField from './FormField';

// Helper format angka dengan titik ribuan
const formatThousand = (val) => {
  if (val === null || val === undefined || val === '') return '';
  const numStr = String(val).replace(/\D/g, '');
  if (!numStr) return '';
  return new Intl.NumberFormat('id-ID').format(Number(numStr));
};

const CurrencyInput = ({ 
  label, 
  name, 
  value, 
  onChange, 
  error, 
  required, 
  placeholder = '0', 
  prefix = 'Rp', 
  className = '', 
  disabled = false,
  ...props 
}) => {
  const displayValue = formatThousand(value);

  const handleInputChange = (e) => {
    const rawVal = e.target.value.replace(/\D/g, '');
    const numVal = rawVal ? parseInt(rawVal, 10) : '';

    if (onChange) {
      if (name) {
        onChange({ target: { name, value: numVal } });
      } else {
        onChange(numVal);
      }
    }
  };

  const inputBaseClasses = 'h-11 w-full min-w-0 rounded-md border border-neutral-300 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body transition-colors';
  const disabledClasses = disabled ? 'bg-neutral-50 text-neutral-500 cursor-not-allowed pointer-events-none' : 'bg-white';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500 ring-danger-500' : '';
  const paddingClass = prefix ? 'pl-14 pr-3' : 'px-3';

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <div className="relative">
        {prefix && (
          <div className="pointer-events-none absolute inset-y-0 left-0 flex w-11 items-center justify-center border-r border-neutral-200 text-sm font-medium text-neutral-500 font-body">
            {prefix}
          </div>
        )}
        <input
          type="text"
          inputMode="numeric"
          name={name}
          value={displayValue}
          onChange={handleInputChange}
          required={required}
          placeholder={placeholder}
          disabled={disabled}
          className={`${inputBaseClasses} ${errorClasses} ${disabledClasses} ${paddingClass}`}
          {...props}
        />
      </div>
    </FormField>
  );
};

export default CurrencyInput;

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

  const inputBaseClasses = 'w-full rounded-md border border-neutral-300 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body transition-colors';
  const disabledClasses = disabled ? 'bg-neutral-50 text-neutral-500 cursor-not-allowed pointer-events-none' : 'bg-white';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500 ring-danger-500' : '';
  const paddingClass = prefix ? 'pl-11 pr-3' : 'px-3';

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <div className="relative">
        {prefix && (
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-neutral-500 text-sm font-medium font-body border-r border-neutral-200 pr-2 my-1.5">
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

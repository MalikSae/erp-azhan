import React from 'react';
import FormField from './FormField';

const Textarea = ({ label, value, onChange, error, required, placeholder, className = '', name, rows = 3 }) => {
  const textareaBaseClasses = 'w-full rounded-xl border border-neutral-200/90 bg-white px-3.5 py-2.5 text-xs md:text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs font-body transition-all';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500/20' : '';

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <textarea
        name={name}
        value={value}
        onChange={onChange}
        required={required}
        placeholder={placeholder}
        rows={rows}
        className={`${textareaBaseClasses} ${errorClasses}`}
      />
    </FormField>
  );
};

export default Textarea;

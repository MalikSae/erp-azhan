import React from 'react';
import FormField from './FormField';

const Textarea = ({ label, value, onChange, error, required, placeholder, className = '', name, id, maxLength, rows = 3 }) => {
  const textareaBaseClasses = 'w-full rounded-lg border border-neutral-200 bg-white px-3.5 py-2.5 text-[13px] text-neutral-900 placeholder:text-neutral-400 focus:outline-none font-body transition-all';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500/20' : '';

  return (
    <FormField label={label} error={error} required={required} className={className} htmlFor={id || name}>
      <textarea
        id={id || name}
        maxLength={maxLength}
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

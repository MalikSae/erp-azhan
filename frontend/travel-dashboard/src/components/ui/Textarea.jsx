import React from 'react';
import FormField from './FormField';

const Textarea = ({ label, value, onChange, error, required, placeholder, className = '', name, rows = 3 }) => {
  const textareaBaseClasses = 'w-full rounded-md border border-neutral-300 px-3 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500' : '';

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

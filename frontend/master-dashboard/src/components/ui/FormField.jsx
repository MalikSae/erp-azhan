import React from 'react';
import Label from './Label';

const FormField = ({ label, error, helpText, required, children, className = '' }) => {
  return (
    <div className={`mb-4 ${className}`}>
      {label && <Label required={required}>{label}</Label>}
      {helpText && <p className="mb-1 text-xs text-neutral-500 font-body">{helpText}</p>}
      {children}
      {error && <p className="mt-1 text-sm text-danger-600 font-body">{error}</p>}
    </div>
  );
};

export default FormField;

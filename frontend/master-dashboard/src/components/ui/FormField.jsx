import React from 'react';
import Label from './Label';

const FormField = ({ label, error, helperText, required, children, className = '' }) => {
  return (
    <div className={`mb-4 ${className}`}>
      {label && <Label required={required}>{label}</Label>}
      {children}
      {error && <p className="mt-1 text-sm text-danger-600 font-body">{error}</p>}
      {!error && helperText && <p className="mt-1 text-xs text-neutral-500 font-body leading-relaxed">{helperText}</p>}
    </div>
  );
};

export default FormField;


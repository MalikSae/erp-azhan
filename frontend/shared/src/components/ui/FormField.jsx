import React from 'react';
import Label from './Label';

const FormField = ({ label, error, helpText, required, children, className = '', htmlFor, errorId, helpId }) => {
  return (
    <div className={`mb-4 ${className}`}>
      {label && <Label htmlFor={htmlFor} required={required}>{label}</Label>}
      {helpText && <p id={helpId} className="mb-1 text-xs text-neutral-500 font-body">{helpText}</p>}
      {children}
      {error && <p id={errorId} className="mt-1 text-sm text-danger-600 font-body">{error}</p>}
    </div>
  );
};

export default FormField;

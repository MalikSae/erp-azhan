import React from 'react';
import Label from './Label';

const FormField = ({ label, error, required, children, htmlFor, className = '' }) => {
  return (
    <div className={`mb-4 ${className}`}>
      {label && <Label required={required} htmlFor={htmlFor}>{label}</Label>}
      {children}
      {error && <p className="mt-1 text-sm text-danger-600 font-body">{error}</p>}
    </div>
  );
};

export default FormField;

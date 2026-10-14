import React from 'react';

const Label = ({ children, required, className = '', htmlFor }) => {
  return (
    <label htmlFor={htmlFor} className={`mb-1.5 block min-h-5 text-[13px] font-medium leading-5 text-neutral-800 font-body ${className}`}>
      <span>{children}</span>
      {required && <span className="text-danger-500 ml-0.5 font-bold" aria-hidden="true">*</span>}
    </label>
  );
};

export default Label;

import React from 'react';

const Label = ({ children, required }) => {
  return (
    <label className="block text-sm font-medium text-neutral-700 mb-1 font-body">
      {children}
      {required && <span className="text-danger-500 ml-1">*</span>}
    </label>
  );
};

export default Label;

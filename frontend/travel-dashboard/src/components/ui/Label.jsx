import React from 'react';

const Label = ({ children, required }) => {
  return (
    <label className="mb-1.5 flex min-h-5 items-center gap-1 text-sm font-medium leading-5 text-neutral-700 font-body">
      <span>{children}</span>
      {required && <span className="text-danger-500" aria-hidden="true">*</span>}
    </label>
  );
};

export default Label;

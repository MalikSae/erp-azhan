import React from 'react';

const MetaBox = ({ title, children, className = '' }) => {
  return (
    <div className={`bg-white border border-neutral-200 rounded-md ${className}`}>
      <div className="bg-neutral-50 border-b border-neutral-200 py-2 px-4 rounded-t-md">
        <h3 className="font-heading text-sm font-semibold uppercase tracking-wide text-neutral-700">
          {title}
        </h3>
      </div>
      <div className="p-4 space-y-4">
        {children}
      </div>
    </div>
  );
};

export default MetaBox;

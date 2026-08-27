import React from 'react';

const MetaBox = ({ title, action, children, className = '' }) => {
  return (
    <div className={`bg-white border border-neutral-200 rounded-md ${className}`}>
      <div className="bg-neutral-50 border-b border-neutral-200 py-2.5 px-4 rounded-t-md flex items-center justify-between">
        <h3 className="font-heading text-sm font-semibold uppercase tracking-wide text-neutral-700">
          {title}
        </h3>
        {action && <div>{action}</div>}
      </div>
      <div className="p-4 space-y-4">
        {children}
      </div>
    </div>
  );
};

export default MetaBox;

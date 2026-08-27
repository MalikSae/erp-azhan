import React from 'react';

const EmptyState = ({ message = 'Data tidak ditemukan' }) => {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
      <div className="text-neutral-500 font-body text-sm">
        {message}
      </div>
    </div>
  );
};

export default EmptyState;

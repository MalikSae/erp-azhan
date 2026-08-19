import React from 'react';

const Card = ({ children, className = '' }) => {
  return (
    <div className={`bg-white border border-neutral-200 rounded-lg p-5 ${className}`}>
      {children}
    </div>
  );
};

export default Card;

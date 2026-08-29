import React from 'react';

const Card = ({ children, className = '' }) => {
  return (
    <div className={`bg-white border border-neutral-200/80 rounded-2xl shadow-card p-5 md:p-6 transition-all ${className}`}>
      {children}
    </div>
  );
};

export default Card;

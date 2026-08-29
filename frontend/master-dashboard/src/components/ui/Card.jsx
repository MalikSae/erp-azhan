import React from 'react';

const Card = ({ children, className = '' }) => {
  return (
    <div className={`bg-white border border-neutral-200/80 rounded-2xl p-5 md:p-6 shadow-card hover:shadow-card-hover transition-all duration-200 ${className}`}>
      {children}
    </div>
  );
};

export default Card;

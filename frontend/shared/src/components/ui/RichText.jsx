import React from 'react';

// Teks sederhana dengan **tebal**. Selain itu semua dianggap teks biasa (tanpa HTML).
const RichText = ({ text = '' }) => {
  const parts = String(text).split('**');
  return (
    <>
      {parts.map((part, index) => (
        index % 2 === 1
          ? <strong key={index} className="font-semibold text-neutral-900">{part}</strong>
          : <React.Fragment key={index}>{part}</React.Fragment>
      ))}
    </>
  );
};

export default RichText;

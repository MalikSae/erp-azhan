import React, { createContext, useState } from 'react';

export const BrandFilterContext = createContext();

export const BrandFilterProvider = ({ children }) => {
  const [selectedBrandId, setSelectedBrandId] = useState(null);

  return (
    <BrandFilterContext.Provider value={{ selectedBrandId, setSelectedBrandId }}>
      {children}
    </BrandFilterContext.Provider>
  );
};

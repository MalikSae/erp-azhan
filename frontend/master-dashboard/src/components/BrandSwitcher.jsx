import React, { useContext, useEffect, useState } from 'react';
import { BrandFilterContext } from '../context/BrandFilterContext';
import { listBrands } from '../api/brands';
import CustomDropdown from './ui/CustomDropdown';

const BrandSwitcher = () => {
  const { selectedBrandId, setSelectedBrandId } = useContext(BrandFilterContext);
  const [brands, setBrands] = useState([]);

  useEffect(() => {
    listBrands().then(data => {
      setBrands(data || []);
    }).catch(err => console.error("Gagal load brand", err));
  }, []);

  return (
    <div className="w-full">
      <CustomDropdown
        variant="dark"
        value={selectedBrandId || ''}
        onChange={(val) => {
          setSelectedBrandId(val === '' ? null : parseInt(val, 10));
        }}
        placeholder="Semua Brand"
        options={[
          { value: '', label: 'Semua Brand' },
          ...brands.map(brand => ({
            value: brand.id,
            label: brand.name
          }))
        ]}
      />
    </div>
  );
};

export default BrandSwitcher;

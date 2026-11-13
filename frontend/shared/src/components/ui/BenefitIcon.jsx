import React from 'react';
import { BENEFIT_ICON_BODY } from './benefitIconData';

// Ikon highlight benefit paket, digambar dari kunci yang disimpan di master (mis. "solar:bed-bold-duotone").
// Warnanya mengikuti warna teks induk (currentColor). Kunci yang tidak dikenal digambar sebagai kotak kosong
// supaya tata letak tidak bergeser.
export default function BenefitIcon({ name, size = 16, className = '' }) {
  const body = BENEFIT_ICON_BODY[name];
  if (!body) {
    return <span aria-hidden="true" className={`inline-block shrink-0 ${className}`} style={{ width: size, height: size }} />;
  }
  return (
    <svg
      aria-hidden="true"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={size}
      height={size}
      className={`shrink-0 ${className}`}
      dangerouslySetInnerHTML={{ __html: body }}
    />
  );
}

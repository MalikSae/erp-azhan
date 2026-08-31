import React from 'react';

const Toggle = ({ id, name, checked, onChange, disabled }) => {
  return (
    <label
      htmlFor={id}
      className={`relative inline-flex items-center ${disabled ? 'cursor-not-allowed pointer-events-none opacity-60' : 'cursor-pointer'}`}
    >
      <input
        type="checkbox"
        id={id}
        name={name}
        className="sr-only peer"
        checked={checked}
        onChange={onChange}
        disabled={disabled}
      />
      <div
        className={`
          w-10 h-6 bg-neutral-300 rounded-full peer 
          peer-focus:ring-2 peer-focus:ring-primary-300 
          peer-checked:after:translate-x-4 peer-checked:after:border-white 
          after:content-[''] after:absolute after:top-[2px] after:left-[2px] 
          after:bg-white after:border-neutral-300 after:border after:rounded-full 
          after:h-5 after:w-5 after:transition-all 
          peer-checked:bg-primary-600
        `}
      ></div>
    </label>
  );
};

export default Toggle;

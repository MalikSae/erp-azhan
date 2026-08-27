import React, { useState, useRef, useEffect } from 'react';
import FormField from './FormField';

const CustomDropdown = ({ 
  label, 
  options = [], 
  value, 
  onChange, 
  placeholder = 'Pilih...', 
  error, 
  required, 
  className = '',
  variant = 'light', // 'light' or 'dark'
  icon,
  prefixIcon
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };
    
    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  const selectedOption = options.find(opt => opt.value === value);

  const isDark = variant === 'dark';
  const activeIcon = prefixIcon || icon;
  const paddingLeftClass = activeIcon ? 'pl-9 pr-3' : 'px-3';

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <div className={`relative ${isOpen ? 'z-50' : 'z-10'}`} ref={dropdownRef}>
        {activeIcon && (
          <div className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400 pointer-events-none z-1 flex items-center justify-center">
            {activeIcon}
          </div>
        )}
        {/* Dropdown Trigger */}
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          className={`w-full flex items-center justify-between rounded-md border py-2 text-sm font-body text-left transition-colors focus:outline-none focus:ring-1 focus:ring-primary-500
            ${paddingLeftClass}
            ${error 
              ? 'border-danger-500 focus:border-danger-500' 
              : isDark ? 'border-neutral-700 focus:border-primary-500' : 'border-neutral-300 focus:border-primary-500'
            }
            ${isOpen ? 'border-primary-500 ring-1 ring-primary-500' : ''}
            ${!selectedOption 
              ? (isDark ? 'text-neutral-400' : 'text-neutral-500') 
              : (isDark ? 'text-white' : 'text-neutral-900')
            }
            ${isDark ? 'bg-neutral-800' : 'bg-white'}
          `}
        >
          <span className="truncate">
            {selectedOption ? selectedOption.label : placeholder}
          </span>
          <svg 
            className={`w-4 h-4 transition-transform duration-200 ${isDark ? 'text-neutral-500' : 'text-neutral-400'} ${isOpen ? 'transform rotate-180' : ''}`} 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {/* Dropdown Menu */}
        {isOpen && (
          <div className={`absolute z-50 w-full mt-1 border rounded-md shadow-lg overflow-hidden animate-in fade-in slide-in-from-top-1 duration-200
            ${isDark ? 'bg-neutral-800 border-neutral-700' : 'bg-white border-neutral-200'}
          `}>
            <ul className="max-h-60 overflow-y-auto py-1 text-sm font-body">
              {options.length === 0 ? (
                <li className={`px-3 py-2 text-center text-xs ${isDark ? 'text-neutral-500' : 'text-neutral-400'} italic`}>
                  Tidak ada pilihan
                </li>
              ) : (
                options.map((opt, idx) => (
                  <li
                    key={idx}
                    onClick={() => {
                      onChange && onChange(opt.value);
                      setIsOpen(false);
                    }}
                    className={`px-3 py-2 cursor-pointer transition-colors
                      ${opt.value === value 
                        ? (isDark ? 'bg-primary-500 text-white font-medium' : 'bg-primary-50 text-primary-700 font-medium')
                        : (isDark ? 'text-neutral-300 hover:bg-neutral-700' : 'text-neutral-700 hover:bg-neutral-100')
                      }
                    `}
                  >
                    {opt.label}
                  </li>
                ))
              )}
            </ul>
          </div>
        )}
      </div>
    </FormField>
  );
};

export default CustomDropdown;

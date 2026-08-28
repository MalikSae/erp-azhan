import React, { useState, useRef, useEffect, Children } from 'react';
import FormField from './FormField';

const CustomDropdown = ({ 
  label, 
  name,
  options = [], 
  value, 
  onChange, 
  placeholder = 'Pilih...', 
  error, 
  required, 
  className = '',
  disabled = false,
  variant = 'light',
  children
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef(null);

  // Parse options from children if provided
  let normalizedOptions = [...options];
  if (children && normalizedOptions.length === 0) {
    Children.forEach(children, (child) => {
      if (child && child.type === 'option') {
        normalizedOptions.push({
          value: child.props.value !== undefined ? child.props.value : child.props.children,
          label: child.props.children
        });
      }
    });
  }

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

  const selectedOption = normalizedOptions.find(opt => String(opt.value) === String(value));
  const isDark = variant === 'dark';

  const handleSelect = (optValue) => {
    if (disabled) return;
    if (onChange) {
      if (name) {
        onChange({ target: { name, value: optValue } });
      } else {
        onChange(optValue);
      }
    }
    setIsOpen(false);
  };

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <div className="relative" ref={dropdownRef}>
        {/* Dropdown Trigger */}
        <button
          type="button"
          disabled={disabled}
          onClick={() => !disabled && setIsOpen(!isOpen)}
          className={`w-full flex items-center justify-between rounded-xl border px-3.5 py-2.5 text-xs md:text-sm font-body text-left transition-all focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs
            ${disabled ? 'bg-neutral-50 text-neutral-400 border-neutral-200 cursor-not-allowed' : isDark ? 'bg-neutral-800' : 'bg-white cursor-pointer'}
            ${error 
              ? 'border-danger-500 focus:border-danger-500' 
              : isDark ? 'border-neutral-700' : 'border-neutral-200/90'
            }
            ${isOpen ? 'border-neutral-400 ring-2 ring-primary-500/40' : ''}
            ${!selectedOption 
              ? (isDark ? 'text-neutral-400' : 'text-neutral-400') 
              : (isDark ? 'text-white' : 'text-neutral-900 font-medium')
            }
          `}
        >
          <span className="truncate">
            {selectedOption ? selectedOption.label : placeholder}
          </span>
          <svg 
            className={`w-4 h-4 transition-transform duration-200 shrink-0 ml-2 ${isDark ? 'text-neutral-500' : 'text-neutral-400'} ${isOpen ? 'transform rotate-180 text-neutral-800' : ''}`} 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {/* Dropdown Menu */}
        {isOpen && !disabled && (
          <div className={`absolute z-30 w-full mt-1.5 border rounded-xl shadow-card overflow-hidden animate-in fade-in slide-in-from-top-1 duration-150
            ${isDark ? 'bg-neutral-800 border-neutral-700' : 'bg-white border-neutral-200/80'}
          `}>
            <ul className="max-h-60 overflow-y-auto py-1 text-xs md:text-sm font-body">
              {normalizedOptions.length === 0 ? (
                <li className={`px-3.5 py-2.5 text-center text-xs ${isDark ? 'text-neutral-500' : 'text-neutral-500'}`}>Tidak ada pilihan</li>
              ) : (
                normalizedOptions.map((opt, idx) => {
                  const isSelected = String(opt.value) === String(value);
                  return (
                    <li
                      key={idx}
                      onClick={() => handleSelect(opt.value)}
                      className={`px-3.5 py-2.5 cursor-pointer transition-colors
                        ${isSelected 
                          ? (isDark ? 'bg-neutral-700 text-primary-500 font-bold' : 'bg-primary-100 text-neutral-900 font-bold')
                          : (isDark ? 'text-neutral-300 hover:bg-neutral-700' : 'text-neutral-700 hover:bg-neutral-50')
                        }
                      `}
                    >
                      {opt.label}
                    </li>
                  );
                })
              )}
            </ul>
          </div>
        )}
      </div>
    </FormField>
  );
};

export default CustomDropdown;

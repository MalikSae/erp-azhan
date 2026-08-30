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
  disabled = false,
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

  const selectedOption = options.find(opt => String(opt.value) === String(value));

  const isDark = variant === 'dark';
  const activeIcon = prefixIcon || icon;
  const paddingLeftClass = activeIcon ? 'pl-9 pr-3' : 'px-3.5';

  const getBadgeClasses = (variantType) => {
    switch (variantType) {
      case 'danger':
        return 'bg-danger-100 text-danger-700 border border-danger-200/60';
      case 'warning':
        return 'bg-amber-100 text-amber-800 border border-amber-200/60';
      case 'neutral':
        return 'bg-neutral-100 text-neutral-700 border border-neutral-200/60';
      case 'success':
      default:
        return 'bg-emerald-100 text-emerald-800 border border-emerald-200/60';
    }
  };

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
          disabled={disabled}
          onClick={() => !disabled && setIsOpen(!isOpen)}
          className={`h-11 w-full min-w-0 flex items-center justify-between rounded-xl border text-xs md:text-sm font-body text-left transition-all focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs
            ${paddingLeftClass}
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
          <div className="flex items-center justify-between gap-2 min-w-0 flex-1 mr-2">
            <span className="truncate">
              {selectedOption ? selectedOption.label : placeholder}
            </span>
            {selectedOption?.badge && (
              <span className={`shrink-0 px-2 py-0.5 rounded-md text-[11px] font-bold ${getBadgeClasses(selectedOption.badgeVariant)}`}>
                {selectedOption.badge}
              </span>
            )}
          </div>
          <svg 
            className={`w-4 h-4 transition-transform duration-200 shrink-0 ${isDark ? 'text-neutral-500' : 'text-neutral-400'} ${isOpen ? 'transform rotate-180' : ''}`} 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {/* Dropdown Menu */}
        {isOpen && !disabled && (
          <div className={`absolute z-50 min-w-full w-max max-w-lg mt-1.5 border rounded-xl shadow-card overflow-hidden animate-in fade-in slide-in-from-top-1 duration-150
            ${isDark ? 'bg-neutral-800 border-neutral-700' : 'bg-white border-neutral-200/80'}
          `}>
            <ul className="max-h-60 overflow-y-auto py-1 text-xs md:text-sm font-body">
              {options.length === 0 ? (
                <li className={`px-3.5 py-2.5 text-center text-xs ${isDark ? 'text-neutral-500' : 'text-neutral-400'} italic`}>
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
                    className={`px-3.5 py-2.5 cursor-pointer transition-colors
                      ${String(opt.value) === String(value)
                        ? (isDark ? 'bg-neutral-700 text-primary-500 font-bold' : 'bg-primary-100 text-neutral-900 font-bold')
                        : (isDark ? 'text-neutral-300 hover:bg-neutral-700' : 'text-neutral-700 hover:bg-neutral-50')
                      }
                    `}
                  >
                    <div className="flex items-center justify-between gap-3 w-full">
                      <span className="truncate">{opt.label}</span>
                      {opt.badge && (
                        <span className={`shrink-0 px-2 py-0.5 rounded-md text-[11px] font-bold ${getBadgeClasses(opt.badgeVariant)}`}>
                          {opt.badge}
                        </span>
                      )}
                    </div>
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

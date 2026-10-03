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
  const [openUp, setOpenUp] = useState(false);
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
    
    const handleEscape = (event) => {
      if (event.key === 'Escape') setIsOpen(false);
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, []);

  const actualValue = (value && typeof value === 'object' && 'target' in value) 
    ? value.target.value 
    : value;

  const selectedOption = normalizedOptions.find(opt => String(opt.value) === String(actualValue));
  const isDark = variant === 'dark';

  // Buka ke atas bila ruang di bawah (dalam area scroll terdekat) tidak cukup,
  // agar menu tidak memanjangkan halaman atau tertutup bar sticky di bawah.
  const toggleOpen = () => {
    if (disabled) return;
    if (!isOpen && dropdownRef.current) {
      const rect = dropdownRef.current.getBoundingClientRect();
      let el = dropdownRef.current.parentElement;
      let bottomEdge = window.innerHeight;
      let topEdge = 0;
      while (el && el !== document.body) {
        const oy = getComputedStyle(el).overflowY;
        if (oy === 'auto' || oy === 'scroll') {
          const r = el.getBoundingClientRect();
          bottomEdge = Math.min(bottomEdge, r.bottom);
          topEdge = Math.max(topEdge, r.top);
          break;
        }
        el = el.parentElement;
      }
      const reserve = 80; // ruang untuk bar aksi sticky
      const menuHeight = Math.min(248, Math.max(1, normalizedOptions.length) * 40 + 10);
      const spaceBelow = bottomEdge - rect.bottom - reserve;
      const spaceAbove = rect.top - topEdge;
      setOpenUp(spaceBelow < menuHeight && spaceAbove > spaceBelow);
    }
    setIsOpen(!isOpen);
  };

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
      <div className="relative" ref={dropdownRef}>
        {/* Dropdown Trigger */}
        <button
          type="button"
          disabled={disabled}
          aria-haspopup="listbox"
          aria-expanded={isOpen}
          onClick={toggleOpen}
          className={`h-10 w-full min-w-0 flex items-center justify-between rounded-lg border px-3.5 text-[13px] font-body text-left transition-all focus:outline-none
            ${disabled ? 'bg-neutral-50 text-neutral-400 border-neutral-200 cursor-not-allowed' : isDark ? 'bg-neutral-800' : 'bg-white cursor-pointer'}
            ${error 
              ? 'border-danger-500 focus:border-danger-500' 
              : isDark ? 'border-neutral-700' : 'border-neutral-200/90'
            }
            
            ${!selectedOption 
              ? (isDark ? 'text-neutral-400' : 'text-neutral-400') 
              : (isDark ? 'text-white' : 'text-neutral-900')
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
            className={`w-4 h-4 transition-transform duration-200 shrink-0 ${isDark ? 'text-neutral-500' : 'text-neutral-400'} ${isOpen ? 'transform rotate-180 text-primary-500' : ''}`} 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {/* Dropdown Menu */}
        {isOpen && !disabled && (
          <div className={`absolute z-50 w-max min-w-full max-w-[360px] border rounded-lg shadow-card overflow-hidden animate-in fade-in duration-150
            ${openUp ? 'bottom-full mb-1.5 slide-in-from-bottom-1' : 'top-full mt-1.5 slide-in-from-top-1'}
            ${isDark ? 'bg-neutral-800 border-neutral-700' : 'bg-white border-neutral-200/80'}
          `}>
            <ul className="max-h-60 overflow-y-auto py-1 text-[13px] font-body [scrollbar-width:thin]">
              {normalizedOptions.length === 0 ? (
                <li className={`px-3.5 py-2.5 text-center text-xs whitespace-nowrap ${isDark ? 'text-neutral-500' : 'text-neutral-500'}`}>Tidak ada pilihan</li>
              ) : (
                normalizedOptions.map((opt, idx) => {
                  const isSelected = String(opt.value) === String(value);
                  return (
                    <li
                      key={idx}
                      onClick={() => handleSelect(opt.value)}
                      title={typeof opt.label === 'string' ? opt.label : undefined}
                      className={`px-3.5 py-2.5 cursor-pointer transition-colors
                        ${isSelected 
                          ? (isDark ? 'bg-neutral-700 text-primary-500 font-medium' : 'bg-primary-50 text-primary-600 font-medium')
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

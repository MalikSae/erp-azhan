import React, { useState, useRef, useEffect, Children } from 'react';
import FormField from './FormField';
import { ChevronDown } from 'lucide-react';

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
          className={`h-11 w-full min-w-0 flex items-center justify-between rounded-md border px-3 text-sm font-body text-left transition-colors focus:outline-none focus:ring-1 focus:ring-primary-500
            ${disabled ? 'bg-neutral-50 text-neutral-400 border-neutral-200 cursor-not-allowed' : 'bg-white cursor-pointer'}
            ${error 
              ? 'border-danger-500 focus:border-danger-500 ring-danger-500' 
              : 'border-neutral-300 focus:border-primary-500'
            }
            ${isOpen ? 'border-primary-500 ring-1 ring-primary-500' : ''}
            ${!selectedOption ? 'text-neutral-400' : 'text-neutral-900 font-medium'}
          `}
        >
          <span className="truncate">
            {selectedOption ? selectedOption.label : placeholder}
          </span>
          <ChevronDown 
            size={16} 
            className={`transition-transform duration-200 text-neutral-400 shrink-0 ml-2 ${isOpen ? 'rotate-180 text-primary-600' : ''}`} 
          />
        </button>

        {/* Dropdown Menu */}
        {isOpen && !disabled && (
          <div className="absolute z-30 w-full mt-1 border border-neutral-200 bg-white rounded-md shadow-lg overflow-hidden animate-in fade-in slide-in-from-top-1 duration-150">
            <ul className="max-h-60 overflow-y-auto py-1 text-sm font-body divide-y divide-neutral-50">
              {normalizedOptions.length === 0 ? (
                <li className="px-3 py-2 text-center text-neutral-400 text-xs">Tidak ada pilihan</li>
              ) : (
                normalizedOptions.map((opt, idx) => {
                  const isSelected = String(opt.value) === String(value);
                  return (
                    <li
                      key={idx}
                      onClick={() => handleSelect(opt.value)}
                      className={`px-3 py-2.5 cursor-pointer transition-colors flex items-center justify-between text-xs sm:text-sm
                        ${isSelected 
                          ? 'bg-primary-50 text-primary-700 font-semibold' 
                          : 'text-neutral-700 hover:bg-neutral-50 hover:text-neutral-900'
                        }
                      `}
                    >
                      <span className="truncate">{opt.label}</span>
                      {isSelected && (
                        <span className="w-1.5 h-1.5 rounded-full bg-primary-600 ml-2 shrink-0" />
                      )}
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

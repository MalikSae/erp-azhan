import React, { useState, useRef, useEffect, useMemo } from 'react';
import FormField from './FormField';
import { X } from 'lucide-react';

const AutocompleteInput = ({
  label,
  name,
  value = '',
  onChange,
  options = [],
  placeholder = 'Ketik nama kota/kabupaten...',
  error,
  required,
  className = '',
  disabled,
  readOnly,
  minChars = 3,
  maxSuggestions = 15,
  ...props
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const containerRef = useRef(null);
  const listRef = useRef(null);
  const inputRef = useRef(null);

  const query = (value || '').trim().toLowerCase();
  const hasMinChars = query.length >= minChars;

  // Filter options berdasarkan input
  const filteredOptions = useMemo(() => {
    if (!hasMinChars) {
      return [];
    }
    return options
      .filter((opt) => {
        const text = typeof opt === 'string' ? opt : opt.label;
        return text.toLowerCase().includes(query);
      })
      .slice(0, maxSuggestions);
  }, [query, hasMinChars, options, maxSuggestions]);

  // Click outside listener
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (containerRef.current && !containerRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelect = (selectedVal) => {
    const val = typeof selectedVal === 'string' ? selectedVal : selectedVal.value || selectedVal.label;
    if (onChange) {
      onChange({ target: { name, value: val } });
    }
    setIsOpen(false);
    setHighlightedIndex(-1);
  };

  const handleKeyDown = (e) => {
    if (!isOpen && hasMinChars && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      setIsOpen(true);
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlightedIndex((prev) => 
        prev < filteredOptions.length - 1 ? prev + 1 : 0
      );
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlightedIndex((prev) => 
        prev > 0 ? prev - 1 : filteredOptions.length - 1
      );
    } else if (e.key === 'Enter') {
      if (isOpen && highlightedIndex >= 0 && highlightedIndex < filteredOptions.length) {
        e.preventDefault();
        handleSelect(filteredOptions[highlightedIndex]);
      }
    } else if (e.key === 'Escape') {
      setIsOpen(false);
    }
  };

  // Scroll active item into view
  useEffect(() => {
    if (highlightedIndex >= 0 && listRef.current) {
      const activeEl = listRef.current.children[highlightedIndex];
      if (activeEl) {
        activeEl.scrollIntoView({ block: 'nearest' });
      }
    }
  }, [highlightedIndex]);

  const inputBaseClasses = 'w-full rounded-md border border-neutral-300 px-3 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body transition-colors';
  const disabledClasses = disabled || readOnly ? 'bg-neutral-50 text-neutral-500 cursor-not-allowed pointer-events-none' : '';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500' : '';

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <div className="relative" ref={containerRef}>
        <div className="relative">
          <input
            ref={inputRef}
            type="text"
            name={name}
            value={value}
            onChange={(e) => {
              onChange && onChange(e);
              setIsOpen(true);
              setHighlightedIndex(0);
            }}
            onFocus={() => {
              if (!disabled && !readOnly && hasMinChars) {
                setIsOpen(true);
              }
            }}
            onKeyDown={handleKeyDown}
            required={required}
            placeholder={placeholder}
            disabled={disabled}
            readOnly={readOnly}
            autoComplete="off"
            className={`${inputBaseClasses} ${errorClasses} ${disabledClasses} ${value ? 'pr-8' : ''}`}
            {...props}
          />

          {value && !disabled && !readOnly && (
            <button
              type="button"
              onClick={() => {
                if (onChange) {
                  onChange({ target: { name, value: '' } });
                }
                inputRef.current?.focus();
              }}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600 transition-colors"
              tabIndex={-1}
            >
              <X size={14} />
            </button>
          )}
        </div>

        {/* Dropdown Suggestions */}
        {isOpen && !disabled && !readOnly && hasMinChars && filteredOptions.length > 0 && (
          <div className="absolute z-50 w-full mt-1 bg-white border border-neutral-200 rounded-md shadow-lg overflow-hidden animate-in fade-in slide-in-from-top-1 duration-150">
            <ul ref={listRef} className="max-h-60 overflow-y-auto py-1 text-sm font-body divide-y divide-neutral-50">
              {filteredOptions.map((opt, idx) => {
                const labelText = typeof opt === 'string' ? opt : opt.label;
                const isSelected = labelText === value;
                const isHighlighted = idx === highlightedIndex;

                return (
                  <li
                    key={idx}
                    onMouseDown={(e) => {
                      // Gunakan onMouseDown agar tidak ter-trigger blur duluan
                      e.preventDefault();
                      handleSelect(opt);
                    }}
                    onMouseEnter={() => setHighlightedIndex(idx)}
                    className={`px-3.5 py-2 cursor-pointer transition-colors
                      ${isHighlighted || isSelected
                        ? 'bg-primary-50 text-primary-900 font-medium'
                        : 'text-neutral-700 hover:bg-neutral-50'
                      }
                    `}
                  >
                    <span className="truncate block">{labelText}</span>
                  </li>
                );
              })}
            </ul>
          </div>
        )}
      </div>
    </FormField>
  );
};

export default AutocompleteInput;

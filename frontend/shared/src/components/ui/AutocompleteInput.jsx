import React, { useState, useRef, useEffect, useMemo } from 'react';
import FormField from './FormField';
import { X } from 'lucide-react';

const AutocompleteInput = ({
  label,
  name,
  value = '',
  onChange,
  onSelect,
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

  const query = String(value ?? '').trim().toLowerCase();
  const hasMinChars = minChars === 0 ? true : query.length >= minChars;

  // Filter options berdasarkan input
  const filteredOptions = useMemo(() => {
    if (!hasMinChars) {
      return [];
    }
    if (!query) {
      return options.slice(0, maxSuggestions);
    }
    return options
      .filter((opt) => {
        const text = typeof opt === 'string' ? opt : (opt.label || String(opt.value || ''));
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
    if (onSelect) {
      onSelect(selectedVal);
    }
    const val = typeof selectedVal === 'string' ? selectedVal : (selectedVal?.label || String(selectedVal?.value ?? ''));
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

  const inputBaseClasses = 'h-11 w-full min-w-0 rounded-xl border border-neutral-200/90 bg-white px-3.5 text-xs md:text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs font-body transition-all';
  const disabledClasses = disabled || readOnly ? 'bg-neutral-50 text-neutral-500 cursor-not-allowed pointer-events-none' : '';
  const errorClasses = error ? 'border-danger-500 focus:border-danger-500 focus:ring-danger-500/20' : '';

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
                if (onSelect) {
                  onSelect(null);
                }
                inputRef.current?.focus();
              }}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 p-1 text-neutral-400 hover:text-neutral-600 rounded-full cursor-pointer"
              tabIndex={-1}
            >
              <X size={14} />
            </button>
          )}
        </div>

        {/* Dropdown Suggestions */}
        {isOpen && hasMinChars && filteredOptions.length > 0 && (
          <ul
            ref={listRef}
            className="absolute z-50 mt-1.5 max-h-60 w-full overflow-auto rounded-xl bg-white py-1.5 text-xs md:text-sm shadow-card border border-neutral-200/80 focus:outline-none"
            role="listbox"
          >
            {filteredOptions.map((opt, idx) => {
              const text = typeof opt === 'string' ? opt : opt.label;
              const isHighlighted = idx === highlightedIndex;

              return (
                <li
                  key={idx}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    handleSelect(opt);
                  }}
                  onMouseEnter={() => setHighlightedIndex(idx)}
                  className={`cursor-pointer select-none px-3.5 py-2.5 transition-colors ${
                    isHighlighted ? 'bg-neutral-100 text-neutral-900 font-bold' : 'text-neutral-700 hover:bg-neutral-50'
                  }`}
                  role="option"
                  aria-selected={isHighlighted}
                >
                  {text}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </FormField>
  );
};

export default AutocompleteInput;

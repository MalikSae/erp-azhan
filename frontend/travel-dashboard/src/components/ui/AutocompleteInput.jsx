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
  placeholder = 'Ketik untuk mencari...',
  error,
  required,
  className = '',
  disabled,
  readOnly,
  minChars = 0,
  maxSuggestions = 30,
  prefixIcon,
  ...props
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const [isTyping, setIsTyping] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const containerRef = useRef(null);
  const listRef = useRef(null);
  const inputRef = useRef(null);

  // Cari option yang cocok dengan value saat ini (baik string maupun ID)
  const selectedOption = useMemo(() => {
    if (value === '' || value === null || value === undefined) return null;
    return options.find((opt) => {
      if (typeof opt === 'string') return opt === value;
      return String(opt.value) === String(value);
    }) || null;
  }, [value, options]);

  // Label tampilan di input
  const selectedLabel = selectedOption
    ? (typeof selectedOption === 'string' ? selectedOption : selectedOption.label)
    : (value !== null && value !== undefined ? String(value) : '');

  const inputValue = isTyping ? searchQuery : selectedLabel;

  const query = String(isTyping ? searchQuery : '').trim().toLowerCase();
  const hasMinChars = minChars === 0 ? true : query.length >= minChars;

  // Filter options berdasarkan input
  const filteredOptions = useMemo(() => {
    if (!hasMinChars) {
      return [];
    }
    if (!isTyping || !query) {
      return options.slice(0, maxSuggestions);
    }
    return options
      .filter((opt) => {
        const text = typeof opt === 'string' ? opt : `${opt.label || ''} ${opt.value || ''}`;
        return text.toLowerCase().includes(query);
      })
      .slice(0, maxSuggestions);
  }, [query, isTyping, hasMinChars, options, maxSuggestions]);

  // Click outside listener
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (containerRef.current && !containerRef.current.contains(event.target)) {
        setIsOpen(false);
        setIsTyping(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelect = (opt) => {
    setIsTyping(false);
    setSearchQuery('');
    setIsOpen(false);
    setHighlightedIndex(-1);

    if (onSelect) {
      onSelect(opt);
    }
    const val = typeof opt === 'string' 
      ? opt 
      : (opt?.value !== undefined ? opt.value : (opt?.label || ''));
    if (onChange) {
      onChange({ target: { name, value: val } });
    }
  };

  const handleKeyDown = (e) => {
    if (!isOpen && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
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
      setIsTyping(false);
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
  const paddingLeftClass = prefixIcon ? 'pl-10' : '';

  return (
    <FormField label={label} error={error} required={required} className={className}>
      <div className="relative" ref={containerRef}>
        <div className="relative">
          {prefixIcon && (
            <div className="absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-400 pointer-events-none z-10">
              {prefixIcon}
            </div>
          )}

          <input
            ref={inputRef}
            type="text"
            name={name}
            value={inputValue}
            onChange={(e) => {
              const val = e.target.value;
              setIsTyping(true);
              setSearchQuery(val);
              setIsOpen(true);
              setHighlightedIndex(0);

              if (!val.trim()) {
                if (onChange) {
                  onChange({ target: { name, value: '' } });
                }
                if (onSelect) {
                  onSelect(null);
                }
              }
            }}
            onFocus={() => {
              if (!disabled && !readOnly) {
                setIsOpen(true);
              }
            }}
            onKeyDown={handleKeyDown}
            required={required}
            placeholder={placeholder}
            disabled={disabled}
            readOnly={readOnly}
            autoComplete="off"
            aria-autocomplete="list"
            aria-expanded={isOpen && filteredOptions.length > 0}
            aria-haspopup="listbox"
            aria-invalid={!!error}
            className={`${inputBaseClasses} ${errorClasses} ${disabledClasses} ${paddingLeftClass} ${inputValue ? 'pr-8' : ''}`}
            {...props}
          />

          {inputValue && !disabled && !readOnly && (
            <button
              type="button"
              onClick={() => {
                setIsTyping(false);
                setSearchQuery('');
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
        {isOpen && filteredOptions.length > 0 && (
          <ul
            ref={listRef}
            className="absolute z-50 mt-1.5 max-h-60 w-full overflow-auto rounded-xl bg-white py-1.5 text-xs md:text-sm shadow-card border border-neutral-200/80 focus:outline-none"
            role="listbox"
          >
            {filteredOptions.map((opt, idx) => {
              const text = typeof opt === 'string' ? opt : opt.label;
              const isSelected = selectedOption && (
                typeof opt === 'string' 
                  ? opt === value 
                  : String(opt.value) === String(selectedOption.value || value)
              );
              const isHighlighted = idx === highlightedIndex;

              return (
                <li
                  key={idx}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    handleSelect(opt);
                  }}
                  onMouseEnter={() => setHighlightedIndex(idx)}
                  className={`cursor-pointer select-none px-3.5 py-2.5 transition-colors flex items-center justify-between ${
                    isHighlighted ? 'bg-neutral-100 text-neutral-900 font-bold' : isSelected ? 'bg-primary-50 text-neutral-900 font-medium' : 'text-neutral-700 hover:bg-neutral-50'
                  }`}
                  role="option"
                  aria-selected={isHighlighted}
                >
                  <span>{text}</span>
                  {isSelected && (
                    <span className="text-xs text-primary-700 font-semibold">✓</span>
                  )}
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

import React, { useState, useRef, useEffect } from 'react';
import { Ellipsis, ChevronDown } from 'lucide-react';

const MENU_WIDTH = 176; // w-44
const ITEM_HEIGHT = 32;

// Menu aksi baris tabel: tombol kotak berbingkai dengan ikon "⋯" (gaya SmartHR).
// Menu memakai posisi fixed agar tidak terpotong oleh kontainer tabel yang bisa digulir.
const ActionMenu = ({ items = [], align = 'right', label }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const menuRef = useRef(null);
  const buttonRef = useRef(null);

  const place = () => {
    const rect = buttonRef.current?.getBoundingClientRect();
    if (!rect) return;
    const menuHeight = items.length * ITEM_HEIGHT + 8;
    const left = align === 'right' ? rect.right - MENU_WIDTH : rect.left;
    const opensUp = rect.bottom + 4 + menuHeight > window.innerHeight && rect.top - 4 - menuHeight > 0;
    setPos({
      top: opensUp ? rect.top - 4 - menuHeight : rect.bottom + 4,
      left: Math.max(8, Math.min(left, window.innerWidth - MENU_WIDTH - 8)),
    });
  };

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (menuRef.current && !menuRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };

    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
      }
    };

    const close = () => setIsOpen(false);

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleKeyDown);
      window.addEventListener('scroll', close, true);
      window.addEventListener('resize', close);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('scroll', close, true);
      window.removeEventListener('resize', close);
    };
  }, [isOpen]);

  return (
    <div className="relative inline-block text-left" ref={menuRef}>
      <button
        ref={buttonRef}
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          if (!isOpen) place();
          setIsOpen((prev) => !prev);
        }}
        title="Pilihan Aksi"
        aria-haspopup="menu"
        aria-expanded={isOpen}
        className={`${
          label ? 'h-[38px] px-4 gap-2 rounded-xl text-sm font-heading font-semibold shadow-2xs' : 'w-[30px] h-[30px] rounded-lg'
        } border flex items-center justify-center transition-colors cursor-pointer ${
          isOpen
            ? 'border-primary-200 bg-primary-50 text-primary-600'
            : 'border-neutral-200 bg-white text-neutral-500 hover:text-neutral-800 hover:bg-neutral-50'
        }`}
      >
        {label ? (
          <>
            <span>{label}</span>
            <ChevronDown size={16} strokeWidth={1.75} />
          </>
        ) : (
          <Ellipsis size={16} strokeWidth={1.75} />
        )}
      </button>

      {isOpen && (
        <div
          role="menu"
          style={{ position: 'fixed', top: pos.top, left: pos.left, width: MENU_WIDTH }}
          className="rounded-xl bg-white border border-neutral-200/90 shadow-lg py-1 z-50 animate-in fade-in-0 zoom-in-95 font-body"
        >
          {items.map((item, index) => {
            const Icon = item.icon;
            if (item.disabled) {
              return (
                <div
                  key={index}
                  role="menuitem"
                  aria-disabled="true"
                  title={item.disabledTooltip || item.tooltip || ''}
                  className="flex items-center gap-2 px-3 py-1.5 text-xs text-neutral-300 cursor-not-allowed select-none"
                >
                  {Icon && <Icon size={14} className="shrink-0 text-neutral-300" />}
                  <span>{item.label}</span>
                </div>
              );
            }

            return (
              <button
                key={index}
                type="button"
                role="menuitem"
                onClick={(e) => {
                  e.stopPropagation();
                  setIsOpen(false);
                  item.onClick && item.onClick();
                }}
                title={item.tooltip || ''}
                className={`flex items-center gap-2 w-full px-3 py-1.5 text-xs font-medium text-left transition-colors cursor-pointer ${
                  item.danger
                    ? 'text-danger-600 hover:bg-danger-50 hover:text-danger-700'
                    : 'text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900'
                }`}
              >
                {Icon && (
                  <Icon
                    size={14}
                    className={`shrink-0 ${
                      item.danger ? 'text-danger-500' : 'text-neutral-500'
                    }`}
                  />
                )}
                <span>{item.label}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default ActionMenu;

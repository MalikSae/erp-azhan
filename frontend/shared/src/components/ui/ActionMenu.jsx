import React, { useState, useRef, useEffect } from 'react';
import { MoreVertical } from 'lucide-react';

const ActionMenu = ({ items = [], align = 'right' }) => {
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef(null);

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

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleKeyDown);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen]);

  return (
    <div className="relative inline-block text-left" ref={menuRef}>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setIsOpen((prev) => !prev);
        }}
        title="Pilihan Aksi"
        className="w-7 h-7 rounded-lg flex items-center justify-center text-neutral-400 hover:text-neutral-800 hover:bg-neutral-100 transition-colors cursor-pointer"
      >
        <MoreVertical size={16} />
      </button>

      {isOpen && (
        <div
          className={`absolute ${
            align === 'right' ? 'right-0' : 'left-0'
          } mt-1 w-36 rounded-xl bg-white border border-neutral-200/90 shadow-lg py-1 z-50 animate-in fade-in-0 zoom-in-95 font-body`}
        >
          {items.map((item, index) => {
            const Icon = item.icon;
            if (item.disabled) {
              return (
                <div
                  key={index}
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

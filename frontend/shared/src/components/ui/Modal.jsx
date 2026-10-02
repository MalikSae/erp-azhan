import React, { useEffect } from 'react';
import { X } from 'lucide-react';

// Modal gaya SmartHR: dialog putih radius 10px tanpa border, backdrop redup
// polos, header kompak dengan tombol tutup kotak berbingkai, footer putih.
const Modal = ({ isOpen, onClose, title, children, footer, size = 'md' }) => {
  // Prevent scrolling on body when modal is open
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.body.style.overflow = 'hidden';
      document.addEventListener('keydown', handleKeyDown);
    } else {
      document.body.style.overflow = 'unset';
    }

    return () => {
      document.body.style.overflow = 'unset';
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const sizeClasses = {
    sm: 'w-full max-w-sm',
    md: 'w-full max-w-md',
    lg: 'w-full max-w-3xl', // ~768px, suitable for nested forms
    xl: 'w-full max-w-xl',
    '2xl': 'w-full max-w-2xl',
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex min-h-full items-center justify-center p-4 sm:p-0">
        {/* Backdrop */}
        <div
          className="fixed inset-0 bg-neutral-900/60 transition-opacity animate-in fade-in duration-200"
          onClick={onClose}
          aria-hidden="true"
        />

        {/* Modal Dialog */}
        <div
          className={`bg-white rounded-[10px] shadow-xl ${sizeClasses[size]} relative z-50 animate-in fade-in zoom-in-95 duration-150 my-8`}
          role="dialog"
          aria-modal="true"
        >
          {/* Header */}
          <div className="px-5 py-4 border-b border-neutral-200 flex items-center justify-between gap-4">
            <h3 className="text-[15px] md:text-base font-bold font-heading text-neutral-900 truncate">
              {title}
            </h3>
            <button
              onClick={onClose}
              className="w-8 h-8 shrink-0 flex items-center justify-center rounded-lg border border-neutral-200 text-neutral-500 hover:text-neutral-800 hover:bg-neutral-50 transition-colors focus:outline-none"
              aria-label="Close modal"
            >
              <X className="w-4 h-4" strokeWidth={2} />
            </button>
          </div>

          {/* Body */}
          <div className="p-5">
            {children}
          </div>

          {/* Footer */}
          {footer && (
            <div className="px-5 py-4 border-t border-neutral-200 flex items-center justify-end gap-2.5">
              {footer}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Modal;

import React, { useEffect } from 'react';

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
          className="fixed inset-0 bg-neutral-900/50 backdrop-blur-sm transition-opacity animate-in fade-in duration-200"
          onClick={onClose}
          aria-hidden="true"
        />

        {/* Modal Dialog */}
        <div 
          className={`bg-white rounded-xl shadow-xl ${sizeClasses[size]} relative z-50 overflow-visible animate-in fade-in zoom-in-95 duration-200 my-8`}
        role="dialog"
        aria-modal="true"
      >
        {/* Header */}
        <div className="px-6 py-4 border-b border-neutral-200 flex items-center justify-between bg-white">
          <h3 className="text-lg font-semibold font-heading text-neutral-900">
            {title}
          </h3>
          <button
            onClick={onClose}
            className="text-neutral-400 hover:text-neutral-600 hover:bg-neutral-100 p-1.5 rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
            aria-label="Close modal"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-4">
          {children}
        </div>

        {/* Footer */}
        {footer && (
          <div className="px-6 py-4 border-t border-neutral-200 bg-neutral-50 flex justify-end gap-3 rounded-b-xl">
            {footer}
          </div>
        )}
      </div>
    </div>
    </div>
  );
};

export default Modal;

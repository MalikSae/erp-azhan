import React, { useState, useRef, useEffect } from 'react';
import { useAuth } from '../../context/AuthContext';
import { LogOut } from "lucide-react";

const Topbar = ({ toggleSidebar }) => {
  const { logout } = useAuth();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <header className="bg-white border-b border-neutral-200 h-16 flex items-center justify-between px-4 md:px-8 shrink-0 sticky top-0 z-30">
      {/* Mobile Hamburger Menu */}
      <button 
        className="md:hidden p-2 -ml-2 text-neutral-600 hover:bg-neutral-100 rounded-md transition-colors focus:outline-none"
        onClick={toggleSidebar}
      >
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
           <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      {/* Desktop empty spacer to keep flex-between working if needed */}
      <div className="hidden md:block"></div>

      {/* Right Side - User Menu */}
      <div className="relative" ref={dropdownRef}>
        <button
          onClick={() => setDropdownOpen(!dropdownOpen)}
          className="flex items-center gap-2 hover:bg-neutral-50 p-1.5 pr-2 rounded-md transition-colors focus:outline-none border border-transparent hover:border-neutral-200"
        >
          <div className="w-8 h-8 bg-primary-100 text-primary-700 rounded-full flex items-center justify-center font-bold font-heading text-sm">
            T
          </div>
          <span className="text-sm font-medium text-neutral-700 hidden sm:block">Admin Travel</span>
          <svg className={`w-4 h-4 text-neutral-500 transition-transform ${dropdownOpen ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {dropdownOpen && (
          <div className="absolute right-0 mt-1 w-48 bg-white rounded-md shadow-lg border border-neutral-200 z-50 animate-in fade-in slide-in-from-top-2">
            <div className="py-1">
              <button
                onClick={() => {
                  setDropdownOpen(false);
                  logout();
                }}
                className="w-full text-left px-4 py-2 text-sm text-danger-600 hover:bg-danger-50 transition-colors flex items-center gap-2"
              >
                <LogOut size={16} />
                Logout
              </button>
            </div>
          </div>
        )}
      </div>
    </header>
  );
};

export default Topbar;

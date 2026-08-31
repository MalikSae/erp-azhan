import React, { useState, useRef, useEffect } from 'react';
import { useAuth } from '../../context/AuthContext';
import { LogOut } from "lucide-react";

function isLightColor(hexColor) {
  if (!hexColor || typeof hexColor !== 'string' || !hexColor.startsWith('#')) return true;
  const hex = hexColor.replace('#', '');
  if (hex.length !== 6 && hex.length !== 3) return true;
  const r = parseInt(hex.length === 3 ? hex[0] + hex[0] : hex.slice(0, 2), 16);
  const g = parseInt(hex.length === 3 ? hex[1] + hex[1] : hex.slice(2, 4), 16);
  const b = parseInt(hex.length === 3 ? hex[2] + hex[2] : hex.slice(4, 6), 16);
  const brightness = (r * 299 + g * 587 + b * 114) / 1000;
  return brightness > 140;
}

const Topbar = ({ toggleSidebar }) => {
  const { user, brandInfo, logout } = useAuth();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef(null);

  const brandColor = brandInfo?.primary_color || '#FED853';
  const isLight = isLightColor(brandColor);

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
    <header className="bg-white border-b border-neutral-200/80 h-16 flex items-center justify-between px-4 md:px-8 shrink-0 sticky top-0 z-30 font-body">
      {/* Mobile Hamburger Menu */}
      <button 
        className="md:hidden p-2 -ml-2 text-neutral-600 hover:bg-neutral-100 rounded-xl transition-colors focus:outline-none"
        onClick={toggleSidebar}
        aria-label="Buka Menu"
      >
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
           <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      {/* Desktop breadcrumb / brand indicator */}
      <div className="hidden md:flex items-center gap-2">
        {brandInfo?.name && (
          <span className="text-sm font-semibold text-neutral-700 font-heading">{brandInfo.name}</span>
        )}
      </div>

      {/* Right Side - User Menu */}
      <div className="relative" ref={dropdownRef}>
        <button
          onClick={() => setDropdownOpen(!dropdownOpen)}
          className="flex items-center gap-2.5 hover:bg-neutral-50 p-1.5 pr-2.5 rounded-xl transition-colors focus:outline-none border border-transparent hover:border-neutral-200 cursor-pointer"
        >
          <div 
            className="w-8 h-8 rounded-xl flex items-center justify-center font-bold font-heading text-xs shadow-2xs shrink-0"
            style={{ backgroundColor: brandColor, color: isLight ? '#14171A' : '#FFFFFF' }}
          >
            {brandInfo?.name ? brandInfo.name.charAt(0).toUpperCase() : 'T'}
          </div>
          <div className="text-left hidden sm:block">
            <span className="text-xs font-bold text-neutral-800 block leading-tight font-heading">
              {brandInfo?.name || "Admin Travel"}
            </span>
            <span className="text-[10px] text-neutral-500 font-medium block leading-tight truncate max-w-[140px]">
              {user?.email || "admin@travel.id"}
            </span>
          </div>
          <svg className={`w-3.5 h-3.5 text-neutral-400 transition-transform duration-200 ${dropdownOpen ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {dropdownOpen && (
          <div className="absolute right-0 mt-1.5 w-52 bg-white rounded-xl shadow-card border border-neutral-200/90 z-50 animate-in fade-in slide-in-from-top-1 duration-150 overflow-hidden">
            <div className="p-3 bg-neutral-50 border-b border-neutral-100">
              <p className="text-xs font-bold text-neutral-900 font-heading truncate">{brandInfo?.name || "Admin Travel"}</p>
              <p className="text-[11px] text-neutral-500 truncate mt-0.5">{user?.email}</p>
            </div>
            <div className="py-1">
              <button
                onClick={() => {
                  setDropdownOpen(false);
                  logout();
                }}
                className="w-full text-left px-3.5 py-2 text-xs font-medium text-danger-600 hover:bg-danger-50 transition-colors flex items-center gap-2 cursor-pointer"
              >
                <LogOut size={14} />
                <span>Keluar (Logout)</span>
              </button>
            </div>
          </div>
        )}
      </div>
    </header>
  );
};

export default Topbar;

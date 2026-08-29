import React, { useContext, useState, useRef, useEffect } from 'react';
import { AuthContext } from '../context/AuthContext';
import { Search, HelpCircle, Bell, ChevronDown, LogOut, ShieldCheck, Menu } from 'lucide-react';

const Topbar = ({ toggleSidebar }) => {
  const { logout, user } = useContext(AuthContext);
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
    <header className="bg-white border-b border-neutral-200/80 h-16 flex items-center justify-between px-4 md:px-8 shrink-0 sticky top-0 z-30 shadow-xs">
      {/* Mobile Hamburger Menu */}
      <button 
        type="button"
        className="md:hidden p-2 -ml-2 text-neutral-600 hover:bg-neutral-100 rounded-xl transition-colors focus:outline-none"
        onClick={toggleSidebar}
        aria-label="Toggle menu"
      >
        <Menu className="w-5 h-5" />
      </button>

      {/* Desktop Quick Search Placeholder */}
      <div className="hidden md:flex items-center gap-2 bg-neutral-50 border border-neutral-200/80 rounded-xl px-3.5 py-1.5 w-72 text-neutral-400 text-sm hover:border-neutral-300 transition-colors cursor-pointer">
        <Search className="w-4 h-4 text-neutral-400" />
        <span className="text-xs text-neutral-500 font-body">Cari paket, booking, jamaah...</span>
        <kbd className="ml-auto text-[10px] font-mono bg-white px-1.5 py-0.5 rounded border border-neutral-200 text-neutral-400 shadow-2xs">
          ⌘K
        </kbd>
      </div>

      {/* Right Side - Actions & User Profile */}
      <div className="flex items-center gap-2 sm:gap-3">
        {/* Help Button */}
        <button
          type="button"
          className="p-2 text-neutral-500 hover:text-neutral-800 hover:bg-neutral-100/80 rounded-xl transition-colors"
          title="Pusat Bantuan & Panduan"
        >
          <HelpCircle className="w-4.5 h-4.5" />
        </button>

        {/* Notification Bell with Badge */}
        <button
          type="button"
          className="p-2 text-neutral-500 hover:text-neutral-800 hover:bg-neutral-100/80 rounded-xl transition-colors relative"
          title="Notifikasi & Peringatan"
        >
          <Bell className="w-4.5 h-4.5" />
          <span className="absolute top-2 right-2 w-2 h-2 bg-danger-500 rounded-full ring-2 ring-white" />
        </button>

        <div className="h-6 w-[1px] bg-neutral-200 mx-1 hidden sm:block" />

        {/* User Profile Card */}
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={() => setDropdownOpen(!dropdownOpen)}
            className="flex items-center gap-3 p-1.5 pl-2 pr-2.5 rounded-xl hover:bg-neutral-50 transition-all border border-transparent hover:border-neutral-200/80"
          >
            <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-amber-500 to-primary-600 text-white flex items-center justify-center font-bold text-xs shadow-xs ring-2 ring-primary-100">
              {user?.name ? user.name.charAt(0).toUpperCase() : 'A'}
            </div>
            <div className="hidden sm:flex flex-col text-left leading-tight">
              <span className="text-xs font-bold text-neutral-900 font-heading">
                {user?.name || 'Admin'}
              </span>
              <span className="text-[10px] text-neutral-500 font-medium font-body flex items-center gap-1">
                <ShieldCheck className="w-3 h-3 text-primary-600 inline" />
                Super Admin
              </span>
            </div>
            <ChevronDown className={`w-3.5 h-3.5 text-neutral-400 transition-transform ${dropdownOpen ? 'rotate-180 text-neutral-600' : ''}`} />
          </button>

          {dropdownOpen && (
            <div className="absolute right-0 mt-2 w-56 bg-white rounded-xl shadow-lg border border-neutral-200/90 z-50 py-1.5 animate-in fade-in slide-in-from-top-2">
              <div className="px-4 py-2 border-b border-neutral-100">
                <p className="text-xs font-bold text-neutral-900 font-heading">{user?.name || 'Admin'}</p>
                <p className="text-[11px] text-neutral-500 font-body truncate">{user?.email || 'admin@azhan.group'}</p>
                <span className="inline-block mt-1 text-[10px] font-semibold uppercase tracking-wider bg-neutral-100 text-neutral-700 border border-neutral-200 px-2 py-0.5 rounded-md">
                  Super Admin
                </span>
              </div>
              <div className="py-1">
                <button
                  onClick={() => {
                    setDropdownOpen(false);
                    logout();
                  }}
                  className="w-full text-left px-4 py-2 text-xs font-medium text-danger-600 hover:bg-danger-50 transition-colors flex items-center gap-2"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  Keluar dari Sistem
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};

export default Topbar;

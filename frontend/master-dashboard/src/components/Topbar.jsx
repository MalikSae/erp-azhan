import React, { useContext, useState, useRef, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import { ChevronDown, LogOut, ShieldCheck, Menu, KeyRound } from 'lucide-react';
import TopbarSearch from './TopbarSearch';
import TopbarNotifikasi from './TopbarNotifikasi';

const Topbar = ({ toggleSidebar }) => {
  const { logout, user, brandInfo, isHolding } = useContext(AuthContext);
  const roleText = user?.roleLabel || 'Admin';
  const subText = isHolding ? roleText : `${roleText}${brandInfo?.name ? ` · ${brandInfo.name}` : ''}`;
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
        type="button"
        className="md:hidden p-2 -ml-2 text-neutral-600 hover:bg-neutral-100 rounded-xl transition-colors focus:outline-none"
        onClick={toggleSidebar}
        aria-label="Toggle menu"
      >
        <Menu className="w-5 h-5" />
      </button>

      {/* Pencarian global (desktop): paket, booking, jamaah. Ctrl+K / ⌘K. */}
      <TopbarSearch />

      {/* Right Side - Actions & User Profile */}
      <div className="flex items-center gap-2 sm:gap-3">
        {/* Lonceng: pekerjaan yang menunggu keputusan */}
        <TopbarNotifikasi />

        <div className="h-6 w-px bg-neutral-200 mx-1 hidden sm:block" />

        {/* User Profile Card */}
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={() => setDropdownOpen(!dropdownOpen)}
            className="flex items-center gap-3 p-1.5 pl-2 pr-2.5 rounded-xl hover:bg-neutral-50 transition-all border border-transparent hover:border-neutral-200"
          >
            <div className="w-8 h-8 rounded-full bg-primary-600 text-white flex items-center justify-center font-semibold text-xs ring-2 ring-primary-100">
              {user?.name ? user.name.charAt(0).toUpperCase() : 'A'}
            </div>
            <div className="hidden sm:flex flex-col text-left leading-tight">
              <span className="text-xs font-semibold text-neutral-900 font-heading">
                {user?.name || 'Admin'}
              </span>
              <span className="text-xs text-neutral-500 font-medium font-body flex items-center gap-1">
                <ShieldCheck className="w-3 h-3 text-primary-600 inline" />
                {subText}
              </span>
            </div>
            <ChevronDown className={`w-3.5 h-3.5 text-neutral-400 transition-transform ${dropdownOpen ? 'rotate-180 text-neutral-600' : ''}`} />
          </button>

          {dropdownOpen && (
            <div className="absolute right-0 mt-2 w-56 bg-white rounded-3xl shadow-popover border border-neutral-200 z-50 py-1.5 animate-in fade-in slide-in-from-top-2">
              <div className="px-4 py-2 border-b border-neutral-100">
                <p className="text-xs font-semibold text-neutral-900 font-heading">{user?.name || 'Admin'}</p>
                <p className="text-xs text-neutral-500 font-body truncate">{user?.email || 'admin@azhan.group'}</p>
                <span className="inline-block mt-1 text-xs font-semibold bg-neutral-100 text-neutral-700 border border-neutral-200 px-2 py-0.5 rounded-md">
                  {roleText}
                </span>
              </div>
              <div className="py-1">
                <Link
                  to="/pengaturan"
                  onClick={() => setDropdownOpen(false)}
                  className="w-full text-left px-4 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 transition-colors flex items-center gap-2"
                >
                  <KeyRound className="w-3.5 h-3.5" />
                  Pengaturan Akun
                </Link>
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

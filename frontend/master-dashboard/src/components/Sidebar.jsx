import React, { useState } from 'react';
import { NavLink } from 'react-router-dom';
import Badge from './ui/Badge';

const Sidebar = ({ isOpen, closeDrawer }) => {

  const [expandedGroups, setExpandedGroups] = useState({
    'MASTER PAKET': true,
  });

  const toggleGroup = (groupName) => {
    setExpandedGroups((prev) => ({
      ...prev,
      [groupName]: !prev[groupName],
    }));
  };

  const adminItems = [
    { name: 'Kelola Brand', path: '/brands', status: 'active' },
    { name: 'Rekening Bank', path: '/bank-accounts', status: 'active' },
    { name: 'Konfirmasi Pembayaran', path: '/payments', status: 'active' },
    { name: 'User Management', path: '/users', status: 'active' }
  ];

  const menuGroups = [
    {
      name: 'MASTER PAKET',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
        </svg>
      ),
      items: [
        { name: 'Kelola Paket', path: '/schedules', status: 'active' },
        { name: 'Kelola Hotel', path: '/hotels', status: 'active' },
        { name: 'Kelola Maskapai', path: '/airlines', status: 'active' },
        { name: 'Kelola Itinerary', path: '/itineraries', status: 'active' },
        { name: 'Kelola Add-On', path: '/addons', status: 'active' },
      ],
    },
    {
      name: 'INVENTORY',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
        </svg>
      ),
      items: [
        { name: 'Kelola Perlengkapan', path: '/inventory/perlengkapan', status: 'active' },
        { name: 'Stok per Brand', path: '/inventory/stok-perlengkapan', status: 'active' },
      ],
    },
    {
      name: 'KOMISI AGEN',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
      items: [
        { name: 'Komisi & Referral', path: '/komisi', status: 'inactive' },
      ],
    },
    {
      name: 'ANALYTICS & LAPORAN',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      ),
      items: [
        { name: 'Analytics Lintas Brand', path: '/analytics/lintas-brand', status: 'inactive' },
        { name: 'Laporan Keuangan Konsolidasi', path: '/analytics/keuangan', status: 'inactive' },
      ],
    },
    {
      name: 'COMPLIANCE',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
      ),
      items: [
        { name: 'Legalitas & Perizinan', path: '/compliance', status: 'inactive' },
      ],
    },
    {
      name: 'ADMINISTRASI',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
      ),
      items: adminItems,
    },
  ];

  return (
    <>

      {isOpen && (
        <div 
          className="md:hidden fixed inset-0 bg-neutral-900/50 backdrop-blur-sm z-40 transition-opacity"
          onClick={closeDrawer}
        />
      )}

      <aside className={`w-56 bg-gradient-to-b from-neutral-900 to-black text-white h-screen flex flex-col fixed left-0 top-0 shadow-2xl border-r border-neutral-800 z-50 transition-transform duration-300 ease-in-out ${isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
      <div className="px-5 py-5 border-b border-neutral-800 shrink-0">
        <h1 className="text-2xl font-heading font-bold tracking-tight text-primary-400">Azhan Grup</h1>
        <p className="text-sm font-body text-neutral-400 mt-1">Master Dashboard</p>
      </div>
      
      <nav className="flex-1 py-4 pl-3 space-y-6 overflow-y-auto font-body scrollbar-hide">
        
        {/* Dashboard - Standalone */}
        <div className="space-y-1">
          <NavLink
            to="/"
            onClick={closeDrawer}
            className={({ isActive }) =>
              `block px-3 py-2 text-sm transition-all rounded-l-md ${
                isActive
                  ? 'bg-gradient-to-r from-primary-600 to-primary-500 text-white font-medium shadow-md shadow-primary-900/30'
                  : 'text-neutral-300 hover:bg-neutral-800 hover:text-primary-300'
              }`
            }
          >
            Dashboard
          </NavLink>
        </div>

        {/* Menu Groups */}
        {menuGroups.map((group) => {
          const isExpanded = expandedGroups[group.name] || false;
          return (
            <div key={group.name} className="space-y-1">
              <button
                onClick={() => toggleGroup(group.name)}
                className="w-full flex items-center justify-between px-3 py-1.5 text-[10px] font-semibold text-neutral-500 hover:text-neutral-300 transition-colors uppercase tracking-widest mb-2"
              >
                <div className="flex items-center space-x-2">
                  {group.icon}
                  <span>{group.name}</span>
                </div>
                <svg
                  className={`w-3.5 h-3.5 transition-transform duration-200 ${isExpanded ? 'rotate-180' : ''}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {isExpanded && (
                <div className="space-y-1 mt-1">
                  {group.items.map((item) => {
                    const isInactive = item.status === 'inactive';
                    
                    return (
                      <NavLink
                        key={item.path}
                        to={item.path}
                        onClick={closeDrawer}
                        className={({ isActive }) => {
                          if (isInactive) {
                            return 'flex items-center justify-between px-3 py-2 text-sm transition-all rounded-l-md text-neutral-400 hover:bg-neutral-800';
                          }
                          
                          return `flex items-center justify-between px-3 py-2 text-sm transition-all rounded-l-md ${
                            isActive
                              ? 'bg-gradient-to-r from-primary-600 to-primary-500 text-white font-medium shadow-md shadow-primary-900/30'
                              : 'text-neutral-300 hover:bg-neutral-800 hover:text-primary-300'
                          }`;
                        }}
                      >
                        <span className="truncate">{item.name}</span>
                      </NavLink>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </nav>

    </aside>
    </>
  );
};

export default Sidebar;

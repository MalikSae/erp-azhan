import React, { useState } from 'react';
import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Package,
  FolderTree,
  Hotel,
  Plane,
  MapPin,
  PlusCircle,
  Users,
  CalendarDays,
  CreditCard,
  Boxes,
  Percent,
  BarChart3,
  ShieldCheck,
  Building2,
  Landmark,
  UserCog,
  ChevronDown
} from 'lucide-react';
import KaabaIcon from '../../../shared/src/components/icons/KaabaIcon';

const Sidebar = ({ isOpen, closeDrawer }) => {
  const [expandedGroups, setExpandedGroups] = useState({
    'MASTER PAKET': true,
    'JAMAAH & BOOKING': true,
    'INVENTORY': false,
    'KOMISI & ANALYTICS': false,
    'ADMINISTRASI': true,
  });

  const toggleGroup = (groupName) => {
    setExpandedGroups((prev) => ({
      ...prev,
      [groupName]: !prev[groupName],
    }));
  };

  const menuGroups = [
    {
      name: 'JAMAAH & BOOKING',
      items: [
        { name: 'Kelola Jamaah', path: '/jamaah', icon: Users, status: 'active' },
        { name: 'Kelola Booking', path: '/bookings', icon: CalendarDays, status: 'active' },
        { name: 'Pembayaran', path: '/payments', icon: CreditCard, status: 'active' },
      ],
    },
    {
      name: 'MASTER PAKET',
      items: [
        { name: 'Kelola Paket', path: '/schedules', icon: Package, status: 'active' },
        { name: 'Kelola Kategori', path: '/categories', icon: FolderTree, status: 'active' },
        { name: 'Kelola Hotel', path: '/hotels', icon: Hotel, status: 'active' },
        { name: 'Maskapai & Bandara', path: '/airlines', icon: Plane, status: 'active' },
        { name: 'Kelola Itinerary', path: '/itineraries', icon: MapPin, status: 'active' },
        { name: 'Kelola Add-On', path: '/addons', icon: PlusCircle, status: 'active' },
      ],
    },
    {
      name: 'INVENTORY',
      items: [
        { name: 'Kelola Perlengkapan', path: '/inventory/perlengkapan', icon: Boxes, status: 'active' },
        { name: 'Stok per Brand', path: '/inventory/stok-perlengkapan', icon: Building2, status: 'active' },
      ],
    },
    {
      name: 'KOMISI & ANALYTICS',
      items: [
        { name: 'Komisi & Referral', path: '/komisi', icon: Percent, status: 'inactive' },
        { name: 'Analytics Lintas Brand', path: '/analytics/lintas-brand', icon: BarChart3, status: 'inactive' },
        { name: 'Legalitas & Perizinan', path: '/compliance', icon: ShieldCheck, status: 'inactive' },
      ],
    },
    {
      name: 'ADMINISTRASI',
      items: [
        { name: 'Kelola Brand', path: '/brands', icon: Building2, status: 'active' },
        { name: 'Rekening Bank', path: '/bank-accounts', icon: Landmark, status: 'active' },
        { name: 'User Management', path: '/users', icon: UserCog, status: 'active' },
      ],
    },
  ];

  return (
    <>
      {isOpen && (
        <div 
          className="md:hidden fixed inset-0 bg-neutral-900/60 backdrop-blur-sm z-40 transition-opacity"
          onClick={closeDrawer}
        />
      )}

      <aside className={`w-64 bg-sidebar-bg text-neutral-300 h-screen flex flex-col fixed left-0 top-0 border-r border-sidebar-border z-50 transition-transform duration-300 ease-in-out select-none ${isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
        {/* Brand Header */}
        <div className="px-5 h-16 border-b border-sidebar-border shrink-0 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-primary-500 flex items-center justify-center text-brand-dark shadow-md shadow-amber-500/20">
              <KaabaIcon className="w-5 h-5" />
            </div>
            <div>
              <h1 className="text-base font-heading font-bold text-white tracking-tight leading-none">
                Azhan Grup
              </h1>
              <p className="text-[11px] font-body text-sidebar-muted mt-1 font-medium">Master ERP</p>
            </div>
          </div>
        </div>
        
        {/* Nav Links */}
        <nav className="flex-1 py-4 px-3 space-y-4 overflow-y-auto font-body scrollbar-dark">
          {/* Dashboard Item */}
          <div>
            <NavLink
              to="/"
              end
              onClick={closeDrawer}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3.5 py-2.5 text-sm transition-all rounded-xl ${
                  isActive
                    ? 'bg-sidebar-active text-sidebar-activeText font-semibold shadow-sm'
                    : 'text-sidebar-muted hover:text-white hover:bg-sidebar-surface font-medium'
                }`
              }
            >
              <LayoutDashboard className="w-4 h-4 shrink-0" />
              <span>Dashboard</span>
            </NavLink>
          </div>

          {/* Menu Groups */}
          {menuGroups.map((group) => {
            const isExpanded = expandedGroups[group.name] !== false;
            return (
              <div key={group.name} className="space-y-1">
                <button
                  type="button"
                  onClick={() => toggleGroup(group.name)}
                  className="w-full flex items-center justify-between px-3.5 py-1.5 text-[11px] font-bold text-sidebar-muted hover:text-neutral-200 transition-colors uppercase tracking-wider"
                >
                  <span>{group.name}</span>
                  <ChevronDown
                    className={`w-3.5 h-3.5 transition-transform duration-200 ${isExpanded ? 'rotate-180 text-neutral-400' : 'text-neutral-600'}`}
                  />
                </button>

                {isExpanded && (
                  <div className="space-y-1 pt-0.5">
                    {group.items.map((item) => {
                      const Icon = item.icon;
                      const isInactive = item.status === 'inactive';
                      
                      return (
                        <NavLink
                          key={item.path}
                          to={item.path}
                          onClick={closeDrawer}
                          className={({ isActive }) => {
                            if (isInactive) {
                              return 'flex items-center justify-between px-3.5 py-2 text-sm transition-all rounded-xl text-neutral-500 hover:bg-sidebar-surface/50 hover:text-neutral-400 font-medium';
                            }
                            
                            return `flex items-center justify-between px-3.5 py-2.5 text-sm transition-all rounded-xl ${
                              isActive
                                ? 'bg-sidebar-active text-sidebar-activeText font-semibold shadow-sm'
                                : 'text-sidebar-muted hover:text-white hover:bg-sidebar-surface font-medium'
                            }`;
                          }}
                        >
                          <div className="flex items-center gap-3 truncate">
                            {Icon && <Icon className="w-4 h-4 shrink-0" />}
                            <span className="truncate">{item.name}</span>
                          </div>
                          {isInactive && (
                            <span className="text-[10px] bg-neutral-800 text-neutral-500 px-1.5 py-0.5 rounded font-mono">
                              Soon
                            </span>
                          )}
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

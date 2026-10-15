import React, { useContext, useEffect, useMemo, useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Package,
  FolderTree,
  Hotel,
  Plane,
  MapPin,
  Users,
  CalendarDays,
  CreditCard,
  Boxes,
  Building2,
  History,
  UserCog,
  ChevronDown,
  Network,
  UserCheck,
  Banknote,
  Wallet,
} from 'lucide-react';
import { AuthContext } from '../context/AuthContext';
import KaabaIcon from '../../../shared/src/components/icons/KaabaIcon';

const Sidebar = ({ isOpen, closeDrawer }) => {
  const { hasPermission, isHolding, brandInfo } = useContext(AuthContext);
  // Admin brand: tampilkan ikon/logo brand bila ada, jika tidak pakai ikon Ka'bah bawaan.
  const [brandIconFailed, setBrandIconFailed] = useState(false);
  const rawIcon = isHolding ? null : (brandInfo?.icon_url || brandInfo?.logo_url);
  const brandIconSrc = rawIcon && !brandIconFailed
    ? (rawIcon.startsWith('http') ? rawIcon : `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${rawIcon.startsWith('/') ? '' : '/'}${rawIcon}`)
    : null;
  // Akordeon: semua grup tertutup secara default dan hanya satu grup yang
  // terbuka pada satu waktu.
  const [openGroup, setOpenGroup] = useState(null);

  // Grup yang berisi halaman aktif dibuka otomatis saat pindah halaman,
  // agar menu aktif selalu terlihat (tetap bisa ditutup manual).
  const location = useLocation();
  useEffect(() => {
    const aktif = menuGroups.find((g) =>
      g.items.some((item) => location.pathname === item.path || location.pathname.startsWith(`${item.path}/`))
    );
    if (aktif) setOpenGroup(aktif.name);
  }, [location.pathname]);

  const toggleGroup = (groupName) => {
    setOpenGroup((prev) => (prev === groupName ? null : groupName));
  };

  // Tiap item punya `perm` (izin yang dibutuhkan); item tanpa izin yang cocok disembunyikan,
  // dan grup tanpa item tidak ditampilkan. Penegakan sebenarnya ada di API.
  const allGroups = [
    {
      name: 'JAMAAH & BOOKING',
      items: [
        { name: 'Kelola Jamaah', path: '/jamaah', icon: Users, perm: 'jamaah.view' },
        { name: 'Duplikat Lintas Brand', path: '/jamaah/duplikat', icon: Users, perm: 'jamaah.view', holdingOnly: true },
        { name: 'Kelola Booking', path: '/bookings', icon: CalendarDays, perm: 'booking.view' },
        { name: 'Pembayaran', path: '/payments', icon: CreditCard, perm: 'payment.view' },
      ],
    },
    {
      name: 'KEUANGAN',
      items: [
        { name: 'Dashboard Keuangan', path: '/keuangan', icon: Banknote, perm: 'report.finance.view' },
      ],
    },
    {
      name: 'AGEN',
      items: [
        { name: 'Daftar Agen', path: '/agen', icon: Users, perm: 'agen.view' },
        { name: 'Riwayat Komisi', path: '/agen/komisi', icon: Wallet, perm: 'komisi.view' },
        { name: 'Pengajuan Agen', path: '/agen/pengajuan', icon: UserCheck, perm: 'agen.approve' },
      ],
    },
    {
      name: 'PAKET',
      items: [
        { name: 'Kelola Paket', path: '/schedules', icon: Package, perm: 'schedule.view' },
        { name: 'Kelola Kategori', path: '/categories', icon: FolderTree, perm: 'masterdata.edit' },
        { name: 'Kelola Hotel', path: '/hotels', icon: Hotel, perm: 'masterdata.edit' },
        { name: 'Maskapai & Bandara', path: '/airlines', icon: Plane, perm: 'masterdata.edit' },
        { name: 'Kelola Itinerary', path: '/itineraries', icon: MapPin, perm: 'masterdata.edit' },
      ],
    },
    {
      name: 'INVENTORY',
      items: [
        { name: 'Perlengkapan', path: '/inventory/perlengkapan', icon: Boxes, perm: 'perlengkapan.view' },
        { name: 'Riwayat Stok', path: '/inventory/riwayat-stok', icon: History, perm: 'perlengkapan.view' },
      ],
    },
    {
      name: 'ADMINISTRASI',
      items: [
        { name: 'Kelola Brand', path: '/brands', icon: Building2, perm: 'brand.edit' },
        { name: 'Pengguna', path: '/users', icon: UserCog, perm: 'user.admin' },
        { name: 'Struktur Organisasi', path: '/organisasi', icon: Network, perm: 'user.admin' },
      ],
    },
  ];

  const menuGroups = useMemo(
    () => allGroups
      .map((g) => ({ ...g, items: g.items.filter((i) => (!i.holdingOnly || isHolding) && (!i.perm || hasPermission(i.perm))) }))
      .filter((g) => g.items.length > 0),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [hasPermission, isHolding],
  );

  return (
    <>
      {isOpen && (
        <div 
          className="md:hidden fixed inset-0 bg-neutral-900/60 backdrop-blur-sm z-40 transition-opacity"
          onClick={closeDrawer}
        />
      )}

      <aside className={`w-64 bg-sidebar-bg text-neutral-600 h-screen flex flex-col fixed left-0 top-0 border-r border-sidebar-border z-50 transition-transform duration-300 ease-in-out select-none ${isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
        {/* Brand Header */}
        <div className="px-5 h-16 border-b border-sidebar-border shrink-0 flex items-center justify-between">
          <div className="flex items-center gap-3">
            {brandIconSrc ? (
              <img
                src={brandIconSrc}
                alt={brandInfo?.name || ''}
                onError={() => setBrandIconFailed(true)}
                className="w-9 h-9 rounded-xl object-contain bg-white border border-neutral-200 shadow-sm"
              />
            ) : (
              <div className="w-9 h-9 rounded-xl bg-primary-500 flex items-center justify-center text-white shadow-sm">
                <KaabaIcon className="w-5 h-5" />
              </div>
            )}
            <div>
              <h1 className="text-base font-heading font-bold text-neutral-900 tracking-tight leading-none truncate max-w-36">
                {isHolding ? 'Azhan Corp' : (brandInfo?.name || 'Azhan Corp')}
              </h1>
              <p className="text-[11px] font-body text-sidebar-muted mt-1 font-medium">Office Dashboard</p>
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
                `flex items-center gap-3 px-3 py-2 text-[13px] transition-all rounded-lg ${
                  isActive
                    ? 'bg-sidebar-active text-sidebar-activeText font-semibold'
                    :'text-sidebar-muted hover:text-neutral-900 hover:bg-sidebar-hover font-medium'
                }`
              }
            >
              <LayoutDashboard className="w-4 h-4 shrink-0" />
              <span>Dashboard</span>
            </NavLink>
          </div>

          {/* Menu Groups — label seksi statis ala SmartHR; semua item terlihat */}
          {menuGroups.map((group) => {
            return (
              <div key={group.name} className="space-y-0.5">
                <div className="px-3.5 pt-2 pb-1 text-[10.5px] font-semibold text-neutral-400 uppercase tracking-[0.08em]">
                  {group.name}
                </div>

                {(
                  <div className="space-y-0.5">
                    {group.items.map((item) => {
                      const Icon = item.icon;
                      const isInactive = item.status === 'inactive';
                      
                      return (
                        <NavLink
                          key={item.path}
                          to={item.path}
                          end={item.path === '/komisi' || item.path === '/agen'}
                          onClick={closeDrawer}
                          className={({ isActive }) => {
                            if (isInactive) {
                              return 'flex items-center justify-between px-3 py-1.5 text-[13px] transition-all rounded-lg text-neutral-500 hover:bg-sidebar-surface/50 hover:text-neutral-400 font-medium';
                            }
                            
                            return `flex items-center justify-between px-3 py-2 text-[13px] transition-all rounded-lg ${
                              isActive
                                ? 'bg-sidebar-active text-sidebar-activeText font-semibold'
                                :'text-sidebar-muted hover:text-neutral-900 hover:bg-sidebar-hover font-medium'
                            }`;
                          }}
                        >
                          <div className="flex items-center gap-3 truncate">
                            {Icon && <Icon className="w-4 h-4 shrink-0" />}
                            <span className="truncate">{item.name}</span>
                          </div>
                          {isInactive && (
                            <span className="text-[10px] bg-neutral-100 text-neutral-400 px-1.5 py-0.5 rounded font-mono">
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

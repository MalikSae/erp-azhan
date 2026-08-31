import { NavLink } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";
import { LayoutDashboard, Users, CalendarDays, Package, Boxes, Banknote } from "lucide-react";

function isLightColor(hexColor) {
  if (!hexColor || typeof hexColor !== 'string' || !hexColor.startsWith('#')) return false;
  const hex = hexColor.replace('#', '');
  if (hex.length !== 6 && hex.length !== 3) return false;
  const r = parseInt(hex.length === 3 ? hex[0] + hex[0] : hex.slice(0, 2), 16);
  const g = parseInt(hex.length === 3 ? hex[1] + hex[1] : hex.slice(2, 4), 16);
  const b = parseInt(hex.length === 3 ? hex[2] + hex[2] : hex.slice(4, 6), 16);
  const brightness = (r * 299 + g * 587 + b * 114) / 1000;
  return brightness > 140;
}

const Sidebar = ({ isOpen, closeDrawer }) => {
  const { brandInfo } = useAuth();

  const menuItems = [
    { name: "Dashboard", path: "/", icon: LayoutDashboard },
    { name: "Daftar Paket", path: "/paket", icon: Package },
    { name: "Stok Perlengkapan", path: "/stok-perlengkapan", icon: Boxes },
    { name: "Kelola Jamaah", path: "/jamaah", icon: Users },
    { name: "Kelola Booking", path: "/bookings", icon: CalendarDays },
    { name: "Pembayaran", path: "/payments", icon: Banknote },
  ];

  const brandColor = brandInfo?.primary_color || '#FED853';
  const isLight = isLightColor(brandColor);

  const logoSrc = brandInfo?.logo_url
    ? (brandInfo.logo_url.startsWith('http')
        ? brandInfo.logo_url
        : `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${brandInfo.logo_url.startsWith('/') ? '' : '/'}${brandInfo.logo_url}`)
    : null;

  return (
    <>
      {isOpen && (
        <div 
          className="md:hidden fixed inset-0 bg-neutral-900/40 backdrop-blur-sm z-40 transition-opacity"
          onClick={closeDrawer}
        />
      )}

      {/* Light Mode Sidebar */}
      <aside className={`w-64 bg-white h-screen flex flex-col fixed left-0 top-0 border-r border-neutral-200 z-50 transition-transform duration-300 ease-in-out select-none ${isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
        
        {/* Brand Header — Logo Only */}
        <div className="px-5 h-16 border-b border-neutral-100 shrink-0 flex items-center justify-center">
          {logoSrc ? (
            <div className="flex items-center justify-center w-full">
              <img
                src={logoSrc}
                alt={brandInfo?.name || "Logo"}
                className="max-h-10 max-w-[160px] object-contain"
                onError={(e) => { e.currentTarget.style.display = 'none'; }}
              />
            </div>
          ) : (
            <div
              className="w-10 h-10 rounded-xl flex items-center justify-center font-heading font-bold text-lg shadow-sm shrink-0"
              style={{ backgroundColor: brandColor, color: isLight ? '#14171A' : '#FFFFFF' }}
            >
              {brandInfo?.name ? brandInfo.name.charAt(0).toUpperCase() : 'T'}
            </div>
          )}
        </div>
        
        {/* Navigation Menu */}
        <nav className="flex-1 py-3 px-3 space-y-0.5 overflow-y-auto font-body">
          {menuItems.map((item) => (
            <NavLink
              key={item.name}
              to={item.path}
              end={item.path === '/'}
              onClick={closeDrawer}
              style={({ isActive }) => 
                isActive 
                  ? { backgroundColor: brandColor, color: isLight ? '#14171A' : '#FFFFFF' }
                  : {}
              }
              className={({ isActive }) =>
                `flex items-center gap-3 px-3.5 py-2.5 text-sm transition-all rounded-xl ${
                  isActive
                    ? 'font-semibold shadow-sm'
                    : 'text-neutral-500 hover:text-neutral-900 hover:bg-neutral-100 font-medium'
                }`
              }
            >
              <item.icon className="w-4 h-4 shrink-0" />
              <span>{item.name}</span>
            </NavLink>
          ))}
        </nav>

        {/* Brand name at bottom */}
        {brandInfo?.name && (
          <div className="px-4 py-3 border-t border-neutral-100">
            <p className="text-[11px] text-neutral-400 font-medium text-center truncate">{brandInfo.name}</p>
          </div>
        )}
      </aside>
    </>
  );
};

export default Sidebar;

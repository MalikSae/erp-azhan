import { NavLink, useLocation } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";
import { LayoutDashboard, Users, CalendarDays, Package, Boxes, Banknote } from "lucide-react";

const Sidebar = ({ isOpen, closeDrawer }) => {
  const { brandInfo } = useAuth();
  const location = useLocation();

  const menuItems = [
    { name: "Dashboard", path: "/", icon: LayoutDashboard },
    { name: "Daftar Paket", path: "/paket", icon: Package },
    { name: "Stok Perlengkapan", path: "/stok-perlengkapan", icon: Boxes },
    { name: "Kelola Jamaah", path: "/jamaah", icon: Users },
    { name: "Kelola Booking", path: "/bookings", icon: CalendarDays },
    { name: "Pembayaran", path: "/payments", icon: Banknote },
  ];

  const logoSrc = brandInfo?.logo_url
    ? (brandInfo.logo_url.startsWith('http')
        ? brandInfo.logo_url
        : `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${brandInfo.logo_url}`)
    : null;

  return (
    <>
      {isOpen && (
        <div 
          className="md:hidden fixed inset-0 bg-neutral-900/50 backdrop-blur-sm z-40 transition-opacity"
          onClick={closeDrawer}
        />
      )}

      <aside className={`w-56 bg-white text-neutral-800 h-screen flex flex-col fixed left-0 top-0 border-r border-neutral-200 z-50 transition-transform duration-300 ease-in-out ${isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
        <div className="px-5 h-16 border-b border-neutral-200 shrink-0 flex items-center">
          {logoSrc ? (
            <div className="h-10 flex items-center w-full">
              <img
                src={logoSrc}
                alt={brandInfo?.name || "Logo"}
                className="max-h-9 max-w-full object-contain"
              />
            </div>
          ) : (
            <h1 className="text-xl font-heading font-bold tracking-tight text-brand truncate">
              {brandInfo?.name || "Memuat..."}
            </h1>
          )}
        </div>
        
        <nav className="flex-1 py-4 px-3 space-y-1 overflow-y-auto font-body scrollbar-hide">
          {menuItems.map((item) => (
            <NavLink
              key={item.name}
              to={item.path}
              onClick={closeDrawer}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 text-sm font-medium transition-all rounded-md ${
                  isActive
                    ? 'bg-brand text-white shadow-sm'
                    : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900'
                }`
              }
            >
              <item.icon size={18} />
              <span>{item.name}</span>
            </NavLink>
          ))}
        </nav>
      </aside>
    </>
  );
};

export default Sidebar;

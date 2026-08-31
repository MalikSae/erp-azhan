import { useState, useEffect } from "react";
import { Outlet } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";
import { applyBrandTheme } from "../../utils/brandTheme";
import Sidebar from "./Sidebar";
import Topbar from "./Topbar";

const DashboardLayout = () => {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const { brandInfo } = useAuth();
  const brandColor = brandInfo?.primary_color || '#FED853';

  useEffect(() => {
    applyBrandTheme(brandColor);
  }, [brandColor]);

  const toggleSidebar = () => setIsSidebarOpen(!isSidebarOpen);
  const closeSidebar = () => setIsSidebarOpen(false);

  return (
    <div className="h-screen bg-page-bg flex overflow-hidden font-body">
      <Sidebar isOpen={isSidebarOpen} closeDrawer={closeSidebar} />
      <div className="flex-1 min-w-0 ml-0 md:ml-64 flex flex-col h-screen">
        <Topbar toggleSidebar={toggleSidebar} />
        <div className="flex-1 p-4 md:p-8 pt-6 md:pt-8 overflow-y-auto relative">
          <div className="max-w-6xl mx-auto min-h-[calc(100vh-8rem)]">
            <Outlet />
          </div>
        </div>
      </div>
    </div>
  );
};

export default DashboardLayout;

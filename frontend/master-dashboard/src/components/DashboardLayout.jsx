import React, { useState } from 'react';
import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import Topbar from './Topbar';
const DashboardLayout = () => {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const toggleSidebar = () => setIsSidebarOpen(!isSidebarOpen);
  const closeSidebar = () => setIsSidebarOpen(false);

  return (
    <div className="h-screen bg-neutral-50 flex overflow-hidden">
      <Sidebar isOpen={isSidebarOpen} closeDrawer={closeSidebar} />
      <div className="flex-1 min-w-0 ml-0 md:ml-56 flex flex-col h-screen">
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

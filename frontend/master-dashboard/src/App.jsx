import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import ProtectedRoute from './components/ProtectedRoute';
import DashboardLayout from './components/DashboardLayout';

import Login from './pages/Login';
import DashboardHome from './pages/DashboardHome';
import HotelsPage from './pages/HotelsPage';
import AirlinesPage from './pages/AirlinesPage';
import ItinerariesPage from './pages/ItinerariesPage';
import ItineraryFormPage from './pages/ItineraryFormPage';
import SchedulesPage from './pages/SchedulesPage';
import ScheduleDetailPage from './pages/ScheduleDetailPage';
import ScheduleFormPage from './pages/ScheduleFormPage';
import CategoriesPage from './pages/CategoriesPage';
import AddOnsPage from './pages/AddOnsPage';
import DesignSystemPage from './pages/DesignSystemPage';
import InventoryPerlengkapanPage from './pages/InventoryPerlengkapanPage';
import InventoryStokPerlengkapanPage from './pages/InventoryStokPerlengkapanPage';
import StokBrandCabangPage from './pages/StokBrandCabangPage';
import KomisiReferralPage from './pages/KomisiReferralPage';
import GantiKaitanAgenPage from './pages/GantiKaitanAgenPage';
import PeringkatAgenPage from './pages/PeringkatAgenPage';
import UserManagementPage from './pages/UserManagementPage';
import UserFormPage from './pages/UserFormPage';
import BrandsPage from './pages/BrandsPage';
import BrandFormPage from './pages/BrandFormPage';
import BankAccountsPage from './pages/BankAccountsPage';

import {
  JamaahPage,
  JamaahDetailPage,
  JamaahFormPage,
  BookingsPage,
  BookingDetailPage,
  BookingFormPage,
  DaftarAgenPage,
  DetailAgenPage,
  RiwayatKomisiPage,
  PaymentConfirmationsPage,
} from 'shared';

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/design-system" element={<DesignSystemPage />} />
          
          <Route element={<ProtectedRoute />}>
            <Route element={<DashboardLayout />}>
              <Route path="/" element={<DashboardHome />} />
              <Route path="/hotels" element={<HotelsPage />} />
              <Route path="/airlines" element={<AirlinesPage />} />
              <Route path="/categories" element={<CategoriesPage />} />
              <Route path="/itineraries" element={<ItinerariesPage />} />
              <Route path="/itineraries/new" element={<ItineraryFormPage />} />
              <Route path="/itineraries/:id/edit" element={<ItineraryFormPage />} />
              <Route path="/schedules" element={<SchedulesPage />} />
              <Route path="/schedules/new" element={<ScheduleFormPage />} />
              <Route path="/schedules/:id" element={<ScheduleDetailPage />} />
              <Route path="/schedules/:id/edit" element={<ScheduleFormPage />} />
              <Route path="/addons" element={<AddOnsPage />} />
              
              {/* Jamaah & Booking (Shared) */}
              <Route path="/jamaah" element={<JamaahPage showBrandColumn={true} />} />
              <Route path="/jamaah/new" element={<JamaahFormPage showBrandColumn={true} />} />
              <Route path="/jamaah/:id" element={<JamaahDetailPage showBrandColumn={true} />} />
              <Route path="/jamaah/:id/edit" element={<JamaahFormPage showBrandColumn={true} />} />
              <Route path="/bookings" element={<BookingsPage showBrandColumn={true} />} />
              <Route path="/bookings/new" element={<BookingFormPage showBrandColumn={true} />} />
              <Route path="/bookings/:id" element={<BookingDetailPage showBrandColumn={true} />} />
              <Route path="/bookings/:id/edit" element={<BookingFormPage showBrandColumn={true} />} />
              
              {/* Inventory */}
              <Route path="/inventory/perlengkapan" element={<InventoryPerlengkapanPage />} />
              <Route path="/inventory/stok-perlengkapan" element={<InventoryStokPerlengkapanPage />} />
              <Route path="/inventory/stok" element={<StokBrandCabangPage />} />
              
              {/* Komisi Agen */}
              <Route path="/komisi" element={<KomisiReferralPage />} />
              <Route path="/komisi/ganti-kaitan" element={<GantiKaitanAgenPage />} />
              <Route path="/agen" element={<DaftarAgenPage showBrandColumn={true} />} />
              <Route path="/agen/komisi" element={<RiwayatKomisiPage showBrandColumn={true} />} />
              <Route path="/agen/:id" element={<DetailAgenPage showBrandColumn={true} />} />
              
              {/* Analytics & Laporan */}
              <Route path="/analytics/peringkat-agen" element={<PeringkatAgenPage />} />
              
              {/* Compliance */}
              
              {/* Administrasi */}
              <Route path="/brands" element={<BrandsPage />} />
              <Route path="/brands/new" element={<BrandFormPage />} />
              <Route path="/brands/:id/edit" element={<BrandFormPage />} />
              <Route path="/users" element={<UserManagementPage />} />
              <Route path="/users/new" element={<UserFormPage />} />
              <Route path="/bank-accounts" element={<BankAccountsPage />} />
              <Route path="/payments" element={<PaymentConfirmationsPage showBrandColumn={true} />} />
            </Route>
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;

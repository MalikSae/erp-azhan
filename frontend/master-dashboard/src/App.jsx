import React, { useContext } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, AuthContext } from './context/AuthContext';
import ProtectedRoute from './components/ProtectedRoute';
import RequireAccess from './components/RequireAccess';
import DashboardLayout from './components/DashboardLayout';

import Login from './pages/Login';
import DashboardHome from './pages/DashboardHome';
import BrandDashboardPage from './pages/BrandDashboardPage';
import KelolaAgenPage from './pages/KelolaAgenPage';
import HalamanAgenPage from './pages/HalamanAgenPage';
import KelolaKomisiPage from './pages/KelolaKomisiPage';
import KelolaJamaahPage from './pages/KelolaJamaahPage';
import PengumumanPage from './pages/PengumumanPage';
import AccountSettingsPage from './pages/AccountSettingsPage';
import HotelsPage from './pages/HotelsPage';
import AirlinesPage from './pages/AirlinesPage';
import ItinerariesPage from './pages/ItinerariesPage';
import ItineraryFormPage from './pages/ItineraryFormPage';
import SchedulesPage from './pages/SchedulesPage';
import ScheduleDetailPage from './pages/ScheduleDetailPage';
import ScheduleFormPage from './pages/ScheduleFormPage';
import CategoriesPage from './pages/CategoriesPage';
import DestinationsPage from './pages/DestinationsPage';
import InventoryPerlengkapanPage from './pages/InventoryPerlengkapanPage';
import InventoryRiwayatStokPage from './pages/InventoryRiwayatStokPage';
import StokBrandCabangPage from './pages/StokBrandCabangPage';
import PenggunaPage from './pages/PenggunaPage';
import PenggunaFormPage from './pages/PenggunaFormPage';
import StrukturOrganisasiPage from './pages/StrukturOrganisasiPage';
import BrandsPage from './pages/BrandsPage';
import BrandFormPage from './pages/BrandFormPage';

import {
  JamaahDetailPage,
  JamaahFormPage,
  BookingsPage,
  BookingDetailPage,
  BookingFormPage,
  DaftarAkunPage,
  DetailAkunPage,
  PaymentConfirmationsPage,
} from 'shared';

function AppRoutes() {
  const { isHolding, hasPermission } = useContext(AuthContext);
  // Holding melihat lintas brand (kolom Brand + filter); admin brand terkunci ke brand-nya.
  const showBrandColumn = isHolding;

  return (
    <Routes>
      <Route path="/login" element={<Login />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<DashboardLayout />}>
          <Route path="/" element={isHolding ? <DashboardHome /> : <BrandDashboardPage />} />
          <Route path="/pengaturan" element={<AccountSettingsPage />} />

          {/* Master paket */}
          <Route path="/hotels" element={<RequireAccess perm="masterdata.view"><HotelsPage /></RequireAccess>} />
          <Route path="/airlines" element={<RequireAccess perm="masterdata.view"><AirlinesPage /></RequireAccess>} />
          <Route path="/categories" element={<RequireAccess perm="masterdata.view"><CategoriesPage /></RequireAccess>} />
          <Route path="/destinations" element={<RequireAccess perm="masterdata.view"><DestinationsPage /></RequireAccess>} />
          <Route path="/itineraries" element={<RequireAccess perm="masterdata.view"><ItinerariesPage /></RequireAccess>} />
          <Route path="/itineraries/new" element={<RequireAccess perm="masterdata.edit"><ItineraryFormPage /></RequireAccess>} />
          <Route path="/itineraries/:id/edit" element={<RequireAccess perm="masterdata.edit"><ItineraryFormPage /></RequireAccess>} />
          <Route path="/schedules" element={<RequireAccess perm="schedule.view"><SchedulesPage /></RequireAccess>} />
          <Route path="/schedules/new" element={<RequireAccess perm="schedule.edit"><ScheduleFormPage /></RequireAccess>} />
          <Route path="/schedules/:id" element={<RequireAccess perm="schedule.view"><ScheduleDetailPage /></RequireAccess>} />
          <Route path="/schedules/:id/edit" element={<RequireAccess perm="schedule.edit"><ScheduleFormPage /></RequireAccess>} />

          {/* Jamaah & Booking (halaman bersama) */}
          <Route path="/akun" element={<RequireAccess perm="akun.view"><DaftarAkunPage showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/akun/:id" element={<RequireAccess perm="akun.view"><DetailAkunPage showBrandColumn={showBrandColumn} izin={{ ubahProfil: hasPermission('akun.edit'), gantiUpline: hasPermission('agen.kaitan.edit'), lihatAgen: hasPermission('agen.view'), setujuiAgen: hasPermission('agen.approve'), lihatBooking: hasPermission('booking.view') }} /></RequireAccess>} />
          <Route path="/jamaah" element={<RequireAccess perm="jamaah.view"><KelolaJamaahPage tab="daftar" showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/jamaah/new" element={<RequireAccess perm="jamaah.edit"><JamaahFormPage showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/jamaah/:id" element={<RequireAccess perm="jamaah.view"><JamaahDetailPage showBrandColumn={showBrandColumn} bolehLihatPerubahan={hasPermission('jamaah.perubahan.approve')} bolehGantiAgen={hasPermission('agen.kaitan.edit')} /></RequireAccess>} />
          <Route path="/jamaah/:id/edit" element={<RequireAccess perm="jamaah.edit"><JamaahFormPage showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/bookings" element={<RequireAccess perm="booking.view"><BookingsPage showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/bookings/new" element={<RequireAccess perm="booking.edit"><BookingFormPage showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/bookings/:id" element={<RequireAccess perm="booking.view"><BookingDetailPage showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/bookings/:id/edit" element={<RequireAccess perm="booking.edit"><BookingFormPage showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/payments" element={<RequireAccess perm="payment.view"><PaymentConfirmationsPage showBrandColumn={showBrandColumn} /></RequireAccess>} />

          {/* Inventory */}
          <Route path="/inventory/perlengkapan" element={<RequireAccess perm="perlengkapan.view"><InventoryPerlengkapanPage /></RequireAccess>} />
          <Route path="/inventory/riwayat-stok" element={<RequireAccess perm="perlengkapan.view"><InventoryRiwayatStokPage /></RequireAccess>} />
          <Route path="/inventory/stok-perlengkapan" element={<Navigate to="/inventory/perlengkapan" replace />} />
          <Route path="/inventory/stok" element={<RequireAccess holdingOnly><StokBrandCabangPage /></RequireAccess>} />

          {/* Agen & komisi */}
          <Route path="/komisi" element={<RequireAccess perm="agen.payout.approve"><KelolaKomisiPage tab="pencairan" showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/pengumuman" element={<RequireAccess perm="pengumuman.manage"><PengumumanPage /></RequireAccess>} />
          <Route path="/jamaah/perubahan" element={<RequireAccess perm="jamaah.perubahan.approve"><KelolaJamaahPage tab="persetujuan" showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/agen/pengajuan" element={<RequireAccess perm="agen.approve"><KelolaAgenPage tab="pengajuan" showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/agen" element={<RequireAccess perm="agen.view"><KelolaAgenPage tab="semua" showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/agen/komisi" element={<RequireAccess perm="komisi.view"><KelolaKomisiPage tab="riwayat" showBrandColumn={showBrandColumn} /></RequireAccess>} />
          <Route path="/agen/:id" element={<RequireAccess perm="agen.view"><HalamanAgenPage showBrandColumn={showBrandColumn} /></RequireAccess>} />

          {/* Peringkat agen menyatu ke daftar agen (diurutkan menurut closing, dengan rentang waktu). */}
          <Route path="/analytics/peringkat-agen" element={<Navigate to="/agen" replace />} />

          {/* Administrasi (holding) */}
          <Route path="/brands" element={<RequireAccess perm="brand.edit"><BrandsPage /></RequireAccess>} />
          <Route path="/brands/new" element={<RequireAccess perm="brand.edit"><BrandFormPage /></RequireAccess>} />
          <Route path="/brands/:id/edit" element={<RequireAccess perm="brand.edit"><BrandFormPage /></RequireAccess>} />
          <Route path="/users" element={<RequireAccess perm="user.admin"><PenggunaPage /></RequireAccess>} />
          <Route path="/users/new" element={<RequireAccess perm="user.admin"><PenggunaFormPage /></RequireAccess>} />
          <Route path="/users/:id" element={<RequireAccess perm="user.admin"><PenggunaFormPage /></RequireAccess>} />
          <Route path="/organisasi" element={<RequireAccess perm="user.admin"><StrukturOrganisasiPage /></RequireAccess>} />
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;

import { Routes, Route } from 'react-router-dom';
import ProtectedRoute from './components/layout/ProtectedRoute';
import DashboardLayout from './components/layout/DashboardLayout';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import PaketPage from './pages/PaketPage';
import PaketDetailPage from './pages/PaketDetailPage';
import StokPerlengkapanPage from './pages/StokPerlengkapanPage';
import PaymentConfirmationsPage from './pages/PaymentConfirmationsPage';

import {
  JamaahPage,
  JamaahDetailPage,
  JamaahFormPage,
  BookingsPage,
  BookingDetailPage,
  BookingFormPage,
} from 'shared';

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      
      <Route element={<ProtectedRoute />}>
        <Route element={<DashboardLayout />}>
          <Route path="/" element={<DashboardPage />} />
          
          <Route path="/paket" element={<PaketPage />} />
          <Route path="/paket/:id" element={<PaketDetailPage />} />
          <Route path="/schedules" element={<PaketPage />} />
          <Route path="/schedules/:id" element={<PaketDetailPage />} />

          <Route path="/stok-perlengkapan" element={<StokPerlengkapanPage />} />
          <Route path="/perlengkapan" element={<StokPerlengkapanPage />} />

          <Route path="/jamaah" element={<JamaahPage showBrandColumn={false} />} />
          <Route path="/jamaah/new" element={<JamaahFormPage showBrandColumn={false} />} />
          <Route path="/jamaah/:id" element={<JamaahDetailPage showBrandColumn={false} />} />
          <Route path="/jamaah/:id/edit" element={<JamaahFormPage showBrandColumn={false} />} />
          
          <Route path="/bookings" element={<BookingsPage showBrandColumn={false} />} />
          <Route path="/bookings/new" element={<BookingFormPage showBrandColumn={false} />} />
          <Route path="/bookings/:id" element={<BookingDetailPage showBrandColumn={false} />} />
          <Route path="/bookings/:id/edit" element={<BookingFormPage showBrandColumn={false} />} />
          <Route path="/payments" element={<PaymentConfirmationsPage />} />
        </Route>
      </Route>
    </Routes>
  );
}

export default App;

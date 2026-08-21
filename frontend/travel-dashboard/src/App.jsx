import { Routes, Route } from 'react-router-dom';
import ProtectedRoute from './components/layout/ProtectedRoute';
import DashboardLayout from './components/layout/DashboardLayout';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import JamaahPage from './pages/JamaahPage';
import JamaahDetailPage from './pages/JamaahDetailPage';
import JamaahFormPage from './pages/JamaahFormPage';
import BookingsPage from './pages/BookingsPage';
import BookingFormPage from './pages/BookingFormPage';
import BookingDetailPage from './pages/BookingDetailPage';
import PaketPage from './pages/PaketPage';
import PaketDetailPage from './pages/PaketDetailPage';
import StokPerlengkapanPage from './pages/StokPerlengkapanPage';
import PaymentConfirmationsPage from './pages/PaymentConfirmationsPage';

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

          <Route path="/jamaah" element={<JamaahPage />} />
          <Route path="/jamaah/new" element={<JamaahFormPage />} />
          <Route path="/jamaah/:id" element={<JamaahDetailPage />} />
          <Route path="/jamaah/:id/edit" element={<JamaahFormPage />} />
          
          <Route path="/bookings" element={<BookingsPage />} />
          <Route path="/bookings/new" element={<BookingFormPage />} />
          <Route path="/bookings/:id" element={<BookingDetailPage />} />
          <Route path="/payments" element={<PaymentConfirmationsPage />} />
        </Route>
      </Route>
    </Routes>
  );
}

export default App;

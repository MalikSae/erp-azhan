import { useNavigate } from 'react-router-dom';
import { SearchX } from 'lucide-react';
import Button from '../components/ui/Button';

// Halaman 404 untuk rute yang tidak dikenal. Dirender di dalam layout dashboard
// supaya sidebar tetap tersedia.
export default function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <div className="flex flex-col items-center justify-center py-24 px-4 text-center bg-white rounded-2xl border border-neutral-200 shadow-card">
      <div className="w-14 h-14 rounded-2xl bg-primary-50 text-primary-600 flex items-center justify-center mb-4">
        <SearchX className="w-7 h-7" strokeWidth={1.75} />
      </div>
      <p className="font-heading text-4xl font-semibold text-neutral-900">404</p>
      <h1 className="mt-2 font-heading text-lg font-semibold text-neutral-900">Halaman tidak ditemukan</h1>
      <p className="mt-1 max-w-sm text-sm font-body text-neutral-500">
        Alamat yang Anda buka tidak tersedia atau sudah dipindahkan.
      </p>
      <div className="mt-6 flex items-center gap-2">
        <Button variant="secondary" onClick={() => navigate(-1)}>Kembali</Button>
        <Button variant="primary" onClick={() => navigate('/')}>Ke Dashboard</Button>
      </div>
    </div>
  );
}

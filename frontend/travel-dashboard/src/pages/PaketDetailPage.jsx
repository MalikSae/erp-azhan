import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getScheduleAdmin } from '../api/schedules';
import PageHeader from '../components/ui/PageHeader';
import Card from '../components/ui/Card';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { 
  ArrowLeft, 
  Plane, 
  Hotel, 
  Users, 
  Calendar, 
  CheckCircle2, 
  XCircle, 
  Tag, 
  Clock, 
  MapPin, 
  Sparkles,
  TicketCheck,
  PlusCircle,
  FileText
} from 'lucide-react';

const formatDate = (dateStr) => {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return dateStr;
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
  return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
};

const formatCurrency = (amount) => {
  if (amount === undefined || amount === null) return '-';
  return 'Rp ' + amount.toLocaleString('id-ID');
};

const PaketDetailPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();

  const [schedule, setSchedule] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchDetail();
  }, [id]);

  const fetchDetail = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getScheduleAdmin(id);
      setSchedule(data);
    } catch (err) {
      setError(err.response?.data?.error || 'Gagal memuat detail paket');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center py-20 bg-white rounded-lg border border-neutral-200 shadow-sm">
        <LoadingSpinner />
      </div>
    );
  }

  if (error || !schedule) {
    return (
      <div className="space-y-4">
        <Button 
          variant="secondary" 
          size="sm" 
          onClick={() => navigate('/paket')}
          className="inline-flex items-center gap-1.5"
        >
          <ArrowLeft size={16} />
          <span>Kembali ke Daftar Paket</span>
        </Button>
        <Alert variant="error" message={error || 'Data paket tidak ditemukan'} />
      </div>
    );
  }

  const filledSeats = (schedule.seat_total || 0) - (schedule.seat_sisa || 0);
  const seatPercent = schedule.seat_total ? Math.round((filledSeats / schedule.seat_total) * 100) : 0;

  // Hitung durasi hari keberangkatan
  let tripDuration = null;
  if (schedule.berangkat_tanggal && schedule.pulang_tanggal) {
    const d1 = new Date(schedule.berangkat_tanggal);
    const d2 = new Date(schedule.pulang_tanggal);
    const diff = Math.round((d2 - d1) / (1000 * 60 * 60 * 24)) + 1;
    if (diff > 0) tripDuration = `${diff} Hari`;
  }

  return (
    <div className="space-y-6 pb-12">
      {/* Header Halaman Konsisten dengan JamaahDetailPage */}
      <PageHeader 
        title={schedule.jadwal_nama} 
        onBack={() => navigate(-1)}
        actionLabel="+ Buat Booking"
        onAction={() => navigate('/bookings/new')}
      />

      {/* Grid Kolom 1 & 2 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom Kiri / Utama (2 Span) */}
        <div className="lg:col-span-2 space-y-6">
          
          {/* Card: Jadwal Penerbangan */}
          <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 bg-neutral-50/50 flex items-center justify-between">
              <div className="flex items-center gap-2 text-primary-700 font-semibold font-heading text-sm">
                <Plane size={17} />
                <span>Jadwal & Rute Penerbangan</span>
              </div>
            </div>

            <div className="p-5 space-y-5">
              {/* Section 1: Maskapai, Tipe Penerbangan, Status Tiket */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 p-4 bg-neutral-50 rounded-lg border border-neutral-200">
                {/* 1. Maskapai Penerbangan */}
                <div className="flex items-center gap-3">
                  {schedule.maskapai?.logo_url ? (
                    <div className="w-12 h-12 rounded-lg bg-white border border-neutral-200 p-1 flex items-center justify-center shrink-0 shadow-xs overflow-hidden">
                      <img 
                        src={schedule.maskapai.logo_url.startsWith('http') ? schedule.maskapai.logo_url : `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${schedule.maskapai.logo_url}`} 
                        alt={schedule.maskapai.name} 
                        className="max-w-full max-h-full object-contain"
                      />
                    </div>
                  ) : (
                    <div className="w-12 h-12 rounded-lg bg-primary-50 border border-primary-100 flex items-center justify-center text-primary-700 shrink-0">
                      <Plane size={22} />
                    </div>
                  )}
                  <div className="min-w-0">
                    <span className="text-xs text-neutral-500 block">Maskapai Penerbangan</span>
                    <div className="flex items-center gap-1.5 mt-0.5 flex-wrap">
                      <span className="text-sm font-bold font-heading text-neutral-900 truncate">
                        {schedule.maskapai?.name || '-'}
                      </span>
                      {schedule.maskapai?.code && (
                        <span className="text-[11px] text-primary-700 font-mono font-bold bg-primary-50 px-1.5 py-0.2 rounded border border-primary-200">
                          {schedule.maskapai.code}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* 2. Tipe Penerbangan */}
                <div className="md:border-l md:border-neutral-200 md:pl-4 flex flex-col justify-center">
                  <span className="text-xs text-neutral-500 block">Tipe Penerbangan</span>
                  <p className="text-sm font-bold font-heading text-neutral-900 mt-0.5">
                    {schedule.is_direct_flight ? 'Direct Flight' : 'Transit'}
                  </p>
                </div>

                {/* 3. Status Tiket */}
                <div className="md:border-l md:border-neutral-200 md:pl-4 flex flex-col justify-center">
                  <span className="text-xs text-neutral-500 block">Status Tiket</span>
                  <div className="mt-1">
                    {schedule.is_ticket_confirmed ? (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-success-50 text-success-700 border border-success-200">
                        <CheckCircle2 size={13} />
                        <span>Tiket Confirmed</span>
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-warning-50 text-warning-800 border border-warning-200">
                        <Clock size={13} />
                        <span>Tiket Pending</span>
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Section 2: Durasi, Keberangkatan, Kepulangan */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 p-4 bg-neutral-50 rounded-lg border border-neutral-200">
                {/* 1. Durasi */}
                <div className="flex flex-col justify-center">
                  <span className="text-xs text-neutral-500 block">Durasi Perjalanan</span>
                  <p className="text-sm font-bold font-heading text-neutral-900 mt-0.5">
                    {tripDuration || '-'}
                  </p>
                </div>

                {/* 2. Keberangkatan */}
                <div className="md:border-l md:border-neutral-200 md:pl-4 space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-neutral-500 block">Keberangkatan</span>
                    {schedule.berangkat_kode_penerbangan && (
                      <span className="font-mono text-[11px] font-semibold text-primary-700 bg-primary-50 px-1.5 py-0.2 rounded border border-primary-200">
                        {schedule.berangkat_kode_penerbangan}
                      </span>
                    )}
                  </div>
                  <p className="text-sm font-bold font-heading text-neutral-900">
                    {formatDate(schedule.berangkat_tanggal)} {schedule.berangkat_jam ? `• ${schedule.berangkat_jam}` : ''}
                  </p>
                  {(schedule.berangkat_bandara_asal || schedule.berangkat_bandara_tujuan) && (
                    <p className="text-xs text-neutral-600 flex items-center gap-1">
                      <MapPin size={12} className="text-primary-600 shrink-0" />
                      <span>{schedule.berangkat_bandara_asal || '-'} → {schedule.berangkat_bandara_tujuan || '-'}</span>
                    </p>
                  )}
                </div>

                {/* 3. Kepulangan */}
                <div className="md:border-l md:border-neutral-200 md:pl-4 space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-neutral-500 block">Kepulangan</span>
                    {schedule.pulang_kode_penerbangan && (
                      <span className="font-mono text-[11px] font-semibold text-primary-700 bg-primary-50 px-1.5 py-0.2 rounded border border-primary-200">
                        {schedule.pulang_kode_penerbangan}
                      </span>
                    )}
                  </div>
                  <p className="text-sm font-bold font-heading text-neutral-900">
                    {formatDate(schedule.pulang_tanggal)} {schedule.pulang_jam ? `• ${schedule.pulang_jam}` : ''}
                  </p>
                  {(schedule.pulang_bandara_asal || schedule.pulang_bandara_tujuan) && (
                    <p className="text-xs text-neutral-600 flex items-center gap-1">
                      <MapPin size={12} className="text-primary-600 shrink-0" />
                      <span>{schedule.pulang_bandara_asal || '-'} → {schedule.pulang_bandara_tujuan || '-'}</span>
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Card: Rincian Harga Kamar */}
          <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 bg-neutral-50/50 flex items-center justify-between">
              <div className="flex items-center gap-2 text-primary-700 font-semibold font-heading text-sm">
                <Tag size={17} />
                <span>Rincian Harga Paket per Tipe Kamar</span>
              </div>
            </div>

            <div className="p-5 grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="p-4 bg-neutral-50 border border-neutral-200 rounded-lg text-center shadow-xs">
                <span className="text-xs font-semibold uppercase text-neutral-500 block mb-1">
                  Kamar Quad (4 Orang)
                </span>
                <span className="text-lg font-bold font-heading text-primary-700">
                  {formatCurrency(schedule.harga_quad)}
                </span>
              </div>

              <div className="p-4 bg-neutral-50 border border-neutral-200 rounded-lg text-center shadow-xs">
                <span className="text-xs font-semibold uppercase text-neutral-500 block mb-1">
                  Kamar Triple (3 Orang)
                </span>
                <span className="text-lg font-bold font-heading text-primary-700">
                  {formatCurrency(schedule.harga_triple)}
                </span>
              </div>

              <div className="p-4 bg-neutral-50 border border-neutral-200 rounded-lg text-center shadow-xs">
                <span className="text-xs font-semibold uppercase text-neutral-500 block mb-1">
                  Kamar Double (2 Orang)
                </span>
                <span className="text-lg font-bold font-heading text-primary-700">
                  {formatCurrency(schedule.harga_double)}
                </span>
              </div>
            </div>
          </div>

          {/* Card: Fasilitas Include & Exclude */}
          <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 bg-neutral-50/50">
              <span className="text-sm font-semibold font-heading text-neutral-900">
                Fasilitas Paket
              </span>
            </div>

            <div className="p-5 grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Include */}
              <div className="space-y-3">
                <div className="flex items-center gap-1.5 text-xs font-bold font-heading uppercase tracking-wider text-success-700">
                  <CheckCircle2 size={16} />
                  <span>Termasuk (Include)</span>
                </div>
                {schedule.include_items && schedule.include_items.length > 0 ? (
                  <ul className="space-y-2 text-xs font-body text-neutral-700">
                    {schedule.include_items.map((inc, i) => (
                      <li key={i} className="flex items-start gap-2">
                        <span className="text-success-600 font-bold">•</span>
                        <span>{inc}</span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-xs text-neutral-400 italic">Tidak ada rincian fasilitas include.</p>
                )}
              </div>

              {/* Exclude */}
              <div className="space-y-3">
                <div className="flex items-center gap-1.5 text-xs font-bold font-heading uppercase tracking-wider text-danger-700">
                  <XCircle size={16} />
                  <span>Tidak Termasuk (Exclude)</span>
                </div>
                {schedule.exclude_items && schedule.exclude_items.length > 0 ? (
                  <ul className="space-y-2 text-xs font-body text-neutral-700">
                    {schedule.exclude_items.map((exc, i) => (
                      <li key={i} className="flex items-start gap-2">
                        <span className="text-danger-600 font-bold">•</span>
                        <span>{exc}</span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-xs text-neutral-400 italic">Tidak ada rincian fasilitas exclude.</p>
                )}
              </div>
            </div>
          </div>

          {/* Card: Itinerary / Rencana Perjalanan (Jika ada) */}
          {schedule.itinerary && (
            <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
              <div className="px-5 py-4 border-b border-neutral-100 bg-neutral-50/50 flex items-center justify-between">
                <div className="flex items-center gap-2 text-primary-700 font-semibold font-heading text-sm">
                  <FileText size={17} />
                  <span>Itinerary: {schedule.itinerary.nama}</span>
                </div>
              </div>
              <div className="p-5 space-y-4">
                {schedule.itinerary.deskripsi && (
                  <p className="text-xs text-neutral-600 leading-relaxed border-b border-neutral-100 pb-3">
                    {schedule.itinerary.deskripsi}
                  </p>
                )}
                {schedule.itinerary.days && schedule.itinerary.days.length > 0 && (
                  <div className="space-y-3">
                    {schedule.itinerary.days.map((day, idx) => (
                      <div key={idx} className="flex gap-3 text-xs">
                        <span className="shrink-0 w-14 font-semibold font-heading text-primary-700 bg-primary-50 px-2 py-1 rounded text-center h-fit border border-primary-200">
                          Hari {day.hari_ke}
                        </span>
                        <div className="space-y-0.5">
                          <p className="font-bold text-neutral-900">{day.judul}</p>
                          <p className="text-neutral-600 leading-relaxed">{day.deskripsi}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Kolom Kanan / Sidebar Info (1 Span) */}
        <div className="space-y-6">
          
          {/* Card: Kuota & Kursi */}
          <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 bg-neutral-50/50 flex items-center gap-2 text-primary-700 font-semibold font-heading text-sm">
              <Users size={17} />
              <span>Status Kuota & Kursi</span>
            </div>

            <div className="p-5 space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Total Kuota:</span>
                <span className="text-sm font-bold text-neutral-900">{schedule.seat_total} Kursi</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Terisi:</span>
                <span className="text-sm font-semibold text-neutral-800">{filledSeats} Kursi ({seatPercent}%)</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Sisa Kursi:</span>
                <span className="text-sm font-bold text-success-600">{schedule.seat_sisa} Kursi Tersedia</span>
              </div>

              {/* Progress bar */}
              <div className="space-y-1 pt-1">
                <div className="h-2 w-full bg-neutral-100 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-primary-600 rounded-full transition-all duration-300"
                    style={{ width: `${Math.min(100, Math.max(0, seatPercent))}%` }}
                  />
                </div>
              </div>
            </div>
          </div>

          {/* Card: Hotel & Akomodasi */}
          <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 bg-neutral-50/50 flex items-center gap-2 text-primary-700 font-semibold font-heading text-sm">
              <Hotel size={17} />
              <span>Hotel & Akomodasi</span>
            </div>

            <div className="p-5 space-y-4">
              {/* Hotel Makkah */}
              <div className="p-3.5 bg-neutral-50 rounded-lg border border-neutral-200 space-y-1.5">
                <span className="text-xs font-bold uppercase tracking-wider text-neutral-500 block">
                  Hotel Makkah
                </span>
                <p className="text-sm font-bold text-neutral-900">
                  {schedule.hotel_mekkah?.name || '-'}
                </p>
                {schedule.hotel_mekkah?.star_rating > 0 ? (
                  <p className="text-xs text-warning-600 font-semibold">
                    {'★'.repeat(schedule.hotel_mekkah.star_rating)} ({schedule.hotel_mekkah.star_rating} Bintang)
                  </p>
                ) : null}
                {schedule.hotel_mekkah?.distance_m !== null && schedule.hotel_mekkah?.distance_m !== undefined ? (
                  <p className="text-xs text-neutral-600 flex items-center gap-1.5 pt-0.5">
                    <MapPin size={13} className="text-primary-600 shrink-0" />
                    <span>±{schedule.hotel_mekkah.distance_m} meter ke Masjidil Haram</span>
                  </p>
                ) : null}
              </div>

              {/* Hotel Madinah */}
              <div className="p-3.5 bg-neutral-50 rounded-lg border border-neutral-200 space-y-1.5">
                <span className="text-xs font-bold uppercase tracking-wider text-neutral-500 block">
                  Hotel Madinah
                </span>
                <p className="text-sm font-bold text-neutral-900">
                  {schedule.hotel_madinah?.name || '-'}
                </p>
                {schedule.hotel_madinah?.star_rating > 0 ? (
                  <p className="text-xs text-warning-600 font-semibold">
                    {'★'.repeat(schedule.hotel_madinah.star_rating)} ({schedule.hotel_madinah.star_rating} Bintang)
                  </p>
                ) : null}
                {schedule.hotel_madinah?.distance_m !== null && schedule.hotel_madinah?.distance_m !== undefined ? (
                  <p className="text-xs text-neutral-600 flex items-center gap-1.5 pt-0.5">
                    <MapPin size={13} className="text-primary-600 shrink-0" />
                    <span>±{schedule.hotel_madinah.distance_m} meter ke Masjid Nabawi</span>
                  </p>
                ) : null}
              </div>
            </div>
          </div>

          {/* Card: Add-On Master (Jika ada) */}
          {schedule.add_ons && schedule.add_ons.length > 0 && (
            <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
              <div className="px-5 py-4 border-b border-neutral-100 bg-neutral-50/50 flex items-center gap-2 text-primary-700 font-semibold font-heading text-sm">
                <Sparkles size={17} />
                <span>Add-On Tersedia</span>
              </div>
              <div className="p-5 space-y-2">
                {schedule.add_ons.map((addon, idx) => (
                  <div key={idx} className="p-2.5 bg-neutral-50 rounded border border-neutral-200 text-xs">
                    <p className="font-semibold text-neutral-900">{addon.nama || addon.add_on_nama}</p>
                    {addon.deskripsi && (
                      <p className="text-neutral-500 text-[11px] mt-0.5">{addon.deskripsi}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default PaketDetailPage;

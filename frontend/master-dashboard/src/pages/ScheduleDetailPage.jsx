import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Plane, 
  Building2, 
  Calendar, 
  Users, 
  CheckCircle2, 
  XCircle, 
  PlusCircle, 
  MapPin, 
  Clock, 
  FileText, 
  ExternalLink, 
  Edit2, 
  Zap, 
  Star,
  ArrowRight,
  Info
} from 'lucide-react';
import PageHeader from '../components/ui/PageHeader';
import MetaBox from '../components/ui/MetaBox';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { getSchedule } from '../api/schedules';
import { getItinerary } from '../api/itineraries';
import { listBrands } from '../api/brands';

const formatRupiah = (amount) => {
  if (amount === undefined || amount === null || amount === '') return '-';
  const num = Number(amount);
  if (isNaN(num) || num <= 0) return '-';
  return 'Rp ' + num.toLocaleString('id-ID');
};

const formatDate = (dateStr) => {
  if (!dateStr) return '-';
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    const months = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
    return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
  } catch (e) {
    return dateStr;
  }
};

const formatTime = (timeStr) => {
  if (!timeStr) return '';
  return timeStr.substring(0, 5);
};

const parseTransitAirports = (rawStr) => {
  if (!rawStr || typeof rawStr !== 'string') {
    return { berangkat: null, pulang: null };
  }
  const str = rawStr.trim();
  if (!str) return { berangkat: null, pulang: null };

  let berangkatRaw = null;
  let pulangRaw = null;

  if (str.toLowerCase().includes('berangkat:') || str.toLowerCase().includes('pulang:')) {
    const parts = str.split('|').map(p => p.trim());
    parts.forEach(p => {
      const lower = p.toLowerCase();
      if (lower.startsWith('berangkat:')) {
        berangkatRaw = p.substring(p.indexOf(':') + 1).trim();
      } else if (lower.startsWith('pulang:')) {
        pulangRaw = p.substring(p.indexOf(':') + 1).trim();
      }
    });
  } else {
    berangkatRaw = str;
    pulangRaw = str;
  }

  const extractCode = (part) => {
    if (!part) return null;
    const cleaned = part.split(',')[0].split('(')[0].trim().toUpperCase();
    const match = cleaned.match(/[A-Z]{3}/);
    if (match) return match[0];
    return cleaned || null;
  };

  return {
    berangkat: extractCode(berangkatRaw),
    pulang: extractCode(pulangRaw)
  };
};

const StarRating = ({ count = 0 }) => {
  const stars = Math.max(0, Math.min(5, Number(count) || 0));
  if (stars === 0) return <span className="text-xs text-neutral-400 font-body">Non-bintang</span>;
  return (
    <div className="flex items-center gap-0.5" title={`${stars} Bintang`}>
      {Array.from({ length: stars }).map((_, i) => (
        <Star key={i} size={14} className="fill-amber-400 text-amber-400" />
      ))}
    </div>
  );
};

const InfoItem = ({ label, value, colSpan = false, children }) => (
  <div className={colSpan ? "md:col-span-2" : ""}>
    <span className="block text-xs text-neutral-500 font-body mb-0.5">{label}</span>
    {children ? children : (
      <p className="text-sm font-medium font-body text-neutral-900 break-words">{value || '-'}</p>
    )}
  </div>
);

const ScheduleDetailPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();

  const [schedule, setSchedule] = useState(null);
  const [itinerary, setItinerary] = useState(null);
  const [brand, setBrand] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true);
      setErrorMessage(null);
      try {
        const scheduleData = await getSchedule(id);
        setSchedule(scheduleData);

        const promises = [];

        // Fetch Itinerary if present
        if (scheduleData.itinerary_id) {
          promises.push(
            getItinerary(scheduleData.itinerary_id)
              .then(res => setItinerary(res))
              .catch(() => setItinerary(null))
          );
        }

        // Fetch Brands to display Brand Name & Logo
        promises.push(
          listBrands()
            .then(brands => {
              const found = (brands || []).find(b => b.id === scheduleData.brand_id);
              setBrand(found || null);
            })
            .catch(() => setBrand(null))
        );

        await Promise.all(promises);
      } catch (error) {
        if (error.response?.status === 404) {
          setErrorMessage("Paket tidak ditemukan.");
        } else {
          setErrorMessage(error.response?.data?.error || "Gagal memuat detail paket.");
        }
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, [id]);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center p-12 bg-white rounded-lg border border-neutral-200 shadow-sm">
        <LoadingSpinner />
      </div>
    );
  }

  if (errorMessage || !schedule) {
    return (
      <div className="space-y-4">
        <Alert variant="error">{errorMessage || "Data paket tidak ditemukan."}</Alert>
        <Button variant="ghost" onClick={() => navigate('/schedules')}>
          ← Kembali ke Kelola Paket
        </Button>
      </div>
    );
  }

  const filledSeats = Math.max(0, (schedule.seat_total || 0) - (schedule.seat_sisa || 0));
  const fillPercent = schedule.seat_total > 0 ? (filledSeats / schedule.seat_total) * 100 : 0;

  return (
    <div className="w-full space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="space-y-1.5">
          <PageHeader
            title={schedule.jadwal_nama}
            onBack={() => navigate('/schedules')}
          />
          <div className="flex flex-wrap items-center gap-2 pl-0 md:pl-8">
            <Badge variant={schedule.status === 'published' ? 'published' : schedule.status === 'draft' ? 'draft' : 'archived'}>
              {schedule.status === 'published' ? 'Published' : schedule.status === 'draft' ? 'Draft' : 'Archived'}
            </Badge>

            {schedule.is_promo && (
              <Badge variant="warning" icon={<Zap size={11} className="fill-current" />}>
                Promo
              </Badge>
            )}

            <Badge variant={schedule.is_ticket_confirmed ? 'success' : 'neutral'}>
              {schedule.is_ticket_confirmed ? 'Tiket Confirmed' : 'Tiket Belum Confirmed'}
            </Badge>

            {schedule.is_direct_flight && (
              <Badge variant="primary">
                Direct Flight
              </Badge>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="primary"
            size="sm"
            onClick={() => navigate(`/schedules/${id}/edit`)}
            className="flex items-center gap-1.5"
          >
            <Edit2 size={14} />
            <span>Edit Paket</span>
          </Button>
        </div>
      </div>

      {/* Main Content Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom Kiri: Info Detail (2 Kolom di Desktop) */}
        <div className="lg:col-span-2 space-y-6">
          
          {/* Penerbangan & Rute */}
          <MetaBox title="Jadwal Penerbangan & Maskapai">
            <div className="space-y-4 font-body">
              {/* Maskapai Banner */}
              <div className="flex items-center gap-3 p-3 bg-neutral-50 rounded-lg border border-neutral-200">
                {schedule.maskapai?.logo_url ? (
                  <img
                    src={schedule.maskapai.logo_url.startsWith('http') ? schedule.maskapai.logo_url : `${import.meta.env.VITE_API_BASE_URL}${schedule.maskapai.logo_url.startsWith('/') ? '' : '/'}${schedule.maskapai.logo_url}`}
                    alt={schedule.maskapai.name}
                    className="w-12 h-12 object-contain bg-white rounded border border-neutral-200 p-1"
                  />
                ) : (
                  <div className="w-12 h-12 bg-neutral-200 rounded border border-neutral-300 flex items-center justify-center text-neutral-500">
                    <Plane size={20} />
                  </div>
                )}
                <div>
                  <span className="text-xs text-neutral-500 block">Maskapai Penerbangan</span>
                  <span className="text-sm font-semibold text-neutral-900">{schedule.maskapai?.name || '-'}</span>
                </div>
              </div>

              {/* Rute Keberangkatan & Kepulangan */}
              {(() => {
                const transitAirports = parseTransitAirports(schedule.transit_bandara);
                return (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
                    {/* Berangkat */}
                    <div className="p-3.5 bg-neutral-50 rounded-lg border border-neutral-200 space-y-2">
                      <div className="flex items-center justify-between border-b border-neutral-200 pb-2">
                        <span className="text-xs font-bold text-neutral-700 uppercase tracking-wider flex items-center gap-1.5">
                          <Plane size={14} className="text-neutral-700 rotate-45" />
                          Penerbangan Berangkat
                        </span>
                        {schedule.berangkat_kode_penerbangan && (
                          <span className="text-xs font-mono font-bold text-neutral-800 bg-neutral-100 px-2 py-0.5 rounded-lg border border-neutral-200/90">
                            {schedule.berangkat_kode_penerbangan}
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-neutral-600 space-y-1">
                        <p><strong>Tanggal:</strong> {formatDate(schedule.berangkat_tanggal)}</p>
                        <p><strong>Jam:</strong> {formatTime(schedule.berangkat_jam) || '-'}</p>
                        <div className="flex items-center gap-1.5 pt-1 text-neutral-900 font-semibold text-xs flex-wrap">
                          <span>{schedule.berangkat_bandara_asal || '-'}</span>
                          <ArrowRight size={13} className="text-neutral-400 shrink-0" />
                          {transitAirports.berangkat && (
                            <>
                              <span>{transitAirports.berangkat}</span>
                              <ArrowRight size={13} className="text-neutral-400 shrink-0" />
                            </>
                          )}
                          <span>{schedule.berangkat_bandara_tujuan || '-'}</span>
                        </div>
                      </div>
                    </div>

                    {/* Pulang */}
                    <div className="p-3.5 bg-neutral-50 rounded-lg border border-neutral-200 space-y-2">
                      <div className="flex items-center justify-between border-b border-neutral-200 pb-2">
                        <span className="text-xs font-bold text-neutral-700 uppercase tracking-wider flex items-center gap-1.5">
                          <Plane size={14} className="text-neutral-700 -rotate-135" />
                          Penerbangan Pulang
                        </span>
                        {schedule.pulang_kode_penerbangan && (
                          <span className="text-xs font-mono font-bold text-neutral-800 bg-neutral-100 px-2 py-0.5 rounded-lg border border-neutral-200/90">
                            {schedule.pulang_kode_penerbangan}
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-neutral-600 space-y-1">
                        <p><strong>Tanggal:</strong> {formatDate(schedule.pulang_tanggal)}</p>
                        <p><strong>Jam:</strong> {formatTime(schedule.pulang_jam) || '-'}</p>
                        <div className="flex items-center gap-1.5 pt-1 text-neutral-900 font-semibold text-xs flex-wrap">
                          <span>{schedule.pulang_bandara_asal || '-'}</span>
                          <ArrowRight size={13} className="text-neutral-400 shrink-0" />
                          {transitAirports.pulang && (
                            <>
                              <span>{transitAirports.pulang}</span>
                              <ArrowRight size={13} className="text-neutral-400 shrink-0" />
                            </>
                          )}
                          <span>{schedule.pulang_bandara_tujuan || '-'}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })()}
            </div>
          </MetaBox>

          {/* Akomodasi Hotel */}
          <MetaBox title="Akomodasi Hotel">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 font-body">
              {/* Hotel Makkah */}
              <div className="p-4 bg-neutral-50 rounded-lg border border-neutral-200 space-y-2.5">
                <div className="flex items-center justify-between border-b border-neutral-200 pb-2">
                  <span className="text-xs font-bold text-neutral-700 uppercase tracking-wider flex items-center gap-1.5">
                    <Building2 size={15} className="text-primary-600" />
                    Hotel Makkah
                  </span>
                  <StarRating count={schedule.hotel_mekkah?.star_rating} />
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-neutral-900">{schedule.hotel_mekkah?.name || '-'}</h4>
                  <p className="text-xs text-neutral-500 flex items-center gap-1 mt-1">
                    <MapPin size={12} className="text-neutral-400" />
                    <span>Jarak ke Masjidil Haram: <strong>{schedule.hotel_mekkah?.distance_m !== null && schedule.hotel_mekkah?.distance_m !== undefined ? `${schedule.hotel_mekkah.distance_m} meter` : '-'}</strong></span>
                  </p>
                </div>
              </div>

              {/* Hotel Madinah */}
              <div className="p-4 bg-neutral-50 rounded-lg border border-neutral-200 space-y-2.5">
                <div className="flex items-center justify-between border-b border-neutral-200 pb-2">
                  <span className="text-xs font-bold text-neutral-700 uppercase tracking-wider flex items-center gap-1.5">
                    <Building2 size={15} className="text-primary-600" />
                    Hotel Madinah
                  </span>
                  <StarRating count={schedule.hotel_madinah?.star_rating} />
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-neutral-900">{schedule.hotel_madinah?.name || '-'}</h4>
                  <p className="text-xs text-neutral-500 flex items-center gap-1 mt-1">
                    <MapPin size={12} className="text-neutral-400" />
                    <span>Jarak ke Masjid Nabawi: <strong>{schedule.hotel_madinah?.distance_m !== null && schedule.hotel_madinah?.distance_m !== undefined ? `${schedule.hotel_madinah.distance_m} meter` : '-'}</strong></span>
                  </p>
                </div>
              </div>
            </div>

            {/* Hotel Transit Jika Ada */}
            {schedule.transit_hotels && schedule.transit_hotels.length > 0 && (
              <div className="mt-4 pt-4 border-t border-neutral-200">
                <h4 className="text-xs font-bold text-neutral-700 uppercase tracking-wider mb-3">Hotel Transit</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 font-body">
                  {schedule.transit_hotels.map((th, idx) => (
                    <div key={idx} className="p-3 bg-neutral-50 rounded-lg border border-neutral-200 flex items-start gap-3">
                      {th.photo_url ? (
                        <img
                          src={th.photo_url.startsWith('http') ? th.photo_url : `${import.meta.env.VITE_API_BASE_URL}${th.photo_url.startsWith('/') ? '' : '/'}${th.photo_url}`}
                          alt={th.nama}
                          className="w-12 h-12 object-cover rounded bg-white border border-neutral-200 shrink-0"
                        />
                      ) : (
                        <div className="w-12 h-12 bg-neutral-200 rounded flex items-center justify-center text-neutral-500 shrink-0">
                          <Building2 size={18} />
                        </div>
                      )}
                      <div className="min-w-0 flex-1">
                        <span className="text-xs font-semibold text-neutral-900 block truncate">{th.nama}</span>
                        <span className="text-xs text-neutral-500 block">{th.kota || 'Kota Transit'}</span>
                        <div className="mt-1">
                          <StarRating count={th.star_rating} />
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </MetaBox>

          {/* Itinerary Hari-per-Hari */}
          <MetaBox title="Itinerary Perjalanan">
            {itinerary && itinerary.days && itinerary.days.length > 0 ? (
              <div className="space-y-4 font-body">
                <div className="p-3 bg-primary-50 border border-primary-200 rounded-md text-xs text-primary-900 flex items-center justify-between">
                  <span><strong>Template Itinerary:</strong> {itinerary.title}</span>
                  <span className="font-semibold">{itinerary.days.length} Hari</span>
                </div>

                <div className="space-y-3">
                  {itinerary.days.map((day, idx) => (
                    <div key={idx} className="p-3.5 bg-neutral-50 rounded-lg border border-neutral-200 space-y-2">
                      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-neutral-200 pb-2">
                        <div className="flex items-center gap-2">
                          <span className="w-6 h-6 rounded-full bg-primary-600 text-white font-heading font-bold text-xs flex items-center justify-center">
                            {day.day_number || (idx + 1)}
                          </span>
                          <h4 className="text-sm font-semibold text-neutral-900">{day.title}</h4>
                        </div>
                        {day.location && (
                          <span className="text-xs text-neutral-500 flex items-center gap-1">
                            <MapPin size={12} className="text-neutral-400" />
                            {day.location}
                          </span>
                        )}
                      </div>

                      {day.activities && day.activities.length > 0 ? (
                        <div className="space-y-1.5 pt-1">
                          {day.activities.map((act, actIdx) => (
                            <div key={actIdx} className="flex items-start gap-2 text-xs">
                              {act.time && (
                                <span className="font-mono font-semibold text-neutral-600 shrink-0 bg-neutral-200/60 px-1.5 py-0.5 rounded">
                                  {act.time}
                                </span>
                              )}
                              <span className="text-neutral-800 leading-relaxed">{act.text}</span>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="text-xs text-neutral-400 italic">Tidak ada rincian kegiatan tercatat.</p>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="p-4 bg-neutral-50 rounded-lg border border-neutral-200 text-center font-body text-neutral-500 text-sm">
                Belum ada data itinerary untuk paket ini.
              </div>
            )}
          </MetaBox>

          {/* Fasilitas Termasuk & Tidak Termasuk */}
          <MetaBox title="Fasilitas Paket">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 font-body">
              {/* Termasuk */}
              <div className="space-y-3">
                <h4 className="text-xs font-bold text-success-700 uppercase tracking-wider flex items-center gap-1.5">
                  <CheckCircle2 size={15} className="text-success-600" />
                  Harga Termasuk (Includes)
                </h4>
                {schedule.include_items && schedule.include_items.length > 0 ? (
                  <ul className="space-y-2 text-xs">
                    {schedule.include_items.map((item, idx) => (
                      <li key={idx} className="flex items-start gap-2 text-neutral-800">
                        <span className="w-1.5 h-1.5 rounded-full bg-success-500 mt-1.5 shrink-0" />
                        <span>{item}</span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-xs text-neutral-400 italic">Tidak ada item tercatat.</p>
                )}
              </div>

              {/* Tidak Termasuk */}
              <div className="space-y-3">
                <h4 className="text-xs font-bold text-neutral-700 uppercase tracking-wider flex items-center gap-1.5">
                  <XCircle size={15} className="text-neutral-400" />
                  Harga Tidak Termasuk (Excludes)
                </h4>
                {schedule.exclude_items && schedule.exclude_items.length > 0 ? (
                  <ul className="space-y-2 text-xs">
                    {schedule.exclude_items.map((item, idx) => (
                      <li key={idx} className="flex items-start gap-2 text-neutral-800">
                        <span className="w-1.5 h-1.5 rounded-full bg-neutral-400 mt-1.5 shrink-0" />
                        <span>{item}</span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-xs text-neutral-400 italic">Tidak ada item tercatat.</p>
                )}
              </div>
            </div>
          </MetaBox>

        </div>

        {/* Kolom Kanan: Sidebar Metadata & Harga (1 Kolom) */}
        <div className="space-y-6">
          
          {/* Brand & Kategori */}
          <MetaBox title="Informasi Brand & Kategori">
            <div className="space-y-3 font-body">
              <div className="flex items-center gap-3 p-3 bg-neutral-50 rounded-lg border border-neutral-200">
                {brand?.icon_url ? (
                  <img
                    src={brand.icon_url.startsWith('http') ? brand.icon_url : `${import.meta.env.VITE_API_BASE_URL}${brand.icon_url.startsWith('/') ? '' : '/'}${brand.icon_url}`}
                    alt={brand.name}
                    className="w-10 h-10 object-contain rounded-full bg-white border border-neutral-200"
                  />
                ) : (
                  <div className="w-10 h-10 rounded-full bg-neutral-200 border border-neutral-300 flex items-center justify-center font-bold text-neutral-700">
                    {brand?.name ? brand.name.charAt(0) : 'B'}
                  </div>
                )}
                <div>
                  <span className="text-xs text-neutral-500 block">Brand Travel</span>
                  <span className="text-sm font-semibold text-neutral-900">{brand?.name || `Brand ID: ${schedule.brand_id}`}</span>
                </div>
              </div>

              <InfoItem label="Kategori Paket" value={schedule.category?.name || '-'} />
            </div>
          </MetaBox>

          {/* Kuota & Seat */}
          <MetaBox title="Kuota & Ketersediaan Seat">
            <div className="space-y-4 font-body">
              <div className="grid grid-cols-2 gap-3 text-center">
                <div className="p-3 bg-neutral-50 rounded-lg border border-neutral-200">
                  <span className="text-xs text-neutral-500 block">Sisa Seat</span>
                  <span className="text-xl font-bold font-heading text-neutral-900">{schedule.seat_sisa ?? 0}</span>
                </div>
                <div className="p-3 bg-neutral-50 rounded-lg border border-neutral-200">
                  <span className="text-xs text-neutral-500 block">Total Kuota</span>
                  <span className="text-xl font-bold font-heading text-neutral-900">{schedule.seat_total ?? 0}</span>
                </div>
              </div>

              {/* Progress Bar */}
              {schedule.seat_total > 0 && (
                <div className="space-y-1.5">
                  <div className="flex justify-between text-xs text-neutral-600">
                    <span>Terisi: {filledSeats} pax</span>
                    <span>{Math.round(fillPercent)}%</span>
                  </div>
                  <div className="w-full h-2.5 bg-neutral-100 rounded-full overflow-hidden border border-neutral-200">
                    <div
                      className="h-full bg-primary-600 rounded-full transition-all duration-300"
                      style={{ width: `${Math.min(100, fillPercent)}%` }}
                    />
                  </div>
                </div>
              )}

              <div className="pt-2 border-t border-neutral-200 text-xs text-neutral-600 flex justify-between">
                <span>Total Booking Terdaftar:</span>
                <span className="font-semibold text-neutral-900">{schedule.booking_count || 0} booking</span>
              </div>
            </div>
          </MetaBox>

          {/* Rincian Harga Kamar */}
          <MetaBox title="Rincian Harga Paket">
            <div className="space-y-3 font-body">
              {schedule.harga_coret && schedule.harga_coret > 0 && (
                <div className="p-2.5 bg-warning-50 border border-warning-200 rounded-md text-xs flex justify-between items-center">
                  <span className="text-warning-800 font-medium">Harga Coret (Promo):</span>
                  <span className="line-through font-semibold text-neutral-500">{formatRupiah(schedule.harga_coret)}</span>
                </div>
              )}

              <div className="divide-y divide-neutral-200 text-xs">
                <div className="py-2.5 flex justify-between items-center">
                  <span className="text-neutral-600 font-medium">Kamar Quad (4 Orang)</span>
                  <span className="text-sm font-bold font-heading text-neutral-900">{formatRupiah(schedule.harga_quad)}</span>
                </div>

                <div className="py-2.5 flex justify-between items-center">
                  <span className="text-neutral-600 font-medium">Kamar Triple (3 Orang)</span>
                  <span className="text-sm font-bold font-heading text-neutral-900">{formatRupiah(schedule.harga_triple)}</span>
                </div>

                <div className="py-2.5 flex justify-between items-center">
                  <span className="text-neutral-600 font-medium">Kamar Double (2 Orang)</span>
                  <span className="text-sm font-bold font-heading text-neutral-900">{formatRupiah(schedule.harga_double)}</span>
                </div>

                {schedule.harga_infant !== null && schedule.harga_infant !== undefined && (
                  <div className="py-2.5 flex justify-between items-center">
                    <span className="text-neutral-600 font-medium">Harga Infant (Bayi)</span>
                    <span className="text-sm font-bold font-heading text-neutral-900">{formatRupiah(schedule.harga_infant)}</span>
                  </div>
                )}
              </div>
            </div>
          </MetaBox>

          {/* Add-On Tersedia */}
          <MetaBox title="Add-On Tersedia">
            {schedule.add_ons && schedule.add_ons.length > 0 ? (
              <div className="flex flex-wrap gap-1.5 font-body">
                {schedule.add_ons.map((addon, idx) => (
                  <span
                    key={idx}
                    className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium bg-neutral-100 text-neutral-800 border border-neutral-200"
                  >
                    <PlusCircle size={12} className="text-primary-600" />
                    {addon.name}
                  </span>
                ))}
              </div>
            ) : (
              <p className="text-xs text-neutral-400 italic font-body">Tidak ada add-on yang dipilih untuk paket ini.</p>
            )}
          </MetaBox>

          {/* Brosur Paket */}
          <MetaBox title="Brosur / Flyer Paket">
            {schedule.brosur_url || schedule.brosur_thumb_url ? (
              <div className="space-y-3 font-body">
                {schedule.brosur_thumb_url ? (
                  <div className="relative rounded-lg overflow-hidden border border-neutral-200 bg-neutral-100 max-h-56 flex items-center justify-center">
                    <img
                      src={schedule.brosur_thumb_url.startsWith('http') ? schedule.brosur_thumb_url : `${import.meta.env.VITE_API_BASE_URL}${schedule.brosur_thumb_url.startsWith('/') ? '' : '/'}${schedule.brosur_thumb_url}`}
                      alt="Thumbnail Brosur"
                      className="w-full h-auto max-h-56 object-contain"
                    />
                  </div>
                ) : null}

                {schedule.brosur_url && (
                  <a
                    href={schedule.brosur_url.startsWith('http') ? schedule.brosur_url : `${import.meta.env.VITE_API_BASE_URL}${schedule.brosur_url.startsWith('/') ? '' : '/'}${schedule.brosur_url}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center justify-center gap-1.5 w-full px-3 py-2 text-xs font-semibold text-primary-700 bg-primary-50 hover:bg-primary-100 border border-primary-200 rounded-md transition-colors"
                  >
                    <FileText size={14} />
                    <span>Buka File Brosur Lengkap</span>
                    <ExternalLink size={12} />
                  </a>
                )}
              </div>
            ) : (
              <p className="text-xs text-neutral-400 italic font-body">Belum ada brosur yang diunggah.</p>
            )}
          </MetaBox>

        </div>
      </div>
    </div>
  );
};

export default ScheduleDetailPage;

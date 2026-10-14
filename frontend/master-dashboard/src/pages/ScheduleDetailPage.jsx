import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Plane,
  Star,
  ArrowRight,
  Check,
  X,
  ExternalLink,
  ChevronDown,
  Printer
} from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import Button from 'shared/ui/Button';
import Badge from 'shared/ui/Badge';
import { Surface, Section, Field, SummaryItem, PILL_STRONG, BrandBadge } from 'shared/ui/DetailLayout';
import RichText from 'shared/ui/RichText';
import { printItinerary } from '../utils/printItinerary';
import { printPackage } from '../utils/printPackage';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import SeatManagement from '../components/SeatManagement';
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

const formatDateShort = (dateStr) => {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return '-';
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
  return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
};

const formatTime = (timeStr) => {
  if (!timeStr) return '';
  return timeStr.substring(0, 5);
};

const assetUrl = (url) =>
  url.startsWith('http') ? url : `${import.meta.env.VITE_API_BASE_URL}${url.startsWith('/') ? '' : '/'}${url}`;

const STATUS_LABEL = { published: 'Published', draft: 'Draft', archived: 'Archived' };

// Warna dipertegas agar tetap terbaca di atas latar halaman abu-abu.
const STATUS_STRONG = {
  published: '!bg-success-100 !border-success-300 !text-success-800',
  draft: '!border-neutral-400 !text-neutral-800',
  archived: '!bg-neutral-200 !border-neutral-300 !text-neutral-800',
};

// Status paket memakai Badge bersama; atribut lain (brand, jenis penerbangan) memakai pil netral.
const StatusBadge = ({ status }) => (
  <Badge variant={STATUS_LABEL[status] ? status : 'archived'} hideIcon className={STATUS_STRONG[status] || STATUS_STRONG.archived}>
    {STATUS_LABEL[status] || '-'}
  </Badge>
);

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
      {Array.from({ length: 5 }).map((_, i) => (
        <Star
          key={i}
          size={14}
          strokeWidth={1.75}
          className={i < stars ? 'fill-warning-400 text-warning-400' : 'text-warning-400'}
        />
      ))}
    </div>
  );
};

// Satu sisi penerbangan (berangkat / pulang), tanpa kotak sendiri.
const FlightLeg = ({ title, iconClass, code, tanggal, jam, asal, transit, tujuan }) => {
  const hasRoute = asal || transit || tujuan;
  return (
    <div className="space-y-3 min-w-0">
      <div className="flex items-center justify-between">
        <span className="font-heading text-sm font-semibold text-neutral-900 flex items-center gap-1.5">
          {title}
        </span>
        {code && <span className="text-xs font-mono font-semibold text-neutral-600">{code}</span>}
      </div>
      <Field label="Jadwal">
        {formatDate(tanggal)}
        {formatTime(jam) && <span className="text-neutral-500 font-normal"> · {formatTime(jam)}</span>}
      </Field>
      {hasRoute ? (
        <Field label="Rute">
          <span className="flex items-center gap-1.5 flex-wrap">
            <span>{asal || '-'}</span>
            <ArrowRight size={13} strokeWidth={1.75} className="text-neutral-400 shrink-0" />
            {transit && (
              <>
                <span>{transit}</span>
                <ArrowRight size={13} strokeWidth={1.75} className="text-neutral-400 shrink-0" />
              </>
            )}
            <span>{tujuan || '-'}</span>
          </span>
        </Field>
      ) : (
        <p className="text-xs text-neutral-400 font-body">Rute belum diisi</p>
      )}
    </div>
  );
};

const HotelItem = ({ title, hotel, distanceLabel }) => {
  const hasDistance = hotel?.distance_m !== null && hotel?.distance_m !== undefined;
  return (
    <div className="min-w-0">
      <div className="flex items-center justify-between gap-2">
        <span className="text-[11px] text-neutral-400 font-body">{title}</span>
        <StarRating count={hotel?.star_rating} />
      </div>
      <h4 className="font-heading text-sm font-semibold text-neutral-900 mt-0.5">{hotel?.name || '-'}</h4>
      {hasDistance && (
        <p className="text-xs text-neutral-500 font-body flex items-center gap-1 mt-1">
          {distanceLabel}: <span className="font-medium text-neutral-700">{hotel.distance_m} meter</span>
        </p>
      )}
    </div>
  );
};

const ITINERARY_PREVIEW_DAYS = 3;

const ItineraryTimeline = ({ itinerary }) => {
  const [expanded, setExpanded] = useState(false);
  const days = itinerary.days;
  const visible = expanded ? days : days.slice(0, ITINERARY_PREVIEW_DAYS);
  const hidden = days.length - visible.length;

  return (
    <div className="font-body">
      <ol>
        {visible.map((day, idx) => {
          const isLast = idx === visible.length - 1;
          const route = day.location || day.title;
          return (
            <li key={idx} className={`relative pl-11 ${isLast ? '' : 'pb-6'}`}>
              <span className="absolute left-0 top-0 w-7 h-7 rounded-full bg-primary-100 text-primary-600 font-heading font-semibold text-xs flex items-center justify-center">
                {day.day_number || (idx + 1)}
              </span>
              {!isLast && <span className="absolute left-[13px] top-8 bottom-1 w-px bg-neutral-200" />}

              <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                <h4 className="font-heading text-sm font-semibold text-neutral-900">
                  Hari {day.day_number || (idx + 1)}{route ? ` · ${route}` : ''}
                </h4>
              </div>

              {day.activities && day.activities.length > 0 ? (
                <ul className="mt-2 list-disc space-y-1.5 pl-4 text-xs leading-relaxed text-neutral-700 marker:text-neutral-400">
                  {day.activities.map((act, actIdx) => (
                    <li key={actIdx}>
                      <RichText text={[act.time, act.text].filter(Boolean).join(' ')} />
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-1 text-xs text-neutral-400">Belum ada rincian kegiatan.</p>
              )}
            </li>
          );
        })}
      </ol>

      {days.length > ITINERARY_PREVIEW_DAYS && (
        <button
          type="button"
          onClick={() => setExpanded(v => !v)}
          className="mt-4 inline-flex items-center gap-1 text-xs font-semibold text-primary-600 hover:text-primary-700 cursor-pointer"
        >
          <ChevronDown size={14} strokeWidth={2} className={expanded ? 'rotate-180' : ''} />
          {expanded ? 'Ringkas itinerary' : `Tampilkan ${hidden} hari lainnya`}
        </button>
      )}
    </div>
  );
};

const FacilityList = ({ title, items, icon: Icon, iconClass }) => (
  <div className="space-y-3">
    <h4 className="font-heading text-sm font-semibold text-neutral-900">{title}</h4>
    {items && items.length > 0 ? (
      <ul className="space-y-2 text-xs">
        {items.map((item, idx) => (
          <li key={idx} className="flex items-start gap-2 text-neutral-800">
            <Icon size={14} strokeWidth={2} className={`mt-px shrink-0 ${iconClass}`} />
            <span>{item}</span>
          </li>
        ))}
      </ul>
    ) : (
      <p className="text-xs text-neutral-400">Tidak ada item tercatat.</p>
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

        // Fetch Brands to display Brand Name
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
      <div className="flex justify-center items-center p-12 bg-white rounded-2xl border border-neutral-200 shadow-card">
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

  const transitAirports = parseTransitAirports(schedule.transit_bandara);

  // Keterangan hari di bawah tanggal berangkat: lama perjalanan (inklusif) dan hitung mundur.
  const dayStamp = (value) => {
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value || '');
    return m ? Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : null;
  };
  const DAY_MS = 24 * 60 * 60 * 1000;
  const berangkatStamp = dayStamp(schedule.berangkat_tanggal);
  const pulangStamp = dayStamp(schedule.pulang_tanggal);
  const now = new Date();
  const todayStamp = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  const durationDays = berangkatStamp !== null && pulangStamp !== null && pulangStamp >= berangkatStamp
    ? Math.round((pulangStamp - berangkatStamp) / DAY_MS) + 1
    : null;
  const daysUntil = berangkatStamp !== null ? Math.round((berangkatStamp - todayStamp) / DAY_MS) : null;
  const departureHint = [
    durationDays ? `Durasi ${durationDays} hari` : null,
    daysUntil === null ? null : (daysUntil > 0 ? `${daysUntil} hari lagi` : (daysUntil === 0 ? 'Berangkat hari ini' : 'Sudah berangkat')),
  ].filter(Boolean).join(' · ');

  const handlePrintItinerary = () => {
    if (!itinerary || !itinerary.days?.length) return;
    const tanggal = [formatDate(schedule.berangkat_tanggal), formatDate(schedule.pulang_tanggal)]
      .filter(v => v && v !== '-').join(' – ');
    printItinerary({
      logoUrl: brand?.logo_url ? assetUrl(brand.logo_url) : (brand?.icon_url ? assetUrl(brand.icon_url) : ''),
      brandName: brand?.name || '',
      title: String(schedule.jadwal_nama || '').toUpperCase(),
      subtitle: tanggal ? `Itinerary Perjalanan · ${tanggal}` : 'Itinerary Perjalanan',
      days: itinerary.days,
      accentColor: brand?.primary_color || '',
      footerItems: [
        { label: 'Maskapai', value: schedule.maskapai?.name },
        { label: 'Hotel Makkah', value: schedule.hotel_mekkah?.name },
        { label: 'Hotel Madinah', value: schedule.hotel_madinah?.name },
      ],
    });
  };

  const handlePrintPackage = () => {
    const tanggalBerangkat = formatDate(schedule.berangkat_tanggal);
    const tanggalPulang = formatDate(schedule.pulang_tanggal);
    const rute = (asal, transit, tujuan) => [asal, transit, tujuan].filter(Boolean).join(' → ');
    const hotelInfo = (hotel) => ({
      nama: hotel?.name || '',
      bintang: hotel?.star_rating > 0 ? `${hotel.star_rating} bintang` : '',
      jarak: hotel?.distance_m !== null && hotel?.distance_m !== undefined ? `${hotel.distance_m} meter` : '',
    });
    const harga = [
      ['Kamar isi 4 (Quad)', schedule.harga_quad],
      ['Kamar isi 3 (Triple)', schedule.harga_triple],
      ['Kamar isi 2 (Double)', schedule.harga_double],
      ['Bayi (Infant)', schedule.harga_infant],
    ].filter(([, value]) => Number(value) > 0).map(([label, value]) => ({ label, value: formatRupiah(value) }));
    const hargaMulai = [schedule.harga_quad, schedule.harga_triple, schedule.harga_double]
      .map(Number).filter(v => v > 0).sort((a, b) => a - b)[0];
    const catatanHarga = [
      Number(schedule.minimal_dp) > 0 ? `Uang muka minimal ${formatRupiah(schedule.minimal_dp)}` : '',
    ].filter(Boolean).join(' · ');

    printPackage({
      title: schedule.jadwal_nama,
      logoUrl: brand?.logo_url ? assetUrl(brand.logo_url) : (brand?.icon_url ? assetUrl(brand.icon_url) : ''),
      brandName: brand?.name || '',
      accentColor: brand?.primary_color || '',
      berangkat: tanggalBerangkat,
      pulang: tanggalPulang,
      durasi: departureHint ? departureHint.split(' · ')[0] : '',
      maskapai: schedule.maskapai?.name || '',
      jenisPenerbangan: schedule.is_direct_flight ? 'Direct' : 'Transit',
      hargaMulai: hargaMulai ? formatRupiah(hargaMulai) : '',
      flights: {
        berangkat: {
          tanggal: tanggalBerangkat,
          jam: formatTime(schedule.berangkat_jam),
          kode: schedule.berangkat_kode_penerbangan,
          rute: rute(schedule.berangkat_bandara_asal, transitAirports.berangkat, schedule.berangkat_bandara_tujuan),
        },
        pulang: {
          tanggal: tanggalPulang,
          jam: formatTime(schedule.pulang_jam),
          kode: schedule.pulang_kode_penerbangan,
          rute: rute(schedule.pulang_bandara_asal, transitAirports.pulang, schedule.pulang_bandara_tujuan),
        },
      },
      hotels: {
        makkah: hotelInfo(schedule.hotel_mekkah),
        madinah: hotelInfo(schedule.hotel_madinah),
        transit: (schedule.transit_hotels || []).map(th => th.nama).filter(Boolean),
      },
      prices: harga,
      priceNote: catatanHarga,
      include: schedule.include_items || [],
      exclude: schedule.exclude_items || [],
      printedAt: new Date().toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' }),
    });
  };

  const seatTotal = Number(schedule.seat_total) || 0;
  const hasSeatSisa = schedule.seat_sisa !== null && schedule.seat_sisa !== undefined;
  const hasItinerary = itinerary && itinerary.days && itinerary.days.length > 0;
  const hasInfant = schedule.harga_infant !== null && schedule.harga_infant !== undefined;
  const hasTransitHotels = schedule.transit_hotels && schedule.transit_hotels.length > 0;
  const hasBrosur = schedule.brosur_url || schedule.brosur_thumb_url;

  return (
    <div className="w-full space-y-6">
      <PageHeader
        title={
          <span className="inline-flex flex-wrap items-center gap-x-3 gap-y-1">
            {schedule.jadwal_nama}
            <StatusBadge status={schedule.status} />
          </span>
        }
        breadcrumb={false}
        subtitle={
          (brand || schedule.category?.name) ? (
            <span className="inline-flex flex-wrap items-center gap-2">
              {brand && <BrandBadge name={brand.name} color={brand.primary_color} />}
              {schedule.category?.name && <Badge variant="neutral" hideIcon className={PILL_STRONG}>{schedule.category.name}</Badge>}
            </span>
          ) : undefined
        }
        onBack={() => navigate('/schedules')}
        actionLabel="Edit Paket"
        onAction={() => navigate(`/schedules/${id}/edit`)}
      >
        <Button variant="secondary" onClick={handlePrintPackage}>
          <Printer size={16} strokeWidth={1.75} />
          <span>Cetak / PDF</span>
        </Button>
      </PageHeader>

      {/* Ringkasan: satu strip, bukan empat card */}
      <Surface className="grid grid-cols-3 divide-x divide-neutral-200">
        <SummaryItem label="Keberangkatan" value={formatDateShort(schedule.berangkat_tanggal)} hint={departureHint || undefined} />
        <SummaryItem
          label="Seat Tersisa"
          value={hasSeatSisa ? schedule.seat_sisa : '-'}
          hint={seatTotal > 0 ? `dari ${seatTotal} pax` : undefined}
        />
        <SummaryItem
          label="Booking"
          value={schedule.booking_count ?? 0}
          hint={(schedule.booking_count || 0) > 0 ? 'Lihat daftar booking →' : 'Belum ada booking'}
          onClick={(schedule.booking_count || 0) > 0 ? () => navigate(`/bookings?schedule=${schedule.id}`) : undefined}
        />
      </Surface>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* Kolom kiri: satu card, section dipisah garis tipis */}
        <Surface className="lg:col-span-2 divide-y divide-neutral-200/80">
          <Section
            title="Penerbangan"
            aside={
              schedule.is_direct_flight
                ? <Badge variant="neutral" hideIcon className="!bg-info-600 !border-info-600 !text-white">Direct</Badge>
                : <Badge variant="warning" hideIcon className="!bg-warning-600 !border-warning-600 !text-white">Transit</Badge>
            }
          >
            <div className="space-y-5 font-body">
              <div className="flex items-center gap-3">
                {schedule.maskapai?.logo_url ? (
                  <img
                    src={assetUrl(schedule.maskapai.logo_url)}
                    alt={schedule.maskapai.name}
                    className="w-10 h-10 object-contain rounded-lg"
                  />
                ) : (
                  <div className="w-10 h-10 bg-neutral-100 rounded-lg flex items-center justify-center text-neutral-500">
                    <Plane size={18} strokeWidth={1.75} />
                  </div>
                )}
                <div>
                  <span className="text-[11px] text-neutral-400 block">Maskapai</span>
                  <span className="font-heading text-sm font-semibold text-neutral-900">{schedule.maskapai?.name || '-'}</span>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 md:gap-0 md:divide-x md:divide-neutral-200/80">
                <div className="md:pr-6">
                  <FlightLeg
                    title="Berangkat"
                    iconClass="rotate-45"
                    code={schedule.berangkat_kode_penerbangan}
                    tanggal={schedule.berangkat_tanggal}
                    jam={schedule.berangkat_jam}
                    asal={schedule.berangkat_bandara_asal}
                    transit={transitAirports.berangkat}
                    tujuan={schedule.berangkat_bandara_tujuan}
                  />
                </div>
                <div className="md:pl-6">
                  <FlightLeg
                    title="Pulang"
                    iconClass="-rotate-135"
                    code={schedule.pulang_kode_penerbangan}
                    tanggal={schedule.pulang_tanggal}
                    jam={schedule.pulang_jam}
                    asal={schedule.pulang_bandara_asal}
                    transit={transitAirports.pulang}
                    tujuan={schedule.pulang_bandara_tujuan}
                  />
                </div>
              </div>
            </div>
          </Section>

          <Section title="Akomodasi Hotel">
            <div className="font-body space-y-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 md:gap-0 md:divide-x md:divide-neutral-200/80">
                <div className="md:pr-6">
                  <HotelItem title="Makkah" hotel={schedule.hotel_mekkah} distanceLabel="Jarak ke Masjidil Haram" />
                </div>
                <div className="md:pl-6">
                  <HotelItem title="Madinah" hotel={schedule.hotel_madinah} distanceLabel="Jarak ke Masjid Nabawi" />
                </div>
              </div>

              {hasTransitHotels && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-y-5 md:gap-x-0 border-t border-neutral-200/80 pt-5">
                  {schedule.transit_hotels.map((th, idx) => (
                    <div
                      key={idx}
                      className={idx % 2 === 0
                        ? 'md:pr-6'
                        : 'md:pl-6 md:border-l md:border-neutral-200/80'}
                    >
                      <HotelItem
                        title={`Hotel Transit${th.kota ? ` · ${th.kota}` : ''}`}
                        hotel={{ name: th.nama, star_rating: th.star_rating }}
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Section>

          <Section
            title="Itinerary"
            aside={hasItinerary && (
              <div className="flex items-center gap-3">
                <span className="text-xs font-medium text-neutral-500">{itinerary.days.length} Hari</span>
                <button
                  type="button"
                  onClick={handlePrintItinerary}
                  className="inline-flex items-center gap-1.5 rounded-md border border-neutral-300 bg-white px-2.5 py-1 text-xs font-semibold text-neutral-700 transition-colors hover:border-primary-300 hover:bg-primary-50 hover:text-primary-700 cursor-pointer"
                >
                  <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 9V3h12v6M6 18H4a1 1 0 01-1-1v-6a2 2 0 012-2h14a2 2 0 012 2v6a1 1 0 01-1 1h-2M7 14h10v7H7z" />
                  </svg>
                  <span>Cetak / PDF</span>
                </button>
              </div>
            )}
          >
            {hasItinerary ? (
              <ItineraryTimeline itinerary={itinerary} />
            ) : (
              <p className="text-sm text-neutral-500 font-body">Belum ada data itinerary untuk paket ini.</p>
            )}
          </Section>

          <Section title="Fasilitas Paket">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 font-body">
              <FacilityList title="Termasuk" items={schedule.include_items} icon={Check} iconClass="text-neutral-700" />
              <FacilityList title="Tidak Termasuk" items={schedule.exclude_items} icon={X} iconClass="text-neutral-400" />
            </div>
          </Section>
        </Surface>

        {/* Kolom kanan: dua card saja */}
        <div className="space-y-6">
          <Surface className="divide-y divide-neutral-200/80">
            <Section title="Harga Paket">
              <div className="font-body">
                <dl className="divide-y divide-neutral-100 text-xs">
                  {[
                    ['Quad (4 orang)', schedule.harga_quad],
                    ['Triple (3 orang)', schedule.harga_triple],
                    ['Double (2 orang)', schedule.harga_double],
                    ...(hasInfant ? [['Infant (bayi)', schedule.harga_infant]] : []),
                  ].map(([label, value]) => (
                    <div key={label} className="py-2.5 flex justify-between items-center first:pt-0 last:pb-0">
                      <dt className="text-neutral-600">{label}</dt>
                      <dd className="text-sm font-semibold font-heading text-neutral-900">{formatRupiah(value)}</dd>
                    </div>
                  ))}
                </dl>
              </div>
            </Section>

            <Section title="Gambar Utama">
              {hasBrosur ? (
                <a
                  href={assetUrl(schedule.brosur_url || schedule.brosur_thumb_url)}
                  target="_blank"
                  rel="noopener noreferrer"
                  title="Buka gambar ukuran penuh"
                  className="group relative block overflow-hidden rounded-lg border border-neutral-200 bg-neutral-50"
                >
                  <img
                    src={assetUrl(schedule.brosur_thumb_url || schedule.brosur_url)}
                    alt={`Gambar utama ${schedule.jadwal_nama}`}
                    className="w-full h-auto max-h-72 object-contain"
                  />
                  <span className="absolute right-2 top-2 inline-flex items-center gap-1 rounded-md bg-white/90 px-2 py-1 text-[11px] font-semibold text-neutral-700 opacity-0 shadow-sm transition-opacity group-hover:opacity-100">
                    Buka penuh
                    <ExternalLink size={11} strokeWidth={1.75} />
                  </span>
                </a>
              ) : (
                <p className="text-xs text-neutral-400 font-body">Belum ada gambar utama.</p>
              )}
            </Section>
          </Surface>

          <Surface>
            <Section title="Kuota Seat">
              <SeatManagement scheduleId={schedule.id} bookingCount={schedule.booking_count} />
            </Section>
          </Surface>
        </div>
      </div>
    </div>
  );
};

export default ScheduleDetailPage;

import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import Alert from '../components/ui/Alert';
import { listBrands } from '../api/brands';
import { listSchedulesAdmin } from '../api/schedules';
import { listHotels } from '../api/hotels';
import { listAirlines } from '../api/airlines';
import { listTransactions30Days } from '../api/analytics';
import {
  Plane,
  Building2,
  Hotel,
  Calendar,
  ArrowRight,
  TrendingUp,
  AlertCircle,
  CheckCircle2,
  Users,
  Plus,
  ArrowUpRight
} from 'lucide-react';
import KaabaIcon from '../../../shared/src/components/icons/KaabaIcon';

const formatDate = (value) => new Intl.DateTimeFormat('id-ID', {
  day: 'numeric', month: 'short', year: 'numeric',
}).format(new Date(`${value}T00:00:00`));

const toDateKey = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const formatCompactRupiah = (value) => new Intl.NumberFormat('id-ID', {
  style: 'currency', currency: 'IDR', notation: 'compact', maximumFractionDigits: 1,
}).format(value);

// Attention Metric Card (Matching Altezza Travel Reference)
const AttentionCard = ({ icon: Icon, title, count, actionLabel, actionLink, badgeText, badgeVariant = 'warning' }) => {
  const isWarning = badgeVariant === 'warning';
  
  return (
    <div className="bg-white border border-neutral-200/80 rounded-2xl p-5 shadow-card hover:shadow-card-hover transition-all duration-200 flex flex-col justify-between group">
      <div>
        <div className="flex items-start justify-between gap-3 mb-3">
          <div className="w-10 h-10 rounded-xl bg-neutral-100/90 text-neutral-700 flex items-center justify-center shrink-0">
            <Icon className="w-5 h-5" />
          </div>
          {badgeText && (
            <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold tracking-wide ${
              isWarning ? 'bg-warning-50 text-warning-700 border border-warning-200/80' : 'bg-success-50 text-success-700 border border-success-200/80'
            }`}>
              {isWarning ? <AlertCircle className="w-3 h-3 text-warning-600" /> : <CheckCircle2 className="w-3 h-3 text-success-600" />}
              <span>{badgeText}</span>
            </span>
          )}
        </div>

        <div className="mt-2">
          <span className="text-3xl font-heading font-extrabold text-neutral-900 tracking-tight">
            {count}
          </span>
          <p className="text-xs font-semibold text-neutral-500 font-heading mt-1 uppercase tracking-wider">
            {title}
          </p>
        </div>
      </div>

      <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between">
        <Link
          to={actionLink}
          className="text-xs font-semibold text-neutral-700 group-hover:text-primary-700 flex items-center gap-1.5 transition-colors"
        >
          <span>{actionLabel}</span>
          <ArrowRight className="w-3.5 h-3.5 transition-transform group-hover:translate-x-0.5" />
        </Link>
      </div>
    </div>
  );
};

const TransactionsChart = ({ brands, transactions, isLoading }) => {
  const width = 900;
  const height = 240;
  const padding = { top: 20, right: 20, bottom: 35, left: 75 };
  const dates = Array.from({ length: 30 }, (_, index) => {
    const date = new Date();
    date.setHours(0, 0, 0, 0);
    date.setDate(date.getDate() - (29 - index));
    return date;
  });
  const activeBrandIds = [...new Set(transactions.map((item) => Number(item.brand_id)))];
  const series = activeBrandIds.map((brandId, index) => {
    const valuesByDate = new Map(
      transactions.filter((item) => Number(item.brand_id) === brandId).map((item) => [item.date, Number(item.total_amount) || 0]),
    );
    return {
      brandId,
      name: brands.find((brand) => Number(brand.id) === brandId)?.name || `Brand ${brandId}`,
      color: `hsl(${(brandId * 137.508 + index * 47) % 360} 68% 42%)`,
      values: dates.map((date) => valuesByDate.get(toDateKey(date)) || 0),
    };
  });
  const maxValue = Math.max(1, ...series.flatMap((item) => item.values));
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;
  const x = (index) => padding.left + (index / 29) * plotWidth;
  const y = (value) => padding.top + plotHeight - (value / maxValue) * plotHeight;
  const labelIndexes = [0, 7, 14, 21, 29];

  return (
    <section className="rounded-2xl border border-neutral-200/80 bg-white shadow-card overflow-hidden" aria-labelledby="transaction-chart-title">
      <div className="flex flex-col gap-3 border-b border-neutral-200/80 px-6 py-5 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-primary-600" />
            <h2 id="transaction-chart-title" className="font-heading font-bold text-neutral-900 text-base">
              Grafik Transaksi Masuk (30 Hari Terakhir)
            </h2>
          </div>
          <p className="mt-1 text-xs text-neutral-500 font-body">Nominal pembayaran terkonfirmasi per brand hingga hari ini</p>
        </div>
        <div className="flex max-w-full flex-wrap gap-x-4 gap-y-2" aria-label="Legenda brand">
          {series.map((item) => (
            <span key={item.brandId} className="flex items-center gap-1.5 text-xs font-semibold text-neutral-700 bg-neutral-50 px-2.5 py-1 rounded-full border border-neutral-200/60">
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.color }} />
              {item.name}
            </span>
          ))}
        </div>
      </div>
      <div className="p-4 md:p-6">
        {isLoading ? (
          <div className="h-60 animate-pulse rounded-xl bg-neutral-100" />
        ) : series.length === 0 ? (
          <div className="flex h-60 flex-col items-center justify-center text-center">
            <p className="text-sm font-medium text-neutral-700 font-heading">Belum ada transaksi terkonfirmasi</p>
            <p className="mt-1 text-xs text-neutral-500 font-body">Transaksi 30 hari terakhir akan tampil di grafik ini.</p>
          </div>
        ) : (
          <div className="overflow-x-auto" role="img" aria-label={`Grafik transaksi 30 hari untuk ${series.length} brand`}>
            <svg viewBox={`0 0 ${width} ${height}`} className="min-w-[680px] w-full" aria-hidden="true">
              {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
                const gridY = padding.top + plotHeight * ratio;
                const value = maxValue * (1 - ratio);
                return (
                  <g key={ratio}>
                    <line x1={padding.left} x2={width - padding.right} y1={gridY} y2={gridY} stroke="#F0F1F3" strokeDasharray="4 5"/>
                    <text x={padding.left - 12} y={gridY + 4} textAnchor="end" fontSize="10" fontWeight="600" fill="#8C95A0" fontFamily="DM Sans">
                      {formatCompactRupiah(value).replace('Rp', '').trim()}
                    </text>
                  </g>
                );
              })}
              {labelIndexes.map((index) => (
                <text key={index} x={x(index)} y={height - 8} textAnchor={index === 0 ? 'start' : index === 29 ? 'end' : 'middle'} fontSize="10" fontWeight="500" fill="#8C95A0" fontFamily="DM Sans">
                  {new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short' }).format(dates[index])}
                </text>
              ))}
              {series.map((item) => {
                const points = item.values.map((value, index) => `${x(index)},${y(value)}`).join(' ');
                return (
                  <g key={item.brandId}>
                    <polyline points={points} fill="none" stroke={item.color} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" vectorEffect="non-scaling-stroke"/>
                    {item.values.map((value, index) => value > 0 && (
                      <circle key={index} cx={x(index)} cy={y(value)} r="4" fill="white" stroke={item.color} strokeWidth="2.5">
                        <title>{item.name} · {formatDate(toDateKey(dates[index]))} · {formatCompactRupiah(value)}</title>
                      </circle>
                    ))}
                  </g>
                );
              })}
            </svg>
          </div>
        )}
      </div>
    </section>
  );
};

const DashboardHome = () => {
  const [data, setData] = useState({ brands: [], schedules: [], hotels: [], airlines: [], transactions: [] });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    Promise.all([listBrands(), listSchedulesAdmin(), listHotels(), listAirlines(), listTransactions30Days()])
      .then(([brands, schedules, hotels, airlines, transactions]) => {
        if (active) setData({ brands: brands || [], schedules: schedules || [], hotels: hotels || [], airlines: airlines || [], transactions: transactions || [] });
      })
      .catch((err) => {
        if (active) setError(err.response?.data?.error || 'Ringkasan dashboard gagal dimuat.');
      })
      .finally(() => {
        if (active) setIsLoading(false);
      });
    return () => { active = false; };
  }, []);

  const summary = useMemo(() => {
    const published = data.schedules.filter((item) => item.status === 'published');
    const draft = data.schedules.filter((item) => item.status === 'draft').length;
    const archived = data.schedules.filter((item) => item.status === 'archived').length;
    const now = new Date();
    now.setHours(0, 0, 0, 0);
    const departures = published
      .filter((item) => item.berangkat_tanggal && new Date(`${item.berangkat_tanggal}T00:00:00`) >= now)
      .sort((a, b) => a.berangkat_tanggal.localeCompare(b.berangkat_tanggal))
      .slice(0, 5);
    const seats = published.reduce((total, item) => total + (Number(item.seat_sisa) || 0), 0);
    return { published, draft, archived, departures, seats };
  }, [data.schedules]);

  const brandById = useMemo(() => Object.fromEntries(data.brands.map((brand) => [brand.id, brand])), [data.brands]);

  return (
    <main className="space-y-6 pb-12">
      {/* Header Banner with Altezza Segmented Badges */}
      <section className="bg-white border border-neutral-200/80 rounded-2xl p-6 md:p-8 shadow-card flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <div className="flex flex-wrap items-center gap-2 mb-3">
            <span className="inline-flex items-center gap-1.5 bg-neutral-900 text-primary-500 px-3 py-1 rounded-full text-xs font-heading font-bold shadow-xs">
              <KaabaIcon className="w-3.5 h-3.5" />
              Azhan ERP System
            </span>
            <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200/60 px-2.5 py-0.5 rounded-full text-xs font-semibold">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
              {data.brands.length} Brand Terhubung
            </span>
          </div>
          <h1 className="text-2xl md:text-3xl font-heading font-extrabold text-neutral-900 tracking-tight">
            Dashboard Utama
          </h1>
          <p className="mt-1 text-sm text-neutral-500 font-body max-w-xl">
            Monitoring terpusat seluruh brand, paket perjalanan umroh, kuota kursi, dan verifikasi transaksi jamaah.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3 shrink-0">
          <Link
            to="/brands/new"
            className="inline-flex items-center gap-2 bg-white hover:bg-neutral-50 text-neutral-800 border border-neutral-200/90 px-4 py-2.5 rounded-xl text-xs font-bold font-heading shadow-xs hover:border-neutral-300 transition-all"
          >
            <Building2 className="w-4 h-4 text-neutral-500" />
            <span>Tambah Brand</span>
          </Link>
          <Link
            to="/schedules/new"
            className="inline-flex items-center gap-2 bg-primary-500 hover:bg-primary-600 text-brand-dark px-4 py-2.5 rounded-xl text-xs font-bold font-heading shadow-xs transition-all hover:shadow"
          >
            <Plus className="w-4 h-4" />
            <span>Buat Paket Baru</span>
          </Link>
        </div>
      </section>

      {error && <Alert variant="error">{error}</Alert>}

      {/* Attention & Operational Summary Cards (Altezza Reference Style) */}
      <div>
        <div className="flex items-center justify-between mb-3 px-1">
          <h2 className="text-xs font-bold font-heading uppercase tracking-wider text-neutral-500">
            Perhatian Operasional & Kesiapan Data
          </h2>
        </div>
        <section aria-label="Perhatian Operasional" className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <AttentionCard
            icon={Plane}
            title="Keberangkatan Mendatang"
            count={isLoading ? '—' : summary.departures.length}
            badgeText="Kesiapan Paket"
            badgeVariant="warning"
            actionLabel="Lihat Jadwal Paket"
            actionLink="/schedules"
          />
          <AttentionCard
            icon={Users}
            title="Kapasitas Kursi Tersedia"
            count={isLoading ? '—' : summary.seats}
            badgeText="Sisa Kuota Aktif"
            badgeVariant="success"
            actionLabel="Kelola Kapasitas"
            actionLink="/schedules"
          />
          <AttentionCard
            icon={AlertCircle}
            title="Paket Berstatus Draft"
            count={isLoading ? '—' : summary.draft}
            badgeText={summary.draft > 0 ? "Perlu Review" : "Semua Terbit"}
            badgeVariant={summary.draft > 0 ? "warning" : "success"}
            actionLabel="Tinjau Draft"
            actionLink="/schedules"
          />
          <AttentionCard
            icon={Hotel}
            title="Mitra Hotel & Maskapai"
            count={isLoading ? '—' : (data.hotels.length + data.airlines.length)}
            badgeText="Master Terverifikasi"
            badgeVariant="success"
            actionLabel="Lihat Master Vendor"
            actionLink="/hotels"
          />
        </section>
      </div>

      {/* Transaction 30 Days Trend */}
      <TransactionsChart brands={data.brands} transactions={data.transactions} isLoading={isLoading} />

      {/* Bottom Grid: Departures & Distribution */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Keberangkatan Terdekat */}
        <section className="lg:col-span-2 rounded-2xl border border-neutral-200/80 bg-white shadow-card overflow-hidden">
          <div className="flex items-center justify-between border-b border-neutral-200/80 px-6 py-5">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-neutral-100 text-neutral-800 flex items-center justify-center">
                <Calendar className="w-4 h-4" />
              </div>
              <div>
                <h2 className="font-heading font-bold text-neutral-900 text-base">Keberangkatan Terdekat</h2>
                <p className="text-xs text-neutral-500 font-body">Paket perjalanan aktif yang akan segera berangkat</p>
              </div>
            </div>
            <Link
              to="/schedules"
              className="inline-flex items-center gap-1.5 text-xs font-bold text-neutral-700 hover:text-neutral-900 bg-neutral-100 hover:bg-neutral-200/80 px-3 py-1.5 rounded-xl transition-colors font-heading"
            >
              <span>Semua Paket</span>
              <ArrowUpRight className="w-3.5 h-3.5 text-neutral-500" />
            </Link>
          </div>

          <div className="divide-y divide-neutral-100 px-6">
            {isLoading && [1, 2, 3].map((item) => (
              <div key={item} className="my-4 h-16 animate-pulse rounded-xl bg-neutral-100" />
            ))}

            {!isLoading && summary.departures.length === 0 && (
              <div className="py-12 text-center">
                <Calendar className="w-8 h-8 text-neutral-300 mx-auto mb-2" />
                <p className="text-sm font-semibold text-neutral-700 font-heading">Belum ada keberangkatan mendatang</p>
                <p className="mt-1 text-xs text-neutral-500 font-body">Terbitkan paket agar tampil di jadwal ini.</p>
              </div>
            )}

            {!isLoading && summary.departures.map((item) => (
              <Link
                key={item.id}
                to={`/schedules/${item.id}/edit`}
                className="group flex items-center justify-between gap-4 py-4 hover:bg-neutral-50/70 -mx-6 px-6 transition-colors"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-10 h-10 rounded-xl bg-neutral-100 flex items-center justify-center text-neutral-600 shrink-0 group-hover:bg-primary-100 group-hover:text-primary-700 transition-colors">
                    <Plane className="w-5 h-5" />
                  </div>
                  <div className="min-w-0">
                    <h3 className="text-sm font-bold text-neutral-900 group-hover:text-primary-700 font-heading truncate">
                      {item.jadwal_nama}
                    </h3>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-[11px] font-semibold bg-neutral-100 text-neutral-700 px-2 py-0.5 rounded-md">
                        {brandById[item.brand_id]?.name || 'Brand'}
                      </span>
                      <span className="text-xs text-neutral-500 font-body">
                        {formatDate(item.berangkat_tanggal)}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="text-right shrink-0">
                  <span className="inline-block text-xs font-bold font-mono px-2.5 py-1 rounded-full bg-emerald-50 text-emerald-700 border border-emerald-200/60">
                    {item.seat_sisa}/{item.seat_total} Kursi
                  </span>
                  <span className="block text-[10px] text-neutral-400 font-body mt-0.5">tersedia</span>
                </div>
              </Link>
            ))}
          </div>
        </section>

        {/* Right Sidebar Widget: Status Paket & Kapasitas */}
        <aside className="space-y-6">
          {/* Status Paket Distribution */}
          <section className="rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-card">
            <h2 className="font-heading font-bold text-neutral-900 text-base">Status Distribusi Paket</h2>
            <p className="text-xs text-neutral-500 font-body mt-0.5">Proporsi seluruh jadwal paket terdaftar</p>
            
            <div className="mt-6 space-y-4">
              {[
                ['Terbit (Published)', summary.published.length, 'bg-emerald-500'],
                ['Draft (Review)', summary.draft, 'bg-amber-400'],
                ['Diarsipkan', summary.archived, 'bg-neutral-400'],
              ].map(([label, count, color]) => {
                const percentage = data.schedules.length ? Math.round((count / data.schedules.length) * 100) : 0;
                return (
                  <div key={label}>
                    <div className="mb-1.5 flex justify-between text-xs font-heading font-bold">
                      <span className="text-neutral-700">{label}</span>
                      <span className="text-neutral-900">{count} ({percentage}%)</span>
                    </div>
                    <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
                      <div className={`h-full rounded-full ${color} transition-all duration-500`} style={{ width: `${percentage}%` }} />
                    </div>
                  </div>
                );
              })}
            </div>
          </section>

          {/* Quick Capacity Alert Card */}
          <section className="rounded-2xl border border-primary-500/60 bg-gradient-to-br from-primary-100 to-white p-6 shadow-card">
            <span className="text-[10px] font-extrabold uppercase tracking-widest text-amber-900 bg-primary-500/30 px-2.5 py-1 rounded-full">
              Kapasitas Aktif Grup
            </span>
            <p className="mt-3 text-4xl font-heading font-extrabold tracking-tight text-neutral-900">
              {isLoading ? '—' : summary.seats}
            </p>
            <p className="mt-1 text-xs text-neutral-600 font-body">
              Total kursi tersisa dari seluruh paket yang telah diterbitkan aktif.
            </p>
            <div className="mt-5">
              <Link
                to="/schedules"
                className="inline-flex items-center gap-1.5 text-xs font-bold text-neutral-900 hover:text-neutral-700 font-heading"
              >
                <span>Kelola kuota kursi paket</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </Link>
            </div>
          </section>
        </aside>
      </div>
    </main>
  );
};

export default DashboardHome;

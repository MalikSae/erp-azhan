import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Plus, Plane, Users, Hotel, AlertCircle, CalendarDays, Home, ChevronRight,
  Wallet, ArrowUpRight, Package, TrendingUp
} from 'lucide-react';
import {
  ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip,
  PieChart, Pie, Cell, Legend
} from 'recharts';
import Alert from '../components/ui/Alert';
import { listBrands } from '../api/brands';
import { listSchedulesAdmin } from '../api/schedules';
import { listHotels } from '../api/hotels';
import { listAirlines } from '../api/airlines';
import { listPax30Days, listTransactions30Days } from '../api/analytics';

// ── Util ─────────────────────────────────────────────────────────────────────
const toDateKey = (date) => {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
};

const formatShortDate = (value) => new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short' }).format(new Date(`${value}T00:00:00`));
const formatLongDate = (value) => new Intl.DateTimeFormat('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }).format(value);
const formatCompactRupiah = (value) => new Intl.NumberFormat('id-ID', { notation: 'compact', maximumFractionDigits: 1 }).format(value || 0);

const BRAND_FALLBACK_COLORS = ['#F26522', '#3B82F6', '#22C55E', '#8B5CF6', '#F59E0B'];

// ── Potongan UI gaya SmartHR ─────────────────────────────────────────────────
// Header kartu: judul tebal kiri + pill kecil kanan.
const CardHeader = ({ title, right }) => (
  <div className="flex items-center justify-between px-5 py-4 border-b border-neutral-100">
    <h3 className="text-[15px] font-heading font-bold text-neutral-900">{title}</h3>
    {right}
  </div>
);

const PillLink = ({ to = '#', children }) => (
  <Link
    to={to}
    className="inline-flex items-center gap-1 text-xs font-medium font-body text-neutral-600 border border-neutral-200 rounded-lg px-2.5 py-1.5 bg-white hover:bg-neutral-50 transition-colors"
  >
    {children}
  </Link>
);

const DeltaChip = ({ value, positive = true }) => (
  <span className={`inline-flex items-center gap-0.5 text-[11px] font-semibold px-1.5 py-0.5 rounded-md ${positive ? 'bg-success-50 text-success-700' : 'bg-danger-50 text-danger-700'}`}>
    <ArrowUpRight className={`w-3 h-3 ${positive ? '' : 'rotate-90'}`} />
    {value}
  </span>
);

// Statistik ringkas: ikon lingkaran berwarna + nilai + label + delta.
const OverviewStat = ({ icon: Icon, tint, value, label, delta, positive }) => (
  <div className="flex items-start gap-3 p-4 rounded-xl border border-neutral-100 bg-white">
    <div className={`w-10 h-10 rounded-full flex items-center justify-center shrink-0 ${tint}`}>
      <Icon className="w-[18px] h-[18px]" strokeWidth={2} />
    </div>
    <div className="min-w-0">
      <div className="flex items-center gap-2">
        <span className="text-xl font-heading font-bold text-neutral-900 leading-tight">{value}</span>
        {delta && <DeltaChip value={delta} positive={positive} />}
      </div>
      <p className="text-xs font-body text-neutral-500 truncate">{label}</p>
    </div>
  </div>
);

// ── Halaman ──────────────────────────────────────────────────────────────────
const DashboardHome = () => {
  const [data, setData] = useState({ brands: [], schedules: [], hotels: [], airlines: [], pax: [], trx: [] });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    Promise.all([
      listBrands(), listSchedulesAdmin(), listHotels(), listAirlines(),
      listPax30Days(), listTransactions30Days(),
    ])
      .then(([brands, schedules, hotels, airlines, pax, trx]) => {
        if (active) setData({ brands: brands || [], schedules: schedules || [], hotels: hotels || [], airlines: airlines || [], pax: pax || [], trx: trx || [] });
      })
      .catch((err) => { if (active) setError(err.response?.data?.error || 'Ringkasan dashboard gagal dimuat.'); })
      .finally(() => { if (active) setIsLoading(false); });
    return () => { active = false; };
  }, []);

  const brandColor = (brandId, index = 0) => {
    const brand = data.brands.find((b) => Number(b.id) === Number(brandId));
    return brand?.primary_color || BRAND_FALLBACK_COLORS[index % BRAND_FALLBACK_COLORS.length];
  };
  const brandName = (brandId) => data.brands.find((b) => Number(b.id) === Number(brandId))?.name || `Brand ${brandId}`;

  const summary = useMemo(() => {
    const published = data.schedules.filter((s) => s.status === 'published');
    const draft = data.schedules.filter((s) => s.status === 'draft');
    const archived = data.schedules.filter((s) => s.status === 'archived');
    const now = new Date(); now.setHours(0, 0, 0, 0);
    const departures = published
      .filter((s) => s.berangkat_tanggal && new Date(`${s.berangkat_tanggal}T00:00:00`) >= now)
      .sort((a, b) => a.berangkat_tanggal.localeCompare(b.berangkat_tanggal));
    const seats = published.reduce((t, s) => t + (Number(s.seat_sisa) || 0), 0);
    const totalPax30 = data.pax.reduce((t, p) => t + (Number(p.pax_count) || 0), 0);
    const totalTrx30 = data.trx.reduce((t, p) => t + (Number(p.total_amount) || 0), 0);
    const trxCount30 = data.trx.reduce((t, p) => t + (Number(p.count) || 0), 0);
    return { published, draft, archived, departures, seats, totalPax30, totalTrx30, trxCount30 };
  }, [data]);

  // Deret 30 hari untuk area chart (kolom per brand aktif).
  const chart = useMemo(() => {
    const dates = Array.from({ length: 30 }, (_, i) => { const d = new Date(); d.setDate(d.getDate() - (29 - i)); return d; });
    const brandIds = [...new Set(data.pax.map((p) => Number(p.brand_id)))];
    const byKey = new Map(data.pax.map((p) => [`${p.brand_id}|${p.date}`, Number(p.pax_count) || 0]));
    const rows = dates.map((d) => {
      const row = { tanggal: formatShortDate(toDateKey(d)) };
      brandIds.forEach((id) => { row[`b${id}`] = byKey.get(`${id}|${toDateKey(d)}`) || 0; });
      return row;
    });
    return { rows, brandIds };
  }, [data.pax, data.brands]);

  // Distribusi pax 30 hari per brand (donut).
  const donut = useMemo(() => {
    const totals = new Map();
    data.pax.forEach((p) => totals.set(Number(p.brand_id), (totals.get(Number(p.brand_id)) || 0) + (Number(p.pax_count) || 0)));
    return [...totals.entries()].map(([id, value], i) => ({ name: brandName(id), value, color: brandColor(id, i) }));
  }, [data.pax, data.brands]);

  const totalPaket = data.schedules.length || 1;
  const segments = [
    { label: 'Published', value: summary.published.length, color: 'bg-primary-500' },
    { label: 'Draft', value: summary.draft.length, color: 'bg-warning-400' },
    { label: 'Archived', value: summary.archived.length, color: 'bg-neutral-300' },
  ];

  return (
    <main className="space-y-4 pb-10">
      {/* Baris judul + breadcrumb + aksi (ala header HR Dashboard) */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <h1 className="text-lg md:text-xl font-heading font-bold text-neutral-900 tracking-tight">Dashboard Utama</h1>
          <div className="mt-1 flex items-center gap-1.5 text-xs font-body text-neutral-500">
            <Home className="w-3.5 h-3.5" />
            <span>Dashboard</span>
            <ChevronRight className="w-3 h-3 text-neutral-300" />
            <span className="text-neutral-700 font-medium">Dashboard Utama</span>
          </div>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <span className="inline-flex items-center gap-1.5 text-xs font-body text-neutral-600 border border-neutral-200 rounded-lg px-2.5 py-1.5 bg-white">
            <CalendarDays className="w-3.5 h-3.5 text-neutral-400" />
            {formatLongDate(new Date())}
          </span>
          <Link
            to="/schedules/new"
            className="inline-flex items-center gap-1.5 bg-primary-500 hover:bg-primary-600 text-white px-3.5 py-2 rounded-lg text-xs font-bold font-heading shadow-xs transition-all"
          >
            <Plus className="w-4 h-4" />
            Buat Paket Baru
          </Link>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {/* Baris 1: status paket (bar bersegmen) + statistik ringkas */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
        <section className="lg:col-span-5 bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden">
          <CardHeader title="Status Paket & Kuota" right={<PillLink to="/schedules">Lihat Semua</PillLink>} />
          <div className="p-5 space-y-4">
            <div className="flex h-3 w-full overflow-hidden rounded-full bg-neutral-100">
              {segments.map((s) => (
                <div key={s.label} className={`${s.color} h-full`} style={{ width: `${Math.max(2, (s.value / totalPaket) * 100)}%` }} />
              ))}
            </div>
            <div className="grid grid-cols-3 divide-x divide-neutral-100">
              {segments.map((s) => (
                <div key={s.label} className="px-3 first:pl-0">
                  <div className="text-xl font-heading font-bold text-neutral-900">{isLoading ? '—' : s.value}</div>
                  <div className="flex items-center gap-1.5 text-[11px] font-body text-neutral-500">
                    <span className={`w-2 h-2 rounded-sm ${s.color}`} />
                    {s.label}
                  </div>
                </div>
              ))}
            </div>
            <div className="flex items-center justify-between rounded-xl bg-neutral-50 border border-neutral-100 px-4 py-3">
              <span className="text-xs font-body text-neutral-500">Kapasitas kursi tersedia (published)</span>
              <span className="text-base font-heading font-bold text-neutral-900">{isLoading ? '—' : summary.seats}</span>
            </div>
          </div>
        </section>

        <section className="lg:col-span-7 bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden">
          <CardHeader title="Statistik Ringkas" right={<span className="text-xs font-body text-neutral-400">30 hari terakhir</span>} />
          <div className="p-4 grid grid-cols-1 sm:grid-cols-2 gap-3">
            <OverviewStat icon={Users} tint="bg-primary-100 text-primary-600" value={isLoading ? '—' : summary.totalPax30} label="Pax Terdaftar (30 Hari)" delta={`${summary.trxCount30} trx`} positive />
            <OverviewStat icon={Wallet} tint="bg-success-100 text-success-600" value={isLoading ? '—' : `Rp ${formatCompactRupiah(summary.totalTrx30)}`} label="Transaksi Terverifikasi (30 Hari)" />
            <OverviewStat icon={Plane} tint="bg-info-100 text-info-600" value={isLoading ? '—' : summary.departures.length} label="Keberangkatan Mendatang" />
            <OverviewStat icon={Hotel} tint="bg-violet-100 text-violet-600" value={isLoading ? '—' : data.hotels.length + data.airlines.length} label="Mitra Hotel & Maskapai" />
          </div>
        </section>
      </div>

      {/* Baris 2: tren pax (area chart) + CTA gelap & draft */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
        <section className="lg:col-span-8 bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden">
          <CardHeader
            title="Tren Pendaftaran Pax"
            right={<span className="inline-flex items-center gap-1 text-xs font-medium font-body text-neutral-600 border border-neutral-200 rounded-lg px-2.5 py-1.5"><TrendingUp className="w-3.5 h-3.5" />30 Hari</span>}
          />
          <div className="px-2 pt-4 pb-2 h-[260px]">
            {isLoading ? (
              <div className="h-full flex items-center justify-center text-sm text-neutral-400 font-body">Memuat grafik…</div>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chart.rows} margin={{ top: 4, right: 16, left: -14, bottom: 0 }}>
                  <defs>
                    {chart.brandIds.map((id, i) => (
                      <linearGradient key={id} id={`grad-b${id}`} x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={brandColor(id, i)} stopOpacity={0.22} />
                        <stop offset="100%" stopColor={brandColor(id, i)} stopOpacity={0.02} />
                      </linearGradient>
                    ))}
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E5E7EB" />
                  <XAxis dataKey="tanggal" tick={{ fontSize: 11, fill: '#9CA3AF' }} tickLine={false} axisLine={false} interval={6} />
                  <YAxis tick={{ fontSize: 11, fill: '#9CA3AF' }} tickLine={false} axisLine={false} allowDecimals={false} />
                  <Tooltip
                    contentStyle={{ borderRadius: 12, border: '1px solid #E5E7EB', fontSize: 12, fontFamily: 'Inter' }}
                    formatter={(value, key) => [value, brandName(String(key).slice(1))]}
                  />
                  {chart.brandIds.map((id, i) => (
                    <Area key={id} type="monotone" dataKey={`b${id}`} stroke={brandColor(id, i)} strokeWidth={2} fill={`url(#grad-b${id})`} dot={false} activeDot={{ r: 4 }} />
                  ))}
                </AreaChart>
              </ResponsiveContainer>
            )}
          </div>
        </section>

        <div className="lg:col-span-4 flex flex-col gap-4">
          {/* Kartu CTA gelap ala "Employees in Training" */}
          <section className="rounded-2xl bg-neutral-900 text-white shadow-card p-5 flex items-center justify-between gap-4">
            <div>
              <p className="text-xs font-body text-neutral-400">Paket menunggu review</p>
              <p className="text-2xl font-heading font-bold mt-0.5">{isLoading ? '—' : summary.draft.length} <span className="text-sm font-medium text-neutral-300">draft</span></p>
            </div>
            <Link to="/schedules" className="inline-flex items-center gap-1.5 bg-primary-500 hover:bg-primary-600 text-white text-xs font-bold font-heading px-3 py-2 rounded-lg transition-colors shrink-0">
              Tinjau <ArrowUpRight className="w-3.5 h-3.5" />
            </Link>
          </section>

          {/* Donut distribusi pax per brand */}
          <section className="flex-1 bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden">
            <CardHeader title="Distribusi Pax per Brand" />
            <div className="h-[170px] px-2">
              {donut.length === 0 ? (
                <div className="h-full flex items-center justify-center text-xs text-neutral-400 font-body">Belum ada data 30 hari terakhir</div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={donut} dataKey="value" nameKey="name" innerRadius={42} outerRadius={62} paddingAngle={3} strokeWidth={0}>
                      {donut.map((d) => <Cell key={d.name} fill={d.color} />)}
                    </Pie>
                    <Legend verticalAlign="middle" align="right" layout="vertical" iconType="circle" iconSize={8} wrapperStyle={{ fontSize: 12, fontFamily: 'Inter' }} />
                    <Tooltip contentStyle={{ borderRadius: 12, border: '1px solid #E5E7EB', fontSize: 12, fontFamily: 'Inter' }} />
                  </PieChart>
                </ResponsiveContainer>
              )}
            </div>
          </section>
        </div>
      </div>

      {/* Baris 3: keberangkatan terdekat */}
      <section className="bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden">
        <CardHeader title="Keberangkatan Terdekat" right={<PillLink to="/schedules">Lihat Semua</PillLink>} />
        {isLoading ? (
          <div className="p-6 text-sm text-neutral-400 font-body">Memuat…</div>
        ) : summary.departures.length === 0 ? (
          <div className="p-6 text-sm text-neutral-400 font-body">Tidak ada keberangkatan terjadwal.</div>
        ) : (
          <ul className="divide-y divide-neutral-100">
            {summary.departures.slice(0, 5).map((s, i) => {
              const color = brandColor(s.brand_id, i);
              const total = Number(s.seat_total) || 0;
              const sisa = Number(s.seat_sisa) || 0;
              const terisi = Math.max(0, total - sisa);
              const pct = total > 0 ? Math.round((terisi / total) * 100) : 0;
              return (
                <li key={s.id} className="flex items-center gap-4 px-5 py-3.5 hover:bg-neutral-50 transition-colors">
                  <div className="w-11 h-11 rounded-xl border border-neutral-200 flex flex-col items-center justify-center shrink-0 bg-white">
                    <span className="text-sm font-heading font-bold text-neutral-900 leading-none">{new Date(`${s.berangkat_tanggal}T00:00:00`).getDate()}</span>
                    <span className="text-[10px] font-body text-neutral-400 uppercase">{new Intl.DateTimeFormat('id-ID', { month: 'short' }).format(new Date(`${s.berangkat_tanggal}T00:00:00`))}</span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-[13px] font-heading font-semibold text-neutral-900 truncate">{s.jadwal_nama || `Paket #${s.id}`}</p>
                    <p className="text-[11px] font-body text-neutral-500 flex items-center gap-1.5">
                      <span className="w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
                      {brandName(s.brand_id)}
                    </p>
                  </div>
                  <div className="hidden sm:block w-40">
                    <div className="flex justify-between text-[11px] font-body text-neutral-500 mb-1">
                      <span>{terisi}/{total} kursi</span><span>{pct}%</span>
                    </div>
                    <div className="h-1.5 rounded-full bg-neutral-100 overflow-hidden">
                      <div className="h-full rounded-full" style={{ width: `${pct}%`, backgroundColor: color }} />
                    </div>
                  </div>
                  <span className={`text-[11px] font-semibold px-2 py-1 rounded-md ${sisa > 0 ? 'bg-success-50 text-success-700' : 'bg-danger-50 text-danger-700'}`}>
                    {sisa > 0 ? `${sisa} tersisa` : 'Penuh'}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <p className="text-center text-[11px] font-body text-neutral-400 pt-2">2026 © Azhan Grup — Master ERP</p>
    </main>
  );
};

export default DashboardHome;

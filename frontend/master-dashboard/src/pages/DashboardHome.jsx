import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import Alert from '../components/ui/Alert';
import { listBrands } from '../api/brands';
import { listSchedulesAdmin } from '../api/schedules';
import { listHotels } from '../api/hotels';
import { listAirlines } from '../api/airlines';
import { listTransactions30Days } from '../api/analytics';

const icons = {
  brand: <><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></>,
  package: <><path d="M12 3 4 7.5v9L12 21l8-4.5v-9L12 3Z"/><path d="m4.5 7.8 7.5 4.1 7.5-4.1M12 12v9"/></>,
  hotel: <><path d="M4 21V5a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v16M9 21v-4h3v4M8 7h1M12 7h1M8 11h1M12 11h1M17 9h2a1 1 0 0 1 1 1v11"/></>,
  plane: <><path d="M10.5 19 9 13l-6-2V9l6 .5L12 3h2l-1 6.5 6 .5v2l-6 1-1 6h-1.5Z"/></>,
  calendar: <><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 10h18"/></>,
  arrow: <><path d="M5 12h14M13 6l6 6-6 6"/></>,
};

const Icon = ({ name, className = 'h-5 w-5' }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    {icons[name]}
  </svg>
);

const StatCard = ({ icon, label, value, detail, tone = 'neutral' }) => {
  const tones = {
    primary: 'bg-primary-50 text-primary-700',
    success: 'bg-success-50 text-success-700',
    neutral: 'bg-neutral-100 text-neutral-700',
    warning: 'bg-warning-50 text-warning-700',
  };
  return (
    <article className="rounded-xl border border-neutral-200 bg-white p-4 md:p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-neutral-500">{label}</p>
          <p className="mt-2 text-2xl font-bold tracking-tight text-neutral-900">{value}</p>
          <p className="mt-1 text-xs text-neutral-500">{detail}</p>
        </div>
        <span className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${tones[tone]}`}><Icon name={icon} /></span>
      </div>
    </article>
  );
};

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

const TransactionsChart = ({ brands, transactions, isLoading }) => {
  const width = 900;
  const height = 250;
  const padding = { top: 18, right: 18, bottom: 35, left: 70 };
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
    <section className="rounded-xl border border-neutral-200 bg-white" aria-labelledby="transaction-chart-title">
      <div className="flex flex-col gap-3 border-b border-neutral-200 px-5 py-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h2 id="transaction-chart-title" className="font-semibold text-neutral-900">Transaksi 30 hari terakhir</h2>
          <p className="mt-1 text-xs text-neutral-500">Nominal pembayaran terkonfirmasi per brand hingga hari ini</p>
        </div>
        <div className="flex max-w-full flex-wrap gap-x-4 gap-y-2" aria-label="Legenda brand">
          {series.map((item) => (
            <span key={item.brandId} className="flex items-center gap-1.5 text-xs font-medium text-neutral-600">
              <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: item.color }} />{item.name}
            </span>
          ))}
        </div>
      </div>
      <div className="p-3 md:p-5">
        {isLoading ? (
          <div className="h-64 animate-pulse rounded-lg bg-neutral-100" />
        ) : series.length === 0 ? (
          <div className="flex h-64 flex-col items-center justify-center text-center">
            <p className="text-sm font-medium text-neutral-700">Belum ada transaksi terkonfirmasi</p>
            <p className="mt-1 text-xs text-neutral-500">Transaksi 30 hari terakhir akan tampil di grafik ini.</p>
          </div>
        ) : (
          <div className="overflow-x-auto" role="img" aria-label={`Grafik transaksi 30 hari untuk ${series.length} brand`}>
            <svg viewBox={`0 0 ${width} ${height}`} className="min-w-[680px] w-full" aria-hidden="true">
              {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
                const gridY = padding.top + plotHeight * ratio;
                const value = maxValue * (1 - ratio);
                return <g key={ratio}><line x1={padding.left} x2={width - padding.right} y1={gridY} y2={gridY} stroke="#E4E4E7" strokeDasharray="4 5"/><text x={padding.left - 10} y={gridY + 4} textAnchor="end" fontSize="10" fill="#71717A">{formatCompactRupiah(value).replace('Rp', '').trim()}</text></g>;
              })}
              {labelIndexes.map((index) => <text key={index} x={x(index)} y={height - 8} textAnchor={index === 0 ? 'start' : index === 29 ? 'end' : 'middle'} fontSize="10" fill="#71717A">{new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short' }).format(dates[index])}</text>)}
              {series.map((item) => {
                const points = item.values.map((value, index) => `${x(index)},${y(value)}`).join(' ');
                return <g key={item.brandId}><polyline points={points} fill="none" stroke={item.color} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" vectorEffect="non-scaling-stroke"/>{item.values.map((value, index) => value > 0 && <circle key={index} cx={x(index)} cy={y(value)} r="3.5" fill="white" stroke={item.color} strokeWidth="2"><title>{item.name} · {formatDate(toDateKey(dates[index]))} · {formatCompactRupiah(value)}</title></circle>)}</g>;
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
      .slice(0, 4);
    const seats = published.reduce((total, item) => total + (Number(item.seat_sisa) || 0), 0);
    return { published, draft, archived, departures, seats };
  }, [data.schedules]);

  const brandById = useMemo(() => Object.fromEntries(data.brands.map((brand) => [brand.id, brand])), [data.brands]);

  return (
    <main className="space-y-5 pb-8">
      <section className="overflow-hidden rounded-2xl bg-neutral-900 text-white">
        <div className="relative px-5 py-6 md:px-7 md:py-7">
          <div className="pointer-events-none absolute -right-16 -top-24 h-56 w-56 rounded-full bg-primary-500/20" />
          <div className="relative flex flex-col gap-5 md:flex-row md:items-end md:justify-between">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-primary-300">Ringkasan grup</p>
              <h1 className="mt-2 text-2xl font-bold tracking-tight md:text-3xl">Selamat datang, Admin</h1>
              <p className="mt-2 max-w-xl text-sm leading-6 text-neutral-300">Pantau kesiapan seluruh brand dan paket perjalanan dari satu tempat.</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Link to="/brands/new" className="rounded-lg border border-white/25 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-primary-300">Tambah brand</Link>
              <Link to="/schedules/new" className="rounded-lg bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-300">Tambah paket</Link>
            </div>
          </div>
        </div>
      </section>

      {error && <Alert variant="error">{error}</Alert>}

      <section aria-label="Ringkasan data" className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard icon="brand" label="Brand aktif" value={isLoading ? '—' : data.brands.length} detail="Travel dalam grup" tone="primary" />
        <StatCard icon="package" label="Paket terbit" value={isLoading ? '—' : summary.published.length} detail={`${summary.draft} paket masih draft`} tone="success" />
        <StatCard icon="hotel" label="Mitra hotel" value={isLoading ? '—' : data.hotels.length} detail="Mekkah dan Madinah" />
        <StatCard icon="plane" label="Maskapai" value={isLoading ? '—' : data.airlines.length} detail="Tersedia untuk paket" tone="warning" />
      </section>

      <TransactionsChart brands={data.brands} transactions={data.transactions} isLoading={isLoading} />

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1.65fr)_minmax(300px,1fr)]">
        <section className="rounded-xl border border-neutral-200 bg-white" aria-labelledby="departure-title">
          <div className="flex items-center justify-between border-b border-neutral-200 px-5 py-4">
            <div>
              <h2 id="departure-title" className="font-semibold text-neutral-900">Keberangkatan terdekat</h2>
              <p className="mt-1 text-xs text-neutral-500">Paket terbit yang akan segera berjalan</p>
            </div>
            <Link to="/schedules" className="flex items-center gap-1 text-sm font-semibold text-primary-700 hover:text-primary-800">Lihat paket <Icon name="arrow" className="h-4 w-4" /></Link>
          </div>
          <div className="divide-y divide-neutral-100 px-5">
            {isLoading && [1, 2, 3].map((item) => <div key={item} className="my-4 h-14 animate-pulse rounded-lg bg-neutral-100" />)}
            {!isLoading && summary.departures.length === 0 && (
              <div className="py-10 text-center"><p className="text-sm font-medium text-neutral-700">Belum ada keberangkatan mendatang</p><p className="mt-1 text-xs text-neutral-500">Terbitkan paket agar tampil di sini.</p></div>
            )}
            {!isLoading && summary.departures.map((item) => (
              <Link key={item.id} to={`/schedules/${item.id}/edit`} className="group flex items-center gap-3 py-4 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary-400">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-700"><Icon name="calendar" /></span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-semibold text-neutral-900 group-hover:text-primary-700">{item.jadwal_nama}</span>
                  <span className="mt-1 block truncate text-xs text-neutral-500">{brandById[item.brand_id]?.name || 'Brand belum tersedia'} · {formatDate(item.berangkat_tanggal)}</span>
                </span>
                <span className="text-right"><span className="block text-sm font-semibold text-neutral-800">{item.seat_sisa}/{item.seat_total}</span><span className="text-xs text-neutral-500">kursi tersisa</span></span>
              </Link>
            ))}
          </div>
        </section>

        <aside className="space-y-5">
          <section className="rounded-xl border border-neutral-200 bg-white p-5" aria-labelledby="status-title">
            <h2 id="status-title" className="font-semibold text-neutral-900">Status paket</h2>
            <p className="mt-1 text-xs text-neutral-500">Distribusi seluruh paket</p>
            <div className="mt-5 space-y-4">
              {[
                ['Terbit', summary.published.length, 'bg-success-500'],
                ['Draft', summary.draft, 'bg-warning-500'],
                ['Diarsipkan', summary.archived, 'bg-neutral-400'],
              ].map(([label, count, color]) => {
                const percentage = data.schedules.length ? Math.round((count / data.schedules.length) * 100) : 0;
                return <div key={label}><div className="mb-1.5 flex justify-between text-sm"><span className="text-neutral-600">{label}</span><span className="font-semibold text-neutral-900">{count}</span></div><div className="h-1.5 overflow-hidden rounded-full bg-neutral-100"><div className={`h-full rounded-full ${color}`} style={{ width: `${percentage}%` }} /></div></div>;
              })}
            </div>
          </section>

          <section className="rounded-xl border border-primary-200 bg-primary-50 p-5">
            <p className="text-xs font-semibold uppercase tracking-wider text-primary-700">Kapasitas aktif</p>
            <p className="mt-2 text-3xl font-bold tracking-tight text-neutral-900">{isLoading ? '—' : summary.seats}</p>
            <p className="mt-1 text-sm text-neutral-600">kursi tersisa dari paket yang telah terbit.</p>
            <Link to="/schedules" className="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-primary-800 hover:text-primary-900">Kelola kapasitas <Icon name="arrow" className="h-4 w-4" /></Link>
          </section>
        </aside>
      </div>
    </main>
  );
};

export default DashboardHome;

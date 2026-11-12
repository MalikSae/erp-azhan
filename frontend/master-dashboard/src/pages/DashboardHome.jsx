import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Plus, Plane, CalendarDays, ChevronRight, FilePen, CheckCircle2, Archive, Armchair, Hotel,
} from 'lucide-react';
import {
  ResponsiveContainer, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip,
} from 'recharts';
import Alert from 'shared/ui/Alert';
import { listBrandOptions } from '../api/brands';
import { listSchedulesAdmin } from '../api/schedules';
import { listHotels } from '../api/hotels';
import { listAirlines } from '../api/airlines';
import { listPax30Days, listTransactions30Days } from '../api/analytics';

// Tata letak meniru beranda dash.cloudflare.com (DESIGN.md): tiga kolom daftar
// bergaris tanpa kartu, lalu blok "Analitik" berisi kartu grafik garis.

// ── Util ─────────────────────────────────────────────────────────────────────
const toDateKey = (date) => {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
};

const formatShortDate = (value) => new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short' }).format(new Date(`${value}T00:00:00`));
const formatLongDate = (value) => new Intl.DateTimeFormat('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }).format(value);
const formatCompact = (value) => new Intl.NumberFormat('id-ID', { notation: 'compact', maximumFractionDigits: 1 }).format(value || 0);

// Recharts menerima warna lewat prop SVG, bukan kelas Tailwind. Nilai di bawah
// sama dengan token preset: neutral-200 (grid), neutral-500 (sumbu), accent-500.
const CHART_GRID = '#E5E5E5';
const CHART_TICK = '#737373';
const CHART_ACCENT = '#FF6A00';
const BRAND_FALLBACK_COLORS = [CHART_ACCENT, '#3B82F6', '#22C55E', '#8B5CF6', '#F59E0B'];
const TOOLTIP_STYLE = { borderRadius: 8, border: `1px solid ${CHART_GRID}`, fontSize: 12, fontFamily: 'Roboto' };
const TICK_STYLE = { fontSize: 12, fill: CHART_TICK, fontFamily: 'Roboto' };

const last30Days = () => Array.from({ length: 30 }, (_, i) => { const d = new Date(); d.setDate(d.getDate() - (29 - i)); return d; });

// ── Potongan UI ──────────────────────────────────────────────────────────────
// Kolom daftar ala "Domains 7 >": judul abu kecil, jumlah, chevron; baris
// dipisah garis tipis, ikon abu di kiri, chevron di kanan.
const ListBlock = ({ title, count, to, isLoading, emptyText, children }) => {
  const rows = React.Children.toArray(children);
  return (
    <section className="min-w-0">
      <Link to={to} className="group inline-flex items-center gap-2 h-8 text-sm font-body text-neutral-500 hover:text-neutral-900 transition-colors">
        <span>{title}</span>
        {count !== undefined && (
          <span className="inline-flex items-center justify-center min-w-6 h-6 px-1.5 rounded-md bg-neutral-100 text-xs text-neutral-600 tabular-nums">{count}</span>
        )}
        <ChevronRight className="w-4 h-4" />
      </Link>
      <ul className="mt-2">
        {isLoading ? (
          <li className="h-12 flex items-center text-sm font-body text-neutral-500 border-b border-neutral-200">Memuat…</li>
        ) : rows.length === 0 ? (
          <li className="h-12 flex items-center text-sm font-body text-neutral-500 border-b border-neutral-200">{emptyText}</li>
        ) : rows}
      </ul>
    </section>
  );
};

const ListRow = ({ to, icon: Icon, title, meta, metaClassName = 'text-neutral-500' }) => (
  <li className="border-b border-neutral-200">
    <Link to={to} className="flex items-center gap-3 h-12 px-1 -mx-1 rounded-md text-sm font-body text-neutral-900 hover:bg-neutral-50 transition-colors">
      <Icon className="w-4 h-4 shrink-0 text-neutral-500" strokeWidth={1.75} />
      <span className="flex-1 min-w-0 truncate">{title}</span>
      {meta !== undefined && <span className={`shrink-0 tabular-nums ${metaClassName}`}>{meta}</span>}
      <ChevronRight className="w-4 h-4 shrink-0 text-neutral-400" />
    </Link>
  </li>
);

// Kartu analitik ala "Total requests": label, angka besar, catatan kecil, grafik.
const MetricCard = ({ label, value, note, isLoading, children, footer }) => (
  <section className="min-w-0 bg-white rounded-2xl border border-neutral-200 p-5">
    <p className="text-sm font-body text-neutral-600">{label}</p>
    <div className="mt-1 flex items-baseline gap-2">
      <span className="text-2xl md:text-3xl font-heading font-semibold tracking-tight tabular-nums text-neutral-900">{isLoading ? '—' : value}</span>
      {note && !isLoading && <span className="text-sm font-body text-neutral-500">{note}</span>}
    </div>
    <div className="mt-4 h-48">
      {isLoading ? (
        <div className="h-full flex items-center justify-center text-sm font-body text-neutral-500">Memuat grafik…</div>
      ) : children}
    </div>
    {footer}
  </section>
);

const EmptyChart = () => (
  <div className="h-full flex items-center justify-center">
    <span className="px-3 py-1 rounded-md border border-neutral-200 text-sm font-body text-neutral-500">Belum ada data</span>
  </div>
);

// ── Halaman ──────────────────────────────────────────────────────────────────
const DashboardHome = () => {
  const [data, setData] = useState({ brands: [], schedules: [], hotels: [], airlines: [], pax: [], trx: [] });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    // allSettled: peran yang tidak memegang semua izin tetap melihat bagian yang boleh ia lihat.
    // Galat hanya ditampilkan bila tidak ada satu pun bagian yang berhasil dimuat.
    Promise.allSettled([
      listBrandOptions(), listSchedulesAdmin(), listHotels(), listAirlines(),
      listPax30Days(), listTransactions30Days(),
    ])
      .then((hasil) => {
        if (!active) return;
        const [brands, schedules, hotels, airlines, pax, trx] = hasil.map((h) => (h.status === 'fulfilled' ? h.value || [] : []));
        setData({ brands, schedules, hotels, airlines, pax, trx });
        const gagal = hasil.filter((h) => h.status === 'rejected');
        if (gagal.length === hasil.length) {
          setError(gagal[0].reason?.response?.data?.error || 'Ringkasan dashboard gagal dimuat.');
        }
      })
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

  // Pax per hari, satu garis per brand.
  const paxChart = useMemo(() => {
    const brandIds = [...new Set(data.pax.map((p) => Number(p.brand_id)))];
    const byKey = new Map(data.pax.map((p) => [`${p.brand_id}|${p.date}`, Number(p.pax_count) || 0]));
    const rows = last30Days().map((d) => {
      const row = { tanggal: formatShortDate(toDateKey(d)) };
      brandIds.forEach((id) => { row[`b${id}`] = byKey.get(`${id}|${toDateKey(d)}`) || 0; });
      return row;
    });
    const totals = brandIds
      .map((id, i) => ({
        id,
        name: brandName(id),
        color: brandColor(id, i),
        total: data.pax.filter((p) => Number(p.brand_id) === id).reduce((t, p) => t + (Number(p.pax_count) || 0), 0),
      }))
      .sort((a, b) => b.total - a.total);
    return { rows, brandIds, totals };
  }, [data.pax, data.brands]);

  // Nominal transaksi terverifikasi per hari, semua brand dijumlah.
  const trxChart = useMemo(() => {
    const byDate = new Map();
    data.trx.forEach((t) => byDate.set(t.date, (byDate.get(t.date) || 0) + (Number(t.total_amount) || 0)));
    return last30Days().map((d) => ({ tanggal: formatShortDate(toDateKey(d)), nominal: byDate.get(toDateKey(d)) || 0 }));
  }, [data.trx]);

  const chartAxes = (formatter) => (
    <>
      <CartesianGrid vertical={false} stroke={CHART_GRID} />
      <XAxis dataKey="tanggal" tick={TICK_STYLE} tickLine={false} axisLine={false} interval={6} padding={{ left: 16, right: 8 }} />
      <YAxis orientation="right" tick={TICK_STYLE} tickLine={false} axisLine={false} allowDecimals={false} width={44} tickFormatter={formatter} />
    </>
  );

  return (
    <main className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <h1 className="text-lg md:text-xl font-heading font-semibold text-neutral-900 tracking-tight">Dashboard Utama</h1>
        <div className="flex items-center gap-2 flex-wrap">
          <span className="inline-flex items-center gap-1.5 h-9 px-3 rounded-2xl border border-neutral-200 bg-white shadow-btn-secondary text-sm font-body text-neutral-700">
            <CalendarDays className="w-4 h-4 text-neutral-500" />
            {formatLongDate(new Date())}
          </span>
          <Link
            to="/schedules/new"
            className="inline-flex items-center gap-1.5 h-9 px-3 rounded-2xl border border-accent-600 bg-accent-500 bg-btn-primary hover:bg-btn-primary-hover shadow-btn-primary text-sm font-medium font-heading text-white transition-colors"
          >
            <Plus className="w-4 h-4" />
            Buat Paket Baru
          </Link>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-x-8 gap-y-10">
        <ListBlock title="Keberangkatan terdekat" count={isLoading ? undefined : summary.departures.length} to="/schedules" isLoading={isLoading} emptyText="Tidak ada keberangkatan terjadwal.">
          {summary.departures.slice(0, 5).map((s) => {
            const sisa = Number(s.seat_sisa) || 0;
            return (
              <ListRow
                key={s.id}
                to={`/schedules/${s.id}`}
                icon={Plane}
                title={s.jadwal_nama || `Paket #${s.id}`}
                meta={sisa > 0 ? formatShortDate(s.berangkat_tanggal) : `${formatShortDate(s.berangkat_tanggal)} · Penuh`}
                metaClassName={sisa > 0 ? 'text-neutral-500' : 'text-danger-700'}
              />
            );
          })}
        </ListBlock>

        <ListBlock title="Menunggu review" count={isLoading ? undefined : summary.draft.length} to="/schedules" isLoading={isLoading} emptyText="Tidak ada paket draft.">
          {summary.draft.slice(0, 5).map((s) => (
            <ListRow key={s.id} to={`/schedules/${s.id}/edit`} icon={FilePen} title={s.jadwal_nama || `Paket #${s.id}`} meta={brandName(s.brand_id)} />
          ))}
        </ListBlock>

        <ListBlock title="Status paket" to="/schedules" isLoading={isLoading}>
          <ListRow to="/schedules" icon={CheckCircle2} title="Published" meta={summary.published.length} />
          <ListRow to="/schedules" icon={FilePen} title="Draft" meta={summary.draft.length} />
          <ListRow to="/schedules" icon={Archive} title="Archived" meta={summary.archived.length} />
          <ListRow to="/schedules" icon={Armchair} title="Seat tersedia (published)" meta={summary.seats} />
          <ListRow to="/hotels" icon={Hotel} title="Mitra hotel & maskapai" meta={data.hotels.length + data.airlines.length} />
        </ListBlock>
      </div>

      <section className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-base md:text-lg font-heading font-semibold text-neutral-900">Analitik</h2>
          <span className="inline-flex items-center gap-1.5 h-9 px-3 rounded-2xl border border-neutral-200 bg-white shadow-btn-secondary text-sm font-body text-neutral-700">
            <CalendarDays className="w-4 h-4 text-neutral-500" />
            30 hari terakhir
          </span>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <MetricCard
            label="Pax terdaftar"
            value={formatCompact(summary.totalPax30)}
            isLoading={isLoading}
            footer={paxChart.totals.length > 0 && (
              <ul className="mt-4 pt-4 border-t border-neutral-200 flex flex-wrap gap-x-5 gap-y-2">
                {paxChart.totals.map((b) => (
                  <li key={b.id} className="inline-flex items-center gap-2 text-sm font-body text-neutral-600">
                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: b.color }} />
                    {b.name}
                    <span className="tabular-nums text-neutral-900">{b.total}</span>
                  </li>
                ))}
              </ul>
            )}
          >
            {paxChart.brandIds.length === 0 ? <EmptyChart /> : (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={paxChart.rows} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
                  {chartAxes()}
                  <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(value, key) => [value, brandName(String(key).slice(1))]} />
                  {paxChart.brandIds.map((id, i) => (
                    <Line key={id} type="monotone" dataKey={`b${id}`} stroke={brandColor(id, i)} strokeWidth={1.5} dot={false} activeDot={{ r: 3 }} />
                  ))}
                </LineChart>
              </ResponsiveContainer>
            )}
          </MetricCard>

          <MetricCard
            label="Transaksi terverifikasi"
            value={`Rp ${formatCompact(summary.totalTrx30)}`}
            note={`${summary.trxCount30} transaksi`}
            isLoading={isLoading}
          >
            {summary.totalTrx30 === 0 ? <EmptyChart /> : (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={trxChart} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
                  {chartAxes(formatCompact)}
                  <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(value) => [`Rp ${new Intl.NumberFormat('id-ID').format(value)}`, 'Nominal']} />
                  <Line type="monotone" dataKey="nominal" stroke={CHART_ACCENT} strokeWidth={1.5} dot={false} activeDot={{ r: 3 }} />
                </LineChart>
              </ResponsiveContainer>
            )}
          </MetricCard>
        </div>
      </section>
    </main>
  );
};

export default DashboardHome;

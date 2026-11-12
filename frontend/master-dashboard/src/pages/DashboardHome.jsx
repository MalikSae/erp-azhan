import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, ChevronRight, ArrowUpRight, ArrowDownRight } from 'lucide-react';
import {
  ResponsiveContainer, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip,
} from 'recharts';
import Alert from 'shared/ui/Alert';
import Badge from 'shared/ui/Badge';
import EmptyState from 'shared/ui/EmptyState';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { Surface, Section } from 'shared/ui/DetailLayout';
import { listTransactions30Days, getMarketingRingkas } from '../api/analytics';

// Dashboard Utama (holding). Keputusan 12 November 2026: tiga KPI marketing di atas
// (pax per channel, 10 agen terbaik, seat terisi 60 hari), lalu blok "Analitik".
// Angka dihitung di server (GET /api/admin/analytics/marketing-ringkas); pax = pax DP+.

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
const persen = (bagian, total) => (total > 0 ? Math.round((bagian * 100) / total) : 0);

// Recharts menerima warna lewat prop SVG, bukan kelas Tailwind. Nilai di bawah
// sama dengan token preset: neutral-200 (grid), neutral-500 (sumbu), accent-500.
const CHART_GRID = '#E5E5E5';
const CHART_TICK = '#737373';
const CHART_ACCENT = '#FF6A00';
const TOOLTIP_STYLE = { borderRadius: 8, border: `1px solid ${CHART_GRID}`, fontSize: 12, fontFamily: 'Roboto' };
const TICK_STYLE = { fontSize: 12, fill: CHART_TICK, fontFamily: 'Roboto' };

const last30Days = () => Array.from({ length: 30 }, (_, i) => { const d = new Date(); d.setDate(d.getDate() - (29 - i)); return d; });

// Seat terisi: maksimal 4 jadwal di kartu; sisanya di menu Jadwal.
const MAKS_SEAT = 4;
const TANDA_SEAT = {
  merah: { tone: 'danger', label: 'Mendesak' },
  kuning: { tone: 'warning', label: 'Perlu dorongan' },
};

// ── Potongan UI ──────────────────────────────────────────────────────────────
const LihatSemua = ({ to }) => (
  <Link to={to} className="shrink-0 whitespace-nowrap inline-flex items-center gap-1 text-sm font-body text-neutral-500 hover:text-neutral-900 transition-colors">
    Lihat semua
    <ChevronRight className="w-4 h-4" />
  </Link>
);

const KpiCard = ({ title, aside, isLoading, children }) => (
  <Surface className="min-w-0">
    <Section title={title} aside={aside}>
      {isLoading ? <LoadingSpinner /> : children}
    </Section>
  </Surface>
);

// Bar porsi dua segmen: segmen utama gelap, segmen kedua pucat.
const BarPorsi = ({ utama, kedua = 0, total, utamaClass = 'bg-accent-500', keduaClass = 'bg-neutral-300' }) => {
  const a = persen(utama, total);
  const b = Math.min(100 - a, persen(kedua, total));
  return (
    <div className="h-2 w-full rounded-sm bg-neutral-100 overflow-hidden flex">
      <div className={utamaClass} style={{ width: `${a}%` }} />
      <div className={keduaClass} style={{ width: `${b}%` }} />
    </div>
  );
};

const Perubahan = ({ sekarang, sebelumnya }) => {
  const selisih = sekarang - sebelumnya;
  if (selisih === 0) return <span className="text-xs font-body text-neutral-500">sama dengan 30 hari sebelumnya</span>;
  const Ikon = selisih > 0 ? ArrowUpRight : ArrowDownRight;
  return (
    <span className="inline-flex items-center gap-0.5 text-xs font-body text-neutral-500 tabular-nums">
      <Ikon className="w-3.5 h-3.5" />
      {selisih > 0 ? '+' : ''}{selisih} vs 30 hari sebelumnya
    </span>
  );
};

const AngkaChannel = ({ label, value, total, sebelumnya, dotClass }) => (
  <div className="min-w-0">
    <p className="inline-flex items-center gap-2 text-sm font-body text-neutral-600">
      <span className={`w-2 h-2 rounded-full ${dotClass}`} />
      {label}
    </p>
    <p className="mt-0.5 flex items-baseline gap-2">
      <span className="text-2xl md:text-3xl font-heading font-semibold tracking-tight tabular-nums text-neutral-900">{value}</span>
      <span className="text-sm font-body text-neutral-500 tabular-nums">{persen(value, total)}%</span>
    </p>
    <Perubahan sekarang={value} sebelumnya={sebelumnya} />
  </div>
);

const PaxChannelCard = ({ data, isLoading }) => {
  const total = (data?.agen || 0) + (data?.tanpa_agen || 0);
  return (
    <KpiCard
      title="Pax per channel · 30 hari"
      aside={!isLoading && total > 0 && <span className="shrink-0 text-sm font-body text-neutral-500 tabular-nums">{total} pax DP+</span>}
      isLoading={isLoading}
    >
      {total === 0 ? <EmptyState variant="inline" message="Belum ada pax DP atau lunas dalam 30 hari terakhir." /> : (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <AngkaChannel label="Dari agen" value={data.agen} total={total} sebelumnya={data.agen_sebelumnya} dotClass="bg-accent-500" />
            <AngkaChannel label="Tanpa agen" value={data.tanpa_agen} total={total} sebelumnya={data.tanpa_agen_sebelumnya} dotClass="bg-neutral-300" />
          </div>
          <BarPorsi utama={data.agen} kedua={data.tanpa_agen} total={total} />
          {data.per_brand.length > 1 && (
            <ul className="pt-4 border-t border-neutral-200 space-y-3">
              {data.per_brand.map((b) => {
                const t = b.agen + b.tanpa_agen;
                return (
                  <li key={b.brand_id} className="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)_auto] items-center gap-3 text-sm font-body">
                    <span className="truncate text-neutral-900">{b.brand_name}</span>
                    <BarPorsi utama={b.agen} kedua={b.tanpa_agen} total={t} />
                    <span className="tabular-nums text-neutral-500">{persen(b.agen, t)}% agen · {t} pax</span>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      )}
    </KpiCard>
  );
};

const AgenTerbaikCard = ({ items, aktivasi, isLoading }) => (
  <KpiCard title="10 agen terbaik · pax closing" aside={<LihatSemua to="/agen" />} isLoading={isLoading}>
    {aktivasi && (
      <div className="mb-4 rounded-xl bg-neutral-50 border border-neutral-200 px-4 py-3">
        <p className="text-sm font-body text-neutral-600">
          <span className="font-heading font-semibold text-neutral-900 tabular-nums">{aktivasi.agen_closing}</span>
          {' '}dari{' '}
          <span className="tabular-nums">{aktivasi.agen_aktif}</span> agen aktif closing dalam 30 hari
          <span className="ml-2 tabular-nums text-neutral-500">({persen(aktivasi.agen_closing, aktivasi.agen_aktif)}%)</span>
        </p>
      </div>
    )}
    {items.length === 0 ? <EmptyState variant="inline" message="Belum ada closing agen dalam 30 hari terakhir." /> : (
      <ol className="-my-2">
        {items.map((a) => (
          <li key={a.akun_id} className="flex items-center gap-3 py-2 border-b border-neutral-200 last:border-b-0 text-sm font-body">
            <span className="w-5 shrink-0 text-right tabular-nums text-neutral-500">{a.peringkat}</span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 min-w-0">
                <Link to={`/agen/${a.akun_id}`} className="truncate font-medium text-neutral-900 hover:text-primary-600 transition-colors">{a.nama_lengkap}</Link>
                {a.status_agen !== 'aktif' && <Badge variant="neutral">Nonaktif</Badge>}
              </div>
              <p className="text-xs text-neutral-500 truncate">{[a.kode_referral, a.brand_name].filter(Boolean).join(' · ')}</p>
            </div>
            <span className="shrink-0 tabular-nums text-neutral-900">{a.pax_closing} pax</span>
          </li>
        ))}
      </ol>
    )}
  </KpiCard>
);

const SeatTerisiCard = ({ items, isLoading }) => (
  <KpiCard
    title={<>Seat terisi · H-30 s/d H-90{!isLoading && <span className="font-body font-normal text-neutral-500 tabular-nums">({items.length})</span>}</>}
    aside={<LihatSemua to="/schedules" />}
    isLoading={isLoading}
  >
    {items.length === 0 ? <EmptyState variant="inline" message="Tidak ada keberangkatan published yang masih bisa dijual (H-30 sampai H-90)." /> : (
      <>
        <ul className="-my-2">
          {items.slice(0, MAKS_SEAT).map((s) => {
            const tanda = TANDA_SEAT[s.tanda];
            return (
              <li key={s.schedule_id} className="border-b border-neutral-200 last:border-b-0">
                <Link to={`/schedules/${s.schedule_id}`} className="block py-3 px-1 -mx-1 rounded-md hover:bg-neutral-50 transition-colors text-sm font-body">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="truncate font-medium text-neutral-900">{s.nama}</span>
                        {tanda && <Badge variant={tanda.tone}>{tanda.label}</Badge>}
                      </div>
                      <p className="text-xs text-neutral-500 truncate">{s.brand_name} · {formatShortDate(s.berangkat_tanggal)} · H-{s.sisa_hari}</p>
                    </div>
                    <div className="shrink-0 text-right">
                      <p className="tabular-nums font-medium text-neutral-900">
                        {s.kurang_seat > 0 ? `kurang ${s.kurang_seat} seat` : 'Target tercapai'}
                      </p>
                      <p className="text-xs tabular-nums text-neutral-500">{s.terisi}/{s.seat_total} · {s.persen}%{s.hold > 0 ? ` · hold ${s.hold}` : ''}</p>
                    </div>
                  </div>
                  <div className="mt-2">
                    <BarPorsi utama={s.terisi} kedua={s.hold} total={s.seat_total} utamaClass="bg-neutral-900" keduaClass="bg-neutral-300" />
                  </div>
                </Link>
              </li>
            );
          })}
        </ul>
        <p className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs font-body text-neutral-500">
          <span className="inline-flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-neutral-900" />Terisi (DP/lunas, tanpa infant)</span>
          <span className="inline-flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-neutral-300" />Hold seat</span>
          <span>Target isian 70%</span>
        </p>
      </>
    )}
  </KpiCard>
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
const CHART_TANPA_AGEN = '#A3A3A3'; // neutral-400

const DashboardHome = () => {
  const [data, setData] = useState({ trx: [], marketing: null });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    // allSettled: peran yang tidak memegang semua izin tetap melihat bagian yang boleh ia lihat.
    // Galat hanya ditampilkan bila tidak ada satu pun bagian yang berhasil dimuat.
    Promise.allSettled([listTransactions30Days(), getMarketingRingkas()])
      .then((hasil) => {
        if (!active) return;
        const nilai = (h, kosong) => (h.status === 'fulfilled' ? h.value || kosong : kosong);
        setData({ trx: nilai(hasil[0], []), marketing: nilai(hasil[1], null) });
        const gagal = hasil.filter((h) => h.status === 'rejected');
        if (gagal.length === hasil.length) {
          setError(gagal[0].reason?.response?.data?.error || 'Ringkasan dashboard gagal dimuat.');
        }
      })
      .finally(() => { if (active) setIsLoading(false); });
    return () => { active = false; };
  }, []);

  const paxChannel = data.marketing?.pax_channel;

  const summary = useMemo(() => {
    const totalTrx30 = data.trx.reduce((t, p) => t + (Number(p.total_amount) || 0), 0);
    const trxCount30 = data.trx.reduce((t, p) => t + (Number(p.count) || 0), 0);
    return { totalTrx30, trxCount30 };
  }, [data.trx]);

  // Pax DP+ per hari, dua garis: dari agen dan tanpa agen (sama dengan kartu Pax per channel).
  const paxChart = useMemo(() => {
    const byDate = new Map((paxChannel?.harian || []).map((h) => [h.tanggal, h]));
    return last30Days().map((d) => {
      const h = byDate.get(toDateKey(d));
      return { tanggal: formatShortDate(toDateKey(d)), agen: h?.agen || 0, tanpa_agen: h?.tanpa_agen || 0 };
    });
  }, [paxChannel]);

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

  const totalPax = (paxChannel?.agen || 0) + (paxChannel?.tanpa_agen || 0);

  return (
    <main className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <h1 className="text-lg md:text-xl font-heading font-semibold text-neutral-900 tracking-tight">Dashboard Utama</h1>
          <p className="mt-1 text-sm font-body text-neutral-500">{formatLongDate(new Date())}</p>
        </div>
        <Link
          to="/schedules/new"
          className="self-start md:self-auto inline-flex items-center gap-1.5 h-9 px-3 rounded-2xl border border-accent-600 bg-accent-500 bg-btn-primary hover:bg-btn-primary-hover shadow-btn-primary text-sm font-medium font-heading text-white transition-colors"
        >
          <Plus className="w-4 h-4" />
          Buat Paket Baru
        </Link>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {/* Kolom kiri: pax per channel + seat terisi; kolom kanan: 10 agen terbaik. Tinggi kedua kolom setara. */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
        <div className="space-y-6 min-w-0">
          <PaxChannelCard data={paxChannel} isLoading={isLoading} />
          <SeatTerisiCard items={data.marketing?.seat_terisi || []} isLoading={isLoading} />
        </div>
        <AgenTerbaikCard items={data.marketing?.agen_terbaik || []} aktivasi={data.marketing?.aktivasi_agen} isLoading={isLoading} />
      </div>

      <section className="space-y-4">
        <div className="flex items-baseline justify-between gap-3">
          <h2 className="text-base md:text-lg font-heading font-semibold text-neutral-900">Analitik</h2>
          <span className="text-sm font-body text-neutral-500">30 hari terakhir</span>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <MetricCard
            label="Pax DP+ per hari"
            value={formatCompact(totalPax)}
            note="booking DP atau lunas"
            isLoading={isLoading}
            footer={totalPax > 0 && (
              <ul className="mt-4 pt-4 border-t border-neutral-200 flex flex-wrap gap-x-5 gap-y-2">
                <li className="inline-flex items-center gap-2 text-sm font-body text-neutral-600">
                  <span className="w-2 h-2 rounded-full bg-accent-500" />
                  Dari agen
                  <span className="tabular-nums text-neutral-900">{paxChannel.agen}</span>
                </li>
                <li className="inline-flex items-center gap-2 text-sm font-body text-neutral-600">
                  <span className="w-2 h-2 rounded-full bg-neutral-400" />
                  Tanpa agen
                  <span className="tabular-nums text-neutral-900">{paxChannel.tanpa_agen}</span>
                </li>
              </ul>
            )}
          >
            {totalPax === 0 ? <EmptyChart /> : (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={paxChart} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
                  {chartAxes()}
                  <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(value, key) => [value, key === 'agen' ? 'Dari agen' : 'Tanpa agen']} />
                  <Line type="monotone" dataKey="agen" stroke={CHART_ACCENT} strokeWidth={1.5} dot={false} activeDot={{ r: 3 }} />
                  <Line type="monotone" dataKey="tanpa_agen" stroke={CHART_TANPA_AGEN} strokeWidth={1.5} dot={false} activeDot={{ r: 3 }} />
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

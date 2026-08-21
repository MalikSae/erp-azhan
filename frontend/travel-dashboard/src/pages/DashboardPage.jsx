import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowRight, Banknote, BarChart3, CalendarDays, CheckCircle2, Clock3, Package, Plane, TicketCheck, TrendingUp, UserRoundCheck, UsersRound } from "lucide-react";
import { listBookings, listPayments } from "../api/bookings";
import { listJamaah } from "../api/jamaah";
import { listSchedulesAdmin } from "../api/schedules";
import { getStokPerlengkapan } from "../api/perlengkapan";
import { useAuth } from "../context/AuthContext";
import { getStatusBadgeConfig } from "../utils/bookingStatus";
import Alert from "../components/ui/Alert";
import Badge from "../components/ui/Badge";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import PageHeader from "../components/ui/PageHeader";

const formatRupiah = (value) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(Number(value) || 0);
const formatCompactRupiah = (value) => {
  const number = Number(value) || 0;
  if (number >= 1_000_000_000) return `Rp ${(number / 1_000_000_000).toFixed(1).replace(".0", "")} M`;
  if (number >= 1_000_000) return `Rp ${(number / 1_000_000).toFixed(1).replace(".0", "")} jt`;
  return formatRupiah(number);
};
const formatDate = (value) => value ? new Date(value).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" }) : "-";
const daysUntil = (value) => Math.ceil((new Date(value).setHours(0, 0, 0, 0) - new Date().setHours(0, 0, 0, 0)) / 86400000);

function MetricCard({ label, value, note, icon: Icon, tone = "primary", onClick }) {
  const tones = { primary: "bg-primary-50 text-primary-700", success: "bg-success-50 text-success-700", warning: "bg-warning-50 text-warning-800", danger: "bg-danger-50 text-danger-700" };
  return <button type="button" onClick={onClick} className="group rounded-lg border border-neutral-200 bg-white p-4 text-left shadow-sm transition-colors hover:border-primary-300">
    <div className="flex items-start justify-between gap-3"><div className="min-w-0"><p className="text-xs font-medium text-neutral-500">{label}</p><p className="mt-1 text-2xl font-bold tracking-tight text-neutral-900">{value}</p><p className="mt-1 truncate text-xs text-neutral-400">{note}</p></div><span className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${tones[tone]}`}><Icon size={19} /></span></div>
  </button>;
}

function SectionHeader({ title, subtitle, action, onAction }) {
  return <div className="flex items-end justify-between gap-3"><div><h2 className="text-base font-bold text-neutral-900">{title}</h2><p className="mt-0.5 text-xs text-neutral-500">{subtitle}</p></div>{action && <button type="button" onClick={onAction} className="flex shrink-0 items-center gap-1 text-xs font-semibold text-primary-700 hover:text-primary-800">{action}<ArrowRight size={13} /></button>}</div>;
}

function PaymentChart({ payments }) {
  const series = useMemo(() => {
    const now = new Date();
    now.setHours(0, 0, 0, 0);
    const values = Array.from({ length: 30 }, (_, index) => {
      const date = new Date(now);
      date.setDate(now.getDate() - (29 - index));
      const key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
      const amount = payments.filter((payment) => payment.status === "confirmed" && String(payment.tanggal || payment.created_at || "").slice(0, 10) === key).reduce((sum, payment) => sum + Number(payment.jumlah || 0), 0);
      return { date, key, amount };
    });
    const max = Math.max(...values.map((item) => item.amount), 1);
    const total = values.reduce((sum, item) => sum + item.amount, 0);
    return { values, max, total, average: total / 30 };
  }, [payments]);

  const width = 800;
  const height = 170;
  const padding = { top: 14, right: 14, bottom: 28, left: 72 };
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;
  const points = series.values.map((item, index) => ({
    ...item,
    x: padding.left + (index / 29) * plotWidth,
    y: padding.top + plotHeight - (item.amount / series.max) * plotHeight
  }));
  const line = points.map((point) => `${point.x},${point.y}`).join(" ");
  const area = `${padding.left},${padding.top + plotHeight} ${line} ${padding.left + plotWidth},${padding.top + plotHeight}`;
  const labelIndexes = [0, 7, 14, 21, 29];

  return <section className="overflow-hidden rounded-lg border border-neutral-200 bg-white shadow-sm">
    <div className="flex flex-col gap-4 border-b border-neutral-100 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-start gap-3"><span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-50 text-primary-700"><BarChart3 size={19} /></span><div><h2 className="text-base font-bold font-heading text-neutral-900">Arus Pembayaran</h2><p className="mt-0.5 text-xs text-neutral-500">30 hari terakhir • pembayaran terkonfirmasi</p></div></div>
      <div className="flex gap-6"><div><p className="text-xs text-neutral-500">Total</p><p className="font-bold text-neutral-900">{formatCompactRupiah(series.total)}</p></div><div className="border-l border-neutral-200 pl-6"><p className="text-xs text-neutral-500">Rata-rata / hari</p><p className="font-bold text-neutral-900">{formatCompactRupiah(series.average)}</p></div></div>
    </div>
    <div className="px-4 pb-3 pt-2 sm:px-5">
      <div className="relative rounded-md bg-neutral-50 px-2 pt-1">
        <svg viewBox={`0 0 ${width} ${height}`} className="h-48 w-full text-primary-600" role="img" aria-label="Grafik pembayaran terkonfirmasi 30 hari terakhir">
          <defs><linearGradient id="paymentArea" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="currentColor" stopOpacity="0.24" /><stop offset="100%" stopColor="currentColor" stopOpacity="0.01" /></linearGradient></defs>
          {[0, 0.5, 1].map((ratio) => <g key={ratio}><line x1={padding.left} y1={padding.top + plotHeight * ratio} x2={padding.left + plotWidth} y2={padding.top + plotHeight * ratio} stroke="#e4e4e7" strokeWidth="1" strokeDasharray="4 5" /><text x={padding.left - 10} y={padding.top + plotHeight * ratio + 3} textAnchor="end" fontSize="9" fill="#71717a">{formatCompactRupiah(series.max * (1 - ratio))}</text></g>)}
          <polygon points={area} fill="url(#paymentArea)" />
          <polyline points={line} fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
          {points.filter((point) => point.amount > 0).map((point) => <circle key={point.key} cx={point.x} cy={point.y} r="4" fill="white" stroke="currentColor" strokeWidth="3"><title>{point.date.toLocaleDateString("id-ID")}: {formatRupiah(point.amount)}</title></circle>)}
          {labelIndexes.map((index) => <text key={index} x={points[index].x} y={height - 7} textAnchor={index === 0 ? "start" : index === 29 ? "end" : "middle"} fontSize="10" fill="#9ca3af">{points[index].date.toLocaleDateString("id-ID", { day: "numeric", month: "short" })}</text>)}
        </svg>
        {series.total === 0 && <div className="pointer-events-none absolute inset-0 flex items-center justify-center"><div className="rounded-lg border border-neutral-200 bg-white/90 px-4 py-2 text-xs font-medium text-neutral-500">Belum ada pembayaran terkonfirmasi dalam 30 hari terakhir</div></div>}
      </div>
    </div>
  </section>;
}

const DashboardPage = () => {
  const navigate = useNavigate();
  const { brandInfo } = useAuth();
  const [data, setData] = useState({ bookings: [], jamaah: [], schedules: [], stock: [], payments: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const [bookings, jamaah, schedules, stock] = await Promise.all([listBookings(), listJamaah(), listSchedulesAdmin(), getStokPerlengkapan()]);
        const paymentLists = await Promise.all((bookings || []).map(async (booking) => {
          try { return ((await listPayments(booking.id)) || []).map((payment) => ({ ...payment, booking_id: booking.id })); } catch { return []; }
        }));
        if (active) setData({ bookings: bookings || [], jamaah: jamaah || [], schedules: schedules || [], stock: stock || [], payments: paymentLists.flat() });
      } catch (err) {
        if (active) setError(err.response?.data?.error || "Gagal memuat ringkasan dashboard.");
      } finally { if (active) setLoading(false); }
    };
    load();
    return () => { active = false; };
  }, []);

  const stats = useMemo(() => {
    const activeBookings = data.bookings.filter((item) => item.status !== "batal");
    const activeBookingIds = new Set(activeBookings.map((item) => item.id));
    const activePayments = data.payments.filter((item) => activeBookingIds.has(item.booking_id));
    const confirmedPaid = activePayments.filter((item) => item.status === "confirmed").reduce((sum, item) => sum + Number(item.jumlah || 0), 0);
    const totalBills = activeBookings.reduce((sum, item) => sum + Number(item.total_harga || 0), 0);
    const pendingPayments = activePayments.filter((item) => item.status === "pending");
    return { activeBookings, blocked: activeBookings.filter((item) => item.is_seat_blocked).length, newBookings: activeBookings.filter((item) => item.status === "baru").length, confirmedPaid, receivable: Math.max(0, totalBills - confirmedPaid), pendingPayments };
  }, [data]);

  const upcomingSchedules = useMemo(() => data.schedules.filter((item) => item.status === "published" && daysUntil(item.berangkat_tanggal) >= 0).sort((a, b) => new Date(a.berangkat_tanggal) - new Date(b.berangkat_tanggal)).slice(0, 3), [data.schedules]);
  const tasks = useMemo(() => {
    const items = [];
    const unlocked = stats.activeBookings.filter((item) => !item.is_seat_blocked);
    const incomplete = stats.activeBookings.filter((item) => !item.siap_berangkat && item.status !== "baru");
    const unconfirmedTickets = data.schedules.filter((item) => item.status === "published" && !item.is_ticket_confirmed);
    if (stats.pendingPayments.length) items.push({ label: "Pembayaran perlu dikonfirmasi", count: stats.pendingPayments.length, icon: Banknote, tone: "warning", path: "/bookings" });
    if (unlocked.length) items.push({ label: "Booking tanpa block seat", count: unlocked.length, icon: TicketCheck, tone: "danger", path: "/bookings" });
    if (incomplete.length) items.push({ label: "Progress jamaah belum lengkap", count: incomplete.length, icon: UserRoundCheck, tone: "warning", path: "/bookings" });
    if (unconfirmedTickets.length) items.push({ label: "Tiket paket belum dikonfirmasi", count: unconfirmedTickets.length, icon: Plane, tone: "danger", path: "/paket" });
    return items.slice(0, 4);
  }, [data, stats]);
  const recentBookings = useMemo(() => [...data.bookings].sort((a, b) => new Date(b.created_at) - new Date(a.created_at)).slice(0, 5), [data.bookings]);
  const lowStock = useMemo(() => [...data.stock].sort((a, b) => a.stok_tersedia - b.stok_tersedia).slice(0, 4), [data.stock]);

  if (loading) return <div className="flex min-h-[420px] items-center justify-center"><LoadingSpinner /></div>;

  return <div className="space-y-5 pb-8">
    <PageHeader title="Dashboard" subtitle={`Ringkasan operasional ${brandInfo?.nama || "travel"} hari ini.`} secondaryActionLabel="Tambah Jamaah" onSecondaryAction={() => navigate("/jamaah/new")} actionLabel="Booking Baru" onAction={() => navigate("/bookings/new")} />
    {error && <Alert variant="error" message={error} onClose={() => setError("")} />}
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      <MetricCard label="Jamaah Terdaftar" value={data.jamaah.length} note={`${stats.activeBookings.length} booking aktif`} icon={UsersRound} onClick={() => navigate("/jamaah")} />
      <MetricCard label="Booking Baru" value={stats.newBookings} note={`${stats.blocked} kursi sedang diblokir`} icon={CalendarDays} tone="warning" onClick={() => navigate("/bookings")} />
      <MetricCard label="Pembayaran Diterima" value={formatCompactRupiah(stats.confirmedPaid)} note={`${stats.pendingPayments.length} pembayaran menunggu`} icon={TrendingUp} tone="success" onClick={() => navigate("/bookings")} />
      <MetricCard label="Sisa Tagihan" value={formatCompactRupiah(stats.receivable)} note="Dari seluruh booking aktif" icon={Banknote} tone="danger" onClick={() => navigate("/bookings")} />
    </div>
    <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
      <div className="md:col-span-2"><PaymentChart payments={data.payments.filter((payment) => stats.activeBookings.some((booking) => booking.id === payment.booking_id))} /></div>
      <section className="rounded-lg border border-neutral-200 bg-white p-4 shadow-sm"><SectionHeader title="Perlu Ditindaklanjuti" subtitle="Prioritas operasional saat ini" /><div className="mt-3 space-y-2">{tasks.length === 0 ? <div className="flex flex-col items-center py-8 text-center"><CheckCircle2 className="text-success-500" size={28} /><p className="mt-2 text-sm font-semibold text-neutral-700">Semua terkendali</p></div> : tasks.map((task) => <button key={task.label} onClick={() => navigate(task.path)} className="flex w-full items-center gap-3 rounded-md border border-neutral-100 px-3 py-2.5 text-left hover:bg-neutral-50"><span className={`flex h-8 w-8 items-center justify-center rounded-md ${task.tone === "danger" ? "bg-danger-50 text-danger-600" : "bg-warning-50 text-warning-700"}`}><task.icon size={15} /></span><span className="min-w-0 flex-1 text-xs font-medium text-neutral-700">{task.label}</span><span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-neutral-900 px-1.5 text-xs font-bold text-white">{task.count}</span></button>)}</div></section>
    </div>
    <div className="grid grid-cols-1 gap-4">
      <section className="rounded-lg border border-neutral-200 bg-white p-4 shadow-sm"><SectionHeader title="Keberangkatan Terdekat" subtitle="Paket aktif berdasarkan tanggal keberangkatan" action="Lihat paket" onAction={() => navigate("/paket")} /><div className="mt-3 space-y-2">{upcomingSchedules.length === 0 ? <p className="py-8 text-center text-sm text-neutral-400">Belum ada jadwal terdekat.</p> : upcomingSchedules.map((schedule) => {
        const filled = Math.max(0, schedule.seat_total - schedule.seat_sisa); const percent = schedule.seat_total ? Math.round((filled / schedule.seat_total) * 100) : 0; const days = daysUntil(schedule.berangkat_tanggal);
        return <button key={schedule.id} onClick={() => navigate("/paket")} className="block w-full rounded-lg border border-neutral-100 bg-neutral-50/60 px-3 py-2.5 text-left hover:border-primary-200"><div className="flex items-start justify-between gap-3"><div className="min-w-0"><p className="truncate text-xs font-semibold text-neutral-900">{schedule.jadwal_nama}</p><p className="mt-0.5 flex items-center gap-1.5 text-[11px] text-neutral-500"><CalendarDays size={12} />{formatDate(schedule.berangkat_tanggal)} • {days === 0 ? "Hari ini" : `${days} hari lagi`}</p></div><Badge variant={schedule.is_ticket_confirmed ? "success" : "warning"}>{schedule.is_ticket_confirmed ? "Tiket Confirm" : "Tiket Pending"}</Badge></div><div className="mt-2"><div className="mb-1 flex justify-between text-[10px]"><span className="text-neutral-500">{filled} dari {schedule.seat_total} kursi terisi</span><span className="font-semibold text-neutral-700">{percent}%</span></div><div className="h-1 overflow-hidden rounded-full bg-neutral-200"><div className="h-full rounded-full bg-primary-600" style={{ width: `${Math.min(100, percent)}%` }} /></div></div></button>;
      })}</div></section>
    </div>
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <section className="overflow-hidden rounded-lg border border-neutral-200 bg-white shadow-sm lg:col-span-2"><div className="p-4 pb-3"><SectionHeader title="Booking Terbaru" subtitle="Aktivitas reservasi jamaah terkini" action="Lihat semua" onAction={() => navigate("/bookings")} /></div><div className="overflow-x-auto"><table className="min-w-full text-left"><thead><tr className="border-y border-neutral-100 bg-neutral-50 text-xs uppercase tracking-wider text-neutral-500"><th className="px-4 py-2 font-semibold">Jamaah</th><th className="px-3 py-2 font-semibold">Paket</th><th className="px-3 py-2 font-semibold">Tagihan</th><th className="px-4 py-2 font-semibold">Status</th></tr></thead><tbody className="divide-y divide-neutral-100">{recentBookings.map((booking) => <tr key={booking.id} onClick={() => navigate(`/bookings/${booking.id}`)} className="cursor-pointer hover:bg-neutral-50"><td className="px-4 py-2.5"><p className="text-xs font-semibold text-neutral-900">{booking.nama_jamaah}</p><p className="mt-0.5 text-xs font-mono text-neutral-400">{booking.id_booking || `ID: ${booking.id}`}</p></td><td className="max-w-48 truncate px-3 py-2.5 text-xs text-neutral-600">{booking.jadwal_nama}</td><td className="px-3 py-2.5 text-xs font-semibold text-neutral-700">{formatRupiah(booking.total_harga)}</td><td className="px-4 py-2.5">{(() => { const [v, l] = getStatusBadgeConfig(booking.status); return <Badge variant={v} hideIcon={true}>{l}</Badge>; })()}</td></tr>)}</tbody></table></div>{recentBookings.length === 0 && <p className="py-8 text-center text-sm text-neutral-400">Belum ada booking.</p>}</section>
      <section className="rounded-lg border border-neutral-200 bg-white p-4 shadow-sm"><SectionHeader title="Kesiapan Perlengkapan" subtitle="Stok terendah yang perlu dipantau" action="Kelola stok" onAction={() => navigate("/stok-perlengkapan")} /><div className="mt-3 space-y-2.5">{lowStock.map((item) => { const isLow = item.stok_tersedia <= 40; return <div key={item.perlengkapan_item_id} className="flex items-center gap-3"><span className={`flex h-8 w-8 items-center justify-center rounded-md ${isLow ? "bg-warning-50 text-warning-700" : "bg-neutral-100 text-neutral-600"}`}><Package size={15} /></span><div className="min-w-0 flex-1"><p className="truncate text-xs font-medium text-neutral-700">{item.nama_item}</p><p className="mt-0.5 text-xs text-neutral-400">{item.qty_per_set} item per set</p></div><span className={`text-sm font-bold ${isLow ? "text-warning-700" : "text-neutral-800"}`}>{item.stok_tersedia}</span></div>; })}</div><div className="mt-4 rounded-md bg-neutral-50 px-3 py-2.5"><div className="flex items-center gap-2 text-xs text-neutral-600"><Clock3 size={13} className="text-primary-600" /><span>Estimasi set lengkap tersedia</span></div><p className="mt-0.5 text-lg font-bold text-neutral-900">{data.stock.length ? Math.min(...data.stock.map((item) => Math.floor(item.stok_tersedia / Math.max(1, item.qty_per_set)))) : 0} set</p></div></section>
    </div>
  </div>;
};

export default DashboardPage;

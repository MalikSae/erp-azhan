import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { 
  getBooking, 
  updateBookingStatus, 
  cancelBookingSeatBlock,
  listPayments, 
  createPayment, 
  updatePaymentStatus, 
  addBookingAddon, 
  deleteBookingAddon, 
  updateBookingDiskon, 
  updateBookingProgress 
} from "../api/bookings";
import { markPerlengkapanDiberikan, batalkanPerlengkapan } from "../api/perlengkapan";
import { getJamaah } from "../api/jamaah";
import { uploadMedia } from "../api/media";
import { useAuth } from "../context/AuthContext";
import PageHeader from "../components/ui/PageHeader";
import Card from "../components/ui/Card";
import Badge from "../components/ui/Badge";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import Table from "../components/ui/Table";
import Modal from "../components/ui/Modal";
import Input from "../components/ui/Input";
import FormField from "../components/ui/FormField";
import CustomDropdown from "../components/ui/CustomDropdown";
import CurrencyInput from "../components/ui/CurrencyInput";
import Toggle from "../components/ui/Toggle";
import { CheckCircle, ExternalLink, FileText, Upload, X, Shield, Calendar, User, Plane, Check, Plus, Trash2, Tag, Percent, Package } from "lucide-react";

const formatRupiah = (angka) => {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(angka || 0);
};

const formatTanggal = (dateStr) => {
  if (!dateStr) return "-";
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("id-ID", {
      day: "numeric",
      month: "long",
      year: "numeric"
    });
  } catch (e) {
    return dateStr;
  }
};

const PROGRESS_MANUAL_ITEMS = [
  { key: 'visa', label: 'Visa' },
  { key: 'siskopatuh', label: 'Siskopatuh' },
  { key: 'tiket', label: 'Tiket Maskapai' },
  { key: 'hotel', label: 'Hotel' },
  { key: 'land_arrangement', label: 'Land Arrangement' },
  { key: 'manasik', label: 'Manasik' },
  { key: 'vaksin_meningitis', label: 'Vaksin Meningitis' },
];

const BookingDetailPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { brandInfo } = useAuth();

  const [booking, setBooking] = useState(null);
  const [jamaah, setJamaah] = useState(null);
  const [payments, setPayments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Payment Modal State
  const [isPaymentModalOpen, setIsPaymentModalOpen] = useState(false);
  const [paymentForm, setPaymentForm] = useState({
    jumlah: "",
    metode: "transfer",
    tanggal: new Date().toISOString().split('T')[0],
    bukti_file: null,
  });
  const [paymentFormError, setPaymentFormError] = useState(null);
  const [paymentSubmitting, setPaymentSubmitting] = useState(false);

  // Add-on Modal State
  const [isAddonModalOpen, setIsAddonModalOpen] = useState(false);
  const [addonForm, setAddonForm] = useState({ nama: "", nominal: "" });
  const [addonError, setAddonError] = useState(null);
  const [addonSubmitting, setAddonSubmitting] = useState(false);

  // Diskon Modal State
  const [isDiskonModalOpen, setIsDiskonModalOpen] = useState(false);
  const [diskonForm, setDiskonForm] = useState({ diskon: "", diskon_keterangan: "" });
  const [diskonError, setDiskonError] = useState(null);
  const [diskonSubmitting, setDiskonSubmitting] = useState(false);

  // Perlengkapan State
  const [perlengkapanLoading, setPerlengkapanLoading] = useState(false);
  const [perlengkapanError, setPerlengkapanError] = useState(null);
  const [isBatalkanModalOpen, setIsBatalkanModalOpen] = useState(false);

  const fetchAll = async () => {
    try {
      setLoading(true);
      const [bookRes, payRes] = await Promise.all([
        getBooking(id),
        listPayments(id)
      ]);
      setBooking(bookRes);
      setPayments(payRes || []);

      if (bookRes?.jamaah_id) {
        try {
          const jRes = await getJamaah(bookRes.jamaah_id);
          setJamaah(jRes);
        } catch (e) {
          // Non-blocking if jamaah detail fails
        }
      }
    } catch (err) {
      setError("Gagal memuat detail booking.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAll();
  }, [id]);

  const handleDistribusiPerlengkapan = async () => {
    setPerlengkapanLoading(true);
    setPerlengkapanError(null);
    try {
      await markPerlengkapanDiberikan(id);
      await fetchAll();
    } catch (err) {
      const msg = err.response?.data?.error || "Gagal mendistribusikan perlengkapan.";
      setPerlengkapanError(msg);
    } finally {
      setPerlengkapanLoading(false);
    }
  };

  const handleBatalkanPerlengkapan = async () => {
    setPerlengkapanLoading(true);
    setPerlengkapanError(null);
    try {
      await batalkanPerlengkapan(id);
      setIsBatalkanModalOpen(false);
      await fetchAll();
    } catch (err) {
      const msg = err.response?.data?.error || "Gagal membatalkan perlengkapan.";
      setPerlengkapanError(msg);
    } finally {
      setPerlengkapanLoading(false);
    }
  };

  const [cancelConfirmModal, setCancelConfirmModal] = useState(false);
  const [cancelSeatBlockModal, setCancelSeatBlockModal] = useState(false);
  const [seatBlockLoading, setSeatBlockLoading] = useState(false);

  const handleBookingStatus = (newStatus) => {
    if (newStatus === "batal") {
      setCancelConfirmModal(true);
      return;
    }
    executeStatusChange(newStatus);
  };

  const executeStatusChange = async (newStatus) => {
    try {
      await updateBookingStatus(id, newStatus);
      setCancelConfirmModal(false);
      fetchAll();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal mengubah status booking");
      setCancelConfirmModal(false);
    }
  };

  const handleCancelSeatBlock = async () => {
    setSeatBlockLoading(true);
    try {
      await cancelBookingSeatBlock(id);
      setCancelSeatBlockModal(false);
      await fetchAll();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal melepas blok kursi");
      setCancelSeatBlockModal(false);
    } finally {
      setSeatBlockLoading(false);
    }
  };

  // Perhitungan Keuangan
  const hargaDasar = booking?.harga_dasar ? parseFloat(booking.harga_dasar) : (parseFloat(booking?.total_harga) || 0);
  const addons = booking?.addons || [];
  const totalAddons = addons.reduce((sum, a) => sum + (parseFloat(a.nominal) || 0), 0);
  const diskon = parseFloat(booking?.diskon) || 0;
  const totalHarga = booking?.total_harga ? parseFloat(booking.total_harga) : Math.max(0, hargaDasar + totalAddons - diskon);

  const totalPaid = payments
    .filter((p) => p.status === "confirmed")
    .reduce((sum, p) => sum + (parseFloat(p.jumlah) || 0), 0);
  const sisaTagihan = Math.max(0, totalHarga - totalPaid);

  // ─── Add-on Handlers ──────────────────────────────────────────────────────────
  const handleAddAddonSubmit = async (e) => {
    e.preventDefault();
    const nominalNum = parseFloat(addonForm.nominal) || 0;
    if (!addonForm.nama.trim()) {
      setAddonError("Nama add-on wajib diisi");
      return;
    }
    if (nominalNum <= 0) {
      setAddonError("Nominal add-on harus lebih dari 0");
      return;
    }

    setAddonSubmitting(true);
    setAddonError(null);
    try {
      await addBookingAddon(id, {
        nama: addonForm.nama.trim(),
        nominal: nominalNum
      });
      setIsAddonModalOpen(false);
      setAddonForm({ nama: "", nominal: "" });
      fetchAll();
    } catch (err) {
      setAddonError(err.response?.data?.error || "Gagal menambahkan add-on");
    } finally {
      setAddonSubmitting(false);
    }
  };

  const handleDeleteAddon = async (addonId) => {
    try {
      await deleteBookingAddon(id, addonId);
      fetchAll();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menghapus add-on");
    }
  };

  // ─── Diskon Handlers ─────────────────────────────────────────────────────────
  const handleOpenDiskonModal = () => {
    setDiskonForm({
      diskon: diskon > 0 ? diskon : "",
      diskon_keterangan: booking?.diskon_keterangan || ""
    });
    setDiskonError(null);
    setIsDiskonModalOpen(true);
  };

  const handleUpdateDiskonSubmit = async (e) => {
    e.preventDefault();
    const diskonNum = parseFloat(diskonForm.diskon) || 0;
    if (diskonNum < 0) {
      setDiskonError("Nominal diskon tidak boleh negatif");
      return;
    }

    setDiskonSubmitting(true);
    setDiskonError(null);
    try {
      await updateBookingDiskon(id, {
        diskon: diskonNum,
        diskon_keterangan: diskonForm.diskon_keterangan ? diskonForm.diskon_keterangan.trim() : null
      });
      setIsDiskonModalOpen(false);
      fetchAll();
    } catch (err) {
      setDiskonError(err.response?.data?.error || "Gagal mengatur diskon");
    } finally {
      setDiskonSubmitting(false);
    }
  };

  const handleRemoveDiskon = async () => {
    setDiskonSubmitting(true);
    try {
      await updateBookingDiskon(id, {
        diskon: 0,
        diskon_keterangan: null
      });
      setIsDiskonModalOpen(false);
      fetchAll();
    } catch (err) {
      setDiskonError(err.response?.data?.error || "Gagal menghapus diskon");
    } finally {
      setDiskonSubmitting(false);
    }
  };

  // ─── Progress Keberangkatan Handler ──────────────────────────────────────────
  const handleToggleProgress = async (key, currentValue) => {
    const newValue = !currentValue;

    // Optimistic Update
    setBooking((prev) => {
      if (!prev) return prev;
      const progressField = `progress_${key}`;
      const updated = { ...prev, [progressField]: newValue };

      // Recalculate computed siap_berangkat (paspor dinamis + 7 toggle manual)
      const allDone = Boolean(updated.progress_paspor) && PROGRESS_MANUAL_ITEMS.every((item) => {
        if (item.key === key) return newValue;
        return Boolean(updated[`progress_${item.key}`]);
      });
      updated.siap_berangkat = allDone;
      return updated;
    });

    try {
      const res = await updateBookingProgress(id, { [key]: newValue });
      setBooking(res);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memperbarui progress keberangkatan");
      fetchAll();
    }
  };

  // ─── Payment Handlers ─────────────────────────────────────────────────────────
  const handlePaymentSubmit = async (e) => {
    e.preventDefault();
    const jumlahNum = parseFloat(paymentForm.jumlah) || 0;

    if (jumlahNum <= 0) {
      setPaymentFormError("Jumlah pembayaran harus lebih dari 0");
      return;
    }
    if (jumlahNum > sisaTagihan) {
      setPaymentFormError(`Jumlah pembayaran tidak boleh lebih dari sisa tagihan (${formatRupiah(sisaTagihan)})`);
      return;
    }

    setPaymentSubmitting(true);
    setPaymentFormError(null);

    try {
      let buktiUrl = null;
      if (paymentForm.bukti_file) {
        buktiUrl = await uploadMedia(paymentForm.bukti_file, "bukti-pembayaran");
      }

      await createPayment(id, {
        jumlah: jumlahNum,
        metode: paymentForm.metode,
        tanggal: paymentForm.tanggal,
        bukti_url: buktiUrl
      });

      setIsPaymentModalOpen(false);
      setPaymentForm({
        jumlah: "",
        metode: "transfer",
        tanggal: new Date().toISOString().split('T')[0],
        bukti_file: null
      });
      fetchAll();
    } catch (err) {
      setPaymentFormError(err.response?.data?.error || "Gagal mencatat pembayaran");
    } finally {
      setPaymentSubmitting(false);
    }
  };

  const handleConfirmPayment = async (paymentId) => {
    try {
      await updatePaymentStatus(paymentId, "confirmed");
      fetchAll();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal konfirmasi pembayaran");
    }
  };

  if (loading) return <div className="p-8 flex justify-center"><LoadingSpinner /></div>;
  if (!booking) return <Alert variant="error" message="Booking tidak ditemukan" />;

  // Status transition logic (Opsi A: Status Lunas otomatis diatur oleh sistem pembayaran)
  const statusTransitions = {
    baru: [
      { label: "Block Seat", status: "dp", variant: "primary" },
      { label: "Batalkan", status: "batal", variant: "danger" }
    ],
    dp: [
      { label: "Batalkan", status: "batal", variant: "danger" }
    ],
    lunas: [
      { label: "Batalkan", status: "batal", variant: "danger" }
    ],
    batal: []  // Terminal
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'baru': return <Badge variant="warning">BARU</Badge>;
      case 'dp': return <Badge variant="primary">SEAT BLOCKED</Badge>;
      case 'lunas': return <Badge variant="success">LUNAS</Badge>;
      case 'dokumen_lengkap': return <Badge variant="success">DOKUMEN LENGKAP</Badge>;
      case 'siap_berangkat': return <Badge variant="success">SIAP BERANGKAT</Badge>;
      case 'batal': return <Badge variant="archived">BATAL</Badge>;
      default: return <Badge variant="neutral">{status.toUpperCase()}</Badge>;
    }
  };

  const nextActions = statusTransitions[booking.status] || [];

  const paymentColumns = [
    { header: "Tanggal", accessor: "tanggal" },
    { header: "Metode", accessor: (row) => <span className="uppercase">{row.metode}</span> },
    { header: "Jumlah", accessor: (row) => formatRupiah(row.jumlah) },
    {
      header: "Bukti",
      accessor: (row) => row.bukti_url ? (
        <a
          href={row.bukti_url.startsWith("http") ? row.bukti_url : `${import.meta.env.VITE_API_BASE_URL || "http://localhost:9090"}${row.bukti_url}`}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1 px-2.5 py-1 rounded text-xs font-medium font-body text-primary-700 bg-primary-50 hover:bg-primary-100 transition-colors"
        >
          <ExternalLink size={12} />
          <span>Lihat Bukti</span>
        </a>
      ) : (
        <span className="text-xs text-neutral-400 font-body">-</span>
      )
    },
    { 
      header: "Status", 
      accessor: (row) => row.status === "confirmed" 
        ? <Badge variant="success">Confirmed</Badge> 
        : <Badge variant="warning">Pending</Badge> 
    },
    {
      header: "Aksi",
      accessor: (row) => row.status === "pending" && (
        <Button 
          variant="ghost" 
          size="sm" 
          onClick={() => handleConfirmPayment(row.id)}
          title="Konfirmasi Pembayaran"
        >
          <CheckCircle size={16} className="text-success-600" />
        </Button>
      )
    }
  ];

  return (
    <div className="space-y-6 max-w-5xl mx-auto pb-12">
      {/* Header & Quick Action Buttons */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <PageHeader 
          title={`Booking #INV-${String(booking.id).padStart(5, '0')}`} 
          onBack={() => navigate(-1)}
        />

        {/* Action Buttons Top Bar */}
        {(nextActions.length > 0 || booking.is_seat_blocked) && (
          <div className="flex items-center gap-2.5 shrink-0 self-start sm:self-auto">
            {booking.is_seat_blocked && booking.status !== 'batal' && (
              <Button
                variant="secondary"
                onClick={() => setCancelSeatBlockModal(true)}
                className="text-xs sm:text-sm font-medium"
              >
                Cancel Block Seat
              </Button>
            )}
            {nextActions.map(action => (
              <Button 
                key={action.status} 
                variant={action.variant}
                onClick={() => handleBookingStatus(action.status)}
                className="text-xs sm:text-sm font-medium"
              >
                {action.label}
              </Button>
            ))}
          </div>
        )}
      </div>

      {error && <Alert variant="error" message={error} onClose={() => setError(null)} />}

      {/* ─── INVOICE / BILLING DOCUMENT CARD ─── */}
      <div className="bg-white rounded-xl border border-neutral-200 shadow-sm overflow-hidden divide-y divide-neutral-200">
        
        {/* 1. Header Invoice */}
        <div className="p-6 sm:p-8 bg-neutral-50/60 flex flex-col sm:flex-row sm:items-start justify-between gap-6">
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <span className="text-xl sm:text-2xl font-bold font-heading text-neutral-900 tracking-tight">
                {brandInfo?.nama || "Hana Tours Travel"}
              </span>
            </div>
            <p className="text-xs font-body text-neutral-500">
              Tagihan Resmi & Konfirmasi Reservasi Umroh
            </p>
            {brandInfo?.legalitas && (
              <p className="text-xs font-body text-neutral-400">
                Izin Kemenag: {brandInfo.legalitas}
              </p>
            )}
          </div>

          <div className="sm:text-right space-y-1.5 border-t sm:border-t-0 pt-4 sm:pt-0 border-neutral-200">
            <div className="flex sm:justify-end items-center gap-2">
              <span className="text-xs font-semibold font-heading uppercase tracking-wider text-neutral-400">INVOICE</span>
              <span className="text-sm font-bold font-heading text-neutral-900">
                #INV-{String(booking.id).padStart(5, '0')}
              </span>
            </div>
            <p className="text-xs font-body text-neutral-500">
              Tgl Booking: <span className="text-neutral-700 font-medium">{formatTanggal(booking.created_at)}</span>
            </p>
            <div className="pt-1 flex sm:justify-end">
              {getStatusBadge(booking.status)}
            </div>
          </div>
        </div>

        {/* 2. Customer (Ditagihkan Kepada) & Travel Detail Grid */}
        <div className="p-6 sm:p-8 grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Kolom Kiri: Data Jamaah */}
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-1 border-b border-neutral-100">
              <div className="flex items-center gap-2">
                <User size={15} className="text-primary-600" />
                <span className="text-xs font-bold font-heading uppercase tracking-wider text-neutral-500">
                  Ditagihkan Kepada
                </span>
              </div>
              {booking.jamaah_id && (
                <button
                  type="button"
                  onClick={() => navigate(`/jamaah/${booking.jamaah_id}`)}
                  className="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 hover:underline transition-colors"
                  title="Buka Halaman Detail Jamaah"
                >
                  <span>Lihat Detail Jamaah</span>
                  <ExternalLink size={12} />
                </button>
              )}
            </div>
            <div className="space-y-1">
              <h3 
                onClick={() => booking.jamaah_id && navigate(`/jamaah/${booking.jamaah_id}`)}
                className={`text-base font-bold font-heading text-neutral-900 ${booking.jamaah_id ? 'cursor-pointer hover:text-primary-600 transition-colors inline-block' : ''}`}
                title={booking.jamaah_id ? "Klik untuk melihat detail data jamaah" : ""}
              >
                {jamaah?.nama_lengkap || booking.nama_jamaah}
              </h3>
              <p className="text-xs font-body text-neutral-600">
                NIK: <span className="font-medium text-neutral-800">{jamaah?.nik || "-"}</span>
              </p>
              <p className="text-xs font-body text-neutral-600">
                No. HP: <span className="font-medium text-neutral-800">{jamaah?.no_hp || "-"}</span>
              </p>
              <p className="text-xs font-body text-neutral-600">
                No. Paspor: <span className="font-medium text-neutral-800">{jamaah?.no_paspor || "-"}</span>
              </p>
              {jamaah?.alamat && (
                <p className="text-xs font-body text-neutral-500 pt-0.5 leading-relaxed">
                  {jamaah.alamat}
                </p>
              )}
            </div>
          </div>

          {/* Kolom Kanan: Detail Keberangkatan */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 pb-1 border-b border-neutral-100">
              <Plane size={15} className="text-primary-600" />
              <span className="text-xs font-bold font-heading uppercase tracking-wider text-neutral-500">
                Detail Keberangkatan
              </span>
            </div>
            <div className="space-y-1">
              <h3 className="text-base font-bold font-heading text-neutral-900">
                {booking.jadwal_nama || booking.schedule?.jadwal_nama}
              </h3>
              <p className="text-xs font-body text-neutral-600">
                Tgl Berangkat: <span className="font-semibold text-neutral-900">{formatTanggal(booking.berangkat_tanggal)}</span>
              </p>
              <p className="text-xs font-body text-neutral-600">
                Tipe Kamar: <span className="font-semibold uppercase text-primary-700">{booking.room_type || booking.tipe_kamar}</span>
              </p>
              <p className="text-xs font-body text-neutral-600">
                Status Kursi:{" "}
                <span className={`font-semibold ${booking.is_seat_blocked ? 'text-success-600' : booking.status === 'batal' ? 'text-neutral-400' : 'text-warning-600'}`}>
                  {booking.is_seat_blocked ? 'Terkunci' : booking.status === 'batal' ? 'Dibatalkan' : 'Tidak Diblokir'}
                </span>
              </p>
            </div>
          </div>
        </div>

        {/* 3. Section: Progress Keberangkatan */}
        <div className="p-6 sm:p-8 border-t border-neutral-200 bg-neutral-50/40">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-5 pb-3 border-b border-neutral-200">
            <div className="flex items-center gap-2">
              <CheckCircle size={16} className="text-primary-600" />
              <h4 className="text-xs font-bold font-heading uppercase tracking-wider text-neutral-600">
                Progress Keberangkatan
              </h4>
            </div>
            <div>
              {booking.siap_berangkat ? (
                <Badge variant="success" className="px-3 py-1 text-xs font-semibold">
                  Siap Berangkat
                </Badge>
              ) : (
                <Badge variant="warning" className="px-3 py-1 text-xs font-semibold">
                  Persiapan Berjalan
                </Badge>
              )}
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* 1. Card Paspor (READ-ONLY - otomatis dari Dokumen Jamaah) */}
            <div className="flex items-center justify-between p-3.5 bg-white rounded-lg border border-neutral-200 shadow-sm">
              <span className="text-sm font-medium font-heading text-neutral-900">
                Paspor
              </span>
              <Toggle
                id="toggle-paspor"
                checked={Boolean(booking.progress_paspor)}
                disabled={true}
                onChange={() => {}}
              />
            </div>

            {/* 2. Card Toggle Manual (7 Item) */}
            {PROGRESS_MANUAL_ITEMS.map((item) => {
              const isChecked = Boolean(booking[`progress_${item.key}`]);
              return (
                <div
                  key={item.key}
                  className="flex items-center justify-between p-3.5 bg-white rounded-lg border border-neutral-200 shadow-sm"
                >
                  <span className="text-sm font-medium font-heading text-neutral-900">
                    {item.label}
                  </span>
                  <Toggle
                    id={`toggle-${item.key}`}
                    checked={isChecked}
                    onChange={() => handleToggleProgress(item.key, isChecked)}
                  />
                </div>
              );
            })}
          </div>
        </div>

        {/* Section: Perlengkapan */}
        <div className="p-6 sm:p-8 border-t border-neutral-200 bg-white">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <Package size={16} className="text-primary-600" />
                <h4 className="text-xs font-bold font-heading uppercase tracking-wider text-neutral-600">
                  Perlengkapan Jamaah
                </h4>
              </div>
              <div className="flex items-center gap-2 pt-1">
                {booking.perlengkapan_status === 'sudah_diberikan' ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="success" className="px-3 py-1 text-xs font-semibold">
                      Sudah Diberikan
                    </Badge>
                    {booking.perlengkapan_tanggal && (
                      <span className="text-xs text-neutral-500 font-body">
                        pada {formatTanggal(booking.perlengkapan_tanggal)}
                      </span>
                    )}
                  </div>
                ) : (
                  <Badge variant="neutral" className="px-3 py-1 text-xs font-semibold">
                    Belum Diberikan
                  </Badge>
                )}
              </div>
            </div>

            <div>
              {booking.perlengkapan_status === 'sudah_diberikan' ? (
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => {
                    setPerlengkapanError(null);
                    setIsBatalkanModalOpen(true);
                  }}
                  disabled={perlengkapanLoading}
                  className="text-xs"
                >
                  Batalkan
                </Button>
              ) : (
                <Button
                  size="sm"
                  variant="primary"
                  onClick={handleDistribusiPerlengkapan}
                  disabled={perlengkapanLoading || booking.status === 'batal'}
                  className="text-xs"
                >
                  {perlengkapanLoading ? "Memproses..." : "Tandai Sudah Diberikan"}
                </Button>
              )}
            </div>
          </div>

          {perlengkapanError && (
            <div className="mt-4">
              <Alert variant="error" onClose={() => setPerlengkapanError(null)}>
                {perlengkapanError}
              </Alert>
            </div>
          )}
        </div>

        {/* 4. Itemized Table (Rincian Biaya Paket, Add-on & Diskon) */}
        <div className="p-6 sm:p-8">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-neutral-200 text-[11px] font-bold font-heading uppercase tracking-wider text-neutral-500">
                  <th className="py-2.5 px-3">Deskripsi Layanan / Item</th>
                  <th className="py-2.5 px-3">Tipe</th>
                  <th className="py-2.5 px-3 text-center">Qty</th>
                  <th className="py-2.5 px-3 text-right">Harga Satuan</th>
                  <th className="py-2.5 px-3 text-right">Jumlah</th>
                  <th className="py-2.5 px-3 text-center w-16">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 text-sm font-body">
                {/* 1. Baris Paket Utama */}
                <tr>
                  <td className="py-3.5 px-3">
                    <p className="font-semibold font-heading text-neutral-900">
                      Paket Umroh: {booking.jadwal_nama || booking.schedule?.jadwal_nama}
                    </p>
                    <p className="text-xs text-neutral-500 font-body">
                      Keberangkatan {formatTanggal(booking.berangkat_tanggal)}
                    </p>
                  </td>
                  <td className="py-3.5 px-3 font-medium uppercase text-neutral-700">
                    Kamar {booking.room_type || booking.tipe_kamar}
                  </td>
                  <td className="py-3.5 px-3 text-center text-neutral-800">
                    1 Pax
                  </td>
                  <td className="py-3.5 px-3 text-right text-neutral-800 font-medium">
                    {formatRupiah(hargaDasar)}
                  </td>
                  <td className="py-3.5 px-3 text-right text-neutral-900 font-bold font-heading">
                    {formatRupiah(hargaDasar)}
                  </td>
                  <td className="py-3.5 px-3 text-center text-neutral-300">
                    -
                  </td>
                </tr>

                {/* 2. Baris Add-ons */}
                {addons.map((addon) => (
                  <tr key={addon.id} className="bg-primary-50/20">
                    <td className="py-3 px-3">
                      <div className="flex items-center gap-2">
                        <span className="text-[11px] font-semibold font-heading px-1.5 py-0.5 rounded bg-primary-100 text-primary-800">
                          Add-on
                        </span>
                        <span className="font-medium text-neutral-800 font-body">
                          {addon.nama}
                        </span>
                      </div>
                    </td>
                    <td className="py-3 px-3 text-xs text-neutral-500 font-body">
                      Tambahan Biaya
                    </td>
                    <td className="py-3 px-3 text-center text-neutral-700 text-xs">
                      1
                    </td>
                    <td className="py-3 px-3 text-right text-primary-700 font-medium">
                      + {formatRupiah(addon.nominal)}
                    </td>
                    <td className="py-3 px-3 text-right text-primary-800 font-semibold font-heading">
                      + {formatRupiah(addon.nominal)}
                    </td>
                    <td className="py-3 px-3 text-center">
                      <button
                        type="button"
                        onClick={() => handleDeleteAddon(addon.id)}
                        className="text-neutral-400 hover:text-danger-600 p-1 transition-colors rounded hover:bg-neutral-100 cursor-pointer"
                        title="Hapus Add-on"
                      >
                        <Trash2 size={15} />
                      </button>
                    </td>
                  </tr>
                ))}

                {/* 3. Baris Diskon */}
                {diskon > 0 && (
                  <tr className="bg-success-50/20">
                    <td className="py-3 px-3">
                      <div className="flex items-center gap-2">
                        <span className="text-[11px] font-semibold font-heading px-1.5 py-0.5 rounded bg-success-100 text-success-800">
                          Diskon
                        </span>
                        <span className="font-medium text-success-900 font-body">
                          {booking.diskon_keterangan || "Potongan Khusus"}
                        </span>
                      </div>
                    </td>
                    <td className="py-3 px-3 text-xs text-success-700 font-body">
                      Potongan Harga
                    </td>
                    <td className="py-3 px-3 text-center text-neutral-400 text-xs">
                      -
                    </td>
                    <td className="py-3 px-3 text-right text-success-600 font-medium">
                      - {formatRupiah(diskon)}
                    </td>
                    <td className="py-3 px-3 text-right text-success-700 font-bold font-heading">
                      - {formatRupiah(diskon)}
                    </td>
                    <td className="py-3 px-3 text-center">
                      <button
                        type="button"
                        onClick={handleOpenDiskonModal}
                        className="text-success-600 hover:text-success-800 p-1 transition-colors rounded hover:bg-success-100 cursor-pointer"
                        title="Kelola Diskon"
                      >
                        <Tag size={15} />
                      </button>
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Tombol Aksi Tambah Add-on & Diskon */}
          <div className="mt-4 pt-3 flex flex-wrap items-center justify-between gap-3 border-t border-neutral-100">
            <div className="flex flex-wrap gap-2">
              <Button 
                size="sm" 
                variant="secondary" 
                onClick={() => {
                  setAddonError(null);
                  setAddonForm({ nama: "", nominal: "" });
                  setIsAddonModalOpen(true);
                }}
                className="text-xs"
              >
                + Tambah Add-on
              </Button>
              <Button 
                size="sm" 
                variant="secondary" 
                onClick={handleOpenDiskonModal}
                className="text-xs"
              >
                {diskon > 0 ? "Kelola Diskon" : "+ Atur Diskon"}
              </Button>
            </div>

            {/* 4. Financial Breakdown Summary (Bottom Right) */}
            <div className="w-full sm:w-80 space-y-2 bg-neutral-50/90 p-4 rounded-lg border border-neutral-200/80">
              <div className="flex justify-between items-center text-xs font-body">
                <span className="text-neutral-500">Subtotal Paket:</span>
                <span className="font-medium text-neutral-800">{formatRupiah(hargaDasar)}</span>
              </div>
              {totalAddons > 0 && (
                <div className="flex justify-between items-center text-xs font-body">
                  <span className="text-neutral-500">Total Add-on ({addons.length}):</span>
                  <span className="font-medium text-primary-700">+ {formatRupiah(totalAddons)}</span>
                </div>
              )}
              {diskon > 0 && (
                <div className="flex justify-between items-center text-xs font-body">
                  <span className="text-neutral-500">Diskon:</span>
                  <span className="font-semibold text-success-600">- {formatRupiah(diskon)}</span>
                </div>
              )}
              <div className="flex justify-between items-center text-sm font-body pt-1.5 border-t border-neutral-200/60">
                <span className="font-semibold font-heading text-neutral-900">Total Tagihan:</span>
                <span className="font-bold font-heading text-neutral-900">{formatRupiah(totalHarga)}</span>
              </div>
              <div className="flex justify-between items-center text-xs font-body">
                <span className="text-neutral-500">Telah Dibayar (Confirmed):</span>
                <span className="font-semibold text-success-600">{formatRupiah(totalPaid)}</span>
              </div>
              <div className="flex justify-between items-center text-base font-body pt-2 border-t border-neutral-200">
                <span className="font-bold font-heading text-neutral-900">Sisa Tagihan:</span>
                <span className={`font-bold font-heading ${sisaTagihan > 0 ? 'text-danger-600' : 'text-success-600'}`}>
                  {formatRupiah(sisaTagihan)}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* 5. Section: Riwayat Pembayaran */}
        <div className="p-6 sm:p-8 bg-neutral-50/30">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
            <div>
              <h3 className="text-base font-heading font-semibold text-neutral-900">
                Riwayat Pembayaran & Bukti Transfer
              </h3>
              <p className="text-xs text-neutral-500 font-body">
                Daftar pembayaran yang telah disetorkan oleh jamaah
              </p>
            </div>
            <Button 
              size="sm" 
              onClick={() => {
                setPaymentFormError(null);
                setIsPaymentModalOpen(true);
              }}
              disabled={sisaTagihan <= 0}
              title={sisaTagihan <= 0 ? "Tagihan sudah lunas" : "Tambah Pembayaran"}
              className="text-xs"
            >
              + Tambah Pembayaran
            </Button>
          </div>

          <div className="bg-white rounded-lg border border-neutral-200 shadow-sm overflow-hidden">
            <Table 
              columns={paymentColumns}
              data={payments}
              emptyMessage="Belum ada riwayat pembayaran yang dicatat."
            />
          </div>
        </div>

      </div>

      {/* Modal Tambah Add-on */}
      <Modal
        isOpen={isAddonModalOpen}
        onClose={() => setIsAddonModalOpen(false)}
        title="Tambah Biaya Add-on"
      >
        <form onSubmit={handleAddAddonSubmit} className="space-y-4">
          {addonError && <Alert variant="error" message={addonError} onClose={() => setAddonError(null)} />}

          <Input
            label="Nama Layanan / Add-on"
            className="!mb-0"
            placeholder="Contoh: Kereta Cepat Haramain, Upgrade Kamar Ka'bah View"
            required
            value={addonForm.nama}
            onChange={(e) => setAddonForm({ ...addonForm, nama: e.target.value })}
          />

          <CurrencyInput
            label="Nominal Biaya (Rp)"
            className="!mb-0"
            placeholder="0"
            required
            value={addonForm.nominal}
            onChange={(val) => setAddonForm({ ...addonForm, nominal: val })}
          />

          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={() => setIsAddonModalOpen(false)}>Batal</Button>
            <Button type="submit" variant="primary" disabled={addonSubmitting}>
              {addonSubmitting ? "Menyimpan..." : "Tambahkan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Kelola Diskon */}
      <Modal
        isOpen={isDiskonModalOpen}
        onClose={() => setIsDiskonModalOpen(false)}
        title="Atur Diskon & Potongan Harga"
      >
        <form onSubmit={handleUpdateDiskonSubmit} className="space-y-4">
          {diskonError && <Alert variant="error" message={diskonError} onClose={() => setDiskonError(null)} />}

          <Input
            label="Keterangan / Alasan Diskon"
            className="!mb-0"
            placeholder="Contoh: Promo Early Bird, Diskon Keluarga, Voucher Milad"
            value={diskonForm.diskon_keterangan}
            onChange={(e) => setDiskonForm({ ...diskonForm, diskon_keterangan: e.target.value })}
          />

          <CurrencyInput
            label="Nominal Diskon (Rp)"
            className="!mb-0"
            placeholder="0"
            required
            value={diskonForm.diskon}
            onChange={(val) => setDiskonForm({ ...diskonForm, diskon: val })}
          />

          <div className="flex justify-between items-center pt-4 border-t border-neutral-200">
            <div>
              {diskon > 0 && (
                <Button 
                  type="button" 
                  variant="danger" 
                  size="sm"
                  onClick={handleRemoveDiskon}
                  disabled={diskonSubmitting}
                >
                  Hapus Diskon
                </Button>
              )}
            </div>
            <div className="flex gap-2">
              <Button type="button" variant="ghost" onClick={() => setIsDiskonModalOpen(false)}>Batal</Button>
              <Button type="submit" variant="primary" disabled={diskonSubmitting}>
                {diskonSubmitting ? "Menyimpan..." : "Simpan Diskon"}
              </Button>
            </div>
          </div>
        </form>
      </Modal>

      {/* Modal Pembayaran */}
      <Modal
        isOpen={isPaymentModalOpen}
        onClose={() => setIsPaymentModalOpen(false)}
        title="Catat Pembayaran Baru"
      >
        <form onSubmit={handlePaymentSubmit} className="space-y-4">
          {paymentFormError && <Alert variant="error" message={paymentFormError} onClose={() => setPaymentFormError(null)} />}

          <div>
            <CurrencyInput 
              label="Jumlah (Rp)"
              className="!mb-0"
              required 
              value={paymentForm.jumlah}
              onChange={(val) => {
                setPaymentForm({...paymentForm, jumlah: val});
                if (parseFloat(val) > sisaTagihan) {
                  setPaymentFormError(`Jumlah melebihi sisa tagihan (${formatRupiah(sisaTagihan)})`);
                } else {
                  setPaymentFormError(null);
                }
              }}
              placeholder="0"
              error={paymentForm.jumlah && parseFloat(paymentForm.jumlah) > sisaTagihan ? "Melebihi sisa tagihan" : null}
            />
            <div className="flex justify-between items-center mt-1 px-0.5">
              <span className="text-xs text-neutral-500 font-body">
                Sisa tagihan: <span className="font-semibold text-neutral-700">{formatRupiah(sisaTagihan)}</span>
              </span>
              {sisaTagihan > 0 && (
                <button
                  type="button"
                  onClick={() => {
                    setPaymentForm(prev => ({ ...prev, jumlah: sisaTagihan }));
                    setPaymentFormError(null);
                  }}
                  className="text-xs text-primary-600 hover:text-primary-700 font-medium font-body underline cursor-pointer"
                >
                  Bayar Penuh
                </button>
              )}
            </div>
          </div>

          <CustomDropdown 
            label="Metode Pembayaran"
            className="!mb-0"
            required
            value={paymentForm.metode}
            onChange={(val) => setPaymentForm({...paymentForm, metode: val})}
            options={[
              { value: 'transfer', label: 'Transfer Bank' },
              { value: 'tunai', label: 'Tunai' },
              { value: 'qris', label: 'QRIS' }
            ]}
          />

          <Input 
            type="date" 
            label="Tanggal Pembayaran"
            className="!mb-0"
            required 
            value={paymentForm.tanggal}
            onChange={(e) => setPaymentForm({...paymentForm, tanggal: e.target.value})}
          />

          {/* Upload Bukti Pembayaran */}
          <FormField label="Bukti Pembayaran (Opsional)">
            <div className="space-y-2">
              {paymentForm.bukti_file ? (
                <div className="flex items-center justify-between p-2.5 bg-neutral-50 rounded-md border border-neutral-200">
                  <div className="flex items-center gap-2 overflow-hidden">
                    <FileText size={16} className="text-primary-600 shrink-0" />
                    <span className="text-xs text-neutral-800 font-body truncate">
                      {paymentForm.bukti_file.name}
                    </span>
                    <span className="text-xs text-neutral-400 font-body shrink-0">
                      ({(paymentForm.bukti_file.size / 1024).toFixed(0)} KB)
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => setPaymentForm(prev => ({ ...prev, bukti_file: null }))}
                    className="text-neutral-400 hover:text-danger-600 p-1 transition-colors"
                    title="Hapus file"
                  >
                    <X size={14} />
                  </button>
                </div>
              ) : (
                <label className="flex flex-col items-center justify-center p-4 border border-dashed border-neutral-300 rounded-md cursor-pointer hover:border-primary-500 hover:bg-neutral-50 transition-colors">
                  <Upload size={18} className="text-neutral-400 mb-1" />
                  <span className="text-xs font-medium text-neutral-700 font-body">Pilih file bukti transfer / kwitansi</span>
                  <span className="text-[11px] text-neutral-400 font-body mt-0.5">JPG, PNG, atau PDF</span>
                  <input
                    type="file"
                    accept="image/*,application/pdf"
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file) setPaymentForm(prev => ({ ...prev, bukti_file: file }));
                    }}
                    className="hidden"
                  />
                </label>
              )}
            </div>
          </FormField>
          
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={() => setIsPaymentModalOpen(false)}>Batal</Button>
            <Button 
              type="submit" 
              variant="primary" 
              disabled={paymentSubmitting || (paymentForm.jumlah && parseFloat(paymentForm.jumlah) > sisaTagihan)}
            >
              {paymentSubmitting ? "Menyimpan..." : "Simpan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Konfirmasi Pembatalan Booking */}
      <Modal
        isOpen={cancelSeatBlockModal}
        onClose={() => !seatBlockLoading && setCancelSeatBlockModal(false)}
        title="Lepaskan Block Seat"
      >
        <div className="space-y-4">
          <p className="text-sm text-neutral-600 font-body">
            Kursi akan dikembalikan ke kuota paket. Booking, data jamaah, pembayaran, dan tagihan tetap aktif serta tidak dibatalkan.
          </p>
          <div className="rounded-lg border border-warning-200 bg-warning-50 p-3 text-xs text-warning-800">
            Status booking tetap <strong>{booking.status.toUpperCase()}</strong> setelah blok kursi dilepas.
          </div>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button variant="ghost" onClick={() => setCancelSeatBlockModal(false)} disabled={seatBlockLoading}>Kembali</Button>
            <Button variant="danger" onClick={handleCancelSeatBlock} disabled={seatBlockLoading}>
              {seatBlockLoading ? "Memproses..." : "Ya, Lepaskan Kursi"}
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={cancelConfirmModal}
        onClose={() => setCancelConfirmModal(false)}
        title="Konfirmasi Pembatalan Booking"
      >
        <div className="space-y-4">
          <p className="text-sm text-neutral-600 font-body">
            Apakah Anda yakin ingin membatalkan booking ini?
            {booking.status === "dp" && (
              <span className="block mt-2 font-medium text-neutral-900">
                1 Kursi akan otomatis dikembalikan ke kuota paket (seat_sisa bertambah 1).
              </span>
            )}
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button variant="ghost" onClick={() => setCancelConfirmModal(false)}>
              Kembali
            </Button>
            <Button variant="danger" onClick={() => executeStatusChange("batal")}>
              Ya, Batalkan Booking
            </Button>
          </div>
        </div>
      </Modal>

      {/* Modal Konfirmasi Batalkan Perlengkapan */}
      <Modal
        isOpen={isBatalkanModalOpen}
        onClose={() => setIsBatalkanModalOpen(false)}
        title="Batalkan Penyerahan Perlengkapan"
      >
        <div className="space-y-4">
          <p className="text-sm text-neutral-600 font-body">
            Yakin batalkan penyerahan perlengkapan untuk jamaah ini? Stok item perlengkapan akan dikembalikan ke inventaris brand.
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              variant="secondary"
              onClick={() => setIsBatalkanModalOpen(false)}
              disabled={perlengkapanLoading}
            >
              Batal
            </Button>
            <Button
              variant="danger"
              onClick={handleBatalkanPerlengkapan}
              disabled={perlengkapanLoading}
            >
              {perlengkapanLoading ? "Membatalkan..." : "Ya, Batalkan"}
            </Button>
          </div>
        </div>
      </Modal>

    </div>
  );
};

export default BookingDetailPage;

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
import { listBrands } from "../api/brands";
import { uploadMedia } from "../api/media";
import { getStatusBadgeConfig, getSeatLockIcon } from "../utils/bookingStatus";
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
import BrandCell from "../components/BrandCell";
import { CheckCircle, ExternalLink, FileText, Upload, X, Shield, Calendar, User, Plane, Check, Plus, Trash2, Tag, Percent, Package, Loader, CircleCheckBig } from "lucide-react";

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
  { key: 'hotel', label: 'Hotel' },
  { key: 'land_arrangement', label: 'Land Arrangement' },
  { key: 'manasik', label: 'Manasik' },
  { key: 'vaksin_meningitis', label: 'Vaksin Meningitis' },
];

export const BookingDetailPage = ({ showBrandColumn = false }) => {
  const { id } = useParams();
  const navigate = useNavigate();

  const [booking, setBooking] = useState(null);
  const [jamaah, setJamaah] = useState(null);
  const [payments, setPayments] = useState([]);
  const [brand, setBrand] = useState(null);
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

      if (bookRes?.brand_id || showBrandColumn) {
        try {
          const brandsData = await listBrands();
          const found = (brandsData || []).find(b => b.id === (bookRes?.brand_id || bookRes?.schedule?.brand_id));
          if (found) setBrand(found);
        } catch (e) {
          // ignore brand fetch error
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
  }, [id, showBrandColumn]);

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
      diskon: booking?.diskon ? booking.diskon.toString() : "",
      diskon_keterangan: booking?.diskon_keterangan || ""
    });
    setDiskonError(null);
    setIsDiskonModalOpen(true);
  };

  const handleDiskonSubmit = async (e) => {
    e.preventDefault();
    const diskonNum = parseFloat(diskonForm.diskon) || 0;
    if (diskonNum < 0) {
      setDiskonError("Diskon tidak boleh bernilai negatif");
      return;
    }

    setDiskonSubmitting(true);
    setDiskonError(null);
    try {
      await updateBookingDiskon(id, {
        diskon: diskonNum,
        diskon_keterangan: diskonForm.diskon_keterangan.trim() || null
      });
      setIsDiskonModalOpen(false);
      fetchAll();
    } catch (err) {
      setDiskonError(err.response?.data?.error || "Gagal memperbarui diskon");
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

      // Recalculate computed siap_berangkat (paspor dinamis + tiket dinamis + 6 toggle manual)
      const allDone = Boolean(updated.progress_paspor) && Boolean(updated.progress_tiket) && PROGRESS_MANUAL_ITEMS.every((item) => {
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
        buktiUrl = await uploadMedia(paymentForm.bukti_file, "payment-proofs");
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
        bukti_file: null,
      });
      fetchAll();
    } catch (err) {
      setPaymentFormError(err.response?.data?.error || "Gagal mencatat pembayaran");
    } finally {
      setPaymentSubmitting(false);
    }
  };

  const handleUpdatePaymentStatus = async (paymentId, newStatus) => {
    try {
      await updatePaymentStatus(paymentId, newStatus);
      fetchAll();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal mengubah status pembayaran");
    }
  };

  // Print Invoice Modal
  const [isInvoiceModalOpen, setIsInvoiceModalOpen] = useState(false);

  if (loading) {
    return (
      <div className="p-12 flex justify-center items-center">
        <LoadingSpinner />
      </div>
    );
  }

  if (!booking) {
    return (
      <div className="space-y-4">
        <Alert variant="error">Booking tidak ditemukan</Alert>
        <Button variant="ghost" onClick={() => navigate("/bookings")}>← Kembali ke Daftar Booking</Button>
      </div>
    );
  }

  const [statusVariant, statusLabel] = getStatusBadgeConfig(booking.status);
  const seatLockInfo = getSeatLockIcon(booking.status, booking.is_seat_blocked);

  return (
    <div className="w-full space-y-6">
      {error && <Alert variant="error">{error}</Alert>}

      {/* Header Info */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 bg-white p-6 rounded-lg border border-neutral-200 shadow-sm">
        <div>
          <div className="flex flex-wrap items-center gap-3">
            <span className="font-mono text-sm font-bold bg-neutral-100 text-neutral-900 px-3 py-1 rounded border border-neutral-200">
              {booking.id_booking || `ID: ${booking.id}`}
            </span>
            <Badge variant={statusVariant} hideIcon={true} className="text-sm font-semibold px-3 py-1">
              Status: {statusLabel}
            </Badge>
            {seatLockInfo && (
              <Badge variant={booking.is_seat_blocked ? "success" : "warning"} className="text-sm px-3 py-1 font-medium flex items-center gap-1.5">
                {booking.is_seat_blocked ? <CircleCheckBig size={14} /> : <Loader size={14} />}
                <span>{seatLockInfo.label}</span>
              </Badge>
            )}
            {showBrandColumn && (brand || booking.brand_id) && (
              <div className="mt-0.5">
                <BrandCell brand={brand} brandId={booking.brand_id} showText={true} />
              </div>
            )}
          </div>
          <p className="text-xs text-neutral-500 font-body mt-2">
            Dibuat pada {formatTanggal(booking.created_at)}
          </p>
        </div>

        {/* Action Controls */}
        <div className="flex flex-wrap items-center gap-2">
          {booking.status === "baru" && (
            <Button size="sm" variant="secondary" onClick={() => handleBookingStatus("dp")}>
              Set Status DP
            </Button>
          )}
          {booking.status === "dp" && sisaTagihan === 0 && (
            <Button size="sm" variant="primary" onClick={() => handleBookingStatus("lunas")}>
              Set Status Lunas
            </Button>
          )}
          {booking.status !== "batal" && (
            <Button size="sm" variant="ghost" className="text-danger-600 hover:bg-danger-50" onClick={() => handleBookingStatus("batal")}>
              Batalkan Booking
            </Button>
          )}
          <Button size="sm" variant="secondary" onClick={() => setIsInvoiceModalOpen(true)} className="flex items-center gap-1.5">
            <FileText size={16} />
            <span>Cetak Invoice</span>
          </Button>
        </div>
      </div>

      {/* Main Grid: Details */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom Kiri: Detail Paket & Jamaah (2 span) */}
        <div className="lg:col-span-2 space-y-6">
          {/* Section: Paket & Kamar */}
          <div className="bg-white p-6 rounded-lg border border-neutral-200 shadow-sm space-y-4">
            <div className="flex items-center gap-2 pb-3 border-b border-neutral-200">
              <Plane size={18} className="text-primary-600" />
              <h3 className="font-semibold text-neutral-900 font-heading">Informasi Paket & Kamar</h3>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <span className="block text-xs text-neutral-500 font-body">Nama Paket</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{booking.jadwal_nama || "-"}</p>
              </div>
              <div>
                <span className="block text-xs text-neutral-500 font-body">Tanggal Keberangkatan</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{formatTanggal(booking.berangkat_tanggal)}</p>
              </div>
              <div>
                <span className="block text-xs text-neutral-500 font-body">Tipe Kamar</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{(booking.room_type || "-").toUpperCase()}</p>
              </div>
              <div>
                <span className="block text-xs text-neutral-500 font-body">Harga Dasar Paket</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{formatRupiah(hargaDasar)}</p>
              </div>
            </div>
          </div>

          {/* Section: Jamaah Info */}
          <div className="bg-white p-6 rounded-lg border border-neutral-200 shadow-sm space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-neutral-200">
              <div className="flex items-center gap-2">
                <User size={18} className="text-primary-600" />
                <h3 className="font-semibold text-neutral-900 font-heading">Data Jamaah</h3>
              </div>
              {booking.jamaah_id && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => navigate(`/jamaah/${booking.jamaah_id}`)}
                  className="text-xs text-primary-600 hover:text-primary-700 flex items-center gap-1"
                >
                  <span>Lihat Profil Lengkap</span>
                  <ExternalLink size={12} />
                </Button>
              )}
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <span className="block text-xs text-neutral-500 font-body">Nama Lengkap</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{booking.nama_jamaah || jamaah?.nama_lengkap || "-"}</p>
              </div>
              <div>
                <span className="block text-xs text-neutral-500 font-body">NIK</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{jamaah?.nik || "-"}</p>
              </div>
              <div>
                <span className="block text-xs text-neutral-500 font-body">No HP / WhatsApp</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{jamaah?.no_hp || "-"}</p>
              </div>
              <div>
                <span className="block text-xs text-neutral-500 font-body">Nomor Paspor</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{jamaah?.no_paspor || "-"}</p>
              </div>
            </div>
          </div>

          {/* Section: Add-ons & Diskon */}
          <div className="bg-white p-6 rounded-lg border border-neutral-200 shadow-sm space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-neutral-200">
              <div className="flex items-center gap-2">
                <Tag size={18} className="text-primary-600" />
                <h3 className="font-semibold text-neutral-900 font-heading">Add-on & Biaya Tambahan</h3>
              </div>
              <Button size="sm" variant="secondary" onClick={() => setIsAddonModalOpen(true)} className="text-xs flex items-center gap-1">
                <Plus size={14} />
                <span>Tambah Add-on</span>
              </Button>
            </div>

            {addons.length === 0 ? (
              <p className="text-xs text-neutral-500 font-body">Tidak ada biaya tambahan/add-on.</p>
            ) : (
              <div className="divide-y divide-neutral-100">
                {addons.map((a) => (
                  <div key={a.id} className="py-2.5 flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-neutral-900 font-body">{a.nama}</p>
                      <span className="text-xs text-neutral-500 font-body">{formatTanggal(a.created_at)}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-semibold text-neutral-900 font-body">{formatRupiah(a.nominal)}</span>
                      <Button variant="ghost" size="sm" onClick={() => handleDeleteAddon(a.id)} title="Hapus Add-on">
                        <Trash2 size={14} className="text-danger-600" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <div className="pt-3 border-t border-neutral-200 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Percent size={16} className="text-primary-600" />
                <span className="text-sm font-medium text-neutral-800 font-body">Diskon Khusus:</span>
                {booking.diskon_keterangan && (
                  <span className="text-xs text-neutral-500 font-body">({booking.diskon_keterangan})</span>
                )}
              </div>
              <div className="flex items-center gap-3">
                <span className="text-sm font-bold text-success-600 font-body">
                  {diskon > 0 ? `- ${formatRupiah(diskon)}` : "Rp 0"}
                </span>
                <Button size="sm" variant="ghost" onClick={handleOpenDiskonModal} className="text-xs text-primary-600">
                  Ubah Diskon
                </Button>
              </div>
            </div>
          </div>

          {/* Section: Riwayat Pembayaran */}
          <div className="bg-white p-6 rounded-lg border border-neutral-200 shadow-sm space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-neutral-200">
              <h3 className="font-semibold text-neutral-900 font-heading">Riwayat Pembayaran</h3>
              {sisaTagihan > 0 && booking.status !== "batal" && (
                <Button size="sm" variant="primary" onClick={() => setIsPaymentModalOpen(true)} className="text-xs flex items-center gap-1">
                  <Plus size={14} />
                  <span>Catat Pembayaran</span>
                </Button>
              )}
            </div>

            {payments.length === 0 ? (
              <p className="text-xs text-neutral-500 font-body">Belum ada pembayaran yang dicatat.</p>
            ) : (
              <Table
                columns={[
                  { header: "Tanggal", key: "tanggal" },
                  { header: "Metode", key: "metode" },
                  { header: "Jumlah", key: "jumlah" },
                  { header: "Bukti", key: "bukti_url" },
                  { header: "Status", key: "status" },
                  { header: "Aksi", key: "aksi" },
                ]}
                data={payments}
                renderCell={(p, key) => {
                  if (key === "tanggal") return formatTanggal(p.tanggal || p.created_at);
                  if (key === "metode") return (p.metode || "transfer").toUpperCase();
                  if (key === "jumlah") return formatRupiah(p.jumlah);
                  if (key === "bukti_url") {
                    return p.bukti_url ? (
                      <a
                        href={p.bukti_url.startsWith('http') ? p.bukti_url : `${import.meta.env.VITE_API_BASE_URL}${p.bukti_url}`}
                        target="_blank"
                        rel="noreferrer"
                        className="text-primary-600 hover:underline flex items-center gap-1 text-xs"
                      >
                        <span>Lihat Bukti</span>
                        <ExternalLink size={12} />
                      </a>
                    ) : "-";
                  }
                  if (key === "status") {
                    const statusConfig = {
                      pending: ["warning", "Menunggu"],
                      confirmed: ["success", "Diterima"],
                      rejected: ["danger", "Ditolak"],
                    };
                    const [v, label] = statusConfig[p.status] || ["neutral", p.status];
                    return <Badge variant={v} hideIcon={true}>{label}</Badge>;
                  }
                  if (key === "aksi") {
                    if (p.status === "pending") {
                      return (
                        <div className="flex items-center gap-1">
                          <Button size="sm" variant="ghost" onClick={() => handleUpdatePaymentStatus(p.id, "confirmed")} title="Konfirmasi">
                            <Check size={14} className="text-success-600" />
                          </Button>
                          <Button size="sm" variant="ghost" onClick={() => handleUpdatePaymentStatus(p.id, "rejected")} title="Tolak">
                            <X size={14} className="text-danger-600" />
                          </Button>
                        </div>
                      );
                    }
                    return <span className="text-xs text-neutral-400">-</span>;
                  }
                  return p[key];
                }}
              />
            )}
          </div>
        </div>

        {/* Kolom Kanan: Ringkasan Tagihan, Progress, & Perlengkapan (1 span) */}
        <div className="space-y-6">
          {/* Ringkasan Finansial Card */}
          <div className="bg-white p-6 rounded-lg border border-neutral-200 shadow-sm space-y-4">
            <h3 className="font-semibold text-neutral-900 font-heading pb-2 border-b border-neutral-200">
              Ringkasan Pembayaran
            </h3>
            <div className="space-y-2.5 font-body">
              <div className="flex justify-between text-sm">
                <span className="text-neutral-500">Harga Paket Dasar:</span>
                <span className="font-medium text-neutral-900">{formatRupiah(hargaDasar)}</span>
              </div>
              {totalAddons > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-neutral-500">Total Add-on:</span>
                  <span className="font-medium text-neutral-900">+ {formatRupiah(totalAddons)}</span>
                </div>
              )}
              {diskon > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-neutral-500">Diskon:</span>
                  <span className="font-medium text-success-600">- {formatRupiah(diskon)}</span>
                </div>
              )}
              <div className="pt-2 border-t border-neutral-200 flex justify-between text-base font-bold">
                <span className="text-neutral-900">Total Tagihan:</span>
                <span className="text-neutral-900">{formatRupiah(totalHarga)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-neutral-500">Sudah Dibayar:</span>
                <span className="font-medium text-success-600">{formatRupiah(totalPaid)}</span>
              </div>
              <div className="pt-2 border-t border-neutral-200 flex justify-between items-center">
                <span className="text-sm font-semibold text-neutral-700">Sisa Tagihan:</span>
                <span className={`text-base font-bold ${sisaTagihan > 0 ? "text-danger-600" : "text-success-600"}`}>
                  {formatRupiah(sisaTagihan)}
                </span>
              </div>
            </div>
          </div>

          {/* Progress Keberangkatan Card */}
          <div className="bg-white p-6 rounded-lg border border-neutral-200 shadow-sm space-y-4">
            <div className="flex items-center justify-between pb-2 border-b border-neutral-200">
              <div className="flex items-center gap-2">
                <Shield size={18} className="text-primary-600" />
                <h3 className="font-semibold text-neutral-900 font-heading">Progress Keberangkatan</h3>
              </div>
              {booking.siap_berangkat ? (
                <Badge variant="success" className="px-2 py-0.5 text-xs font-semibold">
                  Siap Berangkat
                </Badge>
              ) : (
                <Badge variant="warning" className="px-2 py-0.5 text-xs font-semibold">
                  Persiapan Berjalan
                </Badge>
              )}
            </div>

            <div className="space-y-3">
              {/* 1. Paspor Read-only */}
              <div className="flex items-center justify-between p-3 bg-neutral-50 rounded-md border border-neutral-200">
                <div>
                  <span className="text-sm font-medium font-body text-neutral-800 block">Paspor</span>
                  <span className="text-xs text-neutral-500 font-body block">(otomatis dari dokumen)</span>
                </div>
                <Toggle
                  id="toggle-paspor-detail"
                  checked={Boolean(booking.progress_paspor)}
                  disabled={true}
                  onChange={() => {}}
                />
              </div>

              {/* 2. Tiket Maskapai Read-only */}
              <div className="flex items-center justify-between p-3 bg-neutral-50 rounded-md border border-neutral-200">
                <div>
                  <span className="text-sm font-medium font-body text-neutral-800 block">Tiket Maskapai</span>
                  <span className="text-xs text-neutral-500 font-body block">(otomatis dari master paket)</span>
                </div>
                <Toggle
                  id="toggle-tiket-detail"
                  checked={Boolean(booking.progress_tiket)}
                  disabled={true}
                  onChange={() => {}}
                />
              </div>

              {/* 3. Toggle Manual Items */}
              {PROGRESS_MANUAL_ITEMS.map((item) => {
                const isChecked = Boolean(booking[`progress_${item.key}`]);
                return (
                  <div key={item.key} className="flex items-center justify-between p-3 bg-white rounded-md border border-neutral-200">
                    <span className="text-sm font-medium font-body text-neutral-800">{item.label}</span>
                    <Toggle
                      id={`toggle-${item.key}-detail`}
                      checked={isChecked}
                      onChange={() => handleToggleProgress(item.key, isChecked)}
                    />
                  </div>
                );
              })}
            </div>
          </div>

          {/* Perlengkapan Card */}
          <div className="bg-white p-6 rounded-lg border border-neutral-200 shadow-sm space-y-4">
            <div className="flex items-center gap-2 pb-2 border-b border-neutral-200">
              <Package size={18} className="text-primary-600" />
              <h3 className="font-semibold text-neutral-900 font-heading">Perlengkapan Jamaah</h3>
            </div>

            {perlengkapanError && <Alert variant="error">{perlengkapanError}</Alert>}

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-neutral-500 font-body">Status:</span>
                {booking.perlengkapan_status === 'sudah_diberikan' ? (
                  <Badge variant="success">Sudah Diberikan</Badge>
                ) : (
                  <Badge variant="neutral">Belum Diberikan</Badge>
                )}
              </div>

              {booking.perlengkapan_tanggal && (
                <div className="flex justify-between text-xs text-neutral-500 font-body">
                  <span>Tanggal Distribusi:</span>
                  <span>{formatTanggal(booking.perlengkapan_tanggal)}</span>
                </div>
              )}

              <div className="pt-2">
                {booking.perlengkapan_status === 'sudah_diberikan' ? (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="w-full text-danger-600 hover:bg-danger-50 text-xs"
                    disabled={perlengkapanLoading}
                    onClick={() => setIsBatalkanModalOpen(true)}
                  >
                    Batalkan Distribusi
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    variant="secondary"
                    className="w-full text-xs"
                    disabled={perlengkapanLoading}
                    onClick={handleDistribusiPerlengkapan}
                  >
                    {perlengkapanLoading ? "Memproses..." : "Tandai Sudah Diberikan"}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Modal: Tambah Pembayaran */}
      <Modal
        isOpen={isPaymentModalOpen}
        onClose={() => setIsPaymentModalOpen(false)}
        title="Catat Pembayaran Baru"
      >
        <form onSubmit={handlePaymentSubmit} className="space-y-4">
          {paymentFormError && <Alert variant="error">{paymentFormError}</Alert>}
          <CurrencyInput
            label="Jumlah Pembayaran (Rp)"
            value={paymentForm.jumlah}
            onChange={(val) => setPaymentForm(prev => ({ ...prev, jumlah: val }))}
            required
            helperText={`Sisa tagihan saat ini: ${formatRupiah(sisaTagihan)}`}
          />
          <CustomDropdown
            label="Metode Pembayaran"
            value={paymentForm.metode}
            onChange={(val) => setPaymentForm(prev => ({ ...prev, metode: val }))}
            options={[
              { value: "transfer", label: "Transfer Bank" },
              { value: "cash", label: "Tunai / Cash" },
            ]}
          />
          <Input
            label="Tanggal Pembayaran"
            type="date"
            value={paymentForm.tanggal}
            onChange={(e) => setPaymentForm(prev => ({ ...prev, tanggal: e.target.value }))}
            required
          />
          <FormField label="Bukti Pembayaran (Opsional)">
            <Input
              type="file"
              accept="image/*,application/pdf"
              onChange={(e) => setPaymentForm(prev => ({ ...prev, bukti_file: e.target.files?.[0] || null }))}
            />
          </FormField>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={() => setIsPaymentModalOpen(false)}>Batal</Button>
            <Button type="submit" variant="primary" disabled={paymentSubmitting}>
              {paymentSubmitting ? "Menyimpan..." : "Simpan Pembayaran"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Tambah Add-on */}
      <Modal
        isOpen={isAddonModalOpen}
        onClose={() => setIsAddonModalOpen(false)}
        title="Tambah Add-on / Biaya Tambahan"
      >
        <form onSubmit={handleAddAddonSubmit} className="space-y-4">
          {addonError && <Alert variant="error">{addonError}</Alert>}
          <Input
            label="Nama Add-on"
            value={addonForm.nama}
            onChange={(e) => setAddonForm(prev => ({ ...prev, nama: e.target.value }))}
            placeholder="Misal: Upgrade Kamar Double, Visa Tambahan"
            required
          />
          <CurrencyInput
            label="Nominal (Rp)"
            value={addonForm.nominal}
            onChange={(val) => setAddonForm(prev => ({ ...prev, nominal: val }))}
            required
          />
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={() => setIsAddonModalOpen(false)}>Batal</Button>
            <Button type="submit" variant="primary" disabled={addonSubmitting}>
              {addonSubmitting ? "Menyimpan..." : "Tambah Add-on"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Ubah Diskon */}
      <Modal
        isOpen={isDiskonModalOpen}
        onClose={() => setIsDiskonModalOpen(false)}
        title="Pengaturan Diskon Booking"
      >
        <form onSubmit={handleDiskonSubmit} className="space-y-4">
          {diskonError && <Alert variant="error">{diskonError}</Alert>}
          <CurrencyInput
            label="Nominal Diskon (Rp)"
            value={diskonForm.diskon}
            onChange={(val) => setDiskonForm(prev => ({ ...prev, diskon: val }))}
            placeholder="0"
          />
          <Input
            label="Keterangan Diskon"
            value={diskonForm.diskon_keterangan}
            onChange={(e) => setDiskonForm(prev => ({ ...prev, diskon_keterangan: e.target.value }))}
            placeholder="Misal: Promo Early Bird, Diskon Keluarga"
          />
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={() => setIsDiskonModalOpen(false)}>Batal</Button>
            <Button type="submit" variant="primary" disabled={diskonSubmitting}>
              {diskonSubmitting ? "Menyimpan..." : "Simpan Diskon"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Cetak Invoice */}
      <Modal
        isOpen={isInvoiceModalOpen}
        onClose={() => setIsInvoiceModalOpen(false)}
        title="Invoice Booking"
        size="lg"
      >
        <div className="space-y-6 font-body text-neutral-900 p-2">
          {/* Header Invoice */}
          <div className="flex justify-between items-start pb-4 border-b border-neutral-200">
            <div>
              <h2 className="text-xl font-bold font-heading text-neutral-900">
                {brand?.name || "AZHAN TRAVEL GRUP"}
              </h2>
              <p className="text-xs text-neutral-500 mt-1">Invoice Bukti Pemesanan & Pembayaran</p>
            </div>
            <div className="text-right">
              <span className="font-mono text-sm font-bold bg-neutral-100 px-2.5 py-1 rounded border border-neutral-200 block">
                {booking.id_booking || `ID: ${booking.id}`}
              </span>
              <span className="text-xs text-neutral-500 mt-1 block">
                {formatTanggal(new Date())}
              </span>
            </div>
          </div>

          {/* Jamaah & Paket Info */}
          <div className="grid grid-cols-2 gap-4 text-xs">
            <div>
              <span className="text-neutral-500 font-semibold block">DITAGIHKAN KEPADA:</span>
              <p className="font-bold text-sm text-neutral-900 mt-0.5">{booking.nama_jamaah || jamaah?.nama_lengkap || "-"}</p>
              <p className="text-neutral-600">{jamaah?.no_hp || "-"}</p>
              <p className="text-neutral-600">{jamaah?.alamat || "-"}</p>
            </div>
            <div>
              <span className="text-neutral-500 font-semibold block">RINCIAN PAKET:</span>
              <p className="font-bold text-sm text-neutral-900 mt-0.5">{booking.jadwal_nama || "-"}</p>
              <p className="text-neutral-600">Keberangkatan: {formatTanggal(booking.berangkat_tanggal)}</p>
              <p className="text-neutral-600">Tipe Kamar: {(booking.room_type || "-").toUpperCase()}</p>
            </div>
          </div>

          {/* Table Breakdown */}
          <div className="border border-neutral-200 rounded-md overflow-hidden text-xs">
            <table className="w-full text-left">
              <thead className="bg-neutral-50 border-b border-neutral-200">
                <tr>
                  <th className="p-2.5 font-semibold text-neutral-700">Deskripsi</th>
                  <th className="p-2.5 text-right font-semibold text-neutral-700">Nominal</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                <tr>
                  <td className="p-2.5">Harga Dasar Paket ({booking.room_type})</td>
                  <td className="p-2.5 text-right font-medium">{formatRupiah(hargaDasar)}</td>
                </tr>
                {addons.map(a => (
                  <tr key={a.id}>
                    <td className="p-2.5">Add-on: {a.nama}</td>
                    <td className="p-2.5 text-right font-medium">+ {formatRupiah(a.nominal)}</td>
                  </tr>
                ))}
                {diskon > 0 && (
                  <tr>
                    <td className="p-2.5 text-success-600">Diskon {booking.diskon_keterangan ? `(${booking.diskon_keterangan})` : ''}</td>
                    <td className="p-2.5 text-right font-medium text-success-600">- {formatRupiah(diskon)}</td>
                  </tr>
                )}
                <tr className="bg-neutral-50 font-bold">
                  <td className="p-2.5">TOTAL TAGIHAN</td>
                  <td className="p-2.5 text-right">{formatRupiah(totalHarga)}</td>
                </tr>
                <tr>
                  <td className="p-2.5 text-neutral-600">Total Pembayaran Diterima</td>
                  <td className="p-2.5 text-right font-medium text-success-600">{formatRupiah(totalPaid)}</td>
                </tr>
                <tr className="bg-neutral-50 font-bold">
                  <td className="p-2.5 text-neutral-900">SISA TAGIHAN</td>
                  <td className={`p-2.5 text-right ${sisaTagihan > 0 ? "text-danger-600" : "text-success-600"}`}>
                    {formatRupiah(sisaTagihan)}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <Button variant="ghost" onClick={() => setIsInvoiceModalOpen(false)}>Tutup</Button>
            <Button variant="primary" onClick={() => window.print()}>Cetak</Button>
          </div>
        </div>
      </Modal>

      {/* Modal: Batalkan Perlengkapan */}
      <Modal
        isOpen={isBatalkanModalOpen}
        onClose={() => setIsBatalkanModalOpen(false)}
        title="Konfirmasi Pembatalan Distribusi Perlengkapan"
      >
        <p className="text-neutral-600 font-body text-sm mb-6">
          Apakah Anda yakin ingin membatalkan distribusi perlengkapan untuk booking ini?
          Stok barang akan dikembalikan ke inventaris.
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="ghost" onClick={() => setIsBatalkanModalOpen(false)}>Batal</Button>
          <Button variant="danger" disabled={perlengkapanLoading} onClick={handleBatalkanPerlengkapan}>
            {perlengkapanLoading ? "Memproses..." : "Ya, Batalkan"}
          </Button>
        </div>
      </Modal>

      {/* Modal: Batalkan Booking */}
      <Modal
        isOpen={cancelConfirmModal}
        onClose={() => setCancelConfirmModal(false)}
        title="Konfirmasi Pembatalan Booking"
      >
        <p className="text-neutral-600 font-body text-sm mb-6">
          Apakah Anda yakin ingin membatalkan booking ini?
          Jika kursi sempat terkunci, kuota kursi akan dikembalikan ke paket.
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="ghost" onClick={() => setCancelConfirmModal(false)}>Batal</Button>
          <Button variant="danger" onClick={() => executeStatusChange("batal")}>
            Ya, Batalkan Booking
          </Button>
        </div>
      </Modal>
    </div>
  );
};

export default BookingDetailPage;

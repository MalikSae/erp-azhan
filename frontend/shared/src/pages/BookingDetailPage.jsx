import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { 
  getBooking, 
  updateBookingStatus, 
  cancelBookingSeatBlock,
  blockBookingSeat,
  listPayments, 
  createPayment, 
  updatePaymentStatus, 
  addBookingAddon, 
  deleteBookingAddon, 
  addBookingDiscount,
  removeBookingDiscount,
  updateBookingProgress,
  updatePaxProgress,
  cancelBookingPax,
  updatePaxRoomType
} from "../api/bookings";
import { markPerlengkapanDiberikan, batalkanPerlengkapan } from "../api/perlengkapan";
import { getJamaah } from "../api/jamaah";
import { listBrands } from "../api/brands";
import { uploadMedia } from "../api/media";
import { getStatusBadgeConfig, getSeatLockIcon } from "../utils/bookingStatus";
import PageHeader from "../components/ui/PageHeader";
import Card from "../components/ui/Card";
import MetaBox from "../components/ui/MetaBox";
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
import { CheckCircle, ExternalLink, FileText, Upload, X, Shield, Calendar, User, Users, Plane, Check, Plus, Trash2, Tag, Percent, Package, Loader, CircleCheckBig, Building2, CreditCard, Receipt } from "lucide-react";

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

const PROGRESS_HEADER_MANUAL_ITEMS = [
  { key: 'hotel', label: 'Hotel' },
  { key: 'land_arrangement', label: 'Land Arrangement' },
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
  const [selectedBuktiUrl, setSelectedBuktiUrl] = useState(null);

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

  // Pax Action Modal State
  const [isChangeRoomModalOpen, setIsChangeRoomModalOpen] = useState(false);
  const [selectedPaxForRoomChange, setSelectedPaxForRoomChange] = useState(null);
  const [newRoomType, setNewRoomType] = useState("Quad");
  const [changeRoomError, setChangeRoomError] = useState(null);
  const [changeRoomSubmitting, setChangeRoomSubmitting] = useState(false);

  const [isCancelPaxModalOpen, setIsCancelPaxModalOpen] = useState(false);
  const [selectedPaxForCancel, setSelectedPaxForCancel] = useState(null);
  const [cancelPaxSubmitting, setCancelPaxSubmitting] = useState(false);

  // Reject Payment Modal State
  const [selectedPaymentForReject, setSelectedPaymentForReject] = useState(null);
  const [rejectionReason, setRejectionReason] = useState("");
  const [rejectSubmitting, setRejectSubmitting] = useState(false);
  const [rejectError, setRejectError] = useState("");

  // Confirm Payment Modal State
  const [selectedPaymentForConfirm, setSelectedPaymentForConfirm] = useState(null);
  const [confirmSubmitting, setConfirmSubmitting] = useState(false);
  const [confirmError, setConfirmError] = useState("");

  // Seat Block / Release Modal State
  const [cancelSeatBlockModal, setCancelSeatBlockModal] = useState(false);
  const [cancelSeatBlockError, setCancelSeatBlockError] = useState("");
  const [seatBlockLoading, setSeatBlockLoading] = useState(false);

  const [blockSeatModal, setBlockSeatModal] = useState(false);
  const [blockSeatError, setBlockSeatError] = useState("");
  const [blockSeatLoading, setBlockSeatLoading] = useState(false);
  const [blockSeatKey, setBlockSeatKey] = useState("");

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

  const handleDistribusiPerlengkapan = async (paxId) => {
    setPerlengkapanLoading(true);
    setPerlengkapanError(null);
    try {
      await markPerlengkapanDiberikan(id, paxId);
      await fetchAll();
    } catch (err) {
      const msg = err.response?.data?.error || "Gagal mendistribusikan perlengkapan.";
      setPerlengkapanError(msg);
    } finally {
      setPerlengkapanLoading(false);
    }
  };

  const [selectedPaxForCancelPerlengkapan, setSelectedPaxForCancelPerlengkapan] = useState(null);

  const handleBatalkanPerlengkapan = async () => {
    if (!selectedPaxForCancelPerlengkapan) return;
    setPerlengkapanLoading(true);
    setPerlengkapanError(null);
    try {
      await batalkanPerlengkapan(id, selectedPaxForCancelPerlengkapan.id);
      setIsBatalkanModalOpen(false);
      setSelectedPaxForCancelPerlengkapan(null);
      await fetchAll();
    } catch (err) {
      const msg = err.response?.data?.error || "Gagal membatalkan perlengkapan.";
      setPerlengkapanError(msg);
    } finally {
      setPerlengkapanLoading(false);
    }
  };

  const [cancelConfirmModal, setCancelConfirmModal] = useState(false);

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
    setCancelSeatBlockError("");
    try {
      await cancelBookingSeatBlock(id);
      setCancelSeatBlockModal(false);
      await fetchAll();
    } catch (err) {
      setCancelSeatBlockError(err.response?.data?.error || "Gagal membatalkan block seat");
    } finally {
      setSeatBlockLoading(false);
    }
  };

  const handleOpenBlockSeat = () => {
    const key = typeof crypto !== 'undefined' && crypto.randomUUID 
      ? crypto.randomUUID() 
      : 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
          const r = (Math.random() * 16) | 0;
          const v = c === 'x' ? r : (r & 0x3) | 0x8;
          return v.toString(16);
        });
    setBlockSeatKey(key);
    setBlockSeatError("");
    setBlockSeatModal(true);
  };

  const handleBlockSeatSubmit = async () => {
    setBlockSeatLoading(true);
    setBlockSeatError("");
    try {
      await blockBookingSeat(id, blockSeatKey);
      setBlockSeatModal(false);
      await fetchAll();
    } catch (err) {
      setBlockSeatError(err.response?.data?.error || "Gagal melakukan block seat");
    } finally {
      setBlockSeatLoading(false);
    }
  };

  // ─── Pax Action Handlers ───────────────────────────────────────────────────
  const handleOpenChangeRoom = (pax) => {
    setSelectedPaxForRoomChange(pax);
    setNewRoomType(pax.room_type || "Quad");
    setChangeRoomError(null);
    setIsChangeRoomModalOpen(true);
  };

  const handleChangeRoomSubmit = async (e) => {
    e.preventDefault();
    const roomVal = (newRoomType && typeof newRoomType === 'object' && 'target' in newRoomType) 
      ? newRoomType.target.value 
      : newRoomType;

    if (!selectedPaxForRoomChange || !roomVal) return;
    setChangeRoomSubmitting(true);
    setChangeRoomError(null);
    try {
      await updatePaxRoomType(id, selectedPaxForRoomChange.id, roomVal);
      setIsChangeRoomModalOpen(false);
      setSelectedPaxForRoomChange(null);
      fetchAll();
    } catch (err) {
      setChangeRoomError(err.response?.data?.error || "Gagal mengubah tipe kamar pax");
    } finally {
      setChangeRoomSubmitting(false);
    }
  };

  const handleOpenCancelPax = (pax) => {
    setSelectedPaxForCancel(pax);
    setIsCancelPaxModalOpen(true);
  };

  const handleCancelPaxSubmit = async () => {
    if (!selectedPaxForCancel) return;
    setCancelPaxSubmitting(true);
    try {
      await cancelBookingPax(id, selectedPaxForCancel.id);
      setIsCancelPaxModalOpen(false);
      setSelectedPaxForCancel(null);
      fetchAll();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal membatalkan pax");
      setIsCancelPaxModalOpen(false);
    } finally {
      setCancelPaxSubmitting(false);
    }
  };

  // Perhitungan Keuangan
  const hargaDasar = booking?.harga_dasar ? parseFloat(booking.harga_dasar) : (parseFloat(booking?.total_harga) || 0);
  const addons = booking?.addons || [];
  const totalAddons = addons.reduce((sum, a) => sum + (parseFloat(a.nominal) || 0), 0);
  const discounts = booking?.discounts || [];
  const totalDiskon = discounts.reduce((sum, d) => sum + (parseFloat(d.nominal) || 0), 0);
  const totalHarga = booking?.total_harga ? parseFloat(booking.total_harga) : Math.max(0, hargaDasar + totalAddons - totalDiskon);

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
    setDiskonForm({ diskon: "", diskon_keterangan: "" });
    setDiskonError(null);
    setIsDiskonModalOpen(true);
  };

  const handleDiskonSubmit = async (e) => {
    e.preventDefault();
    const diskonNum = parseFloat(diskonForm.diskon) || 0;
    if (!diskonForm.diskon_keterangan.trim()) {
      setDiskonError("Nama/Keterangan diskon wajib diisi");
      return;
    }
    if (diskonNum <= 0) {
      setDiskonError("Nominal diskon harus lebih dari 0");
      return;
    }

    setDiskonSubmitting(true);
    setDiskonError(null);
    try {
      await addBookingDiscount(id, {
        nama: diskonForm.diskon_keterangan.trim(),
        nominal: diskonNum
      });
      setIsDiskonModalOpen(false);
      setDiskonForm({ diskon: "", diskon_keterangan: "" });
      fetchAll();
    } catch (err) {
      setDiskonError(err.response?.data?.error || "Gagal menambahkan diskon");
    } finally {
      setDiskonSubmitting(false);
    }
  };

  const handleDeleteDiscount = async (discountId) => {
    if (!window.confirm("Yakin ingin menghapus diskon ini?")) return;
    try {
      await removeBookingDiscount(id, discountId);
      fetchAll();
    } catch (err) {
      alert(err.response?.data?.error || "Gagal menghapus diskon");
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

      const headerLengkap = Boolean(updated.progress_tiket) && Boolean(updated.progress_hotel) && Boolean(updated.progress_land_arrangement);
      let activePaxCount = 0;
      let allPaxLengkap = true;
      (updated.pax || []).forEach((p) => {
        if (p.pax_status !== 'aktif') return;
        activePaxCount++;
        const manasikReq = p.pax_type === 'infant' ? true : Boolean(p.progress_manasik);
        const pLengkap = Boolean(p.progress_paspor) && Boolean(p.progress_visa) && Boolean(p.progress_siskopatuh) && Boolean(p.progress_vaksin_meningitis) && manasikReq;
        if (!pLengkap) allPaxLengkap = false;
      });

      updated.siap_berangkat = (activePaxCount > 0) && headerLengkap && allPaxLengkap;
      return updated;
    });

    try {
      const res = await updateBookingProgress(id, { [key]: newValue });
      setBooking(res);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memperbarui progress paket");
      fetchAll();
    }
  };

  const handleTogglePaxProgress = async (paxId, key, currentValue) => {
    const newValue = !currentValue;
    const progressField = `progress_${key}`;

    // Optimistic Update
    setBooking((prev) => {
      if (!prev) return prev;
      const updatedPax = (prev.pax || []).map((p) => {
        if (p.id === paxId) {
          return { ...p, [progressField]: newValue };
        }
        return p;
      });

      const headerLengkap = Boolean(prev.progress_tiket) && Boolean(prev.progress_hotel) && Boolean(prev.progress_land_arrangement);
      let activePaxCount = 0;
      let allPaxLengkap = true;
      updatedPax.forEach((p) => {
        if (p.pax_status !== 'aktif') return;
        activePaxCount++;
        const manasikReq = p.pax_type === 'infant' ? true : Boolean(p.progress_manasik);
        const pLengkap = Boolean(p.progress_paspor) && Boolean(p.progress_visa) && Boolean(p.progress_siskopatuh) && Boolean(p.progress_vaksin_meningitis) && manasikReq;
        if (!pLengkap) allPaxLengkap = false;
      });

      return {
        ...prev,
        pax: updatedPax,
        siap_berangkat: (activePaxCount > 0) && headerLengkap && allPaxLengkap
      };
    });

    try {
      const res = await updatePaxProgress(id, paxId, { [key]: newValue });
      setBooking(res);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memperbarui checklist pax");
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

  const handleUpdatePaymentStatus = async (paymentId, newStatus, reason = null) => {
    try {
      await updatePaymentStatus(paymentId, newStatus, reason);
      fetchAll();
    } catch (err) {
      setError(err.response?.data?.error || "Gagal mengubah status pembayaran");
    }
  };

  const handleRejectPaymentSubmit = async (e) => {
    e.preventDefault();
    if (!selectedPaymentForReject || rejectionReason.trim().length < 3) return;

    setRejectSubmitting(true);
    setRejectError("");
    try {
      await updatePaymentStatus(selectedPaymentForReject.id, "rejected", rejectionReason.trim());
      setSelectedPaymentForReject(null);
      setRejectionReason("");
      fetchAll();
    } catch (err) {
      setRejectError(err.response?.data?.error || "Gagal menolak pembayaran");
    } finally {
      setRejectSubmitting(false);
    }
  };

  const handleConfirmPaymentSubmit = async () => {
    if (!selectedPaymentForConfirm) return;

    setConfirmSubmitting(true);
    setConfirmError("");
    try {
      await updatePaymentStatus(selectedPaymentForConfirm.id, "confirmed");
      setSelectedPaymentForConfirm(null);
      fetchAll();
    } catch (err) {
      setConfirmError(err.response?.data?.error || "Gagal mengonfirmasi pembayaran");
    } finally {
      setConfirmSubmitting(false);
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

  const paxList = booking.pax || [];
  const activePaxList = paxList.filter((p) => p.pax_status === 'aktif');
  const regularActiveCount = activePaxList.filter((p) => p.pax_type !== 'infant').length;
  const infantActiveCount = activePaxList.filter((p) => p.pax_type === 'infant').length;

  return (
    <div className="w-full space-y-6">
      {error && <Alert variant="error">{error}</Alert>}

      {/* Header Info */}
      <PageHeader
        title={
          <div className="flex flex-wrap items-center gap-3">
            <span>{booking.id_booking || `Booking #${booking.id}`}</span>
            <Badge variant={statusVariant} hideIcon={true} className="text-xs font-semibold px-2.5 py-0.5">
              Status: {statusLabel}
            </Badge>
            {seatLockInfo && (
              <Badge
                variant={booking.is_seat_blocked ? "success" : "warning"}
                icon={booking.is_seat_blocked ? <CircleCheckBig size={13} /> : <Loader size={13} className="animate-spin" />}
                className="text-xs font-medium px-2.5 py-0.5"
              >
                {seatLockInfo.label}
              </Badge>
            )}
          </div>
        }
        subtitle={`Dibuat pada ${formatTanggal(booking.created_at)}`}
        onBack={() => navigate("/bookings")}
      >
        <div className="flex flex-wrap items-center gap-2">
          {booking.status === "baru" && !booking.is_seat_blocked && (
            <Button size="sm" variant="secondary" onClick={handleOpenBlockSeat}>
              Block Seat
            </Button>
          )}
          {booking.is_seat_blocked && (
            <Button 
              size="sm" 
              variant="secondary" 
              className="text-xs" 
              onClick={() => {
                setCancelSeatBlockError("");
                setCancelSeatBlockModal(true);
              }}
            >
              Cancel Block Seat
            </Button>
          )}
          {booking.status !== "batal" && (
            <Button size="sm" variant="danger" className="text-xs shadow-2xs" onClick={() => handleBookingStatus("batal")}>
              Batalkan Booking
            </Button>
          )}
          <Button size="sm" variant="secondary" onClick={() => setIsInvoiceModalOpen(true)} className="flex items-center gap-1.5 shadow-2xs">
            <FileText size={15} />
            <span>Cetak Invoice</span>
          </Button>
        </div>
      </PageHeader>

      {/* Main Grid: Details */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom Kiri: Detail Paket & Jamaah (2 span) */}
        <div className="lg:col-span-2 space-y-6">
          {/* Section: Informasi Paket */}
          <MetaBox 
            title="Informasi Paket" 
            icon={<Plane size={18} className="text-neutral-700" />}
          >
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <span className="block text-xs text-neutral-500 font-body">Nama Paket</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{booking.jadwal_nama || "-"}</p>
              </div>
              <div>
                <span className="block text-xs text-neutral-500 font-body">Tanggal Keberangkatan</span>
                <p className="text-sm font-semibold text-neutral-900 font-body">{formatTanggal(booking.berangkat_tanggal)}</p>
              </div>
            </div>
          </MetaBox>

          {/* Section: Daftar Jamaah (Pax) */}
          <MetaBox 
            title="Daftar Jamaah (Pax)"
            icon={<Users size={18} className="text-neutral-700" />}
            headerAction={
              <span className="text-xs text-neutral-500 font-body">
                Total: <span className="font-semibold text-neutral-800">{activePaxList.length} Jamaah</span>
                {infantActiveCount > 0 && ` (${infantActiveCount} infant)`}
                {paxList.length > activePaxList.length && ` · ${paxList.length - activePaxList.length} batal`}
              </span>
            }
          >
            {paxList.length === 0 ? (
              <p className="text-xs text-neutral-500 font-body">Tidak ada data pax.</p>
            ) : (
              <div className="overflow-x-auto -mx-5 -mb-4 sm:mx-0 sm:mb-0">
                <table className="w-full text-left text-sm font-body">
                  <thead className="bg-neutral-50 border-b border-neutral-200 text-xs text-neutral-500 uppercase tracking-wider font-semibold">
                    <tr>
                      <th className="py-3 px-4">Nama Jamaah</th>
                      <th className="py-3 px-3">Kamar</th>
                      <th className="py-3 px-3">Harga</th>
                      <th className="py-3 px-4 text-right">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-100">
                    {paxList.map((pax) => {
                      const isPic = Number(pax.jamaah_id) === Number(booking.pic_jamaah_id);
                      const isBatal = pax.pax_status === 'batal';

                      return (
                        <tr key={pax.id} className={isBatal ? "bg-neutral-50/70 opacity-60" : "hover:bg-neutral-50/50"}>
                          <td className="py-3 px-4">
                            <div>
                              <div className="flex items-center gap-2">
                                <button
                                  type="button"
                                  onClick={() => navigate(`/jamaah/${pax.jamaah_id}`)}
                                  className={`font-semibold hover:text-primary-600 hover:underline text-left transition-colors font-body ${
                                    isBatal ? "line-through text-neutral-500" : "text-neutral-900"
                                  }`}
                                  title="Lihat Profil Jamaah"
                                >
                                  {pax.nama_jamaah || `Jamaah #${pax.jamaah_id}`}
                                </button>
                                {isPic && (
                                  <span className="bg-primary-100 text-brand-dark text-[10px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wider">PIC</span>
                                )}
                              </div>
                              {pax.pax_type === 'infant' && (
                                <span className="block text-[11px] text-neutral-500 font-normal">
                                  Infant (Bayi &lt; 2 thn)
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="py-3 px-3">
                            <span className="font-medium text-neutral-800">
                              {pax.pax_type === 'infant' ? '-' : (pax.room_type || '-')}
                            </span>
                          </td>
                          <td className="py-3 px-3 font-semibold text-neutral-900">
                            {formatRupiah(pax.harga_pax)}
                          </td>
                          <td className="py-3 px-4 text-right">
                            {pax.pax_status === 'aktif' && (
                              <div className="flex items-center justify-end gap-1.5">
                                {pax.pax_type === 'reguler' && (
                                  <Button
                                    size="sm"
                                    variant="secondary"
                                    className="text-xs h-7 px-2.5 shadow-none border-neutral-200 text-neutral-700 bg-white"
                                    onClick={() => handleOpenChangeRoom(pax)}
                                  >
                                    Ganti Kamar
                                  </Button>
                                )}
                                <Button
                                  size="sm"
                                  variant="danger-light"
                                  className="text-xs h-7 px-2.5 shadow-none"
                                  title="Batalkan Pax"
                                  onClick={() => handleOpenCancelPax(pax)}
                                >
                                  Batalkan
                                </Button>
                              </div>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </MetaBox>

          {/* Section: Checklist Dokumen & Kesiapan Jamaah */}
          <MetaBox 
            title="Checklist Dokumen & Kesiapan Jamaah"
            icon={<Shield size={18} className="text-neutral-700" />}
          >
            {activePaxList.length === 0 ? (
              <p className="text-xs text-neutral-500 font-body">Tidak ada jamaah aktif.</p>
            ) : (
              <div className="overflow-x-auto -mx-5 -mb-4 sm:mx-0 sm:mb-0">
                <table className="w-full text-left text-sm font-body">
                  <thead className="bg-neutral-50 border-b border-neutral-200 text-xs text-neutral-500 uppercase tracking-wider font-semibold">
                    <tr>
                      <th className="py-3 px-4">Nama Jamaah</th>
                      <th className="py-3 px-3 text-center">Paspor</th>
                      <th className="py-3 px-3 text-center">Vaksin Meningitis</th>
                      <th className="py-3 px-3 text-center">Visa</th>
                      <th className="py-3 px-3 text-center">Siskopatuh</th>
                      <th className="py-3 px-3 text-center">Manasik</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-100">
                    {activePaxList.map((pax) => {
                      const isPic = Number(pax.jamaah_id) === Number(booking.pic_jamaah_id);
                      return (
                        <tr key={pax.id} className="hover:bg-neutral-50/50">
                          <td className="py-3 px-4">
                            <div>
                              <span className="font-semibold text-neutral-900 font-body block">
                                {pax.nama_jamaah || `Jamaah #${pax.jamaah_id}`}
                              </span>
                              {pax.pax_type === 'infant' && (
                                <span className="block text-[11px] text-neutral-500 font-normal">
                                  Infant (Bayi &lt; 2 thn)
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="py-3 px-3 text-center">
                            <div className="flex justify-center">
                              {pax.progress_paspor ? (
                                <span title="Paspor sudah diunggah" className="text-success-600 inline-flex items-center gap-1 text-xs font-medium">
                                  <CheckCircle size={18} />
                                </span>
                              ) : (
                                <span title="Paspor belum diunggah" className="text-neutral-400 inline-flex items-center gap-1 text-xs font-medium">
                                  <X size={18} />
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="py-3 px-3 text-center">
                            <div className="flex justify-center">
                              <input
                                type="checkbox"
                                className="w-4 h-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-600 focus:ring-offset-0 transition-colors cursor-pointer"
                                checked={Boolean(pax.progress_vaksin_meningitis)}
                                onChange={() => handleTogglePaxProgress(pax.id, 'vaksin_meningitis', Boolean(pax.progress_vaksin_meningitis))}
                                title="Tandai Vaksin Meningitis Selesai"
                              />
                            </div>
                          </td>
                          <td className="py-3 px-3 text-center">
                            <div className="flex justify-center">
                              <input
                                type="checkbox"
                                className="w-4 h-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-600 focus:ring-offset-0 transition-colors cursor-pointer"
                                checked={Boolean(pax.progress_visa)}
                                onChange={() => handleTogglePaxProgress(pax.id, 'visa', Boolean(pax.progress_visa))}
                                title="Tandai Visa Selesai"
                              />
                            </div>
                          </td>
                          <td className="py-3 px-3 text-center">
                            <div className="flex justify-center">
                              <input
                                type="checkbox"
                                className="w-4 h-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-600 focus:ring-offset-0 transition-colors cursor-pointer"
                                checked={Boolean(pax.progress_siskopatuh)}
                                onChange={() => handleTogglePaxProgress(pax.id, 'siskopatuh', Boolean(pax.progress_siskopatuh))}
                                title="Tandai Siskopatuh Selesai"
                              />
                            </div>
                          </td>
                          <td className="py-3 px-3 text-center">
                            {pax.pax_type === 'infant' ? (
                              <span className="text-xs text-neutral-400 font-semibold font-body">N/A</span>
                            ) : (
                              <div className="flex justify-center">
                                <input
                                  type="checkbox"
                                  className="w-4 h-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-600 focus:ring-offset-0 transition-colors cursor-pointer"
                                  checked={Boolean(pax.progress_manasik)}
                                  onChange={() => handleTogglePaxProgress(pax.id, 'manasik', Boolean(pax.progress_manasik))}
                                  title="Tandai Manasik Selesai"
                                />
                              </div>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </MetaBox>

          {/* Section: Distribusi Perlengkapan */}
          <MetaBox
            title="Distribusi Perlengkapan"
            icon={<Package size={18} className="text-neutral-700" />}
          >
            {perlengkapanError && <Alert variant="error" className="mb-4">{perlengkapanError}</Alert>}
            
            {activePaxList.length === 0 ? (
              <p className="text-xs text-neutral-500 font-body">Tidak ada jamaah aktif.</p>
            ) : (
              <div className="overflow-x-auto -mx-5 -mb-4 sm:mx-0 sm:mb-0">
                <table className="w-full text-left text-sm font-body">
                  <thead className="bg-neutral-50 border-b border-neutral-200 text-xs text-neutral-500 uppercase tracking-wider font-semibold">
                    <tr>
                      <th className="py-3 px-4">Nama Jamaah</th>
                      <th className="py-3 px-3">Status</th>
                      <th className="py-3 px-3">Tanggal</th>
                      <th className="py-3 px-3 text-right">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-100">
                    {activePaxList.map((pax) => {
                      return (
                        <tr key={pax.id} className="hover:bg-neutral-50/50">
                          <td className="py-3 px-4">
                            <div>
                              <span className="font-semibold text-neutral-900 font-body block">
                                {pax.nama_jamaah || `Jamaah #${pax.jamaah_id}`}
                              </span>
                              {pax.pax_type === 'infant' && (
                                <span className="block text-[11px] text-neutral-500 font-normal">
                                  Infant (Bayi &lt; 2 thn)
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="py-3 px-3">
                            {pax.pax_type === 'infant' ? (
                              <span className="text-xs text-neutral-400 font-semibold font-body">N/A</span>
                            ) : pax.perlengkapan_status === 'sudah_diberikan' ? (
                              <Badge variant="success">Selesai</Badge>
                            ) : (
                              <Badge variant="neutral" hideIcon={true}>Pending</Badge>
                            )}
                          </td>
                          <td className="py-3 px-3 text-xs text-neutral-600">
                            {(pax.pax_type !== 'infant' && pax.perlengkapan_tanggal) ? formatTanggal(pax.perlengkapan_tanggal) : '-'}
                          </td>
                          <td className="py-3 px-3 text-right">
                            {pax.pax_type !== 'infant' && (
                              <div className="flex justify-end">
                                {pax.perlengkapan_status === 'sudah_diberikan' ? (
                                  <Button 
                                    size="sm" 
                                    variant="ghost" 
                                    className="!py-1.5 !px-3 text-xs text-danger-600 hover:bg-danger-50"
                                    disabled={perlengkapanLoading}
                                    onClick={() => {
                                      setSelectedPaxForCancelPerlengkapan(pax);
                                      setIsBatalkanModalOpen(true);
                                    }}
                                  >
                                    Batal
                                  </Button>
                                ) : (
                                  <Button 
                                    size="sm" 
                                    variant="secondary" 
                                    className="!py-1.5 !px-3 text-xs"
                                    disabled={perlengkapanLoading}
                                    onClick={() => handleDistribusiPerlengkapan(pax.id)}
                                  >
                                    {perlengkapanLoading ? "Memproses..." : "Serahkan"}
                                  </Button>
                                )}
                              </div>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </MetaBox>

          {/* Section: Add-on & Biaya Tambahan */}
          <MetaBox
            title="Add-on & Biaya Tambahan"
            icon={<Tag size={18} className="text-neutral-700" />}
            headerAction={
              <Button size="sm" variant="secondary" onClick={() => setIsAddonModalOpen(true)} className="text-xs flex items-center gap-1">
                <Plus size={14} />
                <span>Tambah Add-on</span>
              </Button>
            }
          >
            {addons.length === 0 ? (
              <div className="py-6 flex flex-col items-center justify-center text-center bg-neutral-50/50 rounded-xl border border-neutral-100 border-dashed">
                <Tag size={24} className="text-neutral-300 mb-2" />
                <p className="text-sm font-medium text-neutral-700 font-body">Belum ada Add-on</p>
                <p className="text-xs text-neutral-500 font-body mt-0.5">Tambahkan biaya ekstra jika ada.</p>
              </div>
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

          </MetaBox>

          {/* Section: Diskon Khusus */}
          <MetaBox
            title="Program Diskon"
            icon={<Percent size={18} className="text-neutral-700" />}
            headerAction={
              <Button size="sm" variant="secondary" onClick={handleOpenDiskonModal} className="text-xs flex items-center gap-1">
                <Plus size={14} />
                <span>Tambah Diskon</span>
              </Button>
            }
          >
            {discounts.length === 0 ? (
              <div className="py-6 flex flex-col items-center justify-center text-center bg-neutral-50/50 rounded-xl border border-neutral-100 border-dashed">
                <Percent size={24} className="text-neutral-300 mb-2" />
                <p className="text-sm font-medium text-neutral-700 font-body">Belum ada Diskon</p>
                <p className="text-xs text-neutral-500 font-body mt-0.5">Tambahkan program diskon jika ada.</p>
              </div>
            ) : (
              <div className="divide-y divide-neutral-100">
                {discounts.map((d) => (
                  <div key={d.id} className="py-2.5 flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-neutral-900 font-body">{d.nama}</p>
                      <span className="text-xs text-neutral-500 font-body">{formatTanggal(d.created_at)}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-semibold text-success-600 font-body">- {formatRupiah(d.nominal)}</span>
                      <Button variant="ghost" size="sm" onClick={() => handleDeleteDiscount(d.id)} title="Hapus Diskon">
                        <Trash2 size={14} className="text-danger-600" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </MetaBox>

          {/* Section: Riwayat Pembayaran */}
          <MetaBox
            title="Riwayat Pembayaran"
            icon={<CreditCard size={18} className="text-neutral-700" />}
            headerAction={
              sisaTagihan > 0 && booking.status !== "batal" ? (
                <Button size="sm" variant="primary" onClick={() => setIsPaymentModalOpen(true)} className="text-xs flex items-center gap-1">
                  <Plus size={14} />
                  <span>Catat Pembayaran</span>
                </Button>
              ) : null
            }
          >
            {payments.length === 0 ? (
              <div className="py-8 flex flex-col items-center justify-center text-center bg-neutral-50/50 rounded-xl border border-neutral-100 border-dashed">
                <CreditCard size={32} className="text-neutral-300 mb-2" />
                <p className="text-sm font-medium text-neutral-700 font-body">Belum ada Pembayaran</p>
                <p className="text-xs text-neutral-500 font-body mt-0.5">Catat pembayaran dari jamaah di sini.</p>
              </div>
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
                      <button
                        type="button"
                        onClick={() => {
                          const fullUrl = p.bukti_url.startsWith('http') 
                            ? p.bukti_url 
                            : `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${p.bukti_url.startsWith('/') ? '' : '/'}${p.bukti_url}`;
                          setSelectedBuktiUrl(fullUrl);
                        }}
                        className="text-neutral-900 hover:text-primary-700 font-semibold inline-flex items-center gap-1.5 text-xs bg-neutral-100 hover:bg-neutral-200/80 px-2.5 py-1 rounded-lg border border-neutral-200/90 transition-colors cursor-pointer"
                        title="Lihat Bukti Pembayaran"
                      >
                        <FileText size={13} className="text-neutral-500" />
                        <span>Lihat Bukti</span>
                      </button>
                    ) : (
                      <span className="text-neutral-400 text-xs">-</span>
                    );
                  }
                  if (key === "status") {
                    const statusConfig = {
                      pending: ["warning", "Menunggu"],
                      confirmed: ["success", "Diterima"],
                      rejected: ["danger", "Ditolak"],
                    };
                    const [v, label] = statusConfig[p.status] || ["neutral", p.status];
                    return (
                      <div className="flex flex-col items-start gap-1">
                        <Badge variant={v} hideIcon={true}>{label}</Badge>
                        {p.status === "rejected" && p.rejection_reason && (
                          <span className="text-[11px] text-danger-600 italic font-body max-w-[180px] leading-tight" title={p.rejection_reason}>
                            Alasan: {p.rejection_reason}
                          </span>
                        )}
                      </div>
                    );
                  }
                  if (key === "aksi") {
                    if (p.status === "pending") {
                      return (
                        <div className="flex items-center gap-1">
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                              setSelectedPaymentForConfirm(p);
                              setConfirmError("");
                            }}
                            title="Konfirmasi"
                          >
                            <Check size={14} className="text-success-600" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                              setSelectedPaymentForReject(p);
                              setRejectionReason("");
                              setRejectError("");
                            }}
                            title="Tolak"
                          >
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
          </MetaBox>
        </div>

        {/* Kolom Kanan: Afiliasi Travel, Ringkasan Tagihan, Progress, & Perlengkapan (1 span) */}
        <div className="space-y-6">
          {/* Afiliasi Biro Travel (Khusus Super Admin / Master Dashboard) */}
          {showBrandColumn && (brand || booking.brand_id) && (
            <MetaBox
              title="Afiliasi Biro Travel"
              icon={<Building2 size={18} className="text-neutral-700" />}
            >
              <div className="pt-1">
                <BrandCell brand={brand} brandId={booking.brand_id} showText={true} />
              </div>
            </MetaBox>
          )}

          {/* Ringkasan Finansial Card */}
          <MetaBox
            title="Ringkasan Pembayaran"
            icon={<Receipt size={18} className="text-neutral-700" />}
          >
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
              {totalDiskon > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-neutral-500">Total Diskon:</span>
                  <span className="font-medium text-success-600">- {formatRupiah(totalDiskon)}</span>
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
          </MetaBox>

          {/* Progress Paket Card */}
          <MetaBox
            title="Progress Paket"
            icon={<Shield size={18} className="text-neutral-700" />}
            headerAction={
              booking.siap_berangkat ? (
                <Badge variant="success" className="px-2 py-0.5 text-xs font-semibold">
                  Siap Berangkat
                </Badge>
              ) : (
                <Badge variant="warning" className="px-2 py-0.5 text-xs font-semibold">
                  Persiapan Berjalan
                </Badge>
              )
            }
          >
            <div className="space-y-3">
              {/* 1. Tiket Maskapai Read-only */}
              <div className="flex items-center justify-between p-3 bg-neutral-50 rounded-xl border border-neutral-200/90">
                <div>
                  <span className="text-sm font-medium font-body text-neutral-800 block">Tiket Maskapai</span>
                  <span className="text-xs text-neutral-500 font-body block">(otomatis dari master paket)</span>
                </div>
                {booking.progress_tiket ? (
                  <div className="flex items-center gap-1.5 px-2.5 py-1 bg-success-50 text-success-700 rounded-lg text-xs font-semibold">
                    <CheckCircle size={14} />
                    <span>Selesai</span>
                  </div>
                ) : (
                  <div className="flex items-center gap-1.5 px-2.5 py-1 bg-neutral-100 text-neutral-500 rounded-lg text-xs font-semibold">
                    <Loader size={14} className="animate-spin" />
                    <span>Menunggu</span>
                  </div>
                )}
              </div>

              {/* 2. Hotel */}
              <div className="flex items-center justify-between p-3 bg-white rounded-xl border border-neutral-200/90">
                <span className="text-sm font-medium font-body text-neutral-800">Hotel</span>
                <Toggle
                  id="toggle-hotel-detail"
                  checked={Boolean(booking.progress_hotel)}
                  onChange={() => handleToggleProgress('hotel', Boolean(booking.progress_hotel))}
                />
              </div>

              {/* 3. Land Arrangement */}
              <div className="flex items-center justify-between p-3 bg-white rounded-xl border border-neutral-200/90">
                <span className="text-sm font-medium font-body text-neutral-800">Land Arrangement</span>
                <Toggle
                  id="toggle-land_arrangement-detail"
                  checked={Boolean(booking.progress_land_arrangement)}
                  onChange={() => handleToggleProgress('land_arrangement', Boolean(booking.progress_land_arrangement))}
                />
              </div>
            </div>
          </MetaBox>

          
        </div>
      </div>

      {/* Modal: Tambah Pembayaran */}
      <Modal
        isOpen={isPaymentModalOpen}
        onClose={() => setIsPaymentModalOpen(false)}
        title="Catat Pembayaran Baru"
        size="md"
      >
        <form onSubmit={handlePaymentSubmit} className="space-y-4 font-body">
          {paymentFormError && <Alert variant="error">{paymentFormError}</Alert>}

          {/* Billing Context Card */}
          <div className="p-3.5 bg-neutral-50 rounded-xl border border-neutral-200/80 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div className="space-y-0.5">
              <span className="text-[11px] uppercase tracking-wider text-neutral-500 font-semibold block">Sisa Tagihan</span>
              <div className={`text-base sm:text-lg font-bold font-heading ${sisaTagihan > 0 ? "text-neutral-900" : "text-success-600"}`}>
                {formatRupiah(sisaTagihan)}
              </div>
            </div>
            {sisaTagihan > 0 && (
              <Button
                type="button"
                size="sm"
                variant="secondary"
                onClick={() => setPaymentForm(prev => ({ ...prev, jumlah: sisaTagihan }))}
                className="text-xs shrink-0 self-start sm:self-auto"
              >
                Bayar Penuh Sisa
              </Button>
            )}
          </div>

          <CurrencyInput
            label="Jumlah Pembayaran (Rp)"
            value={paymentForm.jumlah}
            onChange={(val) => setPaymentForm(prev => ({ ...prev, jumlah: val }))}
            required
            placeholder="0"
            className="!mb-0"
          />

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <CustomDropdown
              label="Metode Pembayaran"
              value={paymentForm.metode}
              onChange={(val) => setPaymentForm(prev => ({ ...prev, metode: val }))}
              options={[
                { value: "transfer", label: "Transfer Bank" },
                { value: "cash", label: "Tunai / Cash" },
              ]}
              className="!mb-0"
            />
            <Input
              label="Tanggal Pembayaran"
              type="date"
              value={paymentForm.tanggal}
              onChange={(e) => setPaymentForm(prev => ({ ...prev, tanggal: e.target.value }))}
              required
              className="!mb-0"
            />
          </div>

          <FormField label="Bukti Pembayaran (Opsional)" helperText="Format JPG, PNG, atau PDF (maks. 5MB)">
            {paymentForm.bukti_file ? (
              <div className="flex items-center justify-between p-3 bg-neutral-50 border border-neutral-200 rounded-xl">
                <div className="flex items-center gap-2.5 overflow-hidden">
                  <div className="w-8 h-8 rounded-lg bg-primary-50 border border-primary-200 flex items-center justify-center text-primary-700 shrink-0">
                    <FileText size={16} />
                  </div>
                  <div className="truncate">
                    <p className="text-xs font-semibold text-neutral-800 truncate">{paymentForm.bukti_file.name}</p>
                    <p className="text-[11px] text-neutral-500">{(paymentForm.bukti_file.size / 1024).toFixed(1)} KB</p>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => setPaymentForm(prev => ({ ...prev, bukti_file: null }))}
                  className="text-neutral-400 hover:text-danger-600 p-1 rounded-lg hover:bg-white transition-colors cursor-pointer"
                  title="Hapus file"
                >
                  <X size={16} />
                </button>
              </div>
            ) : (
              <label className="flex flex-col items-center justify-center p-4 border-2 border-neutral-200 border-dashed rounded-xl bg-neutral-50/50 hover:bg-neutral-50 hover:border-primary-400 transition-colors cursor-pointer group">
                <Upload size={20} className="text-neutral-400 group-hover:text-primary-600 transition-colors mb-1.5" />
                <span className="text-xs font-medium text-neutral-700">
                  Unggah bukti transfer / kwitansi
                </span>
                <span className="text-[11px] text-neutral-400 mt-0.5">
                  Klik untuk memilih file
                </span>
                <input
                  type="file"
                  className="hidden"
                  accept="image/*,application/pdf"
                  onChange={(e) => setPaymentForm(prev => ({ ...prev, bukti_file: e.target.files?.[0] || null }))}
                />
              </label>
            )}
          </FormField>

          <div className="flex items-center justify-end gap-3 pt-4 border-t border-neutral-100">
            <Button 
              type="button" 
              variant="ghost" 
              onClick={() => setIsPaymentModalOpen(false)}
              disabled={paymentSubmitting}
            >
              Batal
            </Button>
            <Button 
              type="submit" 
              variant="primary" 
              isLoading={paymentSubmitting}
              disabled={paymentSubmitting}
            >
              {paymentSubmitting ? "Menyimpan..." : "Simpan Pembayaran"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Preview Bukti Pembayaran */}
      <Modal
        isOpen={Boolean(selectedBuktiUrl)}
        onClose={() => setSelectedBuktiUrl(null)}
        title="Bukti Pembayaran"
        size="lg"
        footer={
          <div className="flex items-center justify-between w-full">
            <a
              href={selectedBuktiUrl || "#"}
              target="_blank"
              rel="noreferrer"
              className="text-xs font-semibold text-primary-600 hover:text-primary-700 hover:underline inline-flex items-center gap-1.5"
            >
              <span>Buka di Tab Baru</span>
              <ExternalLink size={13} />
            </a>
            <Button type="button" variant="secondary" onClick={() => setSelectedBuktiUrl(null)}>
              Tutup
            </Button>
          </div>
        }
      >
        <div className="flex flex-col items-center justify-center p-3 bg-neutral-900/5 rounded-xl overflow-hidden min-h-[260px] max-h-[70vh]">
          {selectedBuktiUrl ? (
            selectedBuktiUrl.toLowerCase().endsWith('.pdf') ? (
              <iframe
                src={selectedBuktiUrl}
                title="Bukti Pembayaran PDF"
                className="w-full h-[60vh] rounded-lg border border-neutral-200"
              />
            ) : (
              <img
                src={selectedBuktiUrl}
                alt="Bukti Pembayaran"
                className="max-h-[60vh] w-auto max-w-full object-contain rounded-lg shadow-2xs"
              />
            )
          ) : null}
        </div>
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
                {discounts.map(d => (
                  <tr key={d.id}>
                    <td className="p-2.5 text-success-600">Diskon: {d.nama}</td>
                    <td className="p-2.5 text-right font-medium text-success-600">- {formatRupiah(d.nominal)}</td>
                  </tr>
                ))}
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
          Apakah Anda yakin ingin membatalkan distribusi perlengkapan untuk jamaah {selectedPaxForCancelPerlengkapan?.nama_jamaah || 'ini'}?
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

      {/* Modal: Ganti Tipe Kamar Pax */}
      <Modal
        isOpen={isChangeRoomModalOpen}
        onClose={() => setIsChangeRoomModalOpen(false)}
        title={`Ganti Tipe Kamar - ${selectedPaxForRoomChange?.nama_jamaah || ''}`}
      >
        {changeRoomError && <Alert variant="error" className="mb-4">{changeRoomError}</Alert>}
        <form onSubmit={handleChangeRoomSubmit} className="space-y-5 font-body min-h-[160px] flex flex-col justify-between">
          <div>
            <CustomDropdown
              label="Pilih Tipe Kamar Baru"
              value={newRoomType}
              onChange={(val) => {
                const actual = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
                setNewRoomType(actual);
              }}
              options={[
                { value: "Quad", label: "Quad (4 orang)" },
                { value: "Triple", label: "Triple (3 orang)" },
                { value: "Double", label: "Double (2 orang)" }
              ]}
              required
            />
          </div>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-100">
            <Button type="button" variant="ghost" onClick={() => setIsChangeRoomModalOpen(false)}>Batal</Button>
            <Button type="submit" variant="primary" disabled={changeRoomSubmitting} isLoading={changeRoomSubmitting}>
              {changeRoomSubmitting ? "Menyimpan..." : "Simpan Perubahan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Batalkan Pax */}
      <Modal
        isOpen={isCancelPaxModalOpen}
        onClose={() => setIsCancelPaxModalOpen(false)}
        title="Konfirmasi Pembatalan Pax"
      >
        <p className="text-neutral-600 font-body text-sm mb-6 leading-relaxed">
          Batalkan pax <strong>{selectedPaxForCancel?.nama_jamaah}</strong> dari booking ini? Kuota kursi yang terpakai akan otomatis dikembalikan jika booking masih dalam status kursi terkunci. Perubahan tagihan (jika ada) perlu disesuaikan manual.
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="ghost" onClick={() => setIsCancelPaxModalOpen(false)}>Batal</Button>
          <Button variant="danger" disabled={cancelPaxSubmitting} onClick={handleCancelPaxSubmit}>
            {cancelPaxSubmitting ? "Memproses..." : "Ya, Batalkan Pax"}
          </Button>
        </div>
      </Modal>

      {/* Modal: Tolak Pembayaran */}
      <Modal
        isOpen={Boolean(selectedPaymentForReject)}
        onClose={() => !rejectSubmitting && setSelectedPaymentForReject(null)}
        title="Tolak Pembayaran"
      >
        {rejectError && <Alert variant="error" className="mb-4">{rejectError}</Alert>}

        {selectedPaymentForReject && (
          <div className="p-3.5 bg-neutral-50 rounded-xl border border-neutral-200/80 mb-4 text-xs space-y-1 font-body">
            <div className="flex justify-between">
              <span className="text-neutral-500">Nominal:</span>
              <span className="font-bold text-neutral-900">{formatRupiah(selectedPaymentForReject.jumlah)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Pengirim:</span>
              <span className="font-medium text-neutral-800">{selectedPaymentForReject.sender_name || "-"}</span>
            </div>
          </div>
        )}

        <form onSubmit={handleRejectPaymentSubmit} className="space-y-4 font-body">
          <div>
            <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">
              Alasan Penolakan <span className="text-danger-500">*</span>
            </label>
            <textarea
              rows={3}
              className="w-full rounded-xl border border-neutral-200 bg-white px-3.5 py-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 resize-none disabled:bg-neutral-100"
              placeholder="Contoh: bukti transfer tidak sesuai jumlah"
              value={rejectionReason}
              onChange={(e) => setRejectionReason(e.target.value)}
              required
              disabled={rejectSubmitting}
            />
            <p className="text-[11px] text-neutral-400 mt-1">
              Minimal 3 karakter. Alasan ini akan tercatat dalam riwayat pembayaran.
            </p>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setSelectedPaymentForReject(null)}
              disabled={rejectSubmitting}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="danger"
              disabled={rejectSubmitting || rejectionReason.trim().length < 3}
              isLoading={rejectSubmitting}
            >
              {rejectSubmitting ? "Menolak..." : "Tolak Pembayaran"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Konfirmasi Pembayaran */}
      <Modal
        isOpen={Boolean(selectedPaymentForConfirm)}
        onClose={() => !confirmSubmitting && setSelectedPaymentForConfirm(null)}
        title="Konfirmasi Pembayaran"
      >
        {confirmError && <Alert variant="error" className="mb-4">{confirmError}</Alert>}

        {selectedPaymentForConfirm && (
          <div className="space-y-4 font-body">
            <div className="p-3.5 bg-neutral-50 rounded-xl border border-neutral-200/80 text-xs space-y-2 font-body">
              <div className="flex justify-between items-center">
                <span className="text-neutral-500 font-medium">Nominal:</span>
                <span className="font-bold text-neutral-900 text-sm">{formatRupiah(selectedPaymentForConfirm.jumlah)}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-neutral-500 font-medium">Pengirim:</span>
                <span className="font-medium text-neutral-800">{selectedPaymentForConfirm.sender_name || "-"}</span>
              </div>
              {selectedPaymentForConfirm.destination_bank_name && (
                <div className="flex justify-between items-center">
                  <span className="text-neutral-500 font-medium">Rekening Tujuan:</span>
                  <span className="font-medium text-neutral-800">
                    {selectedPaymentForConfirm.destination_bank_name} - {selectedPaymentForConfirm.destination_account_number} ({selectedPaymentForConfirm.destination_account_holder})
                  </span>
                </div>
              )}
            </div>

            <p className="text-neutral-600 text-sm leading-relaxed">
              Pastikan dana sudah benar-benar masuk ke rekening tujuan. Setelah dikonfirmasi, status booking akan disinkronkan dan alokasi kursi akan dikunci.
            </p>

            <div className="flex justify-end gap-3 pt-2">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setSelectedPaymentForConfirm(null)}
                disabled={confirmSubmitting}
              >
                Batal
              </Button>
              <Button
                type="button"
                variant="primary"
                onClick={handleConfirmPaymentSubmit}
                disabled={confirmSubmitting}
                isLoading={confirmSubmitting}
              >
                {confirmSubmitting ? "Mengonfirmasi..." : "Ya, Konfirmasi Pembayaran"}
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {/* Modal: Cancel Block Seat */}
      <Modal
        isOpen={cancelSeatBlockModal}
        onClose={() => !seatBlockLoading && setCancelSeatBlockModal(false)}
        title="Cancel Block Seat"
      >
        {cancelSeatBlockError && <Alert variant="error" className="mb-4">{cancelSeatBlockError}</Alert>}
        <p className="text-neutral-600 font-body text-sm mb-6 leading-relaxed">
          Apakah Anda yakin ingin membatalkan block seat untuk booking ini? Kuota kursi yang terpakai ({regularActiveCount || booking.seat_count || 1} kursi) akan otomatis dikembalikan ke jadwal paket.
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="ghost" onClick={() => setCancelSeatBlockModal(false)} disabled={seatBlockLoading}>
            Batal
          </Button>
          <Button variant="danger" disabled={seatBlockLoading} isLoading={seatBlockLoading} onClick={handleCancelSeatBlock}>
            {seatBlockLoading ? "Membatalkan..." : "Cancel Block Seat"}
          </Button>
        </div>
      </Modal>

      {/* Modal: Block Seat */}
      <Modal
        isOpen={blockSeatModal}
        onClose={() => !blockSeatLoading && setBlockSeatModal(false)}
        title="Block Seat"
      >
        {blockSeatError && <Alert variant="error" className="mb-4">{blockSeatError}</Alert>}
        <div className="space-y-4 font-body">
          <p className="text-neutral-600 text-sm leading-relaxed">
            Kunci kursi untuk booking ini? Kuota kursi ({regularActiveCount || booking.seat_count || 1} kursi) pada jadwal akan ditahan sampai dilepas secara manual.
          </p>
          <div className="p-3.5 bg-neutral-50 rounded-xl border border-neutral-200/80 text-xs space-y-1.5 font-body">
            <div className="flex justify-between items-center">
              <span className="text-neutral-500 font-medium">Jumlah Kursi:</span>
              <span className="font-semibold text-neutral-900">{regularActiveCount || booking.seat_count || 1} Kursi</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-neutral-500 font-medium">Status Kunci:</span>
              <span className="font-semibold text-neutral-900">Permanen (Manual)</span>
            </div>
            {booking.schedule && booking.schedule.seat_sisa !== undefined && (
              <div className="flex justify-between items-center">
                <span className="text-neutral-500 font-medium">Sisa Kuota Jadwal:</span>
                <span className="font-semibold text-neutral-900">{booking.schedule.seat_sisa} Kursi Tersedia</span>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <Button variant="ghost" onClick={() => setBlockSeatModal(false)} disabled={blockSeatLoading}>
              Batal
            </Button>
            <Button variant="primary" disabled={blockSeatLoading} isLoading={blockSeatLoading} onClick={handleBlockSeatSubmit}>
              {blockSeatLoading ? "Memproses..." : "Block Seat"}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default BookingDetailPage;

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
import { uploadMedia, openProtectedMedia } from "../api/media";
import { getSeatLockIcon, getPaymentStatusMeta } from "../utils/bookingStatus";
import PageHeader from "../components/ui/PageHeader";
import { Surface, Section, TableCard, Field, SummaryItem, PILL_STRONG, BrandBadge } from "../components/ui/DetailLayout";
import Badge from "../components/ui/Badge";
import ActionMenu from "../components/ui/ActionMenu";
import Button from "../components/ui/Button";
import { printInvoice } from "../utils/printInvoice";
import client from "../api/client";
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
import KreditCashbackPanel from "../components/KreditCashbackPanel";
import RefundPanel from "../components/RefundPanel";
import { CheckCircle, ExternalLink, FileText, Printer, Lock, Unlock, Upload, X, Check, Plus, Trash2, BedDouble, UserX } from "lucide-react";

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

const STATUS_STYLE = {
  draft: { label: 'Draft', chip: 'bg-neutral-50 text-neutral-600 border-neutral-200', dot: 'bg-neutral-400' },
  baru: { label: 'Baru', chip: 'bg-info-50 text-info-700 border-info-200', dot: 'bg-info-500' },
  dp: { label: 'DP', chip: 'bg-warning-50 text-warning-700 border-warning-200', dot: 'bg-warning-500' },
  lunas: { label: 'Lunas', chip: 'bg-success-50 text-success-700 border-success-200', dot: 'bg-success-500' },
  batal: { label: 'Batal', chip: 'bg-danger-50 text-danger-700 border-danger-200', dot: 'bg-danger-500' },
};


// Pil status utama: titik berwarna kecil + teks.
const StatusPill = ({ status, styleMap = STATUS_STYLE }) => {
  const s = styleMap[status] || { label: status, chip: 'bg-neutral-50 text-neutral-600 border-neutral-200', dot: 'bg-neutral-400' };
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium font-body border ${s.chip}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${s.dot}`} />
      {s.label}
    </span>
  );
};

// Status ringan: titik + teks tanpa bingkai.
const DotText = ({ tone, children }) => {
  const tones = {
    success: { text: 'text-success-700', dot: 'bg-success-500' },
    warning: { text: 'text-warning-700', dot: 'bg-warning-500' },
    neutral: { text: 'text-neutral-500', dot: 'bg-neutral-400' },
  };
  const t = tones[tone] || tones.neutral;
  return (
    <span className={`inline-flex items-center gap-1.5 text-[12px] font-body whitespace-nowrap ${t.text}`}>
      <span className={`w-2 h-2 rounded-full ${t.dot}`} />
      {children}
    </span>
  );
};

const PROGRESS_HEADER_MANUAL_ITEMS = [
  { key: 'tiket', label: 'Tiket PP' },
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
    bank_account_id: "",
    sender_name: "",
    received_by: "",
    bukti_file: null,
  });
  // Rekening tujuan milik brand booking (opsional; dipakai untuk pembayaran transfer).
  const [bankAccounts, setBankAccounts] = useState([]);
  const [paymentFormError, setPaymentFormError] = useState(null);
  const [paymentSubmitting, setPaymentSubmitting] = useState(false);
  const [selectedBuktiUrl, setSelectedBuktiUrl] = useState(null);


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
  const discounts = booking?.discounts || [];
  const totalDiskon = discounts.reduce((sum, d) => sum + (parseFloat(d.nominal) || 0), 0);
  const totalHarga = booking?.total_harga ? parseFloat(booking.total_harga) : Math.max(0, hargaDasar - totalDiskon);

  const totalPaid = payments
    .filter((p) => p.status === "confirmed")
    .reduce((sum, p) => sum + (parseFloat(p.jumlah) || 0), 0);
  const sisaTagihan = Math.max(0, totalHarga - totalPaid);

  // ─── Cetak / PDF Invoice ──────────────────────────────────────────────────────
  const handlePrintInvoice = async () => {
    const assetSrc = (url) => {
      if (!url) return "";
      if (url.startsWith("http")) return url;
      return `${import.meta.env.VITE_API_BASE_URL || "http://localhost:9090"}${url.startsWith("/") ? "" : "/"}${url}`;
    };
    // Rekening tujuan transfer milik brand (gagal diambil → invoice tetap tercetak tanpa bagian ini).
    let rekening = [];
    try {
      const res = await client.get("/api/admin/bank-accounts/", { params: { brand_id: booking.brand_id } });
      rekening = (Array.isArray(res.data) ? res.data : [])
        .filter((a) => a.is_active && String(a.brand_id) === String(booking.brand_id))
        .map((a) => ({
          bank: a.bank_name,
          nomor: a.account_number,
          pemilik: a.account_holder,
          catatan: a.instructions || "",
          logoUrl: assetSrc(a.logo_url),
        }));
    } catch {
      rekening = [];
    }
    if (rekening.length === 0 && brand?.bank_name && brand?.bank_account_number) {
      rekening = [{ bank: brand.bank_name, nomor: brand.bank_account_number, pemilik: brand.bank_account_holder || "", catatan: "", logoUrl: "" }];
    }
    // Tanggal batas pelunasan = keberangkatan dikurangi batas hari brand.
    const batasPelunasanTgl = (() => {
      const hari = Number(brand?.batas_pelunasan_hari) || 0;
      const d = booking.berangkat_tanggal ? new Date(booking.berangkat_tanggal) : null;
      if (!hari || !d || Number.isNaN(d.getTime())) return "";
      d.setDate(d.getDate() - hari);
      return formatTanggal(d);
    })();
    const aktif = (booking.pax || []).filter((p) => p.pax_status === "aktif");
    const pic = (booking.pax || []).find((p) => Number(p.jamaah_id) === Number(booking.pic_jamaah_id));
    printInvoice({
      kode: booking.id_booking || `Booking #${booking.id}`,
      brandName: brand?.name || "",
      accentColor: brand?.primary_color || "",
      logoUrl: brand?.logo_url ? assetSrc(brand.logo_url) : (brand?.icon_url ? assetSrc(brand.icon_url) : ""),
      kop: {
        alamat: brand?.address || "",
        // Alamat brand sering sudah memuat kota/provinsi; jangan ditulis dua kali.
        kota: brand?.city && !(brand?.address || "").toLowerCase().includes(brand.city.toLowerCase()) ? brand.city : "",
        provinsi: brand?.province && !(brand?.address || "").toLowerCase().includes(brand.province.toLowerCase()) ? brand.province : "",
        telepon: brand?.phone || "",
        whatsapp: brand?.whatsapp_number || "",
        email: brand?.email || "",
        website: brand?.domain || "",
        legalitas: [brand?.ppiu_number && `PPIU ${brand.ppiu_number}`, brand?.pihk_number && `PIHK ${brand.pihk_number}`].filter(Boolean).join("  •  "),
      },
      rekening,
      statusLabel: booking.status === "batal" ? "Dibatalkan" : sisaTagihan <= 0 && totalHarga > 0 ? "Lunas" : totalPaid > 0 ? "Pembayaran Sebagian" : "Belum Dibayar",
      statusTone: booking.status === "batal" ? "batal" : sisaTagihan <= 0 && totalHarga > 0 ? "lunas" : totalPaid > 0 ? "sebagian" : "belum",
      kontakHp: jamaah?.no_hp || "",
      sk: {
        minimalDp: Number(brand?.minimal_dp) > 0 ? formatRupiah(brand.minimal_dp) : "",
        pelunasan: Number(brand?.batas_pelunasan_hari) > 0
          ? `Pelunasan paling lambat H-${brand.batas_pelunasan_hari} sebelum keberangkatan${batasPelunasanTgl ? ` (${batasPelunasanTgl})` : ""}.`
          : "",
        pembatalan: String(brand?.ketentuan_pembatalan || "").split(/\r?\n/).map((t) => t.trim()).filter(Boolean),
      },
      paket: booking.jadwal_nama || "",
      berangkat: formatTanggal(booking.berangkat_tanggal),
      kontak: pic?.nama_jamaah || booking.nama_jamaah || "",
      tanggalBooking: formatTanggal(booking.created_at),
      pax: aktif.map((p) => ({
        nama: p.nama_jamaah || `Jamaah #${p.jamaah_id}`,
        tipe: p.pax_type === "infant" ? "Infant" : (p.room_type ? `Kamar ${p.room_type.charAt(0).toUpperCase()}${p.room_type.slice(1).toLowerCase()}` : ""),
        harga: formatRupiah(p.harga_pax),
      })),
      hargaDasar: formatRupiah(aktif.reduce((sum, p) => sum + (parseFloat(p.harga_pax) || 0), 0) || hargaDasar),
      diskon: discounts.map((d) => ({ nama: d.nama, nominal: formatRupiah(d.nominal) })),
      totalTagihan: formatRupiah(totalHarga),
      totalDibayar: formatRupiah(totalPaid),
      sisaTagihan: formatRupiah(sisaTagihan),
      pembayaran: payments
        .filter((p) => p.status === "confirmed")
        .map((p) => ({
          tanggal: formatTanggal(p.tanggal || p.created_at),
          metode: (p.metode || "transfer").toUpperCase(),
          jumlah: formatRupiah(p.jumlah),
        })),
      printedAt: formatTanggal(new Date()),
    });
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

  // Rekening tujuan: hanya rekening aktif milik brand booking. Gagal memuat (mis. tanpa izin)
  // tidak menghalangi pencatatan; pilihan rekening saja yang tidak tampil.
  useEffect(() => {
    if (!isPaymentModalOpen || !booking?.brand_id) return;
    let cancelled = false;
    client
      .get("/api/admin/bank-accounts/", { params: { brand_id: booking.brand_id, active: "true" } })
      .then((res) => {
        if (cancelled) return;
        const list = Array.isArray(res.data) ? res.data : [];
        setBankAccounts(list.filter((a) => String(a.brand_id) === String(booking.brand_id) && a.is_active !== false));
      })
      .catch(() => { if (!cancelled) setBankAccounts([]); });
    return () => { cancelled = true; };
  }, [isPaymentModalOpen, booking?.brand_id]);

  // ─── Payment Handlers ─────────────────────────────────────────────────────────
  const handlePaymentSubmit = async (e) => {
    e.preventDefault();
    const jumlahNum = parseFloat(paymentForm.jumlah) || 0;

    if (jumlahNum <= 0) {
      setPaymentFormError("Jumlah pembayaran harus lebih dari 0");
      return;
    }
    if (jumlahNum > sisaTagihan) {
      setPaymentFormError(`Maks. ${formatRupiah(sisaTagihan)}`);
      return;
    }
    if (paymentForm.metode === "transfer" && !paymentForm.bank_account_id) {
      setPaymentFormError("Rekening tujuan wajib dipilih untuk pembayaran transfer");
      return;
    }
    if (paymentForm.metode === "cash" && !paymentForm.received_by.trim()) {
      setPaymentFormError("Nama staff penerima wajib diisi untuk pembayaran tunai");
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
        bukti_url: buktiUrl,
        ...(paymentForm.metode === "transfer"
          ? {
              bank_account_id: Number(paymentForm.bank_account_id),
              ...(paymentForm.sender_name.trim() ? { sender_name: paymentForm.sender_name.trim() } : {}),
            }
          : { received_by: paymentForm.received_by.trim() }),
      });

      setIsPaymentModalOpen(false);
      setPaymentForm({
        jumlah: "",
        metode: "transfer",
        tanggal: new Date().toISOString().split('T')[0],
        bank_account_id: "",
        sender_name: "",
        received_by: "",
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

  const seatLockInfo = getSeatLockIcon(booking.status, booking.is_seat_blocked);

  const paxList = booking.pax || [];
  const activePaxList = paxList.filter((p) => p.pax_status === 'aktif');
  const regularActiveCount = activePaxList.filter((p) => p.pax_type !== 'infant').length;
  const infantActiveCount = activePaxList.filter((p) => p.pax_type === 'infant').length;

  return (
    <div className="w-full space-y-6">
      {error && <Alert variant="error">{error}</Alert>}

      {/* Header: judul + status, pil brand / seat, aksi */}
      <PageHeader
        title={booking.id_booking || `Booking #${booking.id}`}
        breadcrumb={false}
        subtitle={
          <span className="inline-flex flex-wrap items-center gap-2">
            {showBrandColumn && (brand || booking.brand_id) && (
              <BrandBadge name={brand?.name || `Brand #${booking.brand_id}`} color={brand?.primary_color} />
            )}
            {seatLockInfo && (
              <Badge
                variant="neutral"
                hideIcon
                className={booking.is_seat_blocked
                  ? "!bg-success-100 !border-success-300 !text-success-800"
                  : "!bg-warning-100 !border-warning-300 !text-warning-800"}
              >
                {seatLockInfo.label}
              </Badge>
            )}
            <span className="text-[11px] font-body text-neutral-500">Dibuat pada {formatTanggal(booking.created_at)}</span>
          </span>
        }
        onBack={() => navigate("/bookings")}
      >
        <div className="flex flex-wrap items-center gap-2">
        <Button variant="secondary" onClick={handlePrintInvoice}>
          <Printer size={16} strokeWidth={1.75} />
          <span>Cetak / PDF</span>
        </Button>
        {(() => {
          const headerMenu = [];
          if (["baru", "dp", "lunas"].includes(booking.status) && !booking.is_seat_blocked) {
            headerMenu.push({ label: "Block Seat", icon: Lock, onClick: handleOpenBlockSeat });
          }
          if (booking.is_seat_blocked) {
            headerMenu.push({
              label: "Cancel Block Seat",
              icon: Unlock,
              onClick: () => {
                setCancelSeatBlockError("");
                setCancelSeatBlockModal(true);
              },
            });
          }
          if (booking.status !== "batal") {
            headerMenu.push({ label: "Batalkan Booking", icon: X, danger: true, onClick: () => handleBookingStatus("batal") });
          }
          return headerMenu.length > 0 ? <ActionMenu label="Aksi" items={headerMenu} align="right" /> : null;
        })()}
        </div>
      </PageHeader>

      {/* Ringkasan: satu strip, bukan empat kartu */}
      <Surface className="grid grid-cols-2 lg:grid-cols-4 divide-x divide-neutral-200">
        <SummaryItem label="Jamaah aktif" value={activePaxList.length} hint={infantActiveCount > 0 ? `${infantActiveCount} infant` : undefined} />
        <SummaryItem label="Total tagihan" value={formatRupiah(totalHarga)} />
        <SummaryItem label="Sudah dibayar" value={formatRupiah(totalPaid)} />
        <SummaryItem label="Sisa tagihan" value={formatRupiah(sisaTagihan)} />
      </Surface>

      {/* Main Grid: Details */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* Kolom kiri: satu card, section dipisah garis tipis */}
        <div className="lg:col-span-2 space-y-6">
          {/* Section: Informasi Paket */}
          <Surface>
          <Section
            title="Informasi Paket"
          >
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4">
              <Field label="Nama Paket">{booking.jadwal_nama || "-"}</Field>
              <Field label="Tanggal Keberangkatan">{formatTanggal(booking.berangkat_tanggal)}</Field>
              {booking.pemesan && (
                <Field label="Pemesan (akun situs)">
                  {booking.pemesan.nama_lengkap}
                  {!booking.pemesan.ikut_berangkat && <span className="text-neutral-500"> · tidak ikut berangkat</span>}
                  <span className="block text-xs text-neutral-500">
                    {[booking.pemesan.no_hp, booking.pemesan.email].filter(Boolean).join(" · ")}
                  </span>
                </Field>
              )}
            </div>
          </Section>
          </Surface>

          {/* Section: Daftar Jamaah (Pax) */}
          <TableCard
            title="Daftar Jamaah"
            aside={
              <span className="text-xs text-neutral-500 font-body">
                Total: <span className="font-semibold text-neutral-800">{activePaxList.length} Jamaah</span>
                {infantActiveCount > 0 && ` (${infantActiveCount} infant)`}
                {paxList.length > activePaxList.length && ` · ${paxList.length - activePaxList.length} batal`}
              </span>
            }
          >
            {paxList.length === 0 ? (
              <p className="p-5 text-xs text-neutral-500 font-body">Tidak ada data pax.</p>
            ) : (
              <div className="overflow-x-auto">
              <Table
                columns={[
                  { header: "Nama Jamaah", key: "nama" },
                  { header: "Kamar", key: "kamar" },
                  { header: "Harga", key: "harga" },
                  { header: "Aksi", key: "aksi", align: "right" },
                ]}
                data={paxList}
                renderCell={(pax, key) => {
                  const isPic = Number(pax.jamaah_id) === Number(booking.pic_jamaah_id);
                  const isBatal = pax.pax_status === 'batal';

                  if (key === "nama") {
                    return (
                      <div className={`min-w-0 ${isBatal ? "opacity-60" : ""}`}>
                        <button
                          type="button"
                          onClick={() => navigate(`/jamaah/${pax.jamaah_id}`)}
                          className={`font-heading font-semibold hover:text-primary-600 hover:underline text-left transition-colors ${
                            isBatal ? "line-through text-neutral-500" : "text-neutral-900"
                          }`}
                          title="Lihat Profil Jamaah"
                        >
                          {pax.nama_jamaah || `Jamaah #${pax.jamaah_id}`}
                        </button>
                        {(isPic || pax.pax_type === 'infant' || isBatal) && (
                          <div className="text-[11px] font-body text-neutral-400">
                            {[
                              isPic ? "PIC" : null,
                              pax.pax_type === 'infant' ? "Infant (Bayi < 2 thn)" : null,
                              isBatal ? "Batal" : null,
                            ].filter(Boolean).join(" · ")}
                          </div>
                        )}
                      </div>
                    );
                  }

                  if (key === "kamar") {
                    return (
                      <span className={`font-medium text-neutral-800 ${isBatal ? "opacity-60" : ""}`}>
                        {pax.pax_type === 'infant' ? '-' : (pax.room_type || '-')}
                      </span>
                    );
                  }

                  if (key === "harga") {
                    return (
                      <span className={`font-semibold text-neutral-900 whitespace-nowrap ${isBatal ? "opacity-60" : ""}`}>
                        {formatRupiah(pax.harga_pax)}
                      </span>
                    );
                  }

                  if (key === "aksi") {
                    if (pax.pax_status !== 'aktif') return null;
                    const paxMenu = [];
                    if (pax.pax_type === 'reguler') {
                      paxMenu.push({ label: 'Ganti Kamar', icon: BedDouble, onClick: () => handleOpenChangeRoom(pax) });
                    }
                    paxMenu.push({ label: 'Batalkan Pax', icon: UserX, danger: true, tooltip: 'Batalkan Pax', onClick: () => handleOpenCancelPax(pax) });
                    return <ActionMenu items={paxMenu} align="right" />;
                  }

                  return pax[key];
                }}
              />
              </div>
            )}
          </TableCard>

          {/* Section: Checklist Dokumen & Kesiapan Jamaah */}
          <TableCard
            title="Kesiapan Dokumen & Operasional"
          >
            {activePaxList.length === 0 ? (
              <p className="p-5 text-xs text-neutral-500 font-body">Tidak ada jamaah aktif.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-[13px] font-body text-neutral-700 border-collapse">
                  <thead className="bg-neutral-50 text-neutral-800 font-heading text-[13px] font-semibold border-b border-neutral-200">
                    <tr className="border-b border-neutral-200/60">
                      <th className="py-2.5 px-4" rowSpan={2}>Nama Jamaah</th>
                      <th className="py-2 px-3 text-center border-x border-neutral-200" colSpan={2}>
                        Dokumen Jamaah
                      </th>
                      <th className="py-2 px-3 text-center" colSpan={3}>
                        Operasional Travel
                      </th>
                    </tr>
                    <tr className="text-[12px] text-neutral-500">
                      <th className="py-2 px-3 text-center border-l border-neutral-200">Paspor</th>
                      <th className="py-2 px-3 text-center border-r border-neutral-200">Vaksin</th>
                      <th className="py-2 px-3 text-center">Visa</th>
                      <th className="py-2 px-3 text-center">Siskopatuh</th>
                      <th className="py-2 px-3 text-center">Manasik</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-100">
                    {activePaxList.map((pax) => {
                      const isPic = Number(pax.jamaah_id) === Number(booking.pic_jamaah_id);
                      return (
                        <tr key={pax.id} className="hover:bg-neutral-50/70 transition-colors">
                          <td className="py-3 px-4">
                            <div>
                              <span className="font-heading font-semibold text-neutral-900 block">
                                {pax.nama_jamaah || `Jamaah #${pax.jamaah_id}`}
                              </span>
                              {pax.pax_type === 'infant' && (
                                <span className="block text-[11px] text-neutral-400 font-body">
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
          </TableCard>

          {/* Section: Distribusi Perlengkapan */}
          <TableCard
            title="Distribusi Perlengkapan"
          >
            {perlengkapanError && <Alert variant="error" className="m-5 mb-0">{perlengkapanError}</Alert>}

            {activePaxList.length === 0 ? (
              <p className="p-5 text-xs text-neutral-500 font-body">Tidak ada jamaah aktif.</p>
            ) : (
              <div className="overflow-x-auto">
              <Table
                columns={[
                  { header: "Nama Jamaah", key: "nama" },
                  { header: "Status", key: "status" },
                  { header: "Tanggal", key: "tanggal" },
                  { header: "Aksi", key: "aksi", align: "right" },
                ]}
                data={activePaxList}
                renderCell={(pax, key) => {
                  if (key === "nama") {
                    return (
                      <div className="min-w-0">
                        <div className="font-heading font-semibold text-neutral-900 leading-snug">
                          {pax.nama_jamaah || `Jamaah #${pax.jamaah_id}`}
                        </div>
                        {pax.pax_type === 'infant' && (
                          <div className="text-[11px] font-body text-neutral-400">
                            Infant (Bayi &lt; 2 thn)
                          </div>
                        )}
                      </div>
                    );
                  }

                  if (key === "status") {
                    if (pax.pax_type === 'infant') return <span className="text-xs text-neutral-400 font-body">N/A</span>;
                    return pax.perlengkapan_status === 'sudah_diberikan'
                      ? <DotText tone="success">Selesai</DotText>
                      : <DotText tone="neutral">Pending</DotText>;
                  }

                  if (key === "tanggal") {
                    return (
                      <span className="text-[13px] text-neutral-600 whitespace-nowrap">
                        {(pax.pax_type !== 'infant' && pax.perlengkapan_tanggal) ? formatTanggal(pax.perlengkapan_tanggal) : '-'}
                      </span>
                    );
                  }

                  if (key === "aksi") {
                    if (pax.pax_type === 'infant') return null;
                    return pax.perlengkapan_status === 'sudah_diberikan' ? (
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
                    );
                  }

                  return pax[key];
                }}
              />
              </div>
            )}
          </TableCard>

          {/* Section: Diskon Khusus */}
          <TableCard
            title="Program Diskon"
            aside={
              <Button size="sm" variant="secondary" onClick={handleOpenDiskonModal} className="text-xs flex items-center gap-1">
                <Plus size={14} strokeWidth={1.75} />
                <span>Tambah Diskon</span>
              </Button>
            }
          >
            {discounts.length === 0 ? (
              <p className="p-5 text-xs text-neutral-500 font-body">Belum ada diskon. Tambahkan program diskon jika ada.</p>
            ) : (
              <div className="divide-y divide-neutral-200/80">
                {discounts.map((d) => (
                  <div key={d.id} className="px-5 py-3 flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-neutral-900 font-body">{d.nama}</p>
                      <span className="text-[11px] text-neutral-400 font-body">{formatTanggal(d.created_at)}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-semibold text-success-600 font-body">- {formatRupiah(d.nominal)}</span>
                      <ActionMenu
                        align="right"
                        items={[{ label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus Diskon', onClick: () => handleDeleteDiscount(d.id) }]}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </TableCard>

          {/* Section: Kredit Cashback (B5) — hanya tampil bila ada pax dengan kredit */}
          <KreditCashbackPanel
            bookingId={id}
            refreshKey={`${booking.total_harga}-${discounts.length}-${booking.status}`}
            onApplied={fetchAll}
          />

          {/* Section: Pengembalian Dana — hanya tampil untuk booking batal yang sudah dibayar */}
          <RefundPanel booking={booking} onRecorded={fetchAll} />

          {/* Section: Riwayat Pembayaran */}
          <TableCard
            title="Riwayat Pembayaran"
            aside={
              sisaTagihan > 0 && booking.status !== "batal" ? (
                <Button size="sm" variant="primary" onClick={() => setIsPaymentModalOpen(true)} className="text-xs flex items-center gap-1">
                  <Plus size={14} strokeWidth={1.75} />
                  <span>Catat Pembayaran</span>
                </Button>
              ) : null
            }
          >
            {payments.length === 0 ? (
              <p className="p-5 text-xs text-neutral-500 font-body">Belum ada pembayaran. Catat pembayaran dari jamaah di sini.</p>
            ) : (
              <div className="overflow-x-auto">
              <Table
                columns={[
                  { header: "Tanggal", key: "tanggal" },
                  { header: "Metode", key: "metode" },
                  { header: "Jumlah", key: "jumlah" },
                  { header: "Bukti", key: "bukti_url" },
                  { header: "Status", key: "status" },
                  { header: "Aksi", key: "aksi", align: "right" },
                ]}
                data={payments}
                renderCell={(p, key) => {
                  if (key === "tanggal") return <span className="whitespace-nowrap">{formatTanggal(p.tanggal || p.created_at)}</span>;
                  if (key === "metode") return (p.metode || "transfer").toUpperCase();
                  if (key === "jumlah") return <span className="font-semibold text-neutral-900 whitespace-nowrap">{formatRupiah(p.jumlah)}</span>;
                  if (key === "bukti_url") {
                    return p.bukti_url ? (
                      <button
                        type="button"
                        onClick={() => openProtectedMedia(p.bukti_url, 'Bukti Pembayaran')}
                        className="text-neutral-700 hover:text-primary-700 font-medium inline-flex items-center gap-1.5 text-xs hover:underline transition-colors cursor-pointer whitespace-nowrap"
                        title="Lihat Bukti Pembayaran"
                      >
                        <FileText size={13} strokeWidth={1.75} className="text-neutral-400" />
                        <span>Lihat Bukti</span>
                      </button>
                    ) : (
                      <span className="text-neutral-400 text-xs">-</span>
                    );
                  }
                  if (key === "status") {
                    return (
                      <div className="flex flex-col items-start gap-1">
                        <Badge variant={getPaymentStatusMeta(p.status).variant} hideIcon className={getPaymentStatusMeta(p.status).className}>
                          {getPaymentStatusMeta(p.status).label}
                        </Badge>
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
                        <ActionMenu
                          align="right"
                          items={[
                            {
                              label: 'Konfirmasi',
                              icon: Check,
                              tooltip: 'Konfirmasi',
                              onClick: () => {
                                setSelectedPaymentForConfirm(p);
                                setConfirmError("");
                              },
                            },
                            {
                              label: 'Tolak',
                              icon: X,
                              danger: true,
                              tooltip: 'Tolak',
                              onClick: () => {
                                setSelectedPaymentForReject(p);
                                setRejectionReason("");
                                setRejectError("");
                              },
                            },
                          ]}
                        />
                      );
                    }
                    return <span className="text-xs text-neutral-400">-</span>;
                  }
                  return p[key];
                }}
              />
              </div>
            )}
          </TableCard>
        </div>

        {/* Kolom kanan: satu card, menempel saat halaman digulir */}
        <Surface className="lg:sticky lg:top-4 divide-y divide-neutral-200/80">
          {/* Ringkasan Finansial Card */}
          <Section
            title="Ringkasan Pembayaran"
          >
            <div className="space-y-2.5 font-body">
              <div className="flex justify-between text-sm">
                <span className="text-neutral-500">Harga Paket Dasar:</span>
                <span className="font-medium text-neutral-900">{formatRupiah(hargaDasar)}</span>
              </div>
              {totalDiskon > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-neutral-500">Total Diskon:</span>
                  <span className="font-medium text-success-600">- {formatRupiah(totalDiskon)}</span>
                </div>
              )}
              <div className="pt-3 border-t border-neutral-200 flex justify-between text-base font-bold">
                <span className="text-neutral-900">Total Tagihan:</span>
                <span className="text-neutral-900">{formatRupiah(totalHarga)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-neutral-500">Sudah Dibayar:</span>
                <span className="font-medium text-success-600">{formatRupiah(totalPaid)}</span>
              </div>
              <div className="pt-3 border-t border-neutral-200 flex justify-between items-center">
                <span className="text-sm font-semibold text-neutral-700">Sisa Tagihan:</span>
                <span className={`text-base font-bold ${sisaTagihan > 0 ? "text-danger-600" : "text-success-600"}`}>
                  {formatRupiah(sisaTagihan)}
                </span>
              </div>
            </div>
          </Section>

          {/* Progress Paket Card */}
          <Section
            title="Fasilitas Paket"
            aside={
              booking.siap_berangkat ? (
                <DotText tone="success">Siap Berangkat</DotText>
              ) : (
                <DotText tone="warning">Persiapan Berjalan</DotText>
              )
            }
          >
            <div className="divide-y divide-neutral-100 [&>div]:py-3 first:[&>div]:pt-0 [&>div:last-child]:pb-0">
              {/* 1. Tiket PP */}
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium font-body text-neutral-800">Tiket PP</span>
                <Toggle
                  id="toggle-tiket-detail"
                  checked={Boolean(booking.progress_tiket)}
                  onChange={() => handleToggleProgress('tiket', Boolean(booking.progress_tiket))}
                />
              </div>

              {/* 2. Hotel */}
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium font-body text-neutral-800">Hotel</span>
                <Toggle
                  id="toggle-hotel-detail"
                  checked={Boolean(booking.progress_hotel)}
                  onChange={() => handleToggleProgress('hotel', Boolean(booking.progress_hotel))}
                />
              </div>

              {/* 3. Land Arrangement */}
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium font-body text-neutral-800">Land Arrangement</span>
                <Toggle
                  id="toggle-land_arrangement-detail"
                  checked={Boolean(booking.progress_land_arrangement)}
                  onChange={() => handleToggleProgress('land_arrangement', Boolean(booking.progress_land_arrangement))}
                />
              </div>
            </div>
          </Section>

        </Surface>
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
                onClick={() => {
                  setPaymentForm(prev => ({ ...prev, jumlah: sisaTagihan }));
                  setPaymentFormError(null);
                }}
                className="text-xs shrink-0 self-start sm:self-auto"
              >
                Bayar Penuh
              </Button>
            )}
          </div>

          <CurrencyInput
            label="Jumlah Pembayaran"
            value={paymentForm.jumlah}
            onChange={(val) => {
              setPaymentForm(prev => ({ ...prev, jumlah: val }));
              if (paymentFormError) setPaymentFormError(null);
            }}
            required
            placeholder="0"
            className="!mb-0"
            error={sisaTagihan > 0 && parseFloat(paymentForm.jumlah) > sisaTagihan ? `Maks. ${formatRupiah(sisaTagihan)}` : undefined}
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

          {paymentForm.metode === "transfer" && (
            <>
              <CustomDropdown
                label="Rekening Tujuan"
                required
                value={paymentForm.bank_account_id}
                onChange={(val) => {
                  setPaymentForm(prev => ({ ...prev, bank_account_id: val?.target ? val.target.value : val }));
                  if (paymentFormError) setPaymentFormError(null);
                }}
                options={[
                  { value: "", label: bankAccounts.length ? "Pilih rekening tujuan" : "Belum ada rekening aktif untuk brand ini" },
                  ...bankAccounts.map((a) => ({
                    value: String(a.id),
                    label: `${a.bank_name} · ${a.account_number} a.n. ${a.account_holder}`,
                  })),
                ]}
                className="!mb-0"
              />
              <Input
                label="Nama Pengirim (Opsional)"
                value={paymentForm.sender_name}
                onChange={(e) => setPaymentForm(prev => ({ ...prev, sender_name: e.target.value }))}
                placeholder="Nama pemilik rekening pengirim"
                className="!mb-0"
              />
            </>
          )}

          {paymentForm.metode === "cash" && (
            <Input
              label="Nama Staff Penerima"
              required
              value={paymentForm.received_by}
              onChange={(e) => {
                setPaymentForm(prev => ({ ...prev, received_by: e.target.value }));
                if (paymentFormError) setPaymentFormError(null);
              }}
              placeholder="Nama staff yang menerima uang tunai"
              className="!mb-0"
            />
          )}

          <FormField label="Bukti Pembayaran (Opsional)">
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
              disabled={paymentSubmitting || (sisaTagihan > 0 && parseFloat(paymentForm.jumlah) > sisaTagihan) || !parseFloat(paymentForm.jumlah) || (paymentForm.metode === "transfer" && !paymentForm.bank_account_id) || (paymentForm.metode === "cash" && !paymentForm.received_by.trim())}
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
          Jika seat sempat terkunci, kuota seat akan dikembalikan ke paket.
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
          Batalkan pax <strong>{selectedPaxForCancel?.nama_jamaah}</strong> dari booking ini? Kuota seat yang terpakai akan otomatis dikembalikan jika booking masih dalam status seat terkunci.
        </p>
        {(booking?.status === "dp" || booking?.status === "lunas") && (
          <p className="text-warning-800 bg-warning-50 border border-warning-200 rounded-xl font-body text-sm px-3 py-2 mb-6 leading-relaxed">
            Booking ini sudah dibayar. Harga pax yang dibatalkan tetap masuk tagihan; pengembalian dana diproses manual oleh admin sesuai ketentuan travel.
          </p>
        )}
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
              Pastikan dana sudah benar-benar masuk ke rekening tujuan. Setelah dikonfirmasi, status booking akan disinkronkan dan alokasi seat akan dikunci.
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
          Apakah Anda yakin ingin membatalkan block seat untuk booking ini? Kuota seat yang terpakai ({regularActiveCount || booking.seat_count || 1} seat) akan otomatis dikembalikan ke jadwal paket.
        </p>
        {Number(booking?.total_dibayar) > 0 && (
          <p className="text-warning-800 bg-warning-50 border border-warning-200 rounded-xl font-body text-sm px-3 py-2 mb-6 leading-relaxed">
            Booking ini sudah dibayar. Seat yang dilepas bisa diambil booking lain; jamaah tidak lagi dijamin mendapat seat sampai seat dikunci ulang lewat tombol Block Seat.
          </p>
        )}
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
            Kunci seat untuk booking ini? Kuota seat ({regularActiveCount || booking.seat_count || 1} seat) pada jadwal akan ditahan sampai dilepas secara manual.
          </p>
          <div className="p-3.5 bg-neutral-50 rounded-xl border border-neutral-200/80 text-xs space-y-1.5 font-body">
            <div className="flex justify-between items-center">
              <span className="text-neutral-500 font-medium">Jumlah Seat:</span>
              <span className="font-semibold text-neutral-900">{regularActiveCount || booking.seat_count || 1} Seat</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-neutral-500 font-medium">Status Kunci:</span>
              <span className="font-semibold text-neutral-900">Permanen (Manual)</span>
            </div>
            {booking.schedule && booking.schedule.seat_sisa !== undefined && (
              <div className="flex justify-between items-center">
                <span className="text-neutral-500 font-medium">Sisa Kuota Jadwal:</span>
                <span className="font-semibold text-neutral-900">{booking.schedule.seat_sisa} Seat Tersedia</span>
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

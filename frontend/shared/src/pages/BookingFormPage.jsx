import React, { useState, useEffect, useMemo } from "react";
import { useNavigate, useParams, useLocation } from "react-router-dom";
import { 
  Plus, 
  Trash2, 
  Users, 
  AlertCircle, 
  Calendar, 
  CalendarDays,
  Plane,
  PlaneTakeoff,
  PlaneLanding,
  Building2,
  MapPin,
  ExternalLink,
  FileText,
  ChevronDown,
  UserCheck, 
  Receipt, 
  CheckCircle2,
  Check,
  UserPlus,
  Save,
  Loader2
} from "lucide-react";
import { 
  createBooking, 
  createDraftBooking, 
  updateDraftBooking, 
  deleteDraftBooking,
  finalizeBooking, 
  getBooking 
} from "../api/bookings";
import { listJamaah, updateJamaah, createJamaah } from "../api/jamaah";
import { listSchedulesAdmin, getScheduleAdmin } from "../api/schedules";
import { listBrands } from "../api/brands";
import { listAirports } from "../api/airports";
import PageHeader from "../components/ui/PageHeader";
import MetaBox from "../components/ui/MetaBox";
import CustomDropdown from "../components/ui/CustomDropdown";
import AutocompleteInput from "../components/ui/AutocompleteInput";
import FormField from "../components/ui/FormField";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import Badge from "../components/ui/Badge";
import Modal from "../components/ui/Modal";

const formatRupiah = (val) => {
  if (val === null || val === undefined || isNaN(Number(val))) return "-";
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0
  }).format(Number(val));
};

const formatIndoDate = (dateStr, options = { day: "numeric", month: "short", year: "numeric" }) => {
  if (!dateStr) return "-";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString("id-ID", options);
};

const formatDurationDays = (startDate, endDate) => {
  if (!startDate || !endDate) return null;
  const start = new Date(startDate);
  const end = new Date(endDate);
  if (isNaN(start.getTime()) || isNaN(end.getTime())) return null;
  const diffTime = end.getTime() - start.getTime();
  const diffDays = Math.round(diffTime / (1000 * 60 * 60 * 24)) + 1;
  return diffDays > 0 ? `${diffDays} Hari` : null;
};

export const calculateAgeDetails = (tanggalLahir, berangkatTanggal) => {
  if (!tanggalLahir || !berangkatTanggal) return null;
  const birthStr = String(tanggalLahir).trim().slice(0, 10);
  const depStr = String(berangkatTanggal).trim().slice(0, 10);
  const birth = new Date(birthStr);
  const dep = new Date(depStr);
  if (isNaN(birth.getTime()) || isNaN(dep.getTime())) return null;

  let years = dep.getFullYear() - birth.getFullYear();
  let months = dep.getMonth() - birth.getMonth();
  let days = dep.getDate() - birth.getDate();

  if (days < 0) {
    months -= 1;
  }
  if (months < 0) {
    years -= 1;
    months += 12;
  }

  const twoYearsAfterBirth = new Date(birth);
  twoYearsAfterBirth.setFullYear(twoYearsAfterBirth.getFullYear() + 2);
  const isInfant = dep < twoYearsAfterBirth;

  let ageText = "";
  if (years > 0) {
    ageText = `${years} thn ${months} bln`;
  } else {
    ageText = `${Math.max(0, months)} bln`;
  }

  return { isInfant, ageText, years, months };
};

export const getContrastTextColor = (hexColor) => {
  if (!hexColor || typeof hexColor !== 'string') return '#FFFFFF';
  const cleanHex = hexColor.replace('#', '').trim();
  if (cleanHex.length !== 6 && cleanHex.length !== 3) return '#FFFFFF';
  let r, g, b;
  if (cleanHex.length === 3) {
    r = parseInt(cleanHex[0] + cleanHex[0], 16);
    g = parseInt(cleanHex[1] + cleanHex[1], 16);
    b = parseInt(cleanHex[2] + cleanHex[2], 16);
  } else {
    r = parseInt(cleanHex.substring(0, 2), 16);
    g = parseInt(cleanHex.substring(2, 4), 16);
    b = parseInt(cleanHex.substring(4, 6), 16);
  }
  const yiq = (r * 299 + g * 587 + b * 114) / 1000;
  return yiq >= 150 ? '#14171A' : '#FFFFFF';
};

export const parseTransitDetails = (rawStr = '', airportMap = {}) => {
  if (!rawStr || typeof rawStr !== 'string') {
    return { isDirect: true, berangkat: null, pulang: null, badgeText: null };
  }

  const str = rawStr.trim();
  if (!str) return { isDirect: true, berangkat: null, pulang: null, badgeText: null };

  let berangkatPart = null;
  let pulangPart = null;

  if (str.toLowerCase().includes('berangkat:') || str.toLowerCase().includes('pulang:')) {
    const parts = str.split('|').map(p => p.trim());
    parts.forEach(p => {
      const lower = p.toLowerCase();
      if (lower.startsWith('berangkat:')) {
        berangkatPart = p.substring(p.indexOf(':') + 1).trim();
      } else if (lower.startsWith('pulang:')) {
        pulangPart = p.substring(p.indexOf(':') + 1).trim();
      }
    });
  } else {
    berangkatPart = str;
    pulangPart = str;
  }

  const formatItem = (itemStr) => {
    if (!itemStr) return null;
    const [rawCode, ...rest] = itemStr.split(',');
    const code = rawCode.trim().toUpperCase();
    const duration = rest.join(',').trim();
    
    // Check if code is airport code (e.g. "KUL") and lookup city from airportMap
    let formattedAirport = code;
    if (!code.includes('-') && airportMap[code]) {
      formattedAirport = airportMap[code];
    }
    return duration ? `${formattedAirport} (${duration})` : formattedAirport;
  };

  const bFormatted = formatItem(berangkatPart);
  const pFormatted = formatItem(pulangPart);

  const codes = [];
  if (berangkatPart) {
    const c = berangkatPart.split(',')[0].split('-')[0].trim().toUpperCase();
    if (c && !codes.includes(c)) codes.push(c);
  }
  if (pulangPart) {
    const c = pulangPart.split(',')[0].split('-')[0].trim().toUpperCase();
    if (c && !codes.includes(c)) codes.push(c);
  }

  const badgeText = codes.length > 0 ? `Transit (${codes.join(', ')})` : 'Transit';

  return {
    isDirect: false,
    berangkat: bFormatted,
    pulang: pFormatted,
    badgeText
  };
};

export const BookingFormPage = ({ showBrandColumn = false }) => {
  const { id } = useParams();
  const location = useLocation();
  const isEditDraft = Boolean(id);
  const navigate = useNavigate();

  const [loadingData, setLoadingData] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [draftSubmitting, setDraftSubmitting] = useState(false);
  const [draftSuccess, setDraftSuccess] = useState(location.state?.draftSuccess || null);
  const [serverError, setServerError] = useState(null);
  const [clientErrors, setClientErrors] = useState({});

  const [jamaahList, setJamaahList] = useState([]);
  const [scheduleList, setScheduleList] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [airportsList, setAirportsList] = useState([]);

  // State pemilihan: Brand dulu (jika Master Dashboard), Bulan, lalu Paket
  const [selectedBrandId, setSelectedBrandId] = useState("");
  const [selectedMonth, setSelectedMonth] = useState("");
  const [selectedScheduleId, setSelectedScheduleId] = useState("");
  const [picJamaahId, setPicJamaahId] = useState("");
  const [paxList, setPaxList] = useState([
    { id: 1, jamaah_id: "", is_infant: false, room_type: "" }
  ]);

  // State expand/collapse detail jadwal paket untuk keperluan front desk
  const [showScheduleDetails, setShowScheduleDetails] = useState(false);

  // Modal konfirmasi cascade delete saat menghapus pax reguler terakhir yang memiliki infant
  const [cascadeDeleteTarget, setCascadeDeleteTarget] = useState(null);

  // Modal konfirmasi hapus draft booking
  const [deleteDraftModalOpen, setDeleteDraftModalOpen] = useState(false);
  const [deleteDraftSubmitting, setDeleteDraftSubmitting] = useState(false);

  // === Quick Create Jamaah Modal ===
  const [quickCreateOpen, setQuickCreateOpen] = useState(false);
  const [quickCreatePaxId, setQuickCreatePaxId] = useState(null); // pax.id yang akan di-fill
  const [quickCreateForm, setQuickCreateForm] = useState({ nama_lengkap: '', jenis_kelamin: '', nik: '', tanggal_lahir: '', no_hp: '' });
  const [quickCreateError, setQuickCreateError] = useState(null);
  const [quickCreateLoading, setQuickCreateLoading] = useState(false);

  // === Inline DOB Quick Save ===
  // Map: jamaah_id -> { value: 'YYYY-MM-DD', saving: bool, error: str }
  const [inlineDobMap, setInlineDobMap] = useState({});

  useEffect(() => {
    const fetchOptions = async () => {
      try {
        setLoadingData(true);
        const promises = [
          listJamaah({ status: "aktif" }),
          listSchedulesAdmin({ status: "published" }),
          listAirports()
        ];
        if (showBrandColumn) {
          promises.push(listBrands());
        }
        if (isEditDraft) {
          promises.push(getBooking(id));
        }

        const results = await Promise.all(promises);
        const jamaahRes = results[0];
        const schedRes = results[1];
        const airportsRes = results[2];
        let brandsRes = null;
        let bookingRes = null;

        if (showBrandColumn && isEditDraft) {
          brandsRes = results[3];
          bookingRes = results[4];
        } else if (showBrandColumn) {
          brandsRes = results[3];
        } else if (isEditDraft) {
          bookingRes = results[3];
        }

        setJamaahList(jamaahRes || []);
        const published = (schedRes || []).filter((s) => s.status === "published");
        setScheduleList(published);
        setAirportsList(airportsRes || []);

        if (brandsRes) {
          const brandData = brandsRes || [];
          setBrands(brandData);
          const bMap = {};
          brandData.forEach((b) => {
            bMap[b.id] = b;
          });
          setBrandsMap(bMap);
        }

        if (isEditDraft && bookingRes) {
          if (bookingRes.status !== "draft") {
            // Bukan draft, redirect ke detail
            navigate(`/bookings/${id}`, { replace: true });
            return;
          }
          if (showBrandColumn && bookingRes.brand_id) {
            setSelectedBrandId(String(bookingRes.brand_id));
          }
          if (bookingRes.schedule_id) {
            setSelectedScheduleId(String(bookingRes.schedule_id));
            const matched = published.find((s) => Number(s.id) === Number(bookingRes.schedule_id));
            if (matched && matched.berangkat_tanggal) {
              setSelectedMonth(matched.berangkat_tanggal.substring(0, 7));
            }
          }
          if (bookingRes.pic_jamaah_id) {
            setPicJamaahId(String(bookingRes.pic_jamaah_id));
          }
          if (bookingRes.pax && bookingRes.pax.length > 0) {
            setPaxList(
              bookingRes.pax.map((p, idx) => ({
                id: idx + 1,
                jamaah_id: p.jamaah_id ? String(p.jamaah_id) : "",
                is_infant: p.pax_type === "infant",
                room_type: p.room_type || ""
              }))
            );
          } else {
            setPaxList([{ id: 1, jamaah_id: "", is_infant: false, room_type: "" }]);
          }
        }
      } catch (err) {
        setServerError(err.response?.data?.error || "Gagal memuat opsi jamaah/paket/draft.");
      } finally {
        setLoadingData(false);
      }
    };
    fetchOptions();
  }, [showBrandColumn, id, isEditDraft, navigate]);

  // Sync draftSuccess message from navigation state
  useEffect(() => {
    if (location.state?.draftSuccess) {
      setDraftSuccess(location.state.draftSuccess);
      window.history.replaceState({}, document.title);
    }
  }, [location.state]);

  // Daftar schedule untuk brand terpilih (atau semua published jika Travel Dashboard)
  const brandScheduleList = useMemo(() => {
    if (!showBrandColumn) return scheduleList;
    if (!selectedBrandId) return [];
    return scheduleList.filter((s) => Number(s.brand_id) === Number(selectedBrandId));
  }, [scheduleList, showBrandColumn, selectedBrandId]);

  // Daftar bulan keberangkatan yang tersedia dari daftar paket brand terpilih
  const availableMonths = useMemo(() => {
    const monthsMap = new Map();
    brandScheduleList.forEach((s) => {
      if (s.berangkat_tanggal) {
        const ym = s.berangkat_tanggal.substring(0, 7); // "YYYY-MM"
        if (!monthsMap.has(ym)) {
          const [year, month] = ym.split("-");
          const d = new Date(Number(year), Number(month) - 1, 1);
          const label = !isNaN(d.getTime())
            ? d.toLocaleDateString("id-ID", { month: "long", year: "numeric" })
            : ym;
          monthsMap.set(ym, label.charAt(0).toUpperCase() + label.slice(1));
        }
      }
    });

    return Array.from(monthsMap.entries())
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([value, label]) => ({ value, label }));
  }, [brandScheduleList]);

  // Filter paket berdasarkan Brand dan Bulan Keberangkatan
  const filteredScheduleList = useMemo(() => {
    let list = brandScheduleList;
    if (selectedMonth) {
      list = list.filter((s) => s.berangkat_tanggal && s.berangkat_tanggal.startsWith(selectedMonth));
    }
    return list;
  }, [brandScheduleList, selectedMonth]);

  // Filter jamaah berdasarkan Brand yang dipilih
  const filteredJamaahList = useMemo(() => {
    if (!showBrandColumn) return jamaahList;
    if (!selectedBrandId) return jamaahList;
    return jamaahList.filter((j) => !j.brand_id || Number(j.brand_id) === Number(selectedBrandId));
  }, [jamaahList, showBrandColumn, selectedBrandId]);

  // Daftar opsi jamaah per baris: mengecualikan jamaah yang sudah dipilih di baris lain,
  // dan khusus Pax #1 (isFirstRow) mengecualikan jamaah yang sudah diketahui infant (<2 tahun pada tanggal keberangkatan)
  const getAvailableJamaahOptions = (currentPaxId, isFirstRow = false) => {
    const selectedInOtherRows = new Set(
      paxList
        .filter((p) => p.id !== currentPaxId && Boolean(p.jamaah_id))
        .map((p) => String(p.jamaah_id))
    );

    return filteredJamaahList
      .filter((j) => {
        // 1. Cek duplikasi di baris lain
        if (selectedInOtherRows.has(String(j.id))) return false;

        // 2. Khusus Pax #1: sembunyikan jamaah yang sudah diketahui infant (<2 tahun) jika tanggal keberangkatan ada
        if (isFirstRow && selectedSchedule?.berangkat_tanggal && j.tanggal_lahir) {
          const ageInfo = calculateAgeDetails(j.tanggal_lahir, selectedSchedule.berangkat_tanggal);
          if (ageInfo?.isInfant) {
            return false;
          }
        }

        return true;
      })
      .map((j) => ({
        value: j.id,
        label: `${j.nama_lengkap} ${j.nik ? `(${j.nik})` : ""}`
      }));
  };

  // State cache rincian lengkap schedule (hotel, maskapai, bandara, dsb)
  const [scheduleDetailMap, setScheduleDetailMap] = useState({});

  // Fetch rincian lengkap schedule saat schedule dipilih
  useEffect(() => {
    if (!selectedScheduleId) return;
    if (scheduleDetailMap[selectedScheduleId]) return;

    let isMounted = true;
    getScheduleAdmin(selectedScheduleId)
      .then((detail) => {
        if (isMounted && detail) {
          setScheduleDetailMap((prev) => ({ ...prev, [selectedScheduleId]: detail }));
        }
      })
      .catch((err) => {
        console.error("Gagal memuat rincian paket:", err);
      });

    return () => {
      isMounted = false;
    };
  }, [selectedScheduleId, scheduleDetailMap]);

  // Cari objek schedule terpilih (menggunakan detail lengkap jika sudah ter-fetch, atau fallback ke list item)
  const selectedSchedule = useMemo(() => {
    if (!selectedScheduleId) return null;
    const basic = scheduleList.find((s) => String(s.id) === String(selectedScheduleId));
    const detailed = scheduleDetailMap[selectedScheduleId];
    if (detailed && basic) {
      return { ...basic, ...detailed };
    }
    return detailed || basic || null;
  }, [scheduleList, selectedScheduleId, scheduleDetailMap]);

  // Lookup map jamaah untuk label cepat
  const jamaahMap = useMemo(() => {
    const map = {};
    jamaahList.forEach((j) => {
      map[j.id] = j;
    });
    return map;
  }, [jamaahList]);

  // Lookup map bandara (kode -> kode - kota)
  const airportMap = useMemo(() => {
    const map = {};
    airportsList.forEach((a) => {
      if (a.code) {
        map[a.code.toUpperCase()] = a.city ? `${a.code} - ${a.city}` : a.code;
      }
    });
    return map;
  }, [airportsList]);

  // Informasi transit terurai untuk penerbangan berangkat dan pulang
  const transitInfo = useMemo(() => {
    if (!selectedSchedule || selectedSchedule.is_direct_flight) {
      return { isDirect: true, berangkat: null, pulang: null, badgeText: "Direct" };
    }
    return parseTransitDetails(selectedSchedule.transit_bandara, airportMap);
  }, [selectedSchedule, airportMap]);

  // Handler saat pemilihan Brand berubah
  const handleBrandChange = (val) => {
    const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
    setSelectedBrandId(actualVal);
    setSelectedMonth("");
    // Reset paket, PIC, dan daftar pax jika berganti travel
    setSelectedScheduleId("");
    setPicJamaahId("");
    setPaxList([{ id: 1, jamaah_id: "", is_infant: false, room_type: "" }]);
    setClientErrors({});
  };

  // Handler saat filter Bulan Keberangkatan berubah
  const handleMonthChange = (val) => {
    const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
    setSelectedMonth(actualVal);
    // Jika schedule terpilih saat ini tidak cocok dengan bulan yang baru dipilih, reset schedule
    if (selectedScheduleId && actualVal) {
      const currentSched = scheduleList.find((s) => String(s.id) === String(selectedScheduleId));
      if (currentSched && (!currentSched.berangkat_tanggal || !currentSched.berangkat_tanggal.startsWith(actualVal))) {
        setSelectedScheduleId("");
        setPicJamaahId("");
        setPaxList((prev) =>
          prev.map((p) => ({
            ...p,
            is_infant: false,
            room_type: ""
          }))
        );
      }
    }
  };

  // Daftar opsi PIC: HANYA jamaah dari pax REGULER yang sudah terisi jamaah_id-nya
  const picOptions = useMemo(() => {
    const regularPax = paxList.filter((p) => !p.is_infant && Boolean(p.jamaah_id));
    const uniqueIds = Array.from(new Set(regularPax.map((p) => p.jamaah_id)));
    return uniqueIds.map((id) => {
      const j = jamaahMap[id];
      return {
        value: id,
        label: j ? `${j.nama_lengkap} ${j.nik ? `(${j.nik})` : ""}` : `Jamaah #${id}`
      };
    });
  }, [paxList, jamaahMap]);

  // Auto-sync PIC: pertahankan jika masih valid (ada di regular pax), re-kalkulasi ke pax reguler pertama jika invalid
  useEffect(() => {
    const isCurrentPicValid = picOptions.some((opt) => String(opt.value) === String(picJamaahId));
    if (isCurrentPicValid) {
      return;
    }

    const firstRegularPax = paxList.find((p) => !p.is_infant && Boolean(p.jamaah_id));
    if (firstRegularPax) {
      setPicJamaahId(String(firstRegularPax.jamaah_id));
    } else {
      setPicJamaahId("");
    }
  }, [picOptions, paxList, picJamaahId]);

  // Hitung harga per baris pax
  const calculatePaxPrice = (pax) => {
    if (!selectedSchedule) return null;
    const j = jamaahMap[pax.jamaah_id];
    // Kondisi A: jika jamaah belum dipilih atau tanggal_lahir kosong, tidak ikut estimasi harga
    if (!pax.jamaah_id || !j?.tanggal_lahir) return null;

    if (pax.is_infant) {
      return selectedSchedule.harga_infant != null ? Number(selectedSchedule.harga_infant) : 0;
    }
    if (!pax.room_type) return null;
    const key = `harga_${pax.room_type.toLowerCase()}`;
    return selectedSchedule[key] != null ? Number(selectedSchedule[key]) : 0;
  };

  // Ringkasan Kuota & Total Harga
  const regularPaxCount = useMemo(() => {
    return paxList.filter((p) => !p.is_infant).length;
  }, [paxList]);

  const infantPaxCount = useMemo(() => {
    return paxList.filter((p) => p.is_infant).length;
  }, [paxList]);

  const seatSisa = selectedSchedule ? Number(selectedSchedule.seat_sisa) : 0;
  const isOverQuota = selectedSchedule ? regularPaxCount > seatSisa : false;

  const totalHarga = useMemo(() => {
    if (!selectedSchedule) return 0;
    return paxList.reduce((sum, p) => {
      const price = calculatePaxPrice(p);
      return sum + (price || 0);
    }, 0);
  }, [paxList, selectedSchedule]);

  // Validasi form kelengkapan untuk disable tombol submit
  const hasEmptyPaxJamaah = paxList.some((p) => !p.jamaah_id);
  const hasEmptyPaxRoom = paxList.some((p) => !p.is_infant && !p.room_type);
  const isSubmitDisabled =
    submitting ||
    isOverQuota ||
    !selectedScheduleId ||
    !picJamaahId ||
    hasEmptyPaxJamaah ||
    hasEmptyPaxRoom ||
    regularPaxCount === 0;

  // Repeater handlers
  const handleAddPax = () => {
    const nextId = paxList.length > 0 ? Math.max(...paxList.map((p) => p.id)) + 1 : 1;
    setPaxList((prev) => [
      ...prev,
      { id: nextId, jamaah_id: "", is_infant: false, room_type: "" }
    ]);
  };

  const handleRemovePax = (idToRemove) => {
    const targetIndex = paxList.findIndex((p) => p.id === idToRemove);
    if (targetIndex === -1) return;

    const targetPax = paxList[targetIndex];

    // Jika hanya ada 1 baris di daftar, reset baris tersebut ke kosong
    if (paxList.length === 1) {
      setPaxList([{ id: 1, jamaah_id: "", is_infant: false, room_type: "" }]);
      setClientErrors({});
      return;
    }

    // Jika baris yang dihapus adalah index 0
    if (targetIndex === 0) {
      // Cari reguler lain yang paling awal di index > 0
      const nextRegularIndex = paxList.findIndex((p, idx) => idx > 0 && !p.is_infant);

      if (nextRegularIndex !== -1) {
        // Ditemukan reguler pengganti: promosikan ke index 0, pertahankan urutan relatif sisanya
        const promotedPax = paxList[nextRegularIndex];
        const otherPax = paxList.filter((p, idx) => idx !== 0 && idx !== nextRegularIndex);
        setPaxList([promotedPax, ...otherPax]);
        cleanClientErrorsForId(idToRemove);
        return;
      }

      // Tidak ada reguler lain sama sekali (semua sisa adalah infant)
      const remainingInfants = paxList.filter((p, idx) => idx > 0 && p.is_infant);
      setCascadeDeleteTarget({
        targetPax,
        infantCount: remainingInfants.length
      });
      return;
    }

    // Jika baris yang dihapus adalah index > 0 (bukan index 0): hapus normal
    setPaxList((prev) => prev.filter((p) => p.id !== idToRemove));
    cleanClientErrorsForId(idToRemove);
  };

  const cleanClientErrorsForId = (idToRemove) => {
    setClientErrors((prev) => {
      const next = { ...prev };
      delete next[`pax_${idToRemove}_jamaah`];
      delete next[`pax_${idToRemove}_room`];
      return next;
    });
  };

  const handleConfirmCascadeDelete = () => {
    if (!cascadeDeleteTarget) return;

    // Reset ke 1 baris kosong baru (reguler index 0)
    setPaxList([{ id: 1, jamaah_id: "", is_infant: false, room_type: "" }]);
    setClientErrors({});
    setCascadeDeleteTarget(null);
  };

  const handleScheduleChange = (val) => {
    const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
    setSelectedScheduleId(actualVal);
    setClientErrors((prev) => {
      const next = { ...prev };
      delete next.schedule_id;
      return next;
    });

    const newSched = scheduleList.find((s) => String(s.id) === String(actualVal));
    if (newSched?.berangkat_tanggal) {
      setPaxList((prev) =>
        prev.map((p, idx) => {
          if (!p.jamaah_id) return p;
          const j = jamaahMap[p.jamaah_id];
          if (!j?.tanggal_lahir) {
            return { ...p, is_infant: false, room_type: "" };
          }
          const ageInfo = calculateAgeDetails(j.tanggal_lahir, newSched.berangkat_tanggal);
          if (!ageInfo) return p;

          if (idx === 0) {
            // Jika Pax #1 yang sudah terpilih SEKARANG jadi infant
            if (ageInfo.isInfant) {
              setClientErrors((prevErr) => ({
                ...prevErr,
                [`pax_${p.id}_jamaah`]: "Pax #1 adalah Kontak Utama dan harus jamaah reguler (bukan infant). Daftarkan jamaah ini di baris Pax #2 atau lainnya."
              }));
              return { ...p, jamaah_id: "", is_infant: false, room_type: "" };
            }
            return { ...p, is_infant: false, room_type: p.room_type || "" };
          }

          const updated = { ...p };
          if (ageInfo.isInfant) {
            updated.is_infant = true;
            updated.room_type = null;
          } else {
            const wasInfant = updated.is_infant;
            updated.is_infant = false;
            if (wasInfant || updated.room_type === null) {
              updated.room_type = "";
            }
          }
          return updated;
        })
      );
    }
  };

  const handlePaxChange = (id, field, value) => {
    const actualVal = (value && typeof value === 'object' && 'target' in value) ? value.target.value : value;

    if (field === "jamaah_id") {
      const targetPaxIndex = paxList.findIndex((p) => p.id === id);
      if (targetPaxIndex === 0 && actualVal) {
        const j = jamaahMap[actualVal];
        if (j?.tanggal_lahir && selectedSchedule?.berangkat_tanggal) {
          const ageInfo = calculateAgeDetails(j.tanggal_lahir, selectedSchedule.berangkat_tanggal);
          if (ageInfo?.isInfant) {
            setClientErrors((prev) => ({
              ...prev,
              [`pax_${id}_jamaah`]: "Pax #1 adalah Kontak Utama dan harus jamaah reguler (bukan infant). Daftarkan jamaah ini di baris Pax #2 atau lainnya."
            }));
            return; // Tolak pemilihan, dropdown tetap kosong
          }
        }
      }
    }

    setPaxList((prev) =>
      prev.map((p) => {
        if (p.id !== id) return p;

        const updated = { ...p, [field]: actualVal };

        if (field === "jamaah_id") {
          const j = jamaahMap[actualVal];
          if (j?.tanggal_lahir && selectedSchedule?.berangkat_tanggal) {
            const ageInfo = calculateAgeDetails(j.tanggal_lahir, selectedSchedule.berangkat_tanggal);
            if (ageInfo) {
              if (ageInfo.isInfant) {
                updated.is_infant = true;
                updated.room_type = null;
              } else {
                const wasInfant = updated.is_infant;
                updated.is_infant = false;
                if (wasInfant || updated.room_type === null) {
                  updated.room_type = "";
                }
              }
            }
          } else {
            updated.is_infant = false;
            updated.room_type = "";
          }
        }

        return updated;
      })
    );

    setClientErrors((prev) => {
      const next = { ...prev };
      if (field === "jamaah_id") delete next[`pax_${id}_jamaah`];
      if (field === "room_type") delete next[`pax_${id}_room`];
      return next;
    });
  };

  // === Handler Quick Create Jamaah ===
  const handleOpenQuickCreate = (paxId) => {
    setQuickCreatePaxId(paxId);
    setQuickCreateForm({ nama_lengkap: '', jenis_kelamin: '', nik: '', tanggal_lahir: '', no_hp: '' });
    setQuickCreateError(null);
    setQuickCreateOpen(true);
  };

  const handleQuickCreateSubmit = async () => {
    if (!quickCreateForm.nama_lengkap.trim()) {
      setQuickCreateError('Nama lengkap wajib diisi');
      return;
    }
    setQuickCreateLoading(true);
    setQuickCreateError(null);
    try {
      const payload = {
        nama_lengkap: quickCreateForm.nama_lengkap.trim(),
        jenis_kelamin: quickCreateForm.jenis_kelamin || null,
        nik: quickCreateForm.nik.trim() || null,
        tanggal_lahir: quickCreateForm.tanggal_lahir || null,
        no_hp: quickCreateForm.no_hp.trim() || null,
        status: 'aktif',
        ...(showBrandColumn && selectedBrandId ? { brand_id: parseInt(selectedBrandId, 10) } : {})
      };
      const newJamaah = await createJamaah(payload);
      // Tambahkan ke jamaahList in-memory
      setJamaahList(prev => [
        ...prev,
        {
          id: newJamaah.id,
          brand_id: newJamaah.brand_id,
          id_jamaah: newJamaah.id_jamaah,
          kode_jamaah: newJamaah.kode_jamaah,
          nama_lengkap: newJamaah.nama_lengkap,
          nik: newJamaah.nik,
          tanggal_lahir: newJamaah.tanggal_lahir,
          no_hp: newJamaah.no_hp,
          status: newJamaah.status,
          created_at: newJamaah.created_at
        }
      ]);
      // Auto-select di pax row yang membuka modal ini
      if (quickCreatePaxId !== null) {
        handlePaxChange(quickCreatePaxId, 'jamaah_id', String(newJamaah.id));
      }
      setQuickCreateOpen(false);
    } catch (err) {
      setQuickCreateError(err.response?.data?.error || 'Gagal membuat data jamaah');
    } finally {
      setQuickCreateLoading(false);
    }
  };

  // === Handler Inline DOB Quick Save ===
  const handleInlineDobChange = (jamaahId, val) => {
    setInlineDobMap(prev => ({
      ...prev,
      [jamaahId]: { ...prev[jamaahId], value: val, error: null }
    }));
  };

  const handleInlineDobSave = async (paxId, jamaahId) => {
    const dob = inlineDobMap[jamaahId]?.value || '';
    if (!dob) {
      setInlineDobMap(prev => ({ ...prev, [jamaahId]: { ...prev[jamaahId], error: 'Pilih tanggal lahir terlebih dahulu' } }));
      return;
    }
    // Ambil data jamaah saat ini sebagai base untuk update (required: nama_lengkap)
    const existingJamaah = jamaahMap[jamaahId];
    if (!existingJamaah) return;
    setInlineDobMap(prev => ({ ...prev, [jamaahId]: { ...prev[jamaahId], saving: true, error: null } }));
    try {
      const payload = {
        nama_lengkap: existingJamaah.nama_lengkap,
        tanggal_lahir: dob,
        nik: existingJamaah.nik || null,
        no_hp: existingJamaah.no_hp || null,
        status: existingJamaah.status || 'aktif'
      };
      await updateJamaah(jamaahId, payload);
      // Update in-memory jamaahList
      setJamaahList(prev => prev.map(j =>
        String(j.id) === String(jamaahId) ? { ...j, tanggal_lahir: dob } : j
      ));
      // Trigger recalculation for the pax row
      handlePaxChange(paxId, 'jamaah_id', String(jamaahId));
      // Bersihkan inline state untuk jamaah ini
      setInlineDobMap(prev => { const n = { ...prev }; delete n[jamaahId]; return n; });
    } catch (err) {
      setInlineDobMap(prev => ({ ...prev, [jamaahId]: { ...prev[jamaahId], saving: false, error: err.response?.data?.error || 'Gagal menyimpan tanggal lahir' } }));
    }
  };

  // Save Draft Handler (Validasi Minimal: Cukup Jadwal/Brand)
  const handleSaveDraft = async () => {
    setServerError(null);
    setDraftSuccess(null);

    const errors = {};
    if (showBrandColumn && !selectedBrandId) {
      errors.brand_id = "Pilih biro travel / brand terlebih dahulu";
    }
    if (!selectedScheduleId) {
      errors.schedule_id = "Pilih paket / jadwal terlebih dahulu untuk menyimpan draft";
    }

    if (Object.keys(errors).length > 0) {
      setClientErrors(errors);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    // Cek duplikasi jamaah di array paxList
    const seenDraftIds = new Set();
    let duplicateDraftName = null;
    for (const p of paxList) {
      if (!p.jamaah_id) continue;
      if (seenDraftIds.has(String(p.jamaah_id))) {
        const j = jamaahMap[p.jamaah_id];
        duplicateDraftName = j ? j.nama_lengkap : `Jamaah #${p.jamaah_id}`;
        break;
      }
      seenDraftIds.add(String(p.jamaah_id));
    }
    if (duplicateDraftName) {
      setServerError(`Jamaah ${duplicateDraftName} didaftarkan lebih dari satu kali dalam booking ini`);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    setDraftSubmitting(true);
    try {
      const draftPayload = {
        schedule_id: parseInt(selectedScheduleId, 10),
        pic_jamaah_id: picJamaahId ? parseInt(picJamaahId, 10) : null,
        pax: paxList
          .filter((p) => Boolean(p.jamaah_id))
          .map((p) => ({
            jamaah_id: parseInt(p.jamaah_id, 10),
            pax_type: p.is_infant ? "infant" : "reguler",
            room_type: p.is_infant ? null : (p.room_type || null)
          }))
      };

      if (isEditDraft) {
        await updateDraftBooking(id, draftPayload);
        setDraftSuccess("Perubahan draft booking berhasil disimpan.");
        window.scrollTo({ top: 0, behavior: "smooth" });
      } else {
        const res = await createDraftBooking(draftPayload);
        navigate(`/bookings/${res.id}/edit`, {
          state: { draftSuccess: "Draft booking berhasil disimpan. Anda berada dalam mode lanjutkan draft." }
        });
      }
    } catch (err) {
      setServerError(err.response?.data?.error || "Gagal menyimpan draft booking");
      window.scrollTo({ top: 0, behavior: "smooth" });
    } finally {
      setDraftSubmitting(false);
    }
  };

  const handleConfirmDeleteDraft = async () => {
    if (!id) return;
    try {
      setDeleteDraftSubmitting(true);
      await deleteDraftBooking(id);
      navigate("/bookings");
    } catch (err) {
      setServerError(err.response?.data?.error || "Gagal menghapus draft booking");
      setDeleteDraftModalOpen(false);
      window.scrollTo({ top: 0, behavior: "smooth" });
    } finally {
      setDeleteDraftSubmitting(false);
    }
  };

  // Submit Handler (Finalisasi atau Buat Booking Baru dengan Validasi Lengkap)
  const handleSubmit = async (e) => {
    e.preventDefault();
    setServerError(null);
    setDraftSuccess(null);

    const errors = {};
    if (showBrandColumn && !selectedBrandId) {
      errors.brand_id = "Pilih biro travel / brand terlebih dahulu";
    }

    if (!selectedScheduleId) {
      errors.schedule_id = "Pilih paket / jadwal terlebih dahulu";
    }

    if (paxList.length === 0) {
      errors.pax = "Minimal harus ada 1 pax dalam booking";
    }

    paxList.forEach((p, idx) => {
      if (!p.jamaah_id) {
        errors[`pax_${p.id}_jamaah`] = `Pilih jamaah untuk Pax #${idx + 1}`;
      }
      if (!p.is_infant && !p.room_type) {
        errors[`pax_${p.id}_room`] = `Pilih tipe kamar untuk Pax #${idx + 1}`;
      }
    });

    if (regularPaxCount === 0) {
      errors.pic_jamaah_id = "Minimal 1 pax reguler diperlukan sebagai kontak utama";
    } else if (!picJamaahId) {
      errors.pic_jamaah_id = "Pilih Kontak Utama (PIC) dari daftar jamaah reguler";
    } else {
      const isValidPic = paxList.some((p) => !p.is_infant && String(p.jamaah_id) === String(picJamaahId));
      if (!isValidPic) {
        errors.pic_jamaah_id = "Kontak Utama (PIC) harus merupakan jamaah reguler (bukan infant)";
      }
    }

    if (isOverQuota) {
      errors.quota = `Jumlah pax reguler (${regularPaxCount}) melebihi sisa kuota kursi (${seatSisa})`;
    }

    if (Object.keys(errors).length > 0) {
      setClientErrors(errors);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    // Cek duplikasi jamaah di array paxList
    const seenSubmitIds = new Set();
    let duplicateSubmitName = null;
    for (const p of paxList) {
      if (!p.jamaah_id) continue;
      if (seenSubmitIds.has(String(p.jamaah_id))) {
        const j = jamaahMap[p.jamaah_id];
        duplicateSubmitName = j ? j.nama_lengkap : `Jamaah #${p.jamaah_id}`;
        break;
      }
      seenSubmitIds.add(String(p.jamaah_id));
    }
    if (duplicateSubmitName) {
      setServerError(`Jamaah ${duplicateSubmitName} didaftarkan lebih dari satu kali dalam booking ini`);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    setClientErrors({});
    setSubmitting(true);

    try {
      const payload = {
        schedule_id: parseInt(selectedScheduleId, 10),
        pic_jamaah_id: parseInt(picJamaahId, 10),
        pax: paxList.map((p) => ({
          jamaah_id: parseInt(p.jamaah_id, 10),
          pax_type: p.is_infant ? "infant" : "reguler",
          room_type: p.is_infant ? null : p.room_type
        }))
      };

      if (isEditDraft) {
        // Update draft with latest complete changes, then finalize
        await updateDraftBooking(id, payload);
        await finalizeBooking(id);
        navigate(`/bookings/${id}`);
      } else {
        const res = await createBooking(payload);
        navigate(`/bookings/${res.id}`);
      }
    } catch (err) {
      setServerError(err.response?.data?.error || (isEditDraft ? "Gagal menyelesaikan booking" : "Gagal membuat booking"));
      setSubmitting(false);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  if (loadingData) {
    return (
      <div className="p-8 flex justify-center">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={isEditDraft ? "Lanjutkan Draft Booking" : "Booking Baru"}
        subtitle={isEditDraft ? "Lengkapi data pendaftaran sebelum finalisasi booking" : "Form pendaftaran jamaah paket umroh"}
        onBack={() => navigate("/bookings")}
      />

      {serverError && (
        <Alert variant="error" className="shadow-sm">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <span>{serverError}</span>
            {paxList.some((p) => p.jamaah_id && !jamaahMap[p.jamaah_id]?.tanggal_lahir) && (
              <div className="flex flex-wrap items-center gap-2 text-xs mt-1 sm:mt-0">
                {paxList
                  .filter((p) => p.jamaah_id && !jamaahMap[p.jamaah_id]?.tanggal_lahir)
                  .map((p) => (
                    <a
                      key={p.id}
                      href={`/jamaah/${p.jamaah_id}/edit`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-bold underline text-danger-900 hover:text-danger-950 inline-flex items-center gap-1"
                      title={`Buka form edit ${jamaahMap[p.jamaah_id]?.nama_lengkap || 'jamaah'} di tab baru`}
                    >
                      Lengkapi {jamaahMap[p.jamaah_id]?.nama_lengkap || `Pax #${p.id}`} →
                    </a>
                  ))}
              </div>
            )}
          </div>
        </Alert>
      )}
      {draftSuccess && (
        <Alert variant="success" className="shadow-sm border-success-300">
          <div className="flex items-center justify-between gap-2">
            <span className="font-medium">{draftSuccess}</span>
            <button
              type="button"
              onClick={() => setDraftSuccess(null)}
              className="text-xs font-bold text-success-800 hover:text-success-900 underline ml-4 cursor-pointer"
            >
              Tutup
            </button>
          </div>
        </Alert>
      )}
      {clientErrors.quota && <Alert variant="error">{clientErrors.quota}</Alert>}

      <form onSubmit={handleSubmit}>
        {/* Responsive 2-Column Grid Layout: Konten Form Pax (Kiri) & Sticky Summary/Actions (Kanan) */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
          
          {/* KOLOM KIRI: Pemilihan Travel, Paket & Manifest Pax */}
          <div className="lg:col-span-2 space-y-6">

            {/* 1. PILIH BIRO TRAVEL & PAKET JADWAL */}
            <MetaBox 
              title="Pilih Paket Umroh" 
              subtitle={showBrandColumn ? "Tentukan biro travel, bulan keberangkatan, dan paket jadwal" : "Pilih bulan keberangkatan dan paket jadwal aktif"}
              icon={<Calendar size={18} className="text-neutral-700" />}
            >
              <div className="space-y-4">
                {/* Mode Master Dashboard: 1. Brand, 2. Filter Bulan, 3. Paket Jadwal */}
                {showBrandColumn ? (
                  <div className="grid grid-cols-1 md:grid-cols-12 gap-4 items-start">
                    <div className="md:col-span-4">
                      <CustomDropdown
                        label="1. Biro Travel"
                        value={selectedBrandId}
                        onChange={handleBrandChange}
                        placeholder="Pilih biro travel..."
                        options={brands.map((b) => ({
                          value: String(b.id),
                          label: b.name
                        }))}
                        error={clientErrors.brand_id}
                        required
                      />
                    </div>

                    <div className="md:col-span-4">
                      <CustomDropdown
                        label="2. Bulan Berangkat"
                        value={selectedMonth}
                        onChange={handleMonthChange}
                        placeholder={
                          !selectedBrandId
                            ? "Pilih travel dulu..."
                            : availableMonths.length === 0
                              ? "Tidak ada jadwal"
                              : "Semua Bulan"
                        }
                        options={[
                          ...(availableMonths.length > 0 ? [{ value: "", label: "Semua Bulan" }] : []),
                          ...availableMonths
                        ]}
                        disabled={!selectedBrandId || availableMonths.length === 0}
                      />
                    </div>

                    <div className="md:col-span-4">
                      <CustomDropdown
                        label="3. Paket Umroh"
                        value={selectedScheduleId}
                        onChange={handleScheduleChange}
                        placeholder={
                          !selectedBrandId 
                            ? "Pilih travel dulu..." 
                            : filteredScheduleList.length === 0
                              ? "Tidak ada paket"
                              : "Pilih paket umroh..."
                        }
                        options={filteredScheduleList.map((s) => {
                          const dateFormatted = formatIndoDate(s.berangkat_tanggal);
                          const datePrefix = s.berangkat_tanggal ? `${dateFormatted} • ` : "";
                          const isFull = s.seat_sisa <= 0;
                          const isLow = s.seat_sisa > 0 && s.seat_sisa <= 5;
                          return {
                            value: s.id,
                            label: `${datePrefix}${s.jadwal_nama}`,
                            badge: isFull ? 'Penuh' : `${s.seat_sisa} Seat`,
                            badgeVariant: isFull ? 'danger' : isLow ? 'warning' : 'success'
                          };
                        })}
                        disabled={!selectedBrandId || filteredScheduleList.length === 0}
                        error={clientErrors.schedule_id}
                        required
                      />
                    </div>
                  </div>
                ) : (
                  /* Mode Travel Dashboard: 1. Filter Bulan, 2. Paket Jadwal */
                  <div className="grid grid-cols-1 md:grid-cols-12 gap-4 items-start">
                    <div className="md:col-span-4">
                      <CustomDropdown
                        label="1. Bulan Berangkat"
                        value={selectedMonth}
                        onChange={handleMonthChange}
                        placeholder={
                          availableMonths.length === 0
                            ? "Tidak ada jadwal"
                            : "Semua Bulan"
                        }
                        options={[
                          ...(availableMonths.length > 0 ? [{ value: "", label: "Semua Bulan" }] : []),
                          ...availableMonths
                        ]}
                        disabled={availableMonths.length === 0}
                      />
                    </div>

                    <div className="md:col-span-8">
                      <CustomDropdown
                        label="2. Paket Umroh"
                        value={selectedScheduleId}
                        onChange={handleScheduleChange}
                        placeholder={
                          filteredScheduleList.length === 0
                            ? "Tidak ada paket di bulan ini"
                            : "Pilih paket umroh..."
                        }
                        options={filteredScheduleList.map((s) => {
                          const dateFormatted = formatIndoDate(s.berangkat_tanggal);
                          const datePrefix = s.berangkat_tanggal ? `${dateFormatted} • ` : "";
                          const isFull = s.seat_sisa <= 0;
                          const isLow = s.seat_sisa > 0 && s.seat_sisa <= 5;
                          return {
                            value: s.id,
                            label: `${datePrefix}${s.jadwal_nama}`,
                            badge: isFull ? 'Penuh' : `${s.seat_sisa} Seat`,
                            badgeVariant: isFull ? 'danger' : isLow ? 'warning' : 'success'
                          };
                        })}
                        disabled={filteredScheduleList.length === 0}
                        error={clientErrors.schedule_id}
                        required
                      />
                    </div>
                  </div>
                )}

                {/* Preview Rincian Paket Terpilih */}
                {selectedSchedule && (
                  <div className="p-4 bg-neutral-50/80 rounded-xl border border-neutral-200/90 space-y-3.5 font-body">
                    {/* Header: Nama Paket + Brand Tag + Tombol Expand */}
                    <div className="flex flex-wrap items-center justify-between gap-2.5">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-heading font-bold text-sm md:text-base text-neutral-900">
                          {selectedSchedule.jadwal_nama}
                        </span>
                        {showBrandColumn && (brandsMap[selectedSchedule.brand_id]?.name || selectedSchedule.brand?.name) && (() => {
                          const brandObj = brandsMap[selectedSchedule.brand_id] || selectedSchedule.brand || brands.find(b => String(b.id) === String(selectedSchedule.brand_id));
                          const brandName = brandObj?.name || '';
                          const brandColor = brandObj?.primary_color || '#181C1F';
                          const textColor = getContrastTextColor(brandColor);
                          return (
                            <span 
                              className="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md shadow-2xs"
                              style={{
                                backgroundColor: brandColor,
                                color: textColor
                              }}
                            >
                              {brandName}
                            </span>
                          );
                        })()}
                      </div>

                      <div className="flex items-center gap-2">
                        {/* Tombol Expand / Collapse Rincian Paket */}
                        <button
                          type="button"
                          onClick={() => setShowScheduleDetails((prev) => !prev)}
                          className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold rounded-lg bg-white border border-neutral-200 text-neutral-700 hover:bg-neutral-100/80 hover:text-neutral-900 transition-colors shadow-2xs cursor-pointer"
                          title={showScheduleDetails ? "Tutup rincian paket" : "Buka rincian lengkap paket"}
                        >
                          <span>{showScheduleDetails ? "Tutup Rincian" : "Lihat Rincian"}</span>
                          <ChevronDown
                            size={14}
                            className={`transition-transform duration-200 ${showScheduleDetails ? "rotate-180 text-neutral-900" : "text-neutral-500"}`}
                          />
                        </button>
                      </div>
                    </div>

                    {/* Ringkasan Cepat: Tanggal & Durasi, Maskapai & Tipe Penerbangan */}
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 py-3 px-3.5 bg-white rounded-lg border border-neutral-200/80 text-xs">
                      {/* 1. Jadwal Keberangkatan & Durasi */}
                      <div className="space-y-1">
                        <div className="flex items-center gap-1.5 text-neutral-500">
                          <CalendarDays size={13} className="text-neutral-500 shrink-0" />
                          <span className="text-[11px] font-medium">Periode Berangkat</span>
                        </div>
                        <div className="font-semibold text-neutral-800">
                          {selectedSchedule.berangkat_tanggal ? (
                            selectedSchedule.pulang_tanggal ? (
                              <>
                                <span>{formatIndoDate(selectedSchedule.berangkat_tanggal)} – {formatIndoDate(selectedSchedule.pulang_tanggal)}</span>
                                {formatDurationDays(selectedSchedule.berangkat_tanggal, selectedSchedule.pulang_tanggal) && (
                                  <span className="ml-1 text-neutral-500 font-normal">
                                    ({formatDurationDays(selectedSchedule.berangkat_tanggal, selectedSchedule.pulang_tanggal)})
                                  </span>
                                )}
                              </>
                            ) : (
                              <span>{formatIndoDate(selectedSchedule.berangkat_tanggal)}</span>
                            )
                          ) : (
                            <span className="text-neutral-400 font-normal">-</span>
                          )}
                        </div>
                      </div>

                      {/* 2. Maskapai & Tipe Penerbangan */}
                      <div className="space-y-1">
                        <div className="flex items-center gap-1.5 text-neutral-500">
                          <Plane size={13} className="text-neutral-500 shrink-0" />
                          <span className="text-[11px] font-medium">Penerbangan</span>
                        </div>
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-semibold text-neutral-800">
                            {selectedSchedule.maskapai?.name || "-"}
                          </span>
                          <span className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                            selectedSchedule.is_direct_flight 
                              ? "bg-success-50 text-success-700 border border-success-200" 
                              : "bg-neutral-100 text-neutral-700 border border-neutral-200"
                          }`}>
                            {selectedSchedule.is_direct_flight ? "Direct" : (transitInfo.badgeText || "Transit")}
                          </span>
                        </div>
                      </div>
                    </div>

                    {/* EXPANDABLE SECTION: Rincian Lengkap untuk Penjelasan Front Desk */}
                    {showScheduleDetails && (
                      <div className="pt-2 border-t border-neutral-200/80 space-y-3.5 text-xs animate-in fade-in slide-in-from-top-2 duration-200">
                        {/* 1. Detail Rute & Jam Penerbangan */}
                        <div className="p-3.5 bg-white rounded-lg border border-neutral-200/80 space-y-3">
                          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-neutral-100 pb-2.5">
                            <span className="font-heading font-semibold text-neutral-900 flex items-center gap-1.5 text-xs md:text-sm">
                              <Plane size={14} className="text-primary-600" />
                              Jadwal Penerbangan
                            </span>
                            {selectedSchedule.is_ticket_confirmed && (
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="inline-flex items-center gap-1 bg-success-600 text-white px-2.5 py-0.5 rounded-full text-[10px] font-semibold shadow-2xs">
                                  <Check size={11} className="stroke-[2.5]" />
                                  <span>Tiket Confirmed</span>
                                </span>
                              </div>
                            )}
                          </div>

                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                            {/* Keberangkatan */}
                            <div className="bg-neutral-50/70 p-3 rounded-lg border border-neutral-200/60 space-y-1.5">
                              <div className="text-[11px] font-semibold text-neutral-700 flex items-center gap-1.5">
                                <PlaneTakeoff size={13} className="text-primary-600" />
                                <span>Keberangkatan</span>
                              </div>
                              <div className="text-neutral-900 font-bold text-xs md:text-sm">
                                {selectedSchedule.berangkat_bandara_asal || "-"} → {selectedSchedule.berangkat_bandara_tujuan || "-"}
                              </div>
                              <div className="text-neutral-600 text-[11px] space-y-0.5">
                                <div>Maskapai: <span className="font-medium text-neutral-800">{selectedSchedule.maskapai?.name || "-"}</span></div>
                                <div>Flight: <span className="font-medium text-neutral-800">{selectedSchedule.berangkat_kode_penerbangan || "-"}</span></div>
                                <div>Waktu: <span className="font-medium text-neutral-800">{selectedSchedule.berangkat_jam || "-"}</span></div>
                                {!selectedSchedule.is_direct_flight && transitInfo.berangkat && (
                                  <div>Transit: <span className="font-medium text-neutral-800">{transitInfo.berangkat}</span></div>
                                )}
                              </div>
                            </div>

                            {/* Kepulangan */}
                            <div className="bg-neutral-50/70 p-3 rounded-lg border border-neutral-200/60 space-y-1.5">
                              <div className="text-[11px] font-semibold text-neutral-700 flex items-center gap-1.5">
                                <PlaneLanding size={13} className="text-primary-600" />
                                <span>Kepulangan</span>
                              </div>
                              <div className="text-neutral-900 font-bold text-xs md:text-sm">
                                {selectedSchedule.pulang_bandara_asal || "-"} → {selectedSchedule.pulang_bandara_tujuan || "-"}
                              </div>
                              <div className="text-neutral-600 text-[11px] space-y-0.5">
                                <div>Maskapai: <span className="font-medium text-neutral-800">{selectedSchedule.maskapai?.name || "-"}</span></div>
                                <div>Flight: <span className="font-medium text-neutral-800">{selectedSchedule.pulang_kode_penerbangan || "-"}</span></div>
                                <div>Waktu: <span className="font-medium text-neutral-800">{selectedSchedule.pulang_jam || "-"}</span></div>
                                {!selectedSchedule.is_direct_flight && transitInfo.pulang && (
                                  <div>Transit: <span className="font-medium text-neutral-800">{transitInfo.pulang}</span></div>
                                )}
                              </div>
                            </div>
                          </div>
                        </div>

                        {/* 2. Detail Hotel Mekkah, Madinah & Transit */}
                        <div className="p-3.5 bg-white rounded-lg border border-neutral-200/80 space-y-3">
                          <span className="font-heading font-semibold text-neutral-900 flex items-center gap-1.5 text-xs md:text-sm border-b border-neutral-100 pb-2.5">
                            <Building2 size={14} className="text-primary-600" />
                            Akomodasi Hotel
                          </span>

                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                            {/* Hotel Mekkah */}
                            <div className="bg-neutral-50/70 p-3 rounded-lg border border-neutral-200/60 space-y-1">
                              <div className="flex items-center justify-between">
                                <span className="text-[11px] font-semibold text-neutral-700">Hotel Mekkah</span>
                                {selectedSchedule.hotel_mekkah?.star_rating > 0 && (
                                  <span className="text-amber-500 font-bold text-xs tracking-wider">
                                    {"★".repeat(selectedSchedule.hotel_mekkah.star_rating)}
                                  </span>
                                )}
                              </div>
                              <div className="font-bold text-neutral-900 text-xs md:text-sm">
                                {selectedSchedule.hotel_mekkah?.name || "-"}
                              </div>
                              {selectedSchedule.hotel_mekkah?.distance_m ? (
                                <div className="text-neutral-600 text-[11px] flex items-center gap-1 pt-0.5">
                                  <MapPin size={12} className="text-neutral-400 shrink-0" />
                                  <span>±{selectedSchedule.hotel_mekkah.distance_m}m ke Masjidil Haram</span>
                                </div>
                              ) : null}
                            </div>

                            {/* Hotel Madinah */}
                            <div className="bg-neutral-50/70 p-3 rounded-lg border border-neutral-200/60 space-y-1">
                              <div className="flex items-center justify-between">
                                <span className="text-[11px] font-semibold text-neutral-700">Hotel Madinah</span>
                                {selectedSchedule.hotel_madinah?.star_rating > 0 && (
                                  <span className="text-amber-500 font-bold text-xs tracking-wider">
                                    {"★".repeat(selectedSchedule.hotel_madinah.star_rating)}
                                  </span>
                                )}
                              </div>
                              <div className="font-bold text-neutral-900 text-xs md:text-sm">
                                {selectedSchedule.hotel_madinah?.name || "-"}
                              </div>
                              {selectedSchedule.hotel_madinah?.distance_m ? (
                                <div className="text-neutral-600 text-[11px] flex items-center gap-1 pt-0.5">
                                  <MapPin size={12} className="text-neutral-400 shrink-0" />
                                  <span>±{selectedSchedule.hotel_madinah.distance_m}m ke Masjid Nabawi</span>
                                </div>
                              ) : null}
                            </div>

                            {/* Hotel Transit (Jika Ada) */}
                            {selectedSchedule.transit_hotels && selectedSchedule.transit_hotels.length > 0 && (
                              selectedSchedule.transit_hotels.map((th, idx) => (
                                <div key={idx} className="bg-neutral-50/70 p-3 rounded-lg border border-neutral-200/60 space-y-1 sm:col-span-2">
                                  <div className="flex items-center justify-between">
                                    <span className="text-[11px] font-semibold text-neutral-700">
                                      Hotel Transit {th.kota ? `(${th.kota})` : ""}
                                    </span>
                                    {th.star_rating > 0 && (
                                      <span className="text-amber-500 font-bold text-xs tracking-wider">
                                        {"★".repeat(th.star_rating)}
                                      </span>
                                    )}
                                  </div>
                                  <div className="font-bold text-neutral-900 text-xs md:text-sm">
                                    {th.nama || "-"}
                                  </div>
                                </div>
                              ))
                            )}
                          </div>
                        </div>

                        {/* 3. Fasilitas Termasuk (Includes) & Tidak Termasuk (Excludes) */}
                        {((selectedSchedule.include_items && selectedSchedule.include_items.length > 0) || 
                          (selectedSchedule.exclude_items && selectedSchedule.exclude_items.length > 0)) && (
                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                            {/* Includes */}
                            {selectedSchedule.include_items && selectedSchedule.include_items.length > 0 && (
                              <div className="p-3.5 bg-white rounded-lg border border-neutral-200/80 space-y-2">
                                <span className="font-heading font-semibold text-success-800 text-xs flex items-center gap-1.5">
                                  <CheckCircle2 size={14} className="text-success-600 shrink-0" />
                                  Fasilitas Termasuk
                                </span>
                                <ul className="space-y-1.5 text-xs text-neutral-700">
                                  {selectedSchedule.include_items.map((item, idx) => (
                                    <li key={idx} className="flex items-start gap-1.5">
                                      <span className="text-success-600 font-bold shrink-0">✓</span>
                                      <span>{item}</span>
                                    </li>
                                  ))}
                                </ul>
                              </div>
                            )}

                            {/* Excludes */}
                            {selectedSchedule.exclude_items && selectedSchedule.exclude_items.length > 0 && (
                              <div className="p-3.5 bg-white rounded-lg border border-neutral-200/80 space-y-2">
                                <span className="font-heading font-semibold text-neutral-800 text-xs flex items-center gap-1.5">
                                  <AlertCircle size={14} className="text-neutral-400 shrink-0" />
                                  Tidak Termasuk
                                </span>
                                <ul className="space-y-1.5 text-xs text-neutral-600">
                                  {selectedSchedule.exclude_items.map((item, idx) => (
                                    <li key={idx} className="flex items-start gap-1.5">
                                      <span className="text-danger-500 font-bold shrink-0">✕</span>
                                      <span>{item}</span>
                                    </li>
                                  ))}
                                </ul>
                              </div>
                            )}
                          </div>
                        )}

                        {/* 4. Brosur & Materi Paket */}
                        {selectedSchedule.brosur_url && (
                          <div className="flex items-center justify-between p-3 bg-primary-50/60 border border-primary-200/70 rounded-lg">
                            <div className="flex items-center gap-2">
                              <FileText size={16} className="text-primary-700" />
                              <span className="text-xs font-medium text-neutral-900">Brosur Paket</span>
                            </div>
                            <a
                              href={selectedSchedule.brosur_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="inline-flex items-center gap-1 text-xs font-semibold text-primary-800 hover:text-primary-900 underline"
                            >
                              <span>Lihat Brosur</span>
                              <ExternalLink size={12} />
                            </a>
                          </div>
                        )}
                      </div>
                    )}

                    {/* Tarif Kamar Grid */}
                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1 text-xs">
                      <div className="bg-white p-2.5 rounded-lg border border-neutral-200/80">
                        <span className="text-neutral-500 block text-[11px]">Quad</span>
                        <span className="font-semibold text-neutral-900">{formatRupiah(selectedSchedule.harga_quad)}</span>
                      </div>
                      <div className="bg-white p-2.5 rounded-lg border border-neutral-200/80">
                        <span className="text-neutral-500 block text-[11px]">Triple</span>
                        <span className="font-semibold text-neutral-900">{formatRupiah(selectedSchedule.harga_triple)}</span>
                      </div>
                      <div className="bg-white p-2.5 rounded-lg border border-neutral-200/80">
                        <span className="text-neutral-500 block text-[11px]">Double</span>
                        <span className="font-semibold text-neutral-900">{formatRupiah(selectedSchedule.harga_double)}</span>
                      </div>
                      <div className="bg-white p-2.5 rounded-lg border border-neutral-200/80">
                        <span className="text-neutral-500 block text-[11px]">Infant</span>
                        <span className="font-semibold text-neutral-900">{formatRupiah(selectedSchedule.harga_infant || 0)}</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </MetaBox>

            {/* 2. REPEATER DAFTAR JAMAAH / PAX */}
            <MetaBox
              title="Daftar Jamaah (Manifest)"
              subtitle={(() => { const filled = paxList.filter(p => p.jamaah_id).length; return filled > 0 ? `${filled} jamaah terdaftar` : 'Belum ada jamaah dipilih'; })()}
              icon={<Users size={18} className="text-neutral-700" />}
              headerActions={
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={handleAddPax}
                  disabled={!selectedScheduleId}
                  className="flex items-center gap-1.5 text-xs"
                >
                  <Plus size={14} />
                  <span>Tambah Jamaah</span>
                </Button>
              }
            >
              <div className="space-y-4">
                {paxList.map((pax, index) => {
                  const paxPrice = calculatePaxPrice(pax);
                  const isPic = String(pax.jamaah_id) === String(picJamaahId) && Boolean(pax.jamaah_id) && !pax.is_infant;
                  const isFirstRow = index === 0;
                  const jamaahError = clientErrors[`pax_${pax.id}_jamaah`];
                  const roomError = clientErrors[`pax_${pax.id}_room`];

                  const currentJamaah = jamaahMap[pax.jamaah_id];
                  const hasDob = Boolean(currentJamaah?.tanggal_lahir);
                  const isMissingDob = Boolean(pax.jamaah_id && !hasDob);
                  const ageDetails = hasDob && selectedSchedule?.berangkat_tanggal
                    ? calculateAgeDetails(currentJamaah.tanggal_lahir, selectedSchedule.berangkat_tanggal)
                    : null;
                  const isInfant = Boolean(ageDetails?.isInfant);

                  return (
                    <div
                      key={pax.id}
                      className="p-4 rounded-xl border border-neutral-200 bg-white space-y-3 relative hover:border-neutral-300 transition-colors shadow-2xs"
                    >
                      {/* Header Baris Pax */}
                      <div className="flex items-center justify-between pb-3 border-b border-neutral-100">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-heading font-semibold text-xs md:text-sm text-neutral-900">
                            Jamaah #{index + 1}
                          </span>
                          {isPic && (
                            <Badge variant="primary" showIcon={false}>
                              PIC
                            </Badge>
                          )}
                          {isInfant && (
                            <Badge variant="warning" showIcon={false}>
                              Infant
                            </Badge>
                          )}
                        </div>

                        <div className="flex items-center gap-3">
                          {/* Tombol Hapus / Reset */}
                          <button
                            type="button"
                            disabled={paxList.length <= 1 && !pax.jamaah_id && !pax.is_infant && !pax.room_type}
                            onClick={() => handleRemovePax(pax.id)}
                            className={`w-8 h-8 flex items-center justify-center rounded-lg text-neutral-400 hover:text-danger-600 hover:bg-danger-50 transition-colors ${
                              paxList.length <= 1 && !pax.jamaah_id && !pax.is_infant && !pax.room_type
                                ? "opacity-30 cursor-not-allowed pointer-events-none"
                                : "cursor-pointer"
                            }`}
                            title={paxList.length <= 1 ? "Reset jamaah ini" : "Hapus jamaah ini"}
                            aria-label={`Hapus Pax #${index + 1}`}
                          >
                            <Trash2 size={16} />
                          </button>
                        </div>
                      </div>

                      {/* Input Grid Terstruktur Sempurna (6-3-3) */}
                      <div className="grid grid-cols-1 sm:grid-cols-12 gap-4 items-start pt-1">
                        {/* 1. Pilih Jamaah (6 col) */}
                        <div className="sm:col-span-6">
                          {/* Row: autocomplete + tombol Jamaah Baru (icon only) */}
                          <div className="flex items-end gap-2">
                            <div className="flex-1 min-w-0">
                              <AutocompleteInput
                                label="Pilih Jamaah"
                                value={pax.jamaah_id}
                                onChange={(e) => handlePaxChange(pax.id, "jamaah_id", e.target.value)}
                                placeholder={!selectedScheduleId ? "Pilih paket dulu..." : "Ketik nama / NIK..."}
                                options={getAvailableJamaahOptions(pax.id, index === 0)}
                                disabled={!selectedScheduleId}
                                error={jamaahError}
                                required
                                maxSuggestions={50}
                              />
                            </div>
                            {!pax.jamaah_id && (
                            <button
                              type="button"
                              disabled={!selectedScheduleId}
                              onClick={() => handleOpenQuickCreate(pax.id)}
                              className="w-11 h-11 shrink-0 inline-flex items-center justify-center rounded-xl border border-neutral-200 bg-white text-neutral-500 hover:bg-primary-50 hover:border-primary-300 hover:text-primary-700 transition-all shadow-2xs disabled:opacity-40 disabled:cursor-not-allowed"
                              title="Tambah jamaah baru langsung dari sini"
                              aria-label="Tambah jamaah baru"
                              style={{ marginBottom: '1rem' }}
                            >
                              <UserPlus size={15} />
                            </button>
                            )}
                          </div>
                          {/* Kondisi A: tanggal_lahir kosong — Inline DOB input */}
                          {isMissingDob && (() => {
                            const inlineState = inlineDobMap[pax.jamaah_id] || {};
                            return (
                              <div className="-mt-2 mb-0 space-y-1">
                                <div className="flex flex-wrap items-center gap-2 text-[11px] text-amber-800 bg-amber-50 border border-amber-200/80 px-3 py-2 rounded-lg">
                                  <AlertCircle size={12} className="shrink-0" />
                                  <span className="font-medium flex-1">Tanggal lahir belum diisi</span>
                                  <div className="flex items-center gap-1.5 w-full sm:w-auto">
                                    <input
                                      type="date"
                                      value={inlineState.value || ''}
                                      onChange={e => handleInlineDobChange(pax.jamaah_id, e.target.value)}
                                      className="h-8 rounded-lg border border-amber-300 bg-white px-2 text-xs text-neutral-900 focus:outline-none focus:ring-2 focus:ring-amber-400/40 font-body flex-1 sm:flex-none sm:w-36"
                                      max={new Date().toISOString().split('T')[0]}
                                    />
                                    <button
                                      type="button"
                                      onClick={() => handleInlineDobSave(pax.id, pax.jamaah_id)}
                                      disabled={inlineState.saving || !inlineState.value}
                                      className="h-8 px-2.5 inline-flex items-center gap-1 rounded-lg bg-amber-600 hover:bg-amber-700 text-white text-[11px] font-semibold transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                      {inlineState.saving ? <Loader2 size={12} className="animate-spin" /> : <Save size={12} />}
                                      <span>Simpan</span>
                                    </button>
                                  </div>
                                </div>
                                {inlineState.error && (
                                  <p className="text-[11px] text-danger-600 pl-1">{inlineState.error}</p>
                                )}
                              </div>
                            );
                          })()}
                        </div>

                        {/* 2. Tipe Kamar (3 col) */}
                        <div className="sm:col-span-3">
                          {isInfant ? (
                            /* Kondisi B: Infant (< 2 tahun) -> Tanpa Kamar */
                            <FormField label="Tipe Kamar">
                              <div className="h-11 w-full rounded-xl border border-neutral-200/90 bg-neutral-50 px-3.5 flex items-center text-xs md:text-sm text-neutral-400 font-body select-none shadow-2xs">
                                Tanpa Kamar
                              </div>
                            </FormField>
                          ) : (
                            /* Kondisi A (disabled) & Kondisi C (aktif normal) */
                            <CustomDropdown
                              label="Tipe Kamar"
                              value={pax.room_type || ""}
                              onChange={(val) => handlePaxChange(pax.id, "room_type", val)}
                              placeholder={
                                !selectedScheduleId
                                  ? "Pilih paket dulu..."
                                  : isMissingDob
                                  ? "Isi tgl lahir dulu..."
                                  : "Pilih tipe kamar..."
                              }
                              options={[
                                { value: "Quad", label: "Quad" },
                                { value: "Triple", label: "Triple" },
                                { value: "Double", label: "Double" }
                              ]}
                              disabled={!selectedScheduleId || isMissingDob || !pax.jamaah_id}
                              error={roomError}
                              required={!isMissingDob}
                            />
                          )}
                        </div>

                        {/* 3. Harga Pax Subtotal (3 col) */}
                        <div className="sm:col-span-3">
                          <FormField label={isInfant ? "Harga Infant" : "Harga / Pax"}>
                            <div className="h-11 w-full rounded-xl border border-neutral-200/90 bg-neutral-50 px-3.5 flex items-center text-xs md:text-sm font-semibold text-neutral-900 font-body select-none shadow-2xs">
                              {paxPrice !== null ? formatRupiah(paxPrice) : "-"}
                            </div>
                          </FormField>
                        </div>
                      </div>
                    </div>
                  );
                })}

                <div className="mt-4 flex justify-center">
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={handleAddPax}
                    disabled={!selectedScheduleId}
                    className="w-full sm:w-auto flex items-center justify-center gap-1.5 text-xs"
                  >
                    <Plus size={14} />
                    <span>Tambah Jamaah</span>
                  </Button>
                </div>
              </div>
            </MetaBox>

          </div>

          {/* KOLOM KANAN: Sticky Sidebar Ringkasan Harga, PIC & Aksi Simpan */}
          <div className="space-y-6 lg:sticky lg:top-6">

            {/* 1. Ringkasan & Total Harga */}
            <MetaBox 
              title="Ringkasan Biaya" 
              subtitle="Estimasi total pembayaran"
              icon={<Receipt size={18} className="text-neutral-700" />}
            >
              <div className="space-y-4">
                {selectedSchedule ? (
                  <>
                    {/* Status Kuota Kursi */}
                    <div
                      className={`p-3 rounded-xl border flex items-start gap-2.5 text-xs ${
                        isOverQuota
                          ? "bg-danger-50 text-danger-800 border-danger-200"
                          : "bg-neutral-50 text-neutral-700 border-neutral-200"
                      }`}
                    >
                      {isOverQuota ? (
                        <AlertCircle className="w-4 h-4 text-danger-600 shrink-0 mt-0.5" />
                      ) : (
                        <CheckCircle2 className="w-4 h-4 text-success-600 shrink-0 mt-0.5" />
                      )}
                      <div className="font-body">
                        {isOverQuota ? (
                          <div>
                            <span className="font-bold">Melebihi Kuota!</span>
                            <p className="mt-0.5">
                              Dibutuhkan <strong>{regularPaxCount} kursi</strong>, sisa <strong>{seatSisa} kursi</strong>.
                            </p>
                          </div>
                        ) : (
                          <div>
                            <span className="font-bold">Kuota Tersedia</span>
                            <p className="mt-0.5 text-neutral-600">
                              <strong>{regularPaxCount}</strong> dari <strong>{seatSisa}</strong> kursi akan terpakai.
                            </p>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* Breakdown Rincian Pax */}
                    <div className="space-y-2 pt-1 text-xs text-neutral-600 font-body border-t border-neutral-100">
                      <div className="flex justify-between">
                        <span>Jamaah Reguler ({regularPaxCount}x)</span>
                        <span className="font-medium text-neutral-900">
                          {formatRupiah(
                            paxList
                              .filter(p => !p.is_infant)
                              .reduce((sum, p) => sum + (calculatePaxPrice(p) || 0), 0)
                          )}
                        </span>
                      </div>
                      {infantPaxCount > 0 && (
                        <div className="flex justify-between">
                          <span>Infant ({infantPaxCount}x)</span>
                          <span className="font-medium text-neutral-900">
                            {formatRupiah(
                              paxList
                                .filter(p => p.is_infant)
                                .reduce((sum, p) => sum + (calculatePaxPrice(p) || 0), 0)
                            )}
                          </span>
                        </div>
                      )}
                    </div>

                    {/* Total Harga Highlight Card */}
                    <div className="p-4 bg-primary-50/70 border border-primary-200/80 rounded-xl">
                      <span className="text-[11px] uppercase tracking-wider font-bold text-neutral-600 block">
                        Total Biaya
                      </span>
                      <div className="mt-1 font-heading font-bold text-xl md:text-2xl text-neutral-900">
                        {formatRupiah(totalHarga)}
                      </div>
                      <p className="text-[11px] text-neutral-500 mt-1">
                        {infantPaxCount > 0 
                          ? `${paxList.length} Jamaah (${regularPaxCount} reguler, ${infantPaxCount} infant)` 
                          : `${paxList.length} Jamaah reguler`
                        }
                      </p>
                    </div>
                  </>
                ) : (
                  <div className="p-3.5 rounded-xl bg-neutral-50 border border-neutral-200 text-neutral-500 text-xs font-body">
                    Pilih travel dan paket umroh untuk melihat kalkulasi biaya.
                  </div>
                )}

                {/* Tombol Submit & Batal */}
                <div className="space-y-2 pt-2 border-t border-neutral-100">
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={handleSaveDraft}
                    disabled={draftSubmitting || submitting || deleteDraftSubmitting || !selectedScheduleId || (showBrandColumn && !selectedBrandId)}
                    className="w-full justify-center shadow-sm"
                  >
                    {draftSubmitting ? "Menyimpan Draft..." : "Simpan Draft"}
                  </Button>
                  <Button
                    type="submit"
                    variant="primary"
                    disabled={isSubmitDisabled || submitting || draftSubmitting || deleteDraftSubmitting}
                    className="w-full justify-center shadow-sm"
                  >
                    {submitting ? "Memproses..." : (isEditDraft ? "Selesaikan Booking" : "Buat Booking")}
                  </Button>
                  <Button 
                    type="button" 
                    variant="ghost" 
                    onClick={() => navigate("/bookings")}
                    disabled={draftSubmitting || submitting || deleteDraftSubmitting}
                    className="w-full justify-center text-neutral-600 hover:text-neutral-900"
                  >
                    Batal
                  </Button>
                  {isEditDraft && (
                    <Button
                      type="button"
                      variant="danger"
                      onClick={() => setDeleteDraftModalOpen(true)}
                      disabled={draftSubmitting || submitting || deleteDraftSubmitting}
                      className="w-full justify-center shadow-sm"
                    >
                      Hapus Draft
                    </Button>
                  )}
                </div>
              </div>
            </MetaBox>

            {/* 2. Kontak Utama (PIC Booking) */}
            <MetaBox 
              title="Kontak Utama (PIC)" 
              subtitle="Penanggung jawab invoice & grup"
              icon={<UserCheck size={18} className="text-neutral-700" />}
            >
              <div className="space-y-3">
                <CustomDropdown
                  label="Pilih PIC"
                  value={picJamaahId}
                  onChange={(val) => {
                    const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
                    setPicJamaahId(actualVal);
                    setClientErrors((prev) => {
                      const next = { ...prev };
                      delete next.pic_jamaah_id;
                      return next;
                    });
                  }}
                  placeholder={
                    picOptions.length === 0
                      ? "Pilih jamaah di daftar pax dulu..."
                      : "Pilih PIC..."
                  }
                  options={picOptions}
                  disabled={picOptions.length === 0}
                  error={clientErrors.pic_jamaah_id}
                  required
                />
                <p className="text-[11px] text-neutral-500 font-body">
                  Penanggung jawab invoice dan komunikasi rombongan.
                </p>
              </div>
            </MetaBox>

          </div>

        </div>
      </form>

      {/* Modal Konfirmasi Cascade Delete */}
      <Modal
        isOpen={Boolean(cascadeDeleteTarget)}
        onClose={() => setCascadeDeleteTarget(null)}
        title="Konfirmasi Hapus Pax"
        footer={
          <div className="flex items-center justify-end gap-3">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setCascadeDeleteTarget(null)}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={handleConfirmCascadeDelete}
            >
              Hapus Semua
            </Button>
          </div>
        }
      >
        <p className="text-sm text-neutral-700 font-body">
          Menghapus <strong>{cascadeDeleteTarget ? (jamaahMap[cascadeDeleteTarget.targetPax.jamaah_id]?.nama_lengkap || `Pax #${paxList.findIndex((p) => p.id === cascadeDeleteTarget.targetPax.id) + 1}`) : ''}</strong> akan menghapus juga <strong>{cascadeDeleteTarget?.infantCount} pax infant</strong> lain yang menyertainya, karena infant tidak bisa terdaftar sendiri tanpa pax reguler penyerta. Lanjutkan?
        </p>
      </Modal>

      {/* Modal Konfirmasi Hapus Draft */}
      <Modal
        isOpen={deleteDraftModalOpen}
        onClose={() => !deleteDraftSubmitting && setDeleteDraftModalOpen(false)}
        title="Konfirmasi Hapus Draft"
        footer={
          <div className="flex items-center justify-end gap-3">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setDeleteDraftModalOpen(false)}
              disabled={deleteDraftSubmitting}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={handleConfirmDeleteDraft}
              disabled={deleteDraftSubmitting}
            >
              {deleteDraftSubmitting ? "Menghapus..." : "Hapus"}
            </Button>
          </div>
        }
      >
        <p className="text-sm text-neutral-700 font-body">
          Hapus draft booking ini? Data pax yang sudah diisi akan ikut terhapus dan tidak bisa dikembalikan.
        </p>
      </Modal>

      {/* Modal Quick Create Jamaah */}
      <Modal
        isOpen={quickCreateOpen}
        onClose={() => !quickCreateLoading && setQuickCreateOpen(false)}
        title="Tambah Jamaah Baru"
        size="md"
        footer={
          <div className="flex items-center justify-end gap-3">
            <Button
              type="button"
              variant="secondary"
              onClick={() => !quickCreateLoading && setQuickCreateOpen(false)}
              disabled={quickCreateLoading}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="primary"
              onClick={handleQuickCreateSubmit}
              disabled={quickCreateLoading || !quickCreateForm.nama_lengkap.trim()}
              className="inline-flex items-center gap-1.5"
            >
              {quickCreateLoading ? (
                <Loader2 size={14} className="animate-spin" />
              ) : (
                <UserPlus size={14} />
              )}
              <span>{quickCreateLoading ? 'Menyimpan...' : 'Simpan Jamaah'}</span>
            </Button>
          </div>
        }
      >
        <div className="space-y-4 font-body">
          <p className="text-xs text-neutral-500">
            Isi data minimal jamaah. Data lebih lengkap dapat dilengkapi nanti melalui menu Jamaah.
          </p>
          {quickCreateError && (
            <div className="flex items-center gap-2 text-xs text-danger-700 bg-danger-50 border border-danger-200 rounded-lg px-3 py-2">
              <AlertCircle size={13} className="shrink-0" />
              <span>{quickCreateError}</span>
            </div>
          )}
          <div className="space-y-3">
            {/* Nama Lengkap */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-700">
                Nama Lengkap <span className="text-danger-500 font-bold">*</span>
              </label>
              <input
                type="text"
                value={quickCreateForm.nama_lengkap}
                onChange={e => setQuickCreateForm(prev => ({ ...prev, nama_lengkap: e.target.value }))}
                placeholder="cth: Ahmad Fauzi"
                className="h-11 w-full rounded-xl border border-neutral-200/90 bg-white px-3.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs"
                autoFocus
              />
            </div>
            {/* Jenis Kelamin */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-700">Jenis Kelamin</label>
              <div className="flex gap-4 items-center h-11 px-1">
                <label className="flex items-center gap-2 text-sm text-neutral-900 cursor-pointer">
                  <input
                    type="radio"
                    name="quick_jenis_kelamin"
                    value="L"
                    checked={quickCreateForm.jenis_kelamin === "L"}
                    onChange={(e) => setQuickCreateForm((prev) => ({ ...prev, jenis_kelamin: e.target.value }))}
                    className="w-4 h-4 text-primary-500 border-neutral-300 focus:ring-primary-500"
                  />
                  Laki-laki
                </label>
                <label className="flex items-center gap-2 text-sm text-neutral-900 cursor-pointer">
                  <input
                    type="radio"
                    name="quick_jenis_kelamin"
                    value="P"
                    checked={quickCreateForm.jenis_kelamin === "P"}
                    onChange={(e) => setQuickCreateForm((prev) => ({ ...prev, jenis_kelamin: e.target.value }))}
                    className="w-4 h-4 text-primary-500 border-neutral-300 focus:ring-primary-500"
                  />
                  Perempuan
                </label>
              </div>
            </div>
            {/* NIK */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-700">NIK</label>
              <input
                type="text"
                inputMode="numeric"
                value={quickCreateForm.nik}
                onChange={e => setQuickCreateForm(prev => ({ ...prev, nik: e.target.value.replace(/\D/g, '').slice(0, 16) }))}
                placeholder="16 digit NIK (opsional)"
                className="h-11 w-full rounded-xl border border-neutral-200/90 bg-white px-3.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs"
              />
            </div>
            {/* Tanggal Lahir */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-700">
                Tanggal Lahir
                <span className="ml-1 text-neutral-400 text-[11px] font-normal">(wajib untuk deteksi usia/infant)</span>
              </label>
              <input
                type="date"
                value={quickCreateForm.tanggal_lahir}
                onChange={e => setQuickCreateForm(prev => ({ ...prev, tanggal_lahir: e.target.value }))}
                max={new Date().toISOString().split('T')[0]}
                className="h-11 w-full rounded-xl border border-neutral-200/90 bg-white px-3.5 text-sm text-neutral-900 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs"
              />
            </div>
            {/* No HP */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-700">No. HP / WhatsApp</label>
              <input
                type="tel"
                value={quickCreateForm.no_hp}
                onChange={e => setQuickCreateForm(prev => ({ ...prev, no_hp: e.target.value }))}
                placeholder="cth: 08123456789"
                className="h-11 w-full rounded-xl border border-neutral-200/90 bg-white px-3.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs"
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default BookingFormPage;

import React, { useState, useEffect, useMemo, useRef } from 'react';
import { useParams, useNavigate, useSearchParams, useLocation } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import CustomDropdown from 'shared/ui/CustomDropdown';
import AutocompleteInput from 'shared/ui/AutocompleteInput';
import Button from 'shared/ui/Button';
import Toggle from 'shared/ui/Toggle';
import FormLayout, { FormSection, SidebarSection, FormAside } from 'shared/ui/FormLayout';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import FileInput from 'shared/ui/FileInput';

import { getSchedule, createSchedule, updateSchedule, updateScheduleStatus } from '../api/schedules';
import {
  listPerlengkapanItems,
  getCategoryPerlengkapan,
  getPerlengkapanSetTemplate,
  getSchedulePerlengkapan,
  putSchedulePerlengkapan,
} from '../api/perlengkapan';
import PerlengkapanSetEditor, { PerlengkapanSetList } from '../components/PerlengkapanSetEditor';
import HighlightBenefitSection, { JUMLAH_BENEFIT_PAKET } from '../components/HighlightBenefitSection';
import LokasiDikunjungiSection, { MAKS_LOKASI_PAKET } from '../components/LokasiDikunjungiSection';
import { listHotels } from '../api/hotels';
import { listAirlines } from '../api/airlines';
import { listAirports } from '../api/airports';
import { listCategories } from '../api/categories';
import { listItineraries, getItinerary } from '../api/itineraries';
import ItineraryDaysEditor, { emptyDay, normalizeDays, daysToPayload, validateDays } from '../components/ItineraryDaysEditor';
import { listBrandOptions } from '../api/brands';
import { uploadMediaWithOptions } from '../api/media';
import FlightLegFields from '../components/paket/FlightLegFields';
import { parseTransit, buildTransit } from '../utils/paketTransit';
import { STATUS_OPTIONS, assetUrl } from '../utils/paketFormat';

// Nama field yang dipakai backend di awal pesan error ("harga_quad wajib ...").
const SERVER_ERROR_FIELDS = new Set([
  'jadwal_nama', 'seat_total', 'brand_id', 'maskapai_id', 'berangkat_tanggal', 'berangkat_jam',
  'berangkat_kode_penerbangan', 'pulang_tanggal', 'pulang_jam', 'pulang_kode_penerbangan',
  'hotel_mekkah_id', 'hotel_madinah_id', 'harga_quad', 'harga_triple', 'harga_double',
  'harga_infant', 'minimal_dp', 'nominal_komisi_langsung',
  'nominal_bonus_pembinaan',
]);

const serverErrorField = (msg) => {
  const m = /^([a-z_]+) /.exec(msg || '');
  if (m && SERVER_ERROR_FIELDS.has(m[1])) return m[1];
  if (/nama dan tanggal berangkat/.test(msg || '')) return 'jadwal_nama';
  if (/highlight benefit|^benefit(_ids)? /.test(msg || '')) return 'benefit_ids';
  if (/(^|\s)lokasi\b/i.test(msg || '')) return 'lokasi_ids';
  return null;
};

// Tautan di ringkasan galat: fokus ke field bermasalah (id field, atau name bila id tidak ada).
const fokusField = (field) => {
  const el = document.getElementById(field) || document.querySelector(`[name="${field}"]`);
  if (!el) return;
  el.scrollIntoView({ block: 'center' });
  el.focus({ preventScroll: true });
};

const SECTION_LINKS = [
  { id: 'sec-dasar', label: 'Dasar' },
  { id: 'sec-harga', label: 'Harga' },
  { id: 'sec-penerbangan', label: 'Penerbangan' },
  { id: 'sec-hotel', label: 'Hotel' },
  { id: 'sec-benefit', label: 'Benefit' },
  { id: 'sec-lokasi', label: 'Lokasi' },
  { id: 'sec-fasilitas', label: 'Fasilitas' },
  { id: 'sec-perlengkapan', label: 'Perlengkapan' },
  { id: 'sec-itinerary', label: 'Itinerary' },
];

const ScheduleFormPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const isEditMode = !!id;
  // Duplikat paket: /schedules/new?from=<id> memuat data paket sumber sebagai draft baru.
  const [searchParams] = useSearchParams();
  const copyFromId = !isEditMode ? searchParams.get('from') : null;
  const sourceId = isEditMode ? id : copyFromId;

  const [isLoadingInitial, setIsLoadingInitial] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isUploadingBrosur, setIsUploadingBrosur] = useState(false);
  const [formError, setFormError] = useState(null);
  // DP dan komisi khusus: mati = ikut brand; nilai tersimpan hanya bila dinyalakan.
  const [aturKhusus, setAturKhusus] = useState(false);
  const [fieldErrors, setFieldErrors] = useState({});
  const errorRef = useRef(null);
  // Snapshot isian setelah data awal selesai dimuat, untuk peringatan "belum disimpan".
  const baselineRef = useRef(null);
  const submittedRef = useRef(false);
  useEffect(() => {
    if (formError) { errorRef.current?.focus(); errorRef.current?.scrollIntoView({ block: 'center' }); }
  }, [formError]);

  const [notFound, setNotFound] = useState(false);
  // Data paket gagal dimuat saat edit: form tidak ditampilkan, supaya isian kosong tidak menimpa paket.
  const [loadFailed, setLoadFailed] = useState(false);
  // Paket tersimpan tetapi set perlengkapan khususnya gagal disimpan (dua panggilan terpisah).
  const [peringatanSimpan, setPeringatanSimpan] = useState(location.state?.peringatanSimpan || null);
  // Lokasi tersimpan lengkap (nama, foto, aktif) dari detail paket, termasuk tempat nonaktif.
  const [lokasiTersimpan, setLokasiTersimpan] = useState([]);

  // Lookups
  const [hotelOptions, setHotelOptions] = useState([]);
  const [airlineOptions, setAirlineOptions] = useState([]);
  const [airportOptions, setAirportOptions] = useState([]);
  const [categoryOptions, setCategoryOptions] = useState([]);
  const [itineraryOptions, setItineraryOptions] = useState([]);
  const [brandOptions, setBrandOptions] = useState([]);

  const [includeItemsText, setIncludeItemsText] = useState('');
  // Perlengkapan jamaah: ikuti set kategori, atau set khusus paket ini.
  const [perlCatalog, setPerlCatalog] = useState([]);
  const [perlCustom, setPerlCustom] = useState(false);
  const [perlItems, setPerlItems] = useState([]);
  const [perlInherited, setPerlInherited] = useState([]);
  const [perlInheritedLabel, setPerlInheritedLabel] = useState('');
  const [perlWasCustom, setPerlWasCustom] = useState(false);
  const [perlOpen, setPerlOpen] = useState(false);
  // Itinerary: pilih dari daftar (master) atau salinan khusus milik paket ini.
  const [itinMode, setItinMode] = useState('master');
  const [itinDays, setItinDays] = useState([]);
  const [itinOpen, setItinOpen] = useState(false);
  const [itinWasKhusus, setItinWasKhusus] = useState(false);
  const [itinError, setItinError] = useState('');
  const [itinCache, setItinCache] = useState({});
  const [itinLoadingId, setItinLoadingId] = useState('');
  const [excludeItemsText, setExcludeItemsText] = useState('');

  const [berangkatFlightNo, setBerangkatFlightNo] = useState('');
  const [berangkatIsCustom, setBerangkatIsCustom] = useState(false);
  const [pulangFlightNo, setPulangFlightNo] = useState('');
  // Baris hotel transit kosong yang baru ditambahkan lewat tombol "+ Tambah Hotel Transit".
  const [extraTransitHotelSlots, setExtraTransitHotelSlots] = useState(0);
  const [pulangIsCustom, setPulangIsCustom] = useState(false);

  const [formData, setFormData] = useState({
    jadwal_nama: '',
    is_direct_flight: false,
    seat_total: '',
    kuota_terisi: '',
    maskapai_id: '',
    berangkat_tanggal: '',
    berangkat_jam: '',
    berangkat_kode_penerbangan: '',
    berangkat_bandara_asal: '',
    berangkat_bandara_tujuan: '',
    pulang_tanggal: '',
    pulang_jam: '',
    pulang_kode_penerbangan: '',
    pulang_bandara_asal: '',
    pulang_bandara_tujuan: '',
    transit_items_berangkat: [{ bandara: '', durasi: '' }],
    transit_items_pulang: [{ bandara: '', durasi: '' }],
    hotel_mekkah_id: '',
    hotel_madinah_id: '',
    transit_hotel_ids: [],
    harga_quad: '',
    harga_triple: '',
    harga_double: '',
    harga_infant: '',
    minimal_dp: '',
    nominal_komisi_langsung: '',
    nominal_bonus_pembinaan: '',
    itinerary_id: '',
    include_items: [''],
    exclude_items: [''],
    brosur_url: '',
    brosur_thumb_url: '',
    status: 'draft',
    brand_id: '',
    category_id: '',
    journey_type: 'umroh',
    // Highlight benefit terpilih, berurutan: [{ id, teks, ikon, is_active }].
    benefits: [],
    // Lokasi dikunjungi: id tempat Jelajah Haramain, berurutan.
    lokasi_ids: []
  });

  useEffect(() => {
    const fetchInitialData = async () => {
      setIsLoadingInitial(true);
      setFormError(null);
      try {
        const promises = [
          listHotels(),
          listAirlines(),
          listCategories(),
          listItineraries(),
          listBrandOptions(),
          listAirports()
        ];

        const results = await Promise.all(promises);
        const hotels = results[0];
        const airlines = results[1];
        const categories = results[2];
        const itineraries = results[3];
        const brands = results[4] || [];
        const airports = results[5] || [];

        setHotelOptions(hotels || []);
        setAirlineOptions(airlines || []);
        setCategoryOptions(categories || []);
        setItineraryOptions(itineraries || []);
        setBrandOptions(brands || []);
        setAirportOptions(airports || []);

        // Perlengkapan jamaah: katalog item + set tersimpan (gagal memuat tidak menghalangi form).
        listPerlengkapanItems().then((d) => setPerlCatalog(d || [])).catch(() => setPerlCatalog([]));
        if (sourceId) {
          getSchedulePerlengkapan(sourceId)
            .then((d) => {
              setPerlCustom(Boolean(d?.custom));
              setPerlItems(d?.custom ? (d.items || []).map((r) => ({ perlengkapan_item_id: r.perlengkapan_item_id, qty: r.qty, untuk: r.untuk || 'semua' })) : []);
              if (isEditMode) setPerlWasCustom(Boolean(d?.custom));
            })
            .catch(() => {});
        }

        if (sourceId) {
          const scheduleData = await getSchedule(sourceId);
          // Itinerary khusus: muat harinya sebagai salinan baru (duplikat tidak memakai ulang id privat).
          const khususId = scheduleData.itinerary_khusus ? (scheduleData.itinerary?.id || scheduleData.itinerary_id) : null;
          if (khususId) {
            setItinMode('khusus');
            setItinWasKhusus(isEditMode);
            getItinerary(khususId)
              .then((d) => setItinDays(normalizeDays(d?.days)))
              .catch(() => setItinError('Gagal memuat itinerary khusus.'));
          }
          const safeArray = (arr) => (Array.isArray(arr) && arr.length > 0) ? arr : [''];

          const parseInitialFlightCode = (rawCode, airlineCode) => {
            if (!rawCode) {
              return { isCustom: false, flightNo: '', fullValue: '' };
            }
            const rawStr = String(rawCode).trim();
            const upperStr = rawStr.toUpperCase();

            if (!airlineCode) {
              return { isCustom: true, flightNo: '', fullValue: rawStr };
            }

            const noSpaces = upperStr.replace(/\s+/g, '');
            const prefix = airlineCode.toUpperCase();

            if (noSpaces.startsWith(prefix)) {
              const rest = noSpaces.slice(prefix.length).replace(/^[^\d]+/, '');
              if (/^\d*$/.test(rest)) {
                return { isCustom: false, flightNo: rest, fullValue: prefix + rest };
              }
            }

            return { isCustom: true, flightNo: '', fullValue: rawStr };
          };

          const schedAirlineId = scheduleData.maskapai?.id || scheduleData.maskapai_id;
          const schedAirline = (airlines || []).find(a => String(a.id) === String(schedAirlineId));
          const schedAirlineCode = schedAirline?.code ? schedAirline.code.trim().toUpperCase() : '';

          const parsedBerangkat = parseInitialFlightCode(scheduleData.berangkat_kode_penerbangan, schedAirlineCode);
          setBerangkatFlightNo(parsedBerangkat.flightNo);
          setBerangkatIsCustom(parsedBerangkat.isCustom);

          const parsedPulang = parseInitialFlightCode(scheduleData.pulang_kode_penerbangan, schedAirlineCode);
          setPulangFlightNo(parsedPulang.flightNo);
          setPulangIsCustom(parsedPulang.isCustom);

          const parsedTransit = parseTransit(scheduleData.transit_bandara);
          const initialTransitHotelIDs = (scheduleData.transit_hotels || []).map(th => th.hotel_id);

          const loadedForm = {
            jadwal_nama: scheduleData.jadwal_nama || '',
            is_direct_flight: scheduleData.is_direct_flight || false,
            seat_total: scheduleData.seat_total || '',
            expected_seat_total: scheduleData.seat_total,
            // Terisi = seat dipegang booking, sama dengan hitungan server saat kuota diubah.
            kuota_terisi: scheduleData.seat_dipegang ?? '',
            maskapai_id: scheduleData.maskapai?.id || scheduleData.maskapai_id || '',
            berangkat_tanggal: scheduleData.berangkat_tanggal ? scheduleData.berangkat_tanggal.split('T')[0] : '',
            // Untuk peringatan: tanggal berangkat menentukan kapan komisi agen jadwal ini tersedia.
            berangkat_tanggal_awal: scheduleData.berangkat_tanggal ? scheduleData.berangkat_tanggal.split('T')[0] : '',
            komisi_aktif: scheduleData.komisi_aktif || 0,
            booking_aktif: scheduleData.booking_aktif || 0,
            alasan_ubah_tanggal: '',
            // Status saat dimuat: status hanya dikirim bila diubah di form ini, supaya simpan dari form yang
            // sudah lama terbuka tidak membatalkan perubahan status oleh admin lain.
            status_awal: scheduleData.status || 'draft',
            berangkat_jam: scheduleData.berangkat_jam || '',
            berangkat_kode_penerbangan: parsedBerangkat.fullValue || scheduleData.berangkat_kode_penerbangan || '',
            berangkat_bandara_asal: scheduleData.berangkat_bandara_asal || '',
            berangkat_bandara_tujuan: scheduleData.berangkat_bandara_tujuan || '',
            pulang_tanggal: scheduleData.pulang_tanggal ? scheduleData.pulang_tanggal.split('T')[0] : '',
            pulang_jam: scheduleData.pulang_jam || '',
            pulang_kode_penerbangan: parsedPulang.fullValue || scheduleData.pulang_kode_penerbangan || '',
            pulang_bandara_asal: scheduleData.pulang_bandara_asal || '',
            pulang_bandara_tujuan: scheduleData.pulang_bandara_tujuan || '',
            transit_items_berangkat: parsedTransit.berangkat,
            transit_items_pulang: parsedTransit.pulang,
            hotel_mekkah_id: scheduleData.hotel_mekkah?.id || scheduleData.hotel_mekkah_id || '',
            hotel_madinah_id: scheduleData.hotel_madinah?.id || scheduleData.hotel_madinah_id || '',
            transit_hotel_ids: initialTransitHotelIDs,
            harga_quad: scheduleData.harga_quad || '',
            harga_triple: scheduleData.harga_triple || '',
            harga_double: scheduleData.harga_double || '',
            harga_infant: scheduleData.harga_infant ?? '',
            minimal_dp: scheduleData.minimal_dp ?? '',
            nominal_komisi_langsung: scheduleData.nominal_komisi_langsung ?? '',
            nominal_bonus_pembinaan: scheduleData.nominal_bonus_pembinaan ?? '',
            itinerary_id: scheduleData.itinerary_khusus ? '' : (scheduleData.itinerary?.id || scheduleData.itinerary_id || ''),
            include_items: safeArray(scheduleData.include_items),
            exclude_items: safeArray(scheduleData.exclude_items),
            brosur_url: scheduleData.brosur_url || '',
            brosur_thumb_url: scheduleData.brosur_thumb_url || '',
            status: scheduleData.status || 'draft',
            brand_id: scheduleData.brand_id || '',
            category_id: scheduleData.category_id || (scheduleData.category?.id || ''),
            journey_type: scheduleData.journey_type || 'umroh',
            benefits: (Array.isArray(scheduleData.benefits) ? scheduleData.benefits : []).map(b => ({ id: b.id, teks: b.teks, ikon: b.ikon, is_active: b.is_active !== false })),
            lokasi_ids: (Array.isArray(scheduleData.lokasi_ids) ? scheduleData.lokasi_ids : []).map(Number)
          };
          setLokasiTersimpan(Array.isArray(scheduleData.lokasi) ? scheduleData.lokasi : []);

          // Salinan: tanggal diisi ulang, seat terisi direset, dan selalu mulai sebagai draft.
          setAturKhusus([scheduleData.minimal_dp, scheduleData.nominal_komisi_langsung, scheduleData.nominal_bonus_pembinaan].some((v) => v !== null && v !== undefined));
          setFormData(isEditMode ? loadedForm : {
            ...loadedForm,
            jadwal_nama: `${loadedForm.jadwal_nama} (Salinan)`.trim(),
            status: 'draft',
            expected_seat_total: undefined,
            kuota_terisi: '',
            berangkat_tanggal: '',
            pulang_tanggal: '',
            // Benefit nonaktif tidak ikut disalin: server menolaknya sebagai pilihan paket baru.
            benefits: loadedForm.benefits.filter(b => b.is_active),
          });

          setIncludeItemsText(safeArray(scheduleData.include_items).join('\n'));
          setExcludeItemsText(safeArray(scheduleData.exclude_items).join('\n'));
        }
      } catch (err) {
        if (isEditMode && err.response?.status === 404) {
          setNotFound(true);
        } else if (isEditMode) {
          setLoadFailed(true);
        } else {
          setFormError("Data hotel, maskapai, dan itinerary gagal dimuat. Muat ulang halaman.");
        }
      } finally {
        setIsLoadingInitial(false);
      }
    };

    fetchInitialData();
  }, [id, isEditMode, sourceId]);

  // Set bawaan yang berlaku bila paket mengikuti kategori: set kategori, atau set standar bila kosong.
  useEffect(() => {
    let cancelled = false;
    const categoryId = formData.category_id;
    const load = async () => {
      try {
        if (categoryId) {
          const d = await getCategoryPerlengkapan(categoryId);
          if (!cancelled && (d?.items || []).length > 0) {
            const nama = categoryOptions.find((c) => String(c.id) === String(categoryId))?.name;
            setPerlInherited(d.items);
            setPerlInheritedLabel(nama ? `Set bawaan kategori ${nama}` : 'Set bawaan kategori');
            return;
          }
        }
        const g = await getPerlengkapanSetTemplate();
        if (!cancelled) {
          setPerlInherited((g || []).map((r) => ({ ...r, untuk: 'semua' })));
          setPerlInheritedLabel('Set standar');
        }
      } catch {
        if (!cancelled) {
          setPerlInherited([]);
          setPerlInheritedLabel('');
        }
      }
    };
    load();
    return () => { cancelled = true; };
  }, [formData.category_id, categoryOptions]);

  useEffect(() => {
    if (!isLoadingInitial && baselineRef.current === null) baselineRef.current = JSON.stringify(formData);
  }, [isLoadingInitial, formData]);

  useEffect(() => {
    const warnBeforeUnload = (e) => {
      if (submittedRef.current || baselineRef.current === null || JSON.stringify(formData) === baselineRef.current) return;
      e.preventDefault();
      e.returnValue = '';
    };
    window.addEventListener('beforeunload', warnBeforeUnload);
    return () => window.removeEventListener('beforeunload', warnBeforeUnload);
  }, [formData]);

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => {
      return { ...prev, [name]: type === 'checkbox' ? checked : value };
    });
  };

  const handleTimeChange = (e) => {
    const { name, value } = e.target;

    let timeInput = value;
    let modifier = '';

    const plusIndex = value.indexOf('+');
    if (plusIndex !== -1) {
      timeInput = value.substring(0, plusIndex);
      modifier = value.substring(plusIndex).replace(/[^\d+]/g, '');
    } else if (value.includes(' ')) {
      const spaceIndex = value.indexOf(' ');
      timeInput = value.substring(0, spaceIndex);
      modifier = value.substring(spaceIndex).replace(/[^\d+]/g, '');
    }

    let val = timeInput.replace(/[^\d:]/g, '');
    let parts = val.split(':');

    if (parts.length > 2) {
      val = parts[0] + ':' + parts.slice(1).join('');
      parts = val.split(':');
    }

    if (parts[0] && parts[0].length > 2) {
      parts[1] = parts[0].substring(2) + (parts[1] || '');
      parts[0] = parts[0].substring(0, 2);
    }
    if (parts[0] && parts[0].length === 2 && parseInt(parts[0], 10) > 23) {
      parts[0] = '23';
    }

    if (parts.length > 1) {
      if (parts[1].length > 2) parts[1] = parts[1].substring(0, 2);
      if (parts[1].length === 2 && parseInt(parts[1], 10) > 59) {
        parts[1] = '59';
      }
    }

    let finalVal = parts.length > 1 ? `${parts[0]}:${parts[1]}` : (parts[0] || '');

    if (modifier) {
      finalVal += (finalVal.length > 0 && !modifier.startsWith(' ') ? ' ' : '') + modifier;
    }

    setFormData(prev => ({ ...prev, [name]: finalVal }));
  };


  const formatCurrency = (val) => {
    if (val === '' || val === null || val === undefined) return '';
    const num = val.toString().replace(/\D/g, '');
    return num.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  };

  const parseCurrency = (val) => {
    if (val === '' || val === null || val === undefined) return 0;
    const digits = val.toString().replace(/\D/g, '');
    return digits ? Number(digits) : 0;
  };

  const handleCurrencyChange = (e) => {
    const { name, value } = e.target;
    const rawValue = value.replace(/\D/g, '');
    setFormData(prev => ({
      ...prev,
      [name]: rawValue
    }));
  };



  const handleTransitChange = (leg, items) => {
    setFormData(prev => ({ ...prev, [`transit_items_${leg}`]: items }));
  };

  const handleBrosurUpload = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setIsUploadingBrosur(true);
    setFormError(null);
    setFieldErrors({});

    try {
      const response = await uploadMediaWithOptions(file, 'schedule-brosur', {
        maxWidth: '1080',
        generateThumbnail: true
      });
      setFormData(prev => ({
        ...prev,
        brosur_url: response.url,
        brosur_thumb_url: response.thumb_url || ''
      }));
    } catch (err) {
      setFormError(err.response?.data?.error || 'Gambar gagal diunggah.');
    } finally {
      setIsUploadingBrosur(false);
      e.target.value = ''; // Reset input
    }
  };

  const handleAddTransitHotel = (hotelId) => {
    if (!hotelId) return;
    const idNum = parseInt(hotelId, 10);
    setFormData(prev => {
      if (prev.transit_hotel_ids.includes(idNum)) return prev;
      return {
        ...prev,
        transit_hotel_ids: [...prev.transit_hotel_ids, idNum]
      };
    });
  };

  // Pilihan hotel untuk satu baris transit: tidak boleh sama dengan hotel Mekkah/Madinah
  // maupun hotel transit lain (hotel milik baris itu sendiri tetap boleh muncul).
  const transitHotelOptionsFor = (currentId) => hotelOptions
    .filter(h =>
      h.id !== parseInt(formData.hotel_mekkah_id, 10) &&
      h.id !== parseInt(formData.hotel_madinah_id, 10) &&
      (h.id === currentId || !formData.transit_hotel_ids.includes(h.id))
    )
    .map(h => ({
      value: h.id,
      label: `${h.name} — ${h.city || 'Lainnya'}${h.star_rating ? ` (${'★'.repeat(h.star_rating)})` : ''}`
    }));

  const handleReplaceTransitHotel = (oldId, newValue) => {
    const newId = parseInt(newValue, 10);
    setFormData(prev => ({
      ...prev,
      transit_hotel_ids: !newId
        ? prev.transit_hotel_ids.filter(id => id !== oldId)
        : prev.transit_hotel_ids
            .map(id => (id === oldId ? newId : id))
            .filter((id, i, arr) => arr.indexOf(id) === i)
    }));
  };

  const handleFillTransitHotelSlot = (newValue) => {
    if (!newValue) return;
    handleAddTransitHotel(newValue);
    setExtraTransitHotelSlots(count => Math.max(0, count - 1));
  };

  const handleRemoveTransitHotel = (hotelId) => {
    setFormData(prev => ({
      ...prev,
      transit_hotel_ids: prev.transit_hotel_ids.filter(id => id !== hotelId)
    }));
  };

  const selectedAirline = airlineOptions.find(a => String(a.id) === String(formData.maskapai_id));
  const currentAirlineCode = selectedAirline?.code ? selectedAirline.code.trim().toUpperCase() : '';

  const handleMaskapaiChange = (val) => {
    const newAirline = airlineOptions.find(a => String(a.id) === String(val));
    const newCode = newAirline?.code ? newAirline.code.trim().toUpperCase() : '';

    setFormData(prev => {
      const updatedBerangkat = (!berangkatIsCustom && newCode)
        ? (newCode + berangkatFlightNo)
        : prev.berangkat_kode_penerbangan;
      const updatedPulang = (!pulangIsCustom && newCode)
        ? (newCode + pulangFlightNo)
        : prev.pulang_kode_penerbangan;

      return {
        ...prev,
        maskapai_id: val,
        berangkat_kode_penerbangan: updatedBerangkat,
        pulang_kode_penerbangan: updatedPulang
      };
    });
  };

  const handleFlightNoChange = (leg, num) => {
    (leg === 'berangkat' ? setBerangkatFlightNo : setPulangFlightNo)(num);
    setFormData(prev => ({
      ...prev,
      [`${leg}_kode_penerbangan`]: currentAirlineCode ? (currentAirlineCode + num) : num
    }));
  };

  const airportDropdownOptions = useMemo(() => {
    return airportOptions.map(a => ({
      value: a.code,
      label: a.city ? `${a.code} - ${a.city}` : a.code
    }));
  }, [airportOptions]);

  // Tanggal berangkat paket yang punya booking aktif diubah: server mewajibkan alasan (audit menu Paket S-04).
  const butuhAlasanTanggal = isEditMode && formData.booking_aktif > 0 && !!formData.berangkat_tanggal_awal
    && formData.berangkat_tanggal !== formData.berangkat_tanggal_awal;

  const ensureItinDays = async (itId) => {
    if (!itId || itinCache[itId]) return itinCache[itId] || null;
    setItinLoadingId(String(itId));
    try {
      const d = await getItinerary(itId);
      const days = normalizeDays(d?.days);
      setItinCache((prev) => ({ ...prev, [itId]: days }));
      return days;
    } catch {
      return null;
    } finally {
      setItinLoadingId('');
    }
  };

  // Rincian itinerary master dimuat hanya saat dibuka.
  useEffect(() => {
    if (itinMode === 'master' && itinOpen && formData.itinerary_id) ensureItinDays(formData.itinerary_id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [itinMode, itinOpen, formData.itinerary_id]);

  const handleItinMode = async (val) => {
    const khusus = val === 'khusus';
    setItinError('');
    if (!khusus) {
      setItinMode('master');
      setItinOpen(false);
      return;
    }
    setItinMode('khusus');
    setItinOpen(true);
    if (itinDays.length > 0) return;
    const masterDays = formData.itinerary_id ? await ensureItinDays(formData.itinerary_id) : null;
    setItinDays(masterDays && masterDays.length > 0 ? JSON.parse(JSON.stringify(masterDays)) : [emptyDay()]);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setItinError('');
    if (itinMode === 'khusus') {
      const msg = validateDays(itinDays);
      if (msg) {
        setItinError(msg);
        setItinOpen(true);
        document.getElementById('sec-itinerary')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
        return;
      }
    }
    if (itinMode === 'master' && itinWasKhusus && !formData.itinerary_id) {
      setItinError('Pilih itinerary.');
      document.getElementById('sec-itinerary')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      return;
    }
    setIsSubmitting(true);
    setFormError(null);
    setFieldErrors({});

    try {
      const hargaQuad = parseCurrency(formData.harga_quad);
      const hargaTriple = parseCurrency(formData.harga_triple);
      const hargaDouble = parseCurrency(formData.harga_double);
      const hargaInfant = formData.harga_infant !== '' ? parseCurrency(formData.harga_infant) : null;
      const minimalDP = aturKhusus && formData.minimal_dp !== '' ? parseCurrency(formData.minimal_dp) : null;
      // Kosong = ikut komisi bawaan brand (NULL); 0 tetap disimpan sebagai 0 = sengaja tanpa komisi.
      const komisiLangsung = aturKhusus && formData.nominal_komisi_langsung !== '' ? parseCurrency(formData.nominal_komisi_langsung) : null;
      const bonusPembinaan = aturKhusus && formData.nominal_bonus_pembinaan !== '' ? parseCurrency(formData.nominal_bonus_pembinaan) : null;

      const errors = {};
      if (minimalDP !== null && (minimalDP < 0 || minimalDP > Math.min(hargaQuad, hargaTriple, hargaDouble))) errors.minimal_dp = 'DP harus antara 0 dan harga kamar termurah.';
if (formData.pulang_tanggal <= formData.berangkat_tanggal) errors.pulang_tanggal = 'Tanggal pulang harus setelah tanggal berangkat.';
      if (hargaInfant === null) errors.harga_infant = 'Isi harga infant.';
      if (butuhAlasanTanggal && !(formData.alasan_ubah_tanggal || '').trim()) errors.alasan_ubah_tanggal = 'Isi alasan ubah tanggal.';
      if (hargaTriple < hargaQuad) errors.harga_triple = 'Harga Triple tidak boleh di bawah harga Quad.';
      if (hargaDouble < hargaTriple) errors.harga_double = 'Harga Double tidak boleh di bawah harga Triple.';
      if (formData.status === 'published' && formData.benefits.length !== JUMLAH_BENEFIT_PAKET) {
        errors.benefit_ids = `Pilih ${JUMLAH_BENEFIT_PAKET} highlight benefit untuk paket published.`;
      }
      if (formData.lokasi_ids.length > MAKS_LOKASI_PAKET) errors.lokasi_ids = `Lokasi maksimal ${MAKS_LOKASI_PAKET}.`;
      if (Object.keys(errors).length) {
        setFieldErrors(errors); setFormError('Periksa isian berikut.'); return;
      }

      const finalTransit = buildTransit({ berangkat: formData.transit_items_berangkat, pulang: formData.transit_items_pulang });

      const finalBerangkatKode = (!berangkatIsCustom && currentAirlineCode)
        ? (berangkatFlightNo ? currentAirlineCode + berangkatFlightNo : '')
        : (formData.berangkat_kode_penerbangan || '').trim();

      const finalPulangKode = (!pulangIsCustom && currentAirlineCode)
        ? (pulangFlightNo ? currentAirlineCode + pulangFlightNo : '')
        : (formData.pulang_kode_penerbangan || '').trim();

      const seatTotalNum = parseInt(formData.seat_total, 10);

      const payload = {
        jadwal_nama: (formData.jadwal_nama || '').trim(),
        brand_id: parseInt(formData.brand_id, 10),
        category_id: formData.category_id ? parseInt(formData.category_id, 10) : null,
        // Edit: status lewat PUT /status setelah paket tersimpan (PUT paket mengabaikan status).
        ...(isEditMode ? {} : { status: formData.status || 'draft' }),
        ...(butuhAlasanTanggal ? { alasan_ubah_tanggal: (formData.alasan_ubah_tanggal || '').trim() } : {}),
        journey_type: formData.journey_type || 'umroh',
        is_direct_flight: !!formData.is_direct_flight,
        seat_total: seatTotalNum,
        ...(isEditMode ? { expected_seat_total: formData.expected_seat_total } : {}),
        maskapai_id: parseInt(formData.maskapai_id, 10),
        berangkat_tanggal: formData.berangkat_tanggal,
        berangkat_jam: formData.berangkat_jam || '',
        berangkat_kode_penerbangan: finalBerangkatKode,
        berangkat_bandara_asal: (formData.berangkat_bandara_asal || '').trim(),
        berangkat_bandara_tujuan: (formData.berangkat_bandara_tujuan || '').trim(),
        pulang_tanggal: formData.pulang_tanggal,
        pulang_jam: formData.pulang_jam || '',
        pulang_kode_penerbangan: finalPulangKode,
        pulang_bandara_asal: (formData.pulang_bandara_asal || '').trim(),
        pulang_bandara_tujuan: (formData.pulang_bandara_tujuan || '').trim(),
        transit_bandara: finalTransit,
        hotel_mekkah_id: parseInt(formData.hotel_mekkah_id, 10),
        hotel_madinah_id: parseInt(formData.hotel_madinah_id, 10),
        transit_hotel_ids: (formData.transit_hotel_ids || []).map(Number),
        harga_quad: hargaQuad,
        harga_triple: hargaTriple,
        harga_double: hargaDouble,
        harga_infant: hargaInfant,
        minimal_dp: minimalDP,
        nominal_komisi_langsung: komisiLangsung,
        nominal_bonus_pembinaan: bonusPembinaan,
        itinerary_id: itinMode === 'khusus' ? null : (formData.itinerary_id ? parseInt(formData.itinerary_id, 10) : null),
        ...(itinMode === 'khusus'
          ? { itinerary_khusus: { title: (formData.jadwal_nama || '').trim(), days: daysToPayload(itinDays) } }
          : {}),
        include_items: includeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        exclude_items: excludeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        brosur_url: formData.brosur_url || '',
        brosur_thumb_url: formData.brosur_thumb_url || '',
        benefit_ids: formData.benefits.map(b => b.id),
        lokasi_ids: formData.lokasi_ids
      };

      let savedId = id;
      // Data yang sudah tersimpan di server: simpan ulang dari form ini tidak boleh tertolak karena nilai awal lama.
      const sinkronTersimpan = (extra = {}) => setFormData(prev => ({
        ...prev,
        expected_seat_total: seatTotalNum,
        berangkat_tanggal_awal: prev.berangkat_tanggal,
        alasan_ubah_tanggal: '',
        ...extra,
      }));
      if (isEditMode) {
        await updateSchedule(id, payload);
        if (formData.status !== formData.status_awal) {
          try {
            await updateScheduleStatus(id, formData.status);
          } catch (err) {
            const sebab = err.response?.data?.error;
            sinkronTersimpan();
            setFormError(`Paket tersimpan, tetapi status gagal diubah${sebab ? `: ${sebab}` : '.'}`);
            const field = serverErrorField(sebab);
            if (field) setFieldErrors({ [field]: sebab });
            return;
          }
          sinkronTersimpan({ status_awal: formData.status });
        }
      } else {
        const created = await createSchedule(payload);
        savedId = created?.id;
      }
      // Set perlengkapan khusus paket disimpan terpisah; mengikuti kategori = hapus set khusus.
      if (savedId && (perlCustom || perlWasCustom)) {
        try {
          await putSchedulePerlengkapan(
            savedId,
            perlCustom,
            perlCustom
              ? perlItems.map((r) => ({ perlengkapan_item_id: r.perlengkapan_item_id, qty: Number(r.qty) || 1, untuk: r.untuk || 'semua' }))
              : [],
          );
        } catch (err) {
          // Paketnya sendiri sudah tersimpan. Pesan harus mengatakan itu, dan paket baru dibuka di mode edit
          // supaya simpan ulang tidak membuat paket kedua (ditolak 409 karena nama dan tanggal sama).
          const sebab = err.response?.data?.error;
          const pesan = `Paket sudah tersimpan, tetapi set perlengkapan khusus gagal disimpan${sebab ? `: ${sebab}` : '.'} Periksa set perlengkapan lalu simpan lagi.`;
          submittedRef.current = true;
          if (isEditMode) {
            setPeringatanSimpan(pesan);
            sinkronTersimpan({ status_awal: formData.status });
            window.scrollTo({ top: 0, behavior: 'smooth' });
          } else {
            navigate(`/schedules/${savedId}/edit`, { replace: true, state: { peringatanSimpan: pesan } });
          }
          return;
        }
      }
      submittedRef.current = true;
      const namaPaket = (formData.jadwal_nama || '').trim();
      navigate('/schedules', {
        state: { successMessage: `Paket "${namaPaket}" ${isEditMode ? 'diperbarui' : 'ditambahkan'}.` },
      });
    } catch (err) {
      const message = err.response?.data?.error || 'Paket gagal disimpan. Coba lagi.';
      setFormError(message);
      const field = serverErrorField(message);
      if (field) setFieldErrors({ [field]: message });
    } finally {
      setIsSubmitting(false);
    }
  };

  const mekkahHotels = hotelOptions.filter(h => {
    const c = (h.city || '').toUpperCase();
    return c === 'MAKKAH' || c === 'MEKKAH';
  });
  const madinahHotels = hotelOptions.filter(h => (h.city || '').toUpperCase() === 'MADINAH');

  const selectedBrandId = parseInt(formData.brand_id, 10);
  // Bawaan brand terpilih untuk DP dan komisi. Null bila brand belum dipilih atau daftar brand tidak termuat.
  const bawaanBrand = brandOptions.find(b => b.id === selectedBrandId) || null;
  const teksIkutBrand = (nominal) => {
    if (!bawaanBrand) return 'Ikut brand';
    return Number(nominal) > 0 ? `Ikut brand: ${formatCurrency(Math.round(Number(nominal)))}` : 'Ikut brand: tidak ada komisi';
  };
  // Kategori nonaktif tidak ditawarkan, kecuali yang sedang dipakai paket ini: paket lama tetap bisa
  // disimpan dengan kategorinya (server hanya menolak kategori nonaktif sebagai pilihan baru).
  const filteredCategories = categoryOptions.filter(c => {
    if (!c.is_active) return String(c.id) === String(formData.category_id);
    if (!c.brands || c.brands.length === 0) return true;
    if (!selectedBrandId) return true;
    return c.brands.some(b => b.id === selectedBrandId);
  });

  if (isLoadingInitial) {
    return (
      <LoadingSpinner variant="card" />
    );
  }

  if (loadFailed) {
    return (
      <div className="space-y-6">
        <PageHeader title="Edit Paket" onBack={() => navigate('/schedules')} />
        <Alert variant="error">Data paket gagal dimuat. Muat ulang halaman.</Alert>
        <Button variant="primary" onClick={() => window.location.reload()}>Muat ulang</Button>
      </div>
    );
  }

  if (notFound) {
    return (
      <div className="space-y-6">
        <PageHeader title="Paket Tidak Ditemukan" onBack={() => navigate('/schedules')} />
        <Alert variant="error">Paket tidak ada atau sudah dihapus.</Alert>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={isEditMode ? 'Edit Paket' : (copyFromId ? 'Duplikat Paket' : 'Tambah Paket')}
        onBack={() => navigate('/schedules')}
      />

      <form onSubmit={handleSubmit} className="space-y-6">
        {peringatanSimpan && (
          <Alert variant="warning" onClose={() => setPeringatanSimpan(null)}>{peringatanSimpan}</Alert>
        )}
        {formError && (
          <div ref={errorRef} tabIndex={-1} className="focus:outline-none">
            <Alert variant="error">
              <p className="font-medium">{formError}</p>
              {Object.keys(fieldErrors).length > 0 && (
                <ul className="mt-1 space-y-1">
                  {Object.entries(fieldErrors).map(([field, message]) => (
                    <li key={field}>
                      <a
                        className="underline underline-offset-2"
                        href={`#${field}`}
                        onClick={e => { e.preventDefault(); fokusField(field); }}
                      >
                        {message}
                      </a>
                    </li>
                  ))}
                </ul>
              )}
            </Alert>
          </div>
        )}
        <FormLayout tabs={SECTION_LINKS}>
            <FormSection id="sec-dasar" title="Informasi Dasar">
              <div className="space-y-4">
                {/* Nama paket + total kuota dalam satu baris */}
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-[minmax(0,1fr)_200px]">
                  <Input
                    label="Nama paket"
                    className="!mb-0"
                    type="text"
                    name="jadwal_nama"
                      error={fieldErrors.jadwal_nama}
                    value={formData.jadwal_nama}
                    onChange={handleChange}
                    required
                    placeholder="Umroh Reguler 9 Hari"
                  />
                  <div>
                    <Input
                      label="Kuota seat"
                      className="!mb-0"
                      type="number"
                      name="seat_total"
                      error={fieldErrors.seat_total}
                      value={formData.seat_total}
                      onChange={handleChange}
                      required
                      min="1"
                      placeholder="45"
                    />
                    {isEditMode && (
                      <p className="mt-1.5 text-xs text-neutral-500 font-body">
                        Terisi {formData.kuota_terisi === '' ? 0 : formData.kuota_terisi} · Sisa {formData.seat_total ? Math.max(0, parseInt(formData.seat_total) - (parseInt(formData.kuota_terisi) || 0)) : 0}
                      </p>
                    )}
                  </div>
                </div>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  {/* Brand */}
                  <div id="brand_id" tabIndex={-1} className="focus:outline-none">
                    <CustomDropdown
                      label="Brand"
                      className="!mb-0"
                      value={formData.brand_id}
                      onChange={(val) => {
                        handleChange({ target: { name: 'brand_id', value: val } });
                        // Benefit milik satu brand: ganti brand mengosongkan pilihan.
                        if (String(val) !== String(formData.brand_id)) setFormData(prev => ({ ...prev, benefits: [] }));
                        const selectedBid = parseInt(val, 10);
                        const currentCat = categoryOptions.find(c => String(c.id) === String(formData.category_id));
                        if (currentCat && currentCat.brands && currentCat.brands.length > 0) {
                          const brandSupported = currentCat.brands.some(b => b.id === selectedBid);
                          if (!brandSupported) {
                            setFormData(prev => ({ ...prev, brand_id: val, category_id: '' }));
                          }
                        }
                      }}
                      required
                      placeholder="Pilih brand"
                      options={brandOptions.map(b => ({ value: b.id, label: b.name }))}
                    />
                  </div>

                  {/* Kategori */}
                  <div>
                    <CustomDropdown
                      label="Kategori"
                      className="!mb-0"
                      value={formData.category_id}
                      onChange={(val) => {
                        handleChange({ target: { name: 'category_id', value: val } });
                        // Itinerary yang sudah dipilih dilepas bila tidak cocok dengan kategori baru.
                        const chosen = itineraryOptions.find(it => String(it.id) === String(formData.itinerary_id));
                        if (val && chosen && chosen.category_id && String(chosen.category_id) !== String(val)) {
                          setFormData(prev => ({ ...prev, category_id: val, itinerary_id: '' }));
                        }
                      }}
                      placeholder="Pilih kategori"
                      options={filteredCategories.map(c => ({ value: c.id, label: c.is_active ? c.name : `${c.name} (nonaktif)` }))}
                    />
                  </div>
                </div>
              </div>
            </FormSection>

            <FormSection id="sec-harga" title="Harga per Pax">
              <div className="space-y-3">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Input
                    label="Quad"
                    className="!mb-0"
                    type="text"
                    name="harga_quad"
                    error={fieldErrors.harga_quad}
                    value={formatCurrency(formData.harga_quad)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="25.000.000"
                  />
                  <Input
                    label="Triple"
                    className="!mb-0"
                    type="text"
                    name="harga_triple"
                    error={fieldErrors.harga_triple}
                    value={formatCurrency(formData.harga_triple)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="26.000.000"
                  />
                  <Input
                    label="Double"
                    className="!mb-0"
                    type="text"
                    name="harga_double"
                    error={fieldErrors.harga_double}
                    value={formatCurrency(formData.harga_double)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="28.000.000"
                  />
                  <Input
                    label="Infant"
                    className="!mb-0"
                    type="text"
                    name="harga_infant"
                    error={fieldErrors.harga_infant}
                    value={formatCurrency(formData.harga_infant)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="5.000.000"
                  />
                </div>

                <div className="border-t border-neutral-200 pt-4 mt-4 flex items-center justify-between gap-3">
                  <label htmlFor="atur_khusus" className="text-sm font-medium text-neutral-800 font-body cursor-pointer">DP dan komisi khusus</label>
                  <Toggle id="atur_khusus" name="atur_khusus" checked={aturKhusus} onChange={(e) => setAturKhusus(e.target.checked)} />
                </div>
                {aturKhusus && (
                  <>
                <div className="border-t border-neutral-200 pt-4 mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Input
                    label="Minimal DP"
                    className="!mb-0"
                    type="text"
                    name="minimal_dp"
                    error={fieldErrors.minimal_dp}
                    value={formatCurrency(formData.minimal_dp)}
                    onChange={handleCurrencyChange}
                    placeholder={bawaanBrand ? `Ikut brand: ${formatCurrency(bawaanBrand.minimal_dp) || '0'}` : 'Ikut brand'}
                    inputMode="numeric"
                  />
                </div>

                {/* Komisi agen per pax, dicatat saat booking lunas. Kosong = ikut bawaan brand; 0 = tanpa komisi. */}
                <div className="border-t border-neutral-200 pt-4 mt-4 space-y-2">
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <Input
                      label="Komisi agen"
                      className="!mb-0"
                      type="text"
                      name="nominal_komisi_langsung"
                      error={fieldErrors.nominal_komisi_langsung}
                      value={formatCurrency(formData.nominal_komisi_langsung)}
                      onChange={handleCurrencyChange}
                      placeholder={teksIkutBrand(bawaanBrand?.ujroh_closing)}
                      inputMode="numeric"
                    />
                    <Input
                      label="Bonus pembinaan upline"
                      className="!mb-0"
                      type="text"
                      name="nominal_bonus_pembinaan"
                      error={fieldErrors.nominal_bonus_pembinaan}
                      value={formatCurrency(formData.nominal_bonus_pembinaan)}
                      onChange={handleCurrencyChange}
                      placeholder={teksIkutBrand(bawaanBrand?.ujroh_pembinaan)}
                      inputMode="numeric"
                    />
                  </div>
                </div>
                  </>
                )}
              </div>
            </FormSection>
            <FormSection id="sec-penerbangan" title="Penerbangan">
              <div className="space-y-6">
                {/* Bagian 1: Data Maskapai */}
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <AutocompleteInput
                    label="Maskapai"
                    className="!mb-0"
                    name="maskapai_id"
                    id="maskapai_id"
                    error={fieldErrors.maskapai_id}
                    value={formData.maskapai_id}
                    onChange={(e) => handleMaskapaiChange(e.target.value)}
                    onSelect={(opt) => handleMaskapaiChange(opt ? opt.value : '')}
                    required
                    placeholder="Cari maskapai"
                    options={airlineOptions.map(a => ({ value: a.id, label: a.code ? `${a.name} (${a.code})` : a.name }))}
                  />
                  <CustomDropdown
                    label="Jenis penerbangan"
                    className="!mb-0"
                    value={formData.is_direct_flight ? 'direct' : 'transit'}
                    onChange={(val) => handleChange({ target: { name: 'is_direct_flight', type: 'checkbox', checked: val === 'direct' } })}
                    options={[
                      { value: 'direct', label: 'Direct' },
                      { value: 'transit', label: 'Transit' }
                    ]}
                  />
                </div>

                {/* Divider */}
                <div className="border-t border-neutral-200" />

                <FlightLegFields
                  leg="berangkat"
                  data={formData}
                  errors={fieldErrors}
                  onFieldChange={handleChange}
                  onTimeChange={handleTimeChange}
                  kodeMaskapai={currentAirlineCode}
                  kodeKhusus={berangkatIsCustom}
                  nomor={berangkatFlightNo}
                  onNomorChange={(num) => handleFlightNoChange('berangkat', num)}
                  airportOptions={airportDropdownOptions}
                  onTransitChange={(items) => handleTransitChange('berangkat', items)}
                >
                  {butuhAlasanTanggal && (
                    <div className="space-y-3">
                      <Alert variant="warning">
                        {formData.booking_aktif} booking aktif. Tanggal berangkat berubah dari{' '}
                        {formData.berangkat_tanggal_awal} ke {formData.berangkat_tanggal || '(kosong)'}. Kabari jamaah.
                      </Alert>
                      <Textarea
                        label="Alasan ubah tanggal"
                        name="alasan_ubah_tanggal"
                        value={formData.alasan_ubah_tanggal || ''}
                        onChange={handleChange}
                        error={fieldErrors.alasan_ubah_tanggal}
                        placeholder="Maskapai memundurkan jadwal"
                        rows={2}
                        maxLength={500}
                        required
                        className="!mb-0"
                      />
                    </div>
                  )}

                  {formData.komisi_aktif > 0 && formData.berangkat_tanggal_awal && formData.berangkat_tanggal !== formData.berangkat_tanggal_awal && (
                    <Alert variant="warning">{formData.komisi_aktif} komisi agen ikut bergeser tanggal cairnya.</Alert>
                  )}
                </FlightLegFields>

                <div className="border-t border-neutral-200" />

                <FlightLegFields
                  leg="pulang"
                  data={formData}
                  errors={fieldErrors}
                  onFieldChange={handleChange}
                  onTimeChange={handleTimeChange}
                  kodeMaskapai={currentAirlineCode}
                  kodeKhusus={pulangIsCustom}
                  nomor={pulangFlightNo}
                  onNomorChange={(num) => handleFlightNoChange('pulang', num)}
                  airportOptions={airportDropdownOptions}
                  onTransitChange={(items) => handleTransitChange('pulang', items)}
                />
              </div>
            </FormSection>

            <FormSection id="sec-hotel" title="Akomodasi Hotel">
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {/* Hotel Mekkah */}
                  <div>
                    <AutocompleteInput
                      label="Hotel Mekkah"
                      className="!mb-0"
                      name="hotel_mekkah_id"
                      id="hotel_mekkah_id"
                      error={fieldErrors.hotel_mekkah_id}
                      value={formData.hotel_mekkah_id}
                      onChange={(e) => handleChange({ target: { name: 'hotel_mekkah_id', value: e.target.value } })}
                      onSelect={(opt) => handleChange({ target: { name: 'hotel_mekkah_id', value: opt ? opt.value : '' } })}
                      required
                      placeholder="Cari hotel"
                      options={mekkahHotels.map(h => ({ value: h.id, label: h.star_rating ? `${h.name} (${'★'.repeat(h.star_rating)})` : h.name }))}
                    />
                  </div>

                  {/* Hotel Madinah */}
                  <div>
                    <AutocompleteInput
                      label="Hotel Madinah"
                      className="!mb-0"
                      name="hotel_madinah_id"
                      id="hotel_madinah_id"
                      error={fieldErrors.hotel_madinah_id}
                      value={formData.hotel_madinah_id}
                      onChange={(e) => handleChange({ target: { name: 'hotel_madinah_id', value: e.target.value } })}
                      onSelect={(opt) => handleChange({ target: { name: 'hotel_madinah_id', value: opt ? opt.value : '' } })}
                      required
                      placeholder="Cari hotel"
                      options={madinahHotels.map(h => ({ value: h.id, label: h.star_rating ? `${h.name} (${'★'.repeat(h.star_rating)})` : h.name }))}
                    />
                  </div>
                </div>

                {/* Hotel Transit (opsional): satu baris per hotel, tambah lewat tombol */}
                <div className="space-y-3">
                  <span className="block text-xs font-semibold text-neutral-700 font-heading">Hotel transit</span>

                  {formData.transit_hotel_ids.map((thId, idx) => (
                    <div key={`th-${thId}`} className="flex items-end gap-3">
                      <div className="min-w-0 flex-1">
                        <AutocompleteInput
                          label={(formData.transit_hotel_ids.length + extraTransitHotelSlots) > 1 ? `Hotel transit #${idx + 1}` : undefined}
                          className="!mb-0"
                          name={`transit_hotel_${idx}`}
                          value={thId}
                          onChange={(e) => handleReplaceTransitHotel(thId, e.target.value)}
                          onSelect={(opt) => handleReplaceTransitHotel(thId, opt ? opt.value : '')}
                          placeholder="Cari hotel"
                          options={transitHotelOptionsFor(thId)}
                        />
                      </div>
                      <Button variant="danger-outline" className="shrink-0"
                        onClick={() => handleRemoveTransitHotel(thId)}
                        title="Hapus"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </Button>
                    </div>
                  ))}

                  {Array.from({ length: extraTransitHotelSlots }).map((_, i) => (
                    <div key={`slot-${i}`} className="flex items-end gap-3">
                      <div className="min-w-0 flex-1">
                        <AutocompleteInput
                          label={(formData.transit_hotel_ids.length + extraTransitHotelSlots) > 1 ? `Hotel transit #${formData.transit_hotel_ids.length + i + 1}` : undefined}
                          className="!mb-0"
                          name={`transit_hotel_new_${i}`}
                          value=""
                          onChange={(e) => { if (e.target.value) handleFillTransitHotelSlot(e.target.value); }}
                          onSelect={(opt) => { if (opt) handleFillTransitHotelSlot(opt.value); }}
                          placeholder="Cari hotel"
                          options={transitHotelOptionsFor(null)}
                        />
                      </div>
                      <Button variant="danger-outline" className="shrink-0"
                        onClick={() => setExtraTransitHotelSlots(count => Math.max(0, count - 1))}
                        title="Hapus"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </Button>
                    </div>
                  ))}

                  <Button variant="dashed" className="w-full"
                    onClick={() => setExtraTransitHotelSlots(count => count + 1)}
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
                    </svg>
                    <span>Tambah hotel</span>
                  </Button>
                </div>
              </div>
            </FormSection>

            <FormSection id="sec-benefit" title="Highlight Benefit" aside={<span className="text-xs font-body text-neutral-500">{formData.benefits.length}/{JUMLAH_BENEFIT_PAKET}</span>}>
              <HighlightBenefitSection
                brandId={formData.brand_id}
                value={formData.benefits}
                onChange={(daftar) => setFormData(prev => ({ ...prev, benefits: daftar }))}
                error={fieldErrors.benefit_ids}
              />
            </FormSection>

            <FormSection id="sec-lokasi" title="Lokasi Dikunjungi" aside={<span className="text-xs font-body text-neutral-500">{formData.lokasi_ids.length}/{MAKS_LOKASI_PAKET}</span>}>
              <LokasiDikunjungiSection
                value={formData.lokasi_ids}
                tersimpan={lokasiTersimpan}
                onChange={(ids) => setFormData(prev => ({ ...prev, lokasi_ids: ids }))}
                error={fieldErrors.lokasi_ids}
              />
            </FormSection>

            <FormSection id="sec-fasilitas" title="Fasilitas Paket">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                {/* Include */}
                <div>
                  <Textarea
                    label="Termasuk"
                    value={includeItemsText}
                    onChange={(e) => setIncludeItemsText(e.target.value)}
                    placeholder="Tiket Pesawat PP&#10;Visa Umroh Resmi&#10;Hotel Mekkah & Madinah&#10;Makan 3x Sehari&#10;Muthawwif"
                    rows={6}
                    className="!mb-0"
                  />
                </div>

                {/* Exclude */}
                <div>
                  <Textarea
                    label="Tidak termasuk"
                    value={excludeItemsText}
                    onChange={(e) => setExcludeItemsText(e.target.value)}
                    placeholder="Paspor&#10;Vaksin Meningitis&#10;Pengeluaran Pribadi / Laundry&#10;Kelebihan Bagasi"
                    rows={6}
                    className="!mb-0"
                  />
                </div>
              </div>
            </FormSection>

            <FormSection id="sec-perlengkapan" title="Perlengkapan Jamaah">
              <div className="space-y-4">
                <CustomDropdown
                  label="Isi perlengkapan"
                  className="!mb-0"
                  value={perlCustom ? 'khusus' : 'ikuti'}
                  onChange={(v) => {
                    const val = v?.target ? v.target.value : v;
                    const custom = val === 'khusus';
                    setPerlCustom(custom);
                    setPerlOpen(custom);
                    // Pertama kali memilih set khusus: mulai dari set bawaan agar tinggal disesuaikan.
                    if (custom && perlItems.length === 0) {
                      setPerlItems(perlInherited.map((r) => ({ perlengkapan_item_id: r.perlengkapan_item_id, qty: r.qty, untuk: r.untuk || 'semua' })));
                    }
                  }}
                  options={[
                    { value: 'ikuti', label: perlInheritedLabel ? `Ikuti ${perlInheritedLabel.charAt(0).toLowerCase()}${perlInheritedLabel.slice(1)}` : 'Ikuti set bawaan' },
                    { value: 'khusus', label: 'Atur khusus untuk paket ini' },
                  ]}
                />

                <div className="flex items-center justify-between gap-3">
                  <p className="min-w-0 truncate text-sm font-body text-neutral-600">
                    {perlCustom
                      ? `${perlItems.length} item`
                      : `${perlInheritedLabel || 'Set bawaan'} · ${perlInherited.length} item`}
                  </p>
                  <button
                    type="button"
                    onClick={() => setPerlOpen((v) => !v)}
                    className="shrink-0 text-xs font-medium text-neutral-500 hover:text-neutral-900 transition-colors cursor-pointer"
                  >
                    {perlOpen ? 'Sembunyikan rincian' : 'Lihat rincian'}
                  </button>
                </div>

                {perlOpen && (perlCustom ? (
                  <PerlengkapanSetEditor catalog={perlCatalog} value={perlItems} onChange={setPerlItems} />
                ) : (
                  <PerlengkapanSetList items={perlInherited} emptyText="Belum ada set bawaan." />
                ))}
              </div>
            </FormSection>

            <FormSection id="sec-itinerary" title="Itinerary">
              <div className="space-y-4">
                <CustomDropdown
                  label="Isi itinerary"
                  className="!mb-0"
                  value={itinMode === 'khusus' ? 'khusus' : (formData.itinerary_id ? String(formData.itinerary_id) : '')}
                  placeholder="Pilih itinerary"
                  onChange={(v) => {
                    const val = v?.target ? v.target.value : v;
                    if (val === 'khusus') {
                      handleItinMode('khusus');
                      return;
                    }
                    handleItinMode('master');
                    handleChange({ target: { name: 'itinerary_id', value: val } });
                  }}
                  options={[
                    ...itineraryOptions
                      .filter(it => !formData.category_id || !it.category_id || String(it.category_id) === String(formData.category_id) || String(it.id) === String(formData.itinerary_id))
                      .map(it => ({ value: String(it.id), label: `${it.title} (${it.day_count || it.duration_days || '-'} hari)` })),
                    { value: 'khusus', label: 'Atur khusus untuk paket ini' },
                  ]}
                />

                {itinMode === 'master' ? (
                  <>
                    {formData.itinerary_id && (() => {
                      const picked = itineraryOptions.find((it) => String(it.id) === String(formData.itinerary_id));
                      const days = itinCache[formData.itinerary_id];
                      const total = days ? days.length : (picked?.day_count || picked?.duration_days || 0);
                      return (
                        <>
                          <div className="flex items-center justify-between gap-3">
                            <p className="min-w-0 truncate text-sm font-body text-neutral-600">
                              {picked?.title || 'Itinerary'} · {total} hari
                            </p>
                            <button
                              type="button"
                              onClick={() => setItinOpen((v) => !v)}
                              className="shrink-0 text-xs font-medium text-neutral-500 hover:text-neutral-900 transition-colors cursor-pointer"
                            >
                              {itinOpen ? 'Sembunyikan rincian' : 'Lihat rincian'}
                            </button>
                          </div>
                          {itinOpen && (
                            itinLoadingId === String(formData.itinerary_id) && !days ? (
                              <LoadingSpinner />
                            ) : (
                              <ol className="divide-y divide-neutral-200 text-sm font-body">
                                {(days || []).map((d, i) => (
                                  <li key={i} className="py-2">
                                    <p className="font-medium text-neutral-800">Hari ke-{i + 1} · {d.location}</p>
                                    <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs text-neutral-600">
                                      {d.activities.map((act, j) => <li key={j}>{act.text}</li>)}
                                    </ul>
                                  </li>
                                ))}
                              </ol>
                            )
                          )}
                        </>
                      );
                    })()}
                  </>
                ) : (
                  <>
                    <div className="flex items-center justify-between gap-3">
                      <p className="min-w-0 truncate text-sm font-body text-neutral-600">{itinDays.length} hari</p>
                      <button
                        type="button"
                        onClick={() => setItinOpen((v) => !v)}
                        className="shrink-0 text-xs font-medium text-neutral-500 hover:text-neutral-900 transition-colors cursor-pointer"
                      >
                        {itinOpen ? 'Sembunyikan rincian' : 'Lihat rincian'}
                      </button>
                    </div>
                    {itinOpen && <ItineraryDaysEditor days={itinDays} onChange={setItinDays} />}
                  </>
                )}

                {itinError && <p className="text-xs font-body text-danger-600">{itinError}</p>}
              </div>
            </FormSection>


          <FormAside>
              <SidebarSection>
                <CustomDropdown
                  label="Status"
                  className="!mb-0"
                  value={formData.status}
                  onChange={(val) => handleChange({ target: { name: 'status', value: val } })}
                  options={STATUS_OPTIONS}
                />
              </SidebarSection>

              <SidebarSection title="Gambar Utama">
              <div className="space-y-3">
                {formData.brosur_url && (
                  <div className="border border-neutral-200 rounded-xl p-1 bg-white overflow-hidden flex justify-center">
                    <img
                      src={assetUrl(formData.brosur_url)}
                      alt="Preview gambar utama"
                      className="max-w-full h-auto max-h-52 object-contain rounded"
                    />
                  </div>
                )}
                <FileInput
                  name="brosur-upload"
                  accept="image/*"
                  className="!mb-0"
                  showPreview={false}
                  onChange={handleBrosurUpload}
                  isUploading={isUploadingBrosur}
                  uploadingText="Mengunggah"
                  placeholder={formData.brosur_url ? 'Ganti gambar' : 'JPG, PNG, WebP'}
                />
              </div>
            </SidebarSection>

            <SidebarSection>
              <Button
                type="submit"
                variant="primary"
                isLoading={isSubmitting}
                disabled={isUploadingBrosur}
                className="w-full justify-center"
              >
                Simpan
              </Button>
            </SidebarSection>
            </FormAside>
        </FormLayout>
      </form>
    </div>
  );
};

export default ScheduleFormPage;

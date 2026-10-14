import React, { useState, useEffect, useMemo, useRef } from 'react';
import { AlertCircle } from 'lucide-react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import FormField from 'shared/ui/FormField';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import CustomDropdown from 'shared/ui/CustomDropdown';
import AutocompleteInput from 'shared/ui/AutocompleteInput';
import Button from 'shared/ui/Button';
import FormLayout, { FormSection, SidebarSection, FormAside } from 'shared/ui/FormLayout';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';

import { getSchedule, createSchedule, updateSchedule } from '../api/schedules';
import { listHotels } from '../api/hotels';
import { listAirlines } from '../api/airlines';
import { listAirports } from '../api/airports';
import { listCategories } from '../api/categories';
import { listItineraries } from '../api/itineraries';
import { listBrands } from '../api/brands';
import { uploadMediaWithOptions } from '../api/media';

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
  return null;
};

const SECTION_LINKS = [
  { id: 'sec-dasar', label: 'Dasar' },
  { id: 'sec-harga', label: 'Harga' },
  { id: 'sec-penerbangan', label: 'Penerbangan' },
  { id: 'sec-hotel', label: 'Hotel' },
  { id: 'sec-fasilitas', label: 'Fasilitas' },
  { id: 'sec-itinerary', label: 'Itinerary' },
];

const ScheduleFormPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEditMode = !!id;
  // Duplikat paket: /schedules/new?from=<id> memuat data paket sumber sebagai draft baru.
  const [searchParams] = useSearchParams();
  const copyFromId = !isEditMode ? searchParams.get('from') : null;
  const sourceId = isEditMode ? id : copyFromId;

  const [isLoadingInitial, setIsLoadingInitial] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isUploadingBrosur, setIsUploadingBrosur] = useState(false);
  const [formError, setFormError] = useState(null);
  const [fieldErrors, setFieldErrors] = useState({});
  const errorRef = useRef(null);
  // Snapshot isian setelah data awal selesai dimuat, untuk peringatan "belum disimpan".
  const baselineRef = useRef(null);
  const submittedRef = useRef(false);
  useEffect(() => {
    if (formError) { errorRef.current?.focus(); errorRef.current?.scrollIntoView({ block: 'center' }); }
  }, [formError]);

  const [notFound, setNotFound] = useState(false);

  // Lookups
  const [hotelOptions, setHotelOptions] = useState([]);
  const [airlineOptions, setAirlineOptions] = useState([]);
  const [airportOptions, setAirportOptions] = useState([]);
  const [categoryOptions, setCategoryOptions] = useState([]);
  const [itineraryOptions, setItineraryOptions] = useState([]);
  const [brandOptions, setBrandOptions] = useState([]);

  const [includeItemsText, setIncludeItemsText] = useState('');
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
    transit_bandara: '',
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
    category_id: ''
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
          listBrands(),
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

        if (sourceId) {
          const scheduleData = await getSchedule(sourceId);
          const safeArray = (arr) => (Array.isArray(arr) && arr.length > 0) ? arr : [''];

          const parseTransitData = (raw) => {
            if (!raw || !raw.trim()) {
              return {
                berangkat: [{ bandara: '', durasi: '' }],
                pulang: [{ bandara: '', durasi: '' }]
              };
            }

            let rawBerangkat = '';
            let rawPulang = '';

            if (raw.includes('Berangkat:') || raw.includes('Pulang:')) {
              const parts = raw.split('|');
              parts.forEach(part => {
                const trimmed = part.trim();
                if (trimmed.startsWith('Berangkat:')) {
                  rawBerangkat = trimmed.replace('Berangkat:', '').trim();
                } else if (trimmed.startsWith('Pulang:')) {
                  rawPulang = trimmed.replace('Pulang:', '').trim();
                }
              });
            } else {
              rawBerangkat = raw;
            }

            const parseList = (str) => {
              if (!str || !str.trim()) return [{ bandara: '', durasi: '' }];
              const segments = str.split(';');
              const items = segments.map(seg => {
                const trimmed = seg.trim();
                if (trimmed.includes(',')) {
                  const p = trimmed.split(',');
                  return { bandara: p[0].trim(), durasi: (p.slice(1).join(',').match(/\d+/) || [''])[0] };
                }
                return { bandara: trimmed, durasi: '' };
              }).filter(i => i.bandara || i.durasi);
              return items.length > 0 ? items : [{ bandara: '', durasi: '' }];
            };

            return {
              berangkat: parseList(rawBerangkat),
              pulang: parseList(rawPulang)
            };
          };

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

          const parsedTransit = parseTransitData(scheduleData.transit_bandara);
          const initialTransitHotelIDs = (scheduleData.transit_hotels || []).map(th => th.hotel_id);

          const loadedForm = {
            jadwal_nama: scheduleData.jadwal_nama || '',
            is_direct_flight: scheduleData.is_direct_flight || false,
            seat_total: scheduleData.seat_total || '',
            expected_seat_total: scheduleData.seat_total,
            kuota_terisi: (scheduleData.seat_total !== undefined && scheduleData.seat_sisa !== undefined) ? (scheduleData.seat_total - scheduleData.seat_sisa) : '',
            maskapai_id: scheduleData.maskapai?.id || scheduleData.maskapai_id || '',
            berangkat_tanggal: scheduleData.berangkat_tanggal ? scheduleData.berangkat_tanggal.split('T')[0] : '',
            berangkat_jam: scheduleData.berangkat_jam || '',
            berangkat_kode_penerbangan: parsedBerangkat.fullValue || scheduleData.berangkat_kode_penerbangan || '',
            berangkat_bandara_asal: scheduleData.berangkat_bandara_asal || '',
            berangkat_bandara_tujuan: scheduleData.berangkat_bandara_tujuan || '',
            pulang_tanggal: scheduleData.pulang_tanggal ? scheduleData.pulang_tanggal.split('T')[0] : '',
            pulang_jam: scheduleData.pulang_jam || '',
            pulang_kode_penerbangan: parsedPulang.fullValue || scheduleData.pulang_kode_penerbangan || '',
            pulang_bandara_asal: scheduleData.pulang_bandara_asal || '',
            pulang_bandara_tujuan: scheduleData.pulang_bandara_tujuan || '',
            transit_bandara: scheduleData.transit_bandara || '',
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
            itinerary_id: scheduleData.itinerary?.id || scheduleData.itinerary_id || '',
            include_items: safeArray(scheduleData.include_items),
            exclude_items: safeArray(scheduleData.exclude_items),
            brosur_url: scheduleData.brosur_url || '',
            brosur_thumb_url: scheduleData.brosur_thumb_url || '',
            status: scheduleData.status || 'draft',
            brand_id: scheduleData.brand_id || '',
            category_id: scheduleData.category_id || (scheduleData.category?.id || '')
          };

          // Salinan: tanggal diisi ulang, kursi terisi direset, dan selalu mulai sebagai draft.
          setFormData(isEditMode ? loadedForm : {
            ...loadedForm,
            jadwal_nama: `${loadedForm.jadwal_nama} (Salinan)`.trim(),
            status: 'draft',
            expected_seat_total: undefined,
            kuota_terisi: '',
            berangkat_tanggal: '',
            pulang_tanggal: '',
          });

          setIncludeItemsText(safeArray(scheduleData.include_items).join('\n'));
          setExcludeItemsText(safeArray(scheduleData.exclude_items).join('\n'));
        }
      } catch (err) {
        if (isEditMode && err.response?.status === 404) {
          setNotFound(true);
        } else {
          setFormError("Gagal memuat data awal (hotel/maskapai/itinerary). Pastikan server backend menyala.");
        }
      } finally {
        setIsLoadingInitial(false);
      }
    };

    fetchInitialData();
  }, [id, isEditMode, sourceId]);

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



  // Handler Transit Keberangkatan
  const handleAddTransitBerangkat = () => {
    setFormData(prev => ({
      ...prev,
      transit_items_berangkat: [...(prev.transit_items_berangkat || []), { bandara: '', durasi: '' }]
    }));
  };

  const handleRemoveTransitBerangkat = (index) => {
    setFormData(prev => {
      const current = prev.transit_items_berangkat || [];
      if (current.length <= 1) return prev;
      return {
        ...prev,
        transit_items_berangkat: current.filter((_, i) => i !== index)
      };
    });
  };

  const handleTransitBerangkatChange = (index, field, value) => {
    setFormData(prev => {
      const current = [...(prev.transit_items_berangkat || [])];
      if (!current[index]) current[index] = { bandara: '', durasi: '' };
      current[index] = { ...current[index], [field]: value };
      return {
        ...prev,
        transit_items_berangkat: current
      };
    });
  };

  // Handler Transit Kepulangan
  const handleAddTransitPulang = () => {
    setFormData(prev => ({
      ...prev,
      transit_items_pulang: [...(prev.transit_items_pulang || []), { bandara: '', durasi: '' }]
    }));
  };

  const handleRemoveTransitPulang = (index) => {
    setFormData(prev => {
      const current = prev.transit_items_pulang || [];
      if (current.length <= 1) return prev;
      return {
        ...prev,
        transit_items_pulang: current.filter((_, i) => i !== index)
      };
    });
  };

  const handleTransitPulangChange = (index, field, value) => {
    setFormData(prev => {
      const current = [...(prev.transit_items_pulang || [])];
      if (!current[index]) current[index] = { bandara: '', durasi: '' };
      current[index] = { ...current[index], [field]: value };
      return {
        ...prev,
        transit_items_pulang: current
      };
    });
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
      setFormError(err.response?.data?.error || 'Gagal mengupload gambar utama.');
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

  const handleBerangkatFlightNoChange = (e) => {
    const num = e.target.value.replace(/\D/g, '').slice(0, 5);
    setBerangkatFlightNo(num);
    setFormData(prev => ({
      ...prev,
      berangkat_kode_penerbangan: currentAirlineCode ? (currentAirlineCode + num) : num
    }));
  };

  const handlePulangFlightNoChange = (e) => {
    const num = e.target.value.replace(/\D/g, '').slice(0, 5);
    setPulangFlightNo(num);
    setFormData(prev => ({
      ...prev,
      pulang_kode_penerbangan: currentAirlineCode ? (currentAirlineCode + num) : num
    }));
  };

  const airportDropdownOptions = useMemo(() => {
    return airportOptions.map(a => ({
      value: a.code,
      label: a.city ? `${a.code} - ${a.city}` : a.code
    }));
  }, [airportOptions]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    setFormError(null);
    setFieldErrors({});

    try {
      const hargaQuad = parseCurrency(formData.harga_quad);
      const hargaTriple = parseCurrency(formData.harga_triple);
      const hargaDouble = parseCurrency(formData.harga_double);
      const hargaInfant = formData.harga_infant !== '' ? parseCurrency(formData.harga_infant) : null;
      const minimalDP = formData.minimal_dp !== '' ? parseCurrency(formData.minimal_dp) : null;
      // Kosong = jadwal tidak ikut program Syiar (NULL), 0 tetap disimpan sebagai 0.
      const komisiLangsung = formData.nominal_komisi_langsung !== '' ? parseCurrency(formData.nominal_komisi_langsung) : null;
      const bonusPembinaan = formData.nominal_bonus_pembinaan !== '' ? parseCurrency(formData.nominal_bonus_pembinaan) : null;

      const errors = {};
      if (minimalDP !== null && (minimalDP < 0 || minimalDP > Math.min(hargaQuad, hargaTriple, hargaDouble))) errors.minimal_dp = 'DP harus antara 0 dan harga kamar termurah.';
if (formData.pulang_tanggal <= formData.berangkat_tanggal) errors.pulang_tanggal = 'Tanggal pulang harus setelah tanggal berangkat.';
      if (hargaInfant === null) errors.harga_infant = 'Harga bayi wajib diisi.';
      if (hargaTriple < hargaQuad) errors.harga_triple = 'Harga Triple tidak boleh lebih murah dari harga Quad.';
      if (hargaDouble < hargaTriple) errors.harga_double = 'Harga Double tidak boleh lebih murah dari harga Triple.';
      if (Object.keys(errors).length) {
        setFieldErrors(errors); setFormError('Periksa kembali isian paket berikut.'); return;
      }

      const validBerangkat = (formData.transit_items_berangkat || [])
        .map(t => ({ bandara: t.bandara?.trim() || '', durasi: t.durasi?.trim() || '' }))
        .filter(t => t.bandara || t.durasi);

      const validPulang = (formData.transit_items_pulang || [])
        .map(t => ({ bandara: t.bandara?.trim() || '', durasi: t.durasi?.trim() || '' }))
        .filter(t => t.bandara || t.durasi);

      let transitParts = [];
      if (validBerangkat.length > 0) {
        const strB = validBerangkat.map(t => t.durasi ? `${t.bandara}, ${t.durasi} Jam` : t.bandara).join('; ');
        transitParts.push(`Berangkat: ${strB}`);
      }
      if (validPulang.length > 0) {
        const strP = validPulang.map(t => t.durasi ? `${t.bandara}, ${t.durasi} Jam` : t.bandara).join('; ');
        transitParts.push(`Pulang: ${strP}`);
      }

      const finalTransit = transitParts.join(' | ');

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
        status: formData.status || 'draft',
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
        itinerary_id: formData.itinerary_id ? parseInt(formData.itinerary_id, 10) : null,
        include_items: includeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        exclude_items: excludeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        brosur_url: formData.brosur_url || '',
        brosur_thumb_url: formData.brosur_thumb_url || ''
      };

      if (isEditMode) {
        await updateSchedule(id, payload);
      } else {
        await createSchedule(payload);
      }
      submittedRef.current = true;
      const namaPaket = (formData.jadwal_nama || '').trim();
      navigate('/schedules', {
        state: { successMessage: `Paket "${namaPaket}" berhasil ${isEditMode ? 'diperbarui' : 'ditambahkan'}.` },
      });
    } catch (err) {
      const message = err.response?.data?.error || `Gagal ${isEditMode ? 'mengupdate' : 'menyimpan'} paket.`;
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
  const filteredCategories = categoryOptions.filter(c => {
    if (!c.is_active) return false;
    if (!c.brands || c.brands.length === 0) return true;
    if (!selectedBrandId) return true;
    return c.brands.some(b => b.id === selectedBrandId);
  });

  if (isLoadingInitial) {
    return (
      <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card">
        <LoadingSpinner />
      </div>
    );
  }

  if (notFound) {
    return (
      <div className="space-y-6">
        <PageHeader title="Paket Tidak Ditemukan" />
        <Alert variant="error">Paket dengan ID {id} tidak ditemukan.</Alert>
        <Button variant="ghost" onClick={() => navigate('/schedules')}>Kembali ke Data Paket</Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={isEditMode ? 'Edit Paket Umroh' : (copyFromId ? 'Duplikat Paket' : 'Tambah Paket Baru')}
        onBack={() => navigate('/schedules')}
      />

      <form onSubmit={handleSubmit}>
        {formError && (
          <div
            ref={errorRef}
            tabIndex={-1}
            role="alert"
            aria-labelledby="package-errors"
            className="mb-5 flex gap-3 rounded-xl border border-danger-200 bg-danger-50 p-4 font-body focus:outline-none"
          >
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-danger-600" strokeWidth={2} aria-hidden="true" />
            <div className="min-w-0">
              <h2 id="package-errors" className="font-heading text-sm font-semibold leading-5 text-danger-800">{formError}</h2>
              {Object.keys(fieldErrors).length > 0 && (
                <ul className="mt-1.5 space-y-1">
                  {Object.entries(fieldErrors).map(([field, message]) => (
                    <li key={field} className="text-[13px] leading-5 text-danger-700">
                      <a
                        className="underline decoration-danger-300 underline-offset-2 hover:text-danger-900 hover:decoration-danger-600"
                        href={`#${field}`}
                        onClick={e => { e.preventDefault(); document.getElementById(field)?.focus(); }}
                      >
                        {message}
                      </a>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        )}
        <FormLayout tabs={SECTION_LINKS}>
            <FormSection id="sec-dasar" title="Informasi Dasar">
              <div className="space-y-4">
                {/* Nama paket + total kuota dalam satu baris */}
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-[minmax(0,1fr)_200px]">
                  <Input
                    label="Nama Paket"
                    className="!mb-0"
                    type="text"
                    name="jadwal_nama"
                      error={fieldErrors.jadwal_nama}
                    value={formData.jadwal_nama}
                    onChange={handleChange}
                    required
                    placeholder="mis. Umroh Reguler 9 Hari"
                  />
                  <div>
                    <Input
                      label="Total Kuota"
                      className="!mb-0"
                      type="number"
                      name="seat_total"
                      error={fieldErrors.seat_total}
                      value={formData.seat_total}
                      onChange={handleChange}
                      required
                      min="1"
                      placeholder="mis. 45"
                    />
                    {isEditMode && (
                      <p className="mt-1.5 text-xs text-neutral-500 font-body">
                        Terisi {formData.kuota_terisi === '' ? 0 : formData.kuota_terisi} · Sisa {formData.seat_total ? Math.max(0, parseInt(formData.seat_total) - (parseInt(formData.kuota_terisi) || 0)) : 0}
                      </p>
                    )}
                  </div>
                </div>
              </div>
            </FormSection>

            <FormSection id="sec-harga" title="Harga per Orang (Rp)">
              <div className="space-y-3">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Input
                    label="Kamar isi 4 (Quad)"
                    className="!mb-0"
                    type="text"
                    name="harga_quad"
                    error={fieldErrors.harga_quad}
                    value={formatCurrency(formData.harga_quad)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="mis. 25.000.000"
                  />
                  <Input
                    label="Kamar isi 3 (Triple)"
                    className="!mb-0"
                    type="text"
                    name="harga_triple"
                    error={fieldErrors.harga_triple}
                    value={formatCurrency(formData.harga_triple)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="mis. 26.000.000"
                  />
                  <Input
                    label="Kamar isi 2 (Double)"
                    className="!mb-0"
                    type="text"
                    name="harga_double"
                    error={fieldErrors.harga_double}
                    value={formatCurrency(formData.harga_double)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="mis. 28.000.000"
                  />
                  <Input
                    label="Bayi (Infant)"
                    className="!mb-0"
                    type="text"
                    name="harga_infant"
                    error={fieldErrors.harga_infant}
                    value={formatCurrency(formData.harga_infant)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="mis. 5.000.000"
                  />
                </div>
                <p className="text-xs text-neutral-500 font-body">
                  Harga per orang. Dewasa mengikuti isi kamar; bayi memakai harga sendiri.
                </p>
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
                    error={fieldErrors.maskapai_id}
                    value={formData.maskapai_id}
                    onChange={(e) => handleMaskapaiChange(e.target.value)}
                    onSelect={(opt) => handleMaskapaiChange(opt ? opt.value : '')}
                    required
                    placeholder="Ketik / cari maskapai..."
                    options={airlineOptions.map(a => ({ value: a.id, label: a.code ? `${a.name} (${a.code})` : a.name }))}
                  />
                  <CustomDropdown
                    label="Jenis Penerbangan"
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

                {/* Bagian 2: Penerbangan Berangkat */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-neutral-600 text-[13px] font-semibold font-heading">
                    <span>Penerbangan Berangkat</span>
                  </div>

                  {/* Baris 1: Waktu & Kode Penerbangan */}
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Input
                      label="Tanggal Berangkat"
                      className="!mb-0"
                      type="date"
                      name="berangkat_tanggal"
                    error={fieldErrors.berangkat_tanggal}
                      value={formData.berangkat_tanggal}
                      onChange={handleChange}
                      required
                    />
                    <Input
                      label="Jam Berangkat"
                      className="!mb-0"
                      type="text"
                      name="berangkat_jam"
                    error={fieldErrors.berangkat_jam}
                      value={formData.berangkat_jam}
                      onChange={handleTimeChange}
                      placeholder="mis. 09:00"
                    />
                    {!berangkatIsCustom && currentAirlineCode ? (
                      <div>
                        <label className="mb-1.5 block min-h-5 text-[13px] font-medium leading-5 text-neutral-800 font-body">
                          Kode Penerbangan
                        </label>
                        <div className="flex">
                          <span className="inline-flex h-10 items-center px-3 rounded-l-lg border border-r-0 border-neutral-200 bg-neutral-50 text-neutral-700 font-mono font-semibold text-[13px] select-none">
                            {currentAirlineCode}
                          </span>
                          <input
                            type="text"
                            name="berangkat_flight_no"
                            value={berangkatFlightNo}
                            onChange={handleBerangkatFlightNoChange}
                            placeholder="mis. 822"
                            maxLength={5}
                            inputMode="numeric"
                            className="h-10 flex-1 min-w-0 block w-full px-3.5 rounded-none rounded-r-lg border border-neutral-200 focus:outline-none focus:border-primary-500 text-[13px] font-mono text-neutral-900 bg-white placeholder:text-neutral-400"
                          />
                        </div>
                      </div>
                    ) : (
                      <Input
                        label="Kode Penerbangan"
                        className="!mb-0"
                        name="berangkat_kode_penerbangan"
                    error={fieldErrors.berangkat_kode_penerbangan}
                        value={formData.berangkat_kode_penerbangan}
                        onChange={handleChange}
                        placeholder="mis. SV 822"
                      />
                    )}
                  </div>

                  {/* Baris 2: Bandara Asal & Tujuan */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <AutocompleteInput
                      label="Bandara Asal"
                      className="!mb-0"
                      name="berangkat_bandara_asal"
                      value={formData.berangkat_bandara_asal}
                      onChange={(e) => setFormData(prev => ({ ...prev, berangkat_bandara_asal: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, berangkat_bandara_asal: opt?.value || '' }))}
                      placeholder="Ketik kode / kota (mis. CGK / Jakarta)"
                      options={airportDropdownOptions}
                    />
                    <AutocompleteInput
                      label="Bandara Tujuan Akhir"
                      className="!mb-0"
                      name="berangkat_bandara_tujuan"
                      value={formData.berangkat_bandara_tujuan}
                      onChange={(e) => setFormData(prev => ({ ...prev, berangkat_bandara_tujuan: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, berangkat_bandara_tujuan: opt?.value || '' }))}
                      placeholder="Ketik kode / kota (mis. JED / Jeddah)"
                      options={airportDropdownOptions}
                    />
                  </div>

                  {/* Transit Keberangkatan */}
                  {!formData.is_direct_flight && (
                    <div className="pt-2 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-semibold text-neutral-700 font-heading flex items-center gap-1.5">
                          <span>Transit Keberangkatan</span>
                        </span>
                      </div>

                      <div className="space-y-3">
                        {(formData.transit_items_berangkat || [{ bandara: '', durasi: '' }]).map((item, idx) => (
                          <div key={idx} className="flex flex-col sm:flex-row items-start sm:items-end gap-3 pb-2.5 border-b border-neutral-100 last:border-b-0">
                            <div className="flex-1 w-full grid grid-cols-1 sm:grid-cols-3 gap-3">
                              <div className="sm:col-span-2">
                                <AutocompleteInput
                                  label={`Bandara Transit ${formData.transit_items_berangkat.length > 1 ? `#${idx + 1}` : ''}`}
                                  className="!mb-0"
                                  name={`transit_berangkat_${idx}`}
                                  value={item.bandara}
                                  onChange={(e) => handleTransitBerangkatChange(idx, 'bandara', e.target.value)}
                                  onSelect={(opt) => handleTransitBerangkatChange(idx, 'bandara', opt?.value || '')}
                                  placeholder="Ketik kode / kota (mis. KUL / DOH)"
                                  options={airportDropdownOptions}
                                />
                              </div>
                              <div>
                                <Input
                                  label="Durasi Transit (jam)"
                                  className="!mb-0"
                                  value={item.durasi}
                                  onChange={(e) => handleTransitBerangkatChange(idx, 'durasi', e.target.value.replace(/\D/g, '').slice(0, 3))}
                                  inputMode="numeric"
                                  placeholder="mis. 2"
                                />
                              </div>
                            </div>

                            {(formData.transit_items_berangkat || []).length > 1 && (
                              <button
                                type="button"
                                onClick={() => handleRemoveTransitBerangkat(idx)}
                                className="h-10 px-3 inline-flex items-center gap-1 text-xs text-danger-600 hover:text-danger-700 hover:bg-danger-50 rounded-md border border-neutral-200 transition-colors cursor-pointer shrink-0"
                                title="Hapus titik transit"
                              >
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                </svg>
                                <span className="sm:hidden">Hapus</span>
                              </button>
                            )}
                          </div>
                        ))}

                        <button
                          type="button"
                          onClick={handleAddTransitBerangkat}
                          className="w-full py-2 px-4 rounded-md border border-dashed border-neutral-300 hover:border-primary-400 hover:bg-neutral-50 text-neutral-700 hover:text-primary-600 font-medium text-xs flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
                          </svg>
                          <span>Tambah Transit</span>
                        </button>
                      </div>
                    </div>
                  )}
                </div>

                {/* Divider */}
                <div className="border-t border-neutral-200" />

                {/* Bagian 3: Penerbangan Pulang */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-neutral-600 text-[13px] font-semibold font-heading">
                    <span>Penerbangan Pulang</span>
                  </div>

                  {/* Baris 1: Waktu & Kode Penerbangan */}
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Input
                      label="Tanggal Pulang"
                      className="!mb-0"
                      type="date"
                      name="pulang_tanggal"
                    error={fieldErrors.pulang_tanggal}
                      value={formData.pulang_tanggal}
                      onChange={handleChange}
                      required
                    />
                    <Input
                      label="Jam Pulang"
                      className="!mb-0"
                      type="text"
                      name="pulang_jam"
                    error={fieldErrors.pulang_jam}
                      value={formData.pulang_jam}
                      onChange={handleTimeChange}
                      placeholder="mis. 15:30"
                    />
                    {!pulangIsCustom && currentAirlineCode ? (
                      <div>
                        <label className="mb-1.5 block min-h-5 text-[13px] font-medium leading-5 text-neutral-800 font-body">
                          Kode Penerbangan
                        </label>
                        <div className="flex">
                          <span className="inline-flex h-10 items-center px-3 rounded-l-lg border border-r-0 border-neutral-200 bg-neutral-50 text-neutral-700 font-mono font-semibold text-[13px] select-none">
                            {currentAirlineCode}
                          </span>
                          <input
                            type="text"
                            name="pulang_flight_no"
                            value={pulangFlightNo}
                            onChange={handlePulangFlightNoChange}
                            placeholder="mis. 823"
                            maxLength={5}
                            inputMode="numeric"
                            className="h-10 flex-1 min-w-0 block w-full px-3.5 rounded-none rounded-r-lg border border-neutral-200 focus:outline-none focus:border-primary-500 text-[13px] font-mono text-neutral-900 bg-white placeholder:text-neutral-400"
                          />
                        </div>
                      </div>
                    ) : (
                      <Input
                        label="Kode Penerbangan"
                        className="!mb-0"
                        name="pulang_kode_penerbangan"
                    error={fieldErrors.pulang_kode_penerbangan}
                        value={formData.pulang_kode_penerbangan}
                        onChange={handleChange}
                        placeholder="mis. SV 823"
                      />
                    )}
                  </div>

                  {/* Baris 2: Bandara Asal & Tujuan */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <AutocompleteInput
                      label="Bandara Asal"
                      className="!mb-0"
                      name="pulang_bandara_asal"
                      value={formData.pulang_bandara_asal}
                      onChange={(e) => setFormData(prev => ({ ...prev, pulang_bandara_asal: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, pulang_bandara_asal: opt?.value || '' }))}
                      placeholder="Ketik kode / kota (mis. MED / Madinah)"
                      options={airportDropdownOptions}
                    />
                    <AutocompleteInput
                      label="Bandara Tujuan Akhir"
                      className="!mb-0"
                      name="pulang_bandara_tujuan"
                      value={formData.pulang_bandara_tujuan}
                      onChange={(e) => setFormData(prev => ({ ...prev, pulang_bandara_tujuan: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, pulang_bandara_tujuan: opt?.value || '' }))}
                      placeholder="Ketik kode / kota (mis. CGK / Jakarta)"
                      options={airportDropdownOptions}
                    />
                  </div>

                  {/* Transit Kepulangan */}
                  {!formData.is_direct_flight && (
                    <div className="pt-2 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-semibold text-neutral-700 font-heading flex items-center gap-1.5">
                          <span>Transit Kepulangan</span>
                        </span>
                      </div>

                      <div className="space-y-3">
                        {(formData.transit_items_pulang || [{ bandara: '', durasi: '' }]).map((item, idx) => (
                          <div key={idx} className="flex flex-col sm:flex-row items-start sm:items-end gap-3 pb-2.5 border-b border-neutral-100 last:border-b-0">
                            <div className="flex-1 w-full grid grid-cols-1 sm:grid-cols-3 gap-3">
                              <div className="sm:col-span-2">
                                <AutocompleteInput
                                  label={`Bandara Transit ${formData.transit_items_pulang.length > 1 ? `#${idx + 1}` : ''}`}
                                  className="!mb-0"
                                  name={`transit_pulang_${idx}`}
                                  value={item.bandara}
                                  onChange={(e) => handleTransitPulangChange(idx, 'bandara', e.target.value)}
                                  onSelect={(opt) => handleTransitPulangChange(idx, 'bandara', opt?.value || '')}
                                  placeholder="Ketik kode / kota (mis. DOH / DXB)"
                                  options={airportDropdownOptions}
                                />
                              </div>
                              <div>
                                <Input
                                  label="Durasi Transit (jam)"
                                  className="!mb-0"
                                  value={item.durasi}
                                  onChange={(e) => handleTransitPulangChange(idx, 'durasi', e.target.value.replace(/\D/g, '').slice(0, 3))}
                                  inputMode="numeric"
                                  placeholder="mis. 3"
                                />
                              </div>
                            </div>

                            {(formData.transit_items_pulang || []).length > 1 && (
                              <button
                                type="button"
                                onClick={() => handleRemoveTransitPulang(idx)}
                                className="h-10 px-3 inline-flex items-center gap-1 text-xs text-danger-600 hover:text-danger-700 hover:bg-danger-50 rounded-md border border-neutral-200 transition-colors cursor-pointer shrink-0"
                                title="Hapus titik transit"
                              >
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                </svg>
                                <span className="sm:hidden">Hapus</span>
                              </button>
                            )}
                          </div>
                        ))}

                        <button
                          type="button"
                          onClick={handleAddTransitPulang}
                          className="w-full py-2 px-4 rounded-md border border-dashed border-neutral-300 hover:border-primary-400 hover:bg-neutral-50 text-neutral-700 hover:text-primary-600 font-medium text-xs flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
                          </svg>
                          <span>Tambah Transit</span>
                        </button>
                      </div>
                    </div>
                  )}
                </div>
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
                    error={fieldErrors.hotel_mekkah_id}
                      value={formData.hotel_mekkah_id}
                      onChange={(e) => handleChange({ target: { name: 'hotel_mekkah_id', value: e.target.value } })}
                      onSelect={(opt) => handleChange({ target: { name: 'hotel_mekkah_id', value: opt ? opt.value : '' } })}
                      required
                      placeholder="Ketik / cari hotel di Mekkah..."
                      options={mekkahHotels.map(h => ({ value: h.id, label: h.star_rating ? `${h.name} (${'★'.repeat(h.star_rating)})` : h.name }))}
                    />
                  </div>

                  {/* Hotel Madinah */}
                  <div>
                    <AutocompleteInput
                      label="Hotel Madinah"
                      className="!mb-0"
                      name="hotel_madinah_id"
                      value={formData.hotel_madinah_id}
                      onChange={(e) => handleChange({ target: { name: 'hotel_madinah_id', value: e.target.value } })}
                      onSelect={(opt) => handleChange({ target: { name: 'hotel_madinah_id', value: opt ? opt.value : '' } })}
                      required
                      placeholder="Ketik / cari hotel di Madinah..."
                      options={madinahHotels.map(h => ({ value: h.id, label: h.star_rating ? `${h.name} (${'★'.repeat(h.star_rating)})` : h.name }))}
                    />
                  </div>
                </div>

                {/* Hotel Transit (opsional): satu baris per hotel, tambah lewat tombol */}
                <div className="space-y-3">
                  <span className="block text-xs font-semibold text-neutral-700 font-heading">Hotel Transit (Opsional)</span>

                  {formData.transit_hotel_ids.map((thId, idx) => (
                    <div key={`th-${thId}`} className="flex items-end gap-3">
                      <div className="min-w-0 flex-1">
                        <AutocompleteInput
                          label={(formData.transit_hotel_ids.length + extraTransitHotelSlots) > 1 ? `Hotel Transit #${idx + 1}` : undefined}
                          className="!mb-0"
                          name={`transit_hotel_${idx}`}
                          value={thId}
                          onChange={(e) => handleReplaceTransitHotel(thId, e.target.value)}
                          onSelect={(opt) => handleReplaceTransitHotel(thId, opt ? opt.value : '')}
                          placeholder="Ketik / cari hotel..."
                          options={transitHotelOptionsFor(thId)}
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => handleRemoveTransitHotel(thId)}
                        className="h-10 px-3 inline-flex items-center text-danger-600 hover:text-danger-700 hover:bg-danger-50 rounded-md border border-neutral-200 transition-colors cursor-pointer shrink-0"
                        title="Hapus hotel transit"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  ))}

                  {Array.from({ length: extraTransitHotelSlots }).map((_, i) => (
                    <div key={`slot-${i}`} className="flex items-end gap-3">
                      <div className="min-w-0 flex-1">
                        <AutocompleteInput
                          label={(formData.transit_hotel_ids.length + extraTransitHotelSlots) > 1 ? `Hotel Transit #${formData.transit_hotel_ids.length + i + 1}` : undefined}
                          className="!mb-0"
                          name={`transit_hotel_new_${i}`}
                          value=""
                          onChange={(e) => { if (e.target.value) handleFillTransitHotelSlot(e.target.value); }}
                          onSelect={(opt) => { if (opt) handleFillTransitHotelSlot(opt.value); }}
                          placeholder="Ketik / cari hotel..."
                          options={transitHotelOptionsFor(null)}
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => setExtraTransitHotelSlots(count => Math.max(0, count - 1))}
                        className="h-10 px-3 inline-flex items-center text-danger-600 hover:text-danger-700 hover:bg-danger-50 rounded-md border border-neutral-200 transition-colors cursor-pointer shrink-0"
                        title="Batalkan baris ini"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  ))}

                  <button
                    type="button"
                    onClick={() => setExtraTransitHotelSlots(count => count + 1)}
                    className="w-full py-2 px-4 rounded-md border border-dashed border-neutral-300 hover:border-primary-400 hover:bg-neutral-50 text-neutral-700 hover:text-primary-600 font-medium text-xs flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
                    </svg>
                    <span>Tambah Hotel Transit</span>
                  </button>
                </div>
              </div>
            </FormSection>

            <FormSection id="sec-fasilitas" title="Fasilitas Paket">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                {/* Include */}
                <div>
                  <Textarea
                    label="Fasilitas Termasuk (Include)"
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
                    label="Tidak Termasuk (Exclude)"
                    value={excludeItemsText}
                    onChange={(e) => setExcludeItemsText(e.target.value)}
                    placeholder="Paspor&#10;Vaksin Meningitis&#10;Pengeluaran Pribadi / Laundry&#10;Kelebihan Bagasi"
                    rows={6}
                    className="!mb-0"
                  />
                </div>
              </div>
            </FormSection>

            <FormSection id="sec-itinerary" title="Itinerary">
              <div className="space-y-4">
                {/* Itinerary Dropdown */}
                <div className="w-full">
                  <CustomDropdown
                    label="Master Itinerary"
                    className="!mb-0"
                    value={formData.itinerary_id}
                    onChange={(val) => handleChange({ target: { name: 'itinerary_id', value: val } })}
                    placeholder="-- Pilih Itinerary --"
                    options={itineraryOptions
                      .filter(it => !formData.category_id || !it.category_id || String(it.category_id) === String(formData.category_id) || String(it.id) === String(formData.itinerary_id))
                      .map(it => ({ value: it.id, label: `${it.title} (${it.day_count || it.duration_days || '-'} Hari)` }))}
                  />
                </div>
              </div>
            </FormSection>


          <FormAside>
              <SidebarSection title="Publikasi">
                <div className="space-y-3">
                  <CustomDropdown
                    label="Status"
                    className="!mb-0"
                    value={formData.status}
                    onChange={(val) => handleChange({ target: { name: 'status', value: val } })}
                    options={[
                      { value: 'draft', label: 'Draft' },
                      { value: 'published', label: 'Published' },
                      { value: 'archived', label: 'Archived' }
                    ]}
                  />
                  <p className="text-xs text-neutral-500 font-body">
                    Pastikan tanggal, hotel, maskapai, dan kuota sudah sesuai sebelum menyimpan.
                  </p>
                  <div className="pt-1">
                    <Button
                      type="submit"
                      variant="primary"
                      isLoading={isSubmitting}
                      disabled={isUploadingBrosur}
                      className="w-full justify-center"
                    >
                      {isSubmitting ? 'Menyimpan...' : (isEditMode ? 'Perbarui Paket' : 'Simpan Paket')}
                    </Button>
                  </div>
                </div>
              </SidebarSection>

              <SidebarSection title="Brand & Kategori">
                <div className="space-y-4">
                  {/* Brand */}
                  <div>
                    <CustomDropdown
                      label="Brand"
                      className="!mb-0"
                      value={formData.brand_id}
                      onChange={(val) => {
                        handleChange({ target: { name: 'brand_id', value: val } });
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
                      placeholder="-- Pilih Brand --"
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
                      placeholder="-- Pilih Kategori --"
                      options={filteredCategories.map(c => ({ value: c.id, label: c.name }))}
                    />
                  </div>
                </div>
              </SidebarSection>

              <SidebarSection title="Gambar Utama">
              <div>
                {formData.brosur_url ? (
                  <div className="space-y-3">
                    <div className="border border-neutral-200 rounded-xl p-1 bg-white overflow-hidden flex justify-center shadow-2xs">
                      <img
                        src={formData.brosur_url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${formData.brosur_url}` : formData.brosur_url}
                        alt="Preview gambar utama"
                        className="max-w-full h-auto max-h-52 object-contain rounded"
                      />
                    </div>
                    <label className="flex items-center justify-center gap-2 px-4 py-2 w-full bg-white border border-neutral-300 rounded-lg text-xs font-semibold text-neutral-700 hover:bg-neutral-50 cursor-pointer shadow-2xs transition-colors text-center">
                        <span>{isUploadingBrosur ? 'Mengupload...' : 'Ganti Gambar'}</span>
                      <input type="file" className="hidden" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                    </label>
                  </div>
                ) : (
                  <div className="flex justify-center px-4 pt-6 pb-6 border-2 border-neutral-300 border-dashed rounded-xl hover:border-primary-400 transition-colors bg-neutral-50">
                    <div className="space-y-2 text-center">
                      <div className="text-xs text-neutral-600">
                        <label htmlFor="brosur-upload" className="cursor-pointer font-semibold text-primary-600 hover:text-primary-700">
                          <span>{isUploadingBrosur ? 'Mengupload...' : 'Upload Gambar Utama'}</span>
                          <input id="brosur-upload" name="brosur-upload" type="file" className="sr-only" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                        </label>
                        <p className="text-[11px] text-neutral-400 mt-0.5">JPG, PNG, WebP</p>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </SidebarSection>
            </FormAside>
        </FormLayout>
      </form>
    </div>
  );
};

export default ScheduleFormPage;

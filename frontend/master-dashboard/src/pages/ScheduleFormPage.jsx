import React, { useState, useEffect, useMemo, useRef } from 'react';
import { AlertCircle } from 'lucide-react';
import { useParams, useNavigate, useSearchParams, useLocation } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import FormField from 'shared/ui/FormField';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import CustomDropdown from 'shared/ui/CustomDropdown';
import AutocompleteInput from 'shared/ui/AutocompleteInput';
import Button from 'shared/ui/Button';
import Toggle from 'shared/ui/Toggle';
import FormLayout, { FormSection, SidebarSection, FormAside } from 'shared/ui/FormLayout';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';

import { getSchedule, createSchedule, updateSchedule } from '../api/schedules';
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
        ...(isEditMode && formData.status === formData.status_awal ? {} : { status: formData.status || 'draft' }),
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
      if (isEditMode) {
        await updateSchedule(id, payload);
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
            setFormData(prev => ({ ...prev, status_awal: prev.status, berangkat_tanggal_awal: prev.berangkat_tanggal, alasan_ubah_tanggal: '' }));
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

      <form onSubmit={handleSubmit}>
        {peringatanSimpan && (
          <Alert variant="warning" onClose={() => setPeringatanSimpan(null)} className="mb-4">{peringatanSimpan}</Alert>
        )}
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
                    <li key={field} className="text-sm leading-5 text-danger-700">
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
                  <div>
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
                    helpText="Isi 0 untuk tanpa DP."
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
                  <p className="text-xs text-neutral-500 font-body">Isi 0 untuk tanpa komisi.</p>
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

                {/* Bagian 2: Penerbangan Berangkat */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-neutral-600 text-sm font-semibold font-heading">
                    <span>Berangkat</span>
                  </div>

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
                    <Alert variant="warning">
                      {formData.komisi_aktif} komisi agen ikut bergeser tanggal cairnya. Komisi yang sudah dicairkan bisa
                      membuat saldo agen minus.
                    </Alert>
                  )}

                  {/* Baris 1: Waktu & Kode Penerbangan */}
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Input
                      label="Tanggal berangkat"
                      className="!mb-0"
                      type="date"
                      name="berangkat_tanggal"
                    error={fieldErrors.berangkat_tanggal}
                      value={formData.berangkat_tanggal}
                      onChange={handleChange}
                      required
                    />
                    <Input
                      label="Jam berangkat"
                      className="!mb-0"
                      type="text"
                      name="berangkat_jam"
                    error={fieldErrors.berangkat_jam}
                      value={formData.berangkat_jam}
                      onChange={handleTimeChange}
                      placeholder="09:00"
                    />
                    {!berangkatIsCustom && currentAirlineCode ? (
                      <div>
                        <label className="mb-1.5 block min-h-5 text-sm font-medium leading-5 text-neutral-800 font-body">
                          Kode penerbangan
                        </label>
                        <div className="flex">
                          <span className="inline-flex h-9 items-center px-3 rounded-l-lg border border-r-0 border-neutral-300 bg-neutral-50 text-neutral-700 font-mono font-semibold text-sm select-none">
                            {currentAirlineCode}
                          </span>
                          <input
                            type="text"
                            name="berangkat_flight_no"
                            value={berangkatFlightNo}
                            onChange={handleBerangkatFlightNoChange}
                            placeholder="822"
                            maxLength={5}
                            inputMode="numeric"
                            className="h-9 flex-1 min-w-0 block w-full px-3 rounded-none rounded-r-lg border border-neutral-300 focus:outline-none focus:border-primary-500 text-sm font-mono text-neutral-900 bg-white placeholder:text-neutral-400"
                          />
                        </div>
                      </div>
                    ) : (
                      <Input
                        label="Kode penerbangan"
                        className="!mb-0"
                        name="berangkat_kode_penerbangan"
                    error={fieldErrors.berangkat_kode_penerbangan}
                        value={formData.berangkat_kode_penerbangan}
                        onChange={handleChange}
                        placeholder="SV 822"
                      />
                    )}
                  </div>

                  {/* Baris 2: Bandara Asal & Tujuan */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <AutocompleteInput
                      label="Bandara asal"
                      className="!mb-0"
                      name="berangkat_bandara_asal"
                      value={formData.berangkat_bandara_asal}
                      onChange={(e) => setFormData(prev => ({ ...prev, berangkat_bandara_asal: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, berangkat_bandara_asal: opt?.value || '' }))}
                      placeholder="Cari bandara"
                      options={airportDropdownOptions}
                    />
                    <AutocompleteInput
                      label="Bandara tujuan akhir"
                      className="!mb-0"
                      name="berangkat_bandara_tujuan"
                      value={formData.berangkat_bandara_tujuan}
                      onChange={(e) => setFormData(prev => ({ ...prev, berangkat_bandara_tujuan: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, berangkat_bandara_tujuan: opt?.value || '' }))}
                      placeholder="Cari bandara"
                      options={airportDropdownOptions}
                    />
                  </div>

                  {/* Transit Keberangkatan */}
                  {!formData.is_direct_flight && (
                    <div className="pt-2 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-semibold text-neutral-700 font-heading flex items-center gap-1.5">
                          <span>Transit</span>
                        </span>
                      </div>

                      <div className="space-y-3">
                        {(formData.transit_items_berangkat || [{ bandara: '', durasi: '' }]).map((item, idx) => (
                          <div key={idx} className="flex flex-col sm:flex-row items-start sm:items-end gap-3 pb-2.5 border-b border-neutral-100 last:border-b-0">
                            <div className="flex-1 w-full grid grid-cols-1 sm:grid-cols-3 gap-3">
                              <div className="sm:col-span-2">
                                <AutocompleteInput
                                  label={`Bandara transit ${formData.transit_items_berangkat.length > 1 ? `#${idx + 1}` : ''}`}
                                  className="!mb-0"
                                  name={`transit_berangkat_${idx}`}
                                  value={item.bandara}
                                  onChange={(e) => handleTransitBerangkatChange(idx, 'bandara', e.target.value)}
                                  onSelect={(opt) => handleTransitBerangkatChange(idx, 'bandara', opt?.value || '')}
                                  placeholder="Cari bandara"
                                  options={airportDropdownOptions}
                                />
                              </div>
                              <div>
                                <Input
                                  label="Durasi (jam)"
                                  className="!mb-0"
                                  value={item.durasi}
                                  onChange={(e) => handleTransitBerangkatChange(idx, 'durasi', e.target.value.replace(/\D/g, '').slice(0, 3))}
                                  inputMode="numeric"
                                  placeholder="2"
                                />
                              </div>
                            </div>

                            {(formData.transit_items_berangkat || []).length > 1 && (
                              <Button variant="danger-outline" className="shrink-0"
                                onClick={() => handleRemoveTransitBerangkat(idx)}
                                title="Hapus"
                              >
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                </svg>
                                <span className="sm:hidden">Hapus</span>
                              </Button>
                            )}
                          </div>
                        ))}

                        <Button variant="dashed" className="w-full"
                          onClick={handleAddTransitBerangkat}
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
                          </svg>
                          <span>Tambah transit</span>
                        </Button>
                      </div>
                    </div>
                  )}
                </div>

                {/* Divider */}
                <div className="border-t border-neutral-200" />

                {/* Bagian 3: Penerbangan Pulang */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-neutral-600 text-sm font-semibold font-heading">
                    <span>Pulang</span>
                  </div>

                  {/* Baris 1: Waktu & Kode Penerbangan */}
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Input
                      label="Tanggal pulang"
                      className="!mb-0"
                      type="date"
                      name="pulang_tanggal"
                    error={fieldErrors.pulang_tanggal}
                      value={formData.pulang_tanggal}
                      onChange={handleChange}
                      required
                    />
                    <Input
                      label="Jam pulang"
                      className="!mb-0"
                      type="text"
                      name="pulang_jam"
                    error={fieldErrors.pulang_jam}
                      value={formData.pulang_jam}
                      onChange={handleTimeChange}
                      placeholder="15:30"
                    />
                    {!pulangIsCustom && currentAirlineCode ? (
                      <div>
                        <label className="mb-1.5 block min-h-5 text-sm font-medium leading-5 text-neutral-800 font-body">
                          Kode penerbangan
                        </label>
                        <div className="flex">
                          <span className="inline-flex h-9 items-center px-3 rounded-l-lg border border-r-0 border-neutral-300 bg-neutral-50 text-neutral-700 font-mono font-semibold text-sm select-none">
                            {currentAirlineCode}
                          </span>
                          <input
                            type="text"
                            name="pulang_flight_no"
                            value={pulangFlightNo}
                            onChange={handlePulangFlightNoChange}
                            placeholder="823"
                            maxLength={5}
                            inputMode="numeric"
                            className="h-9 flex-1 min-w-0 block w-full px-3 rounded-none rounded-r-lg border border-neutral-300 focus:outline-none focus:border-primary-500 text-sm font-mono text-neutral-900 bg-white placeholder:text-neutral-400"
                          />
                        </div>
                      </div>
                    ) : (
                      <Input
                        label="Kode penerbangan"
                        className="!mb-0"
                        name="pulang_kode_penerbangan"
                    error={fieldErrors.pulang_kode_penerbangan}
                        value={formData.pulang_kode_penerbangan}
                        onChange={handleChange}
                        placeholder="SV 823"
                      />
                    )}
                  </div>

                  {/* Baris 2: Bandara Asal & Tujuan */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <AutocompleteInput
                      label="Bandara asal"
                      className="!mb-0"
                      name="pulang_bandara_asal"
                      value={formData.pulang_bandara_asal}
                      onChange={(e) => setFormData(prev => ({ ...prev, pulang_bandara_asal: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, pulang_bandara_asal: opt?.value || '' }))}
                      placeholder="Cari bandara"
                      options={airportDropdownOptions}
                    />
                    <AutocompleteInput
                      label="Bandara tujuan akhir"
                      className="!mb-0"
                      name="pulang_bandara_tujuan"
                      value={formData.pulang_bandara_tujuan}
                      onChange={(e) => setFormData(prev => ({ ...prev, pulang_bandara_tujuan: e.target.value }))}
                      onSelect={(opt) => setFormData(prev => ({ ...prev, pulang_bandara_tujuan: opt?.value || '' }))}
                      placeholder="Cari bandara"
                      options={airportDropdownOptions}
                    />
                  </div>

                  {/* Transit Kepulangan */}
                  {!formData.is_direct_flight && (
                    <div className="pt-2 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-semibold text-neutral-700 font-heading flex items-center gap-1.5">
                          <span>Transit</span>
                        </span>
                      </div>

                      <div className="space-y-3">
                        {(formData.transit_items_pulang || [{ bandara: '', durasi: '' }]).map((item, idx) => (
                          <div key={idx} className="flex flex-col sm:flex-row items-start sm:items-end gap-3 pb-2.5 border-b border-neutral-100 last:border-b-0">
                            <div className="flex-1 w-full grid grid-cols-1 sm:grid-cols-3 gap-3">
                              <div className="sm:col-span-2">
                                <AutocompleteInput
                                  label={`Bandara transit ${formData.transit_items_pulang.length > 1 ? `#${idx + 1}` : ''}`}
                                  className="!mb-0"
                                  name={`transit_pulang_${idx}`}
                                  value={item.bandara}
                                  onChange={(e) => handleTransitPulangChange(idx, 'bandara', e.target.value)}
                                  onSelect={(opt) => handleTransitPulangChange(idx, 'bandara', opt?.value || '')}
                                  placeholder="Cari bandara"
                                  options={airportDropdownOptions}
                                />
                              </div>
                              <div>
                                <Input
                                  label="Durasi (jam)"
                                  className="!mb-0"
                                  value={item.durasi}
                                  onChange={(e) => handleTransitPulangChange(idx, 'durasi', e.target.value.replace(/\D/g, '').slice(0, 3))}
                                  inputMode="numeric"
                                  placeholder="3"
                                />
                              </div>
                            </div>

                            {(formData.transit_items_pulang || []).length > 1 && (
                              <Button variant="danger-outline" className="shrink-0"
                                onClick={() => handleRemoveTransitPulang(idx)}
                                title="Hapus"
                              >
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                </svg>
                                <span className="sm:hidden">Hapus</span>
                              </Button>
                            )}
                          </div>
                        ))}

                        <Button variant="dashed" className="w-full"
                          onClick={handleAddTransitPulang}
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
                          </svg>
                          <span>Tambah transit</span>
                        </Button>
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
                    <span>Tambah hotel transit</span>
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
                  options={[
                    { value: 'draft', label: 'Draft' },
                    { value: 'published', label: 'Published' },
                    { value: 'archived', label: 'Archived' }
                  ]}
                />
              </SidebarSection>

              <SidebarSection title="Gambar Utama">
              <div>
                {formData.brosur_url ? (
                  <div className="space-y-3">
                    <div className="border border-neutral-200 rounded-xl p-1 bg-white overflow-hidden flex justify-center">
                      <img
                        src={formData.brosur_url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${formData.brosur_url}` : formData.brosur_url}
                        alt="Preview gambar utama"
                        className="max-w-full h-auto max-h-52 object-contain rounded"
                      />
                    </div>
                    <label className="flex items-center justify-center gap-2 px-4 py-2 w-full bg-white border border-neutral-300 rounded-lg text-xs font-semibold text-neutral-700 hover:bg-neutral-50 cursor-pointer transition-colors text-center">
                        <span>{isUploadingBrosur ? 'Mengunggah...' : 'Ganti gambar'}</span>
                      <input type="file" className="hidden" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                    </label>
                  </div>
                ) : (
                  <div className="flex justify-center px-4 pt-6 pb-6 border-2 border-neutral-300 border-dashed rounded-xl hover:border-primary-400 transition-colors bg-neutral-50">
                    <div className="space-y-2 text-center">
                      <div className="text-xs text-neutral-600">
                        <label htmlFor="brosur-upload" className="cursor-pointer font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900">
                          <span>{isUploadingBrosur ? 'Mengunggah...' : 'Unggah gambar'}</span>
                          <input id="brosur-upload" name="brosur-upload" type="file" className="sr-only" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                        </label>
                        <p className="text-xs text-neutral-400 mt-0.5">JPG, PNG, WebP</p>
                      </div>
                    </div>
                  </div>
                )}
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
                {isSubmitting ? 'Menyimpan...' : 'Simpan'}
              </Button>
            </SidebarSection>
            </FormAside>
        </FormLayout>
      </form>
    </div>
  );
};

export default ScheduleFormPage;

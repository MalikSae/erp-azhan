import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import Textarea from '../components/ui/Textarea';
import CustomDropdown from '../components/ui/CustomDropdown';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import MetaBox from '../components/ui/MetaBox';
import Badge from '../components/ui/Badge';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import Toggle from '../components/ui/Toggle';

import { getSchedule, createSchedule, updateSchedule } from '../api/schedules';
import { listHotels } from '../api/hotels';
import { listAirlines } from '../api/airlines';
import { listCategories } from '../api/categories';
import { listItineraries } from '../api/itineraries';
import { listBrands } from '../api/brands';
import { listAddOns } from '../api/addons';
import { uploadMediaWithOptions } from '../api/media';

const ScheduleFormPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEditMode = !!id;

  const [isLoadingInitial, setIsLoadingInitial] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isUploadingBrosur, setIsUploadingBrosur] = useState(false);
  const [formError, setFormError] = useState(null);
  const [notFound, setNotFound] = useState(false);

  // Lookups
  const [hotelOptions, setHotelOptions] = useState([]);
  const [airlineOptions, setAirlineOptions] = useState([]);
  const [categoryOptions, setCategoryOptions] = useState([]);
  const [itineraryOptions, setItineraryOptions] = useState([]);
  const [brandOptions, setBrandOptions] = useState([]);
  const [addOnOptions, setAddOnOptions] = useState([]);

  const [includeItemsText, setIncludeItemsText] = useState('');
  const [excludeItemsText, setExcludeItemsText] = useState('');

  const [berangkatFlightNo, setBerangkatFlightNo] = useState('');
  const [berangkatIsCustom, setBerangkatIsCustom] = useState(false);
  const [pulangFlightNo, setPulangFlightNo] = useState('');
  const [pulangIsCustom, setPulangIsCustom] = useState(false);

  const [formData, setFormData] = useState({
    jadwal_nama: '',
    is_promo: false,
    is_ticket_confirmed: false,
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
    harga_coret: '',
    itinerary_id: '',
    include_items: [''],
    exclude_items: [''],
    add_on_ids: [],
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
          listAddOns()
        ];

        const results = await Promise.all(promises);
        const hotels = results[0];
        const airlines = results[1];
        const categories = results[2];
        const itineraries = results[3];
        const brands = results[4] || [];
        const addons = results[5] || [];

        setHotelOptions(hotels || []);
        setAirlineOptions(airlines || []);
        setCategoryOptions(categories || []);
        setItineraryOptions(itineraries || []);
        setBrandOptions(brands || []);
        setAddOnOptions(addons || []);

        if (isEditMode) {
          const scheduleData = await getSchedule(id);
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
                  return { bandara: p[0].trim(), durasi: p.slice(1).join(',').trim() };
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

          setFormData({
            jadwal_nama: scheduleData.jadwal_nama || '',
            is_promo: scheduleData.is_promo || false,
            is_ticket_confirmed: scheduleData.is_ticket_confirmed || false,
            is_direct_flight: scheduleData.is_direct_flight || false,
            seat_total: scheduleData.seat_total || '',
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
            harga_infant: scheduleData.harga_infant || '',
            harga_coret: scheduleData.harga_coret || '',
            itinerary_id: scheduleData.itinerary?.id || scheduleData.itinerary_id || '',
            include_items: safeArray(scheduleData.include_items),
            exclude_items: safeArray(scheduleData.exclude_items),
            add_on_ids: scheduleData.add_ons ? scheduleData.add_ons.map(a => a.id) : [],
            brosur_url: scheduleData.brosur_url || '',
            brosur_thumb_url: scheduleData.brosur_thumb_url || '',
            status: scheduleData.status || 'draft',
            brand_id: scheduleData.brand_id || '',
            category_id: scheduleData.category_id || (scheduleData.category?.id || '')
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
  }, [id, isEditMode]);

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => {
      const updates = { [name]: type === 'checkbox' ? checked : value };
      if (name === 'is_promo' && !checked) {
        updates.harga_coret = '';
      }
      return { ...prev, ...updates };
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



  const handleAddOnCheckboxChange = (id, checked) => {
    setFormData(prev => {
      const currentIds = [...prev.add_on_ids];
      if (checked) {
        if (!currentIds.includes(id)) currentIds.push(id);
      } else {
        const index = currentIds.indexOf(id);
        if (index > -1) currentIds.splice(index, 1);
      }
      return { ...prev, add_on_ids: currentIds };
    });
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
      setFormError(err.response?.data?.error || 'Gagal mengupload brosur.');
    } finally {
      setIsUploadingBrosur(false);
      e.target.value = ''; // Reset input
    }
  };

  const handleAddTransitHotel = (hotelId) => {
    if (!hotelId) return;
    const idNum = parseInt(hotelId, 10);
    if (formData.transit_hotel_ids.includes(idNum)) return;
    setFormData(prev => ({
      ...prev,
      transit_hotel_ids: [...prev.transit_hotel_ids, idNum]
    }));
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

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    setFormError(null);

    try {
      const hargaQuad = parseCurrency(formData.harga_quad);
      const hargaTriple = parseCurrency(formData.harga_triple);
      const hargaDouble = parseCurrency(formData.harga_double);
      const hargaInfant = formData.harga_infant ? parseCurrency(formData.harga_infant) : null;
      const hargaCoret = formData.is_promo && formData.harga_coret
        ? parseCurrency(formData.harga_coret)
        : null;

      if (hargaCoret !== null && hargaCoret <= hargaQuad) {
        setFormError('Harga coret harus lebih besar dari Harga Quad karena merupakan harga sebelum promo.');
        return;
      }

      const validBerangkat = (formData.transit_items_berangkat || [])
        .map(t => ({ bandara: t.bandara?.trim() || '', durasi: t.durasi?.trim() || '' }))
        .filter(t => t.bandara || t.durasi);

      const validPulang = (formData.transit_items_pulang || [])
        .map(t => ({ bandara: t.bandara?.trim() || '', durasi: t.durasi?.trim() || '' }))
        .filter(t => t.bandara || t.durasi);

      let transitParts = [];
      if (validBerangkat.length > 0) {
        const strB = validBerangkat.map(t => t.durasi ? `${t.bandara}, ${t.durasi}` : t.bandara).join('; ');
        transitParts.push(`Berangkat: ${strB}`);
      }
      if (validPulang.length > 0) {
        const strP = validPulang.map(t => t.durasi ? `${t.bandara}, ${t.durasi}` : t.bandara).join('; ');
        transitParts.push(`Pulang: ${strP}`);
      }

      const finalTransit = transitParts.join(' | ');

      const finalBerangkatKode = (!berangkatIsCustom && currentAirlineCode)
        ? (currentAirlineCode + berangkatFlightNo)
        : (formData.berangkat_kode_penerbangan || '').trim();

      const finalPulangKode = (!pulangIsCustom && currentAirlineCode)
        ? (currentAirlineCode + pulangFlightNo)
        : (formData.pulang_kode_penerbangan || '').trim();

      const payload = {
        ...formData,
        berangkat_kode_penerbangan: finalBerangkatKode,
        pulang_kode_penerbangan: finalPulangKode,
        transit_bandara: finalTransit,
        is_promo: !!formData.is_promo,
        is_ticket_confirmed: !!formData.is_ticket_confirmed,
        is_direct_flight: !!formData.is_direct_flight,
        seat_total: parseInt(formData.seat_total, 10),
        seat_sisa: parseInt(formData.seat_total, 10) - parseInt(formData.kuota_terisi || 0, 10),
        maskapai_id: parseInt(formData.maskapai_id, 10),
        hotel_mekkah_id: parseInt(formData.hotel_mekkah_id, 10),
        hotel_madinah_id: parseInt(formData.hotel_madinah_id, 10),
        transit_hotel_ids: (formData.transit_hotel_ids || []).map(Number),
        harga_quad: hargaQuad,
        harga_triple: hargaTriple,
        harga_double: hargaDouble,
        harga_infant: hargaInfant,
        harga_coret: hargaCoret,
        status: formData.status,
        itinerary_id: formData.itinerary_id ? parseInt(formData.itinerary_id, 10) : null,
        include_items: includeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        exclude_items: excludeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        add_on_ids: formData.add_on_ids,
        brand_id: parseInt(formData.brand_id, 10),
        category_id: formData.category_id ? parseInt(formData.category_id, 10) : null
      };

      if (isEditMode) {
        await updateSchedule(id, payload);
      } else {
        await createSchedule(payload);
      }
      navigate('/schedules');
    } catch (err) {
      setFormError(err.response?.data?.error || `Gagal ${isEditMode ? 'mengupdate' : 'menyimpan'} paket.`);
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
      <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
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
    <div className="space-y-6 pb-20">
      <PageHeader
        title={isEditMode ? 'Edit Paket' : 'Tambah Paket Baru'}
        onBack={() => navigate('/schedules')}
      />

      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 md:grid-cols-[1fr_320px] gap-6 md:items-start">

          {/* Kolom KIRI (Konten Utama) */}
          <div className="space-y-6">

            {/* 1. SECTION: INFORMASI DASAR & KUOTA */}
            <MetaBox
              title="Informasi Dasar"
              icon={
                <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
                </svg>
              }
            >
              <div className="space-y-4">
                {/* Input Nama Paket */}
                <Input
                  label="Nama Paket"
                  type="text"
                  name="jadwal_nama"
                  value={formData.jadwal_nama}
                  onChange={handleChange}
                  required
                  placeholder="mis. Umroh Reguler 9 Hari"
                  prefixIcon={
                    <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
                    </svg>
                  }
                />

                {/* Grid Alokasi Kuota */}
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  {/* Total Kuota */}
                  <div>
                    <Input
                      label="Total Kuota"
                      className="!mb-0"
                      type="number"
                      name="seat_total"
                      value={formData.seat_total}
                      onChange={handleChange}
                      required
                      min="1"
                      placeholder="mis. 45"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                        </svg>
                      }
                    />
                  </div>

                  {/* Kuota Terisi */}
                  <div>
                    <Input
                      label="Kuota Terisi"
                      className="!mb-0"
                      type="number"
                      name="kuota_terisi"
                      value={formData.kuota_terisi}
                      onChange={handleChange}
                      min="0"
                      placeholder="mis. 10"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                      }
                    />
                    {parseInt(formData.kuota_terisi || 0) > parseInt(formData.seat_total || 0) && (
                      <p className="text-xs text-danger-600 mt-1">Melebihi total kuota!</p>
                    )}
                  </div>

                  {/* Sisa Kuota */}
                  <div>
                    <Input
                      label="Sisa Kuota"
                      className="!mb-0"
                      type="text"
                      value={formData.seat_total ? `${Math.max(0, parseInt(formData.seat_total) - (parseInt(formData.kuota_terisi) || 0))} Kursi` : '-'}
                      readOnly
                      placeholder="Otomatis"
                      prefixIcon={
                        <svg className="w-4 h-4 text-success-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7" />
                        </svg>
                      }
                    />
                  </div>
                </div>
              </div>
            </MetaBox>

            {/* 2. SECTION: PENERBANGAN & RUTE */}
            <MetaBox
              title="Penerbangan & Maskapai"
              icon={
                <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
              }
            >
              <div className="space-y-6">
                {/* Bagian 1: Data Maskapai */}
                <div className="flex flex-col lg:flex-row lg:items-end justify-between gap-5">
                  <div className="flex-1 min-w-[240px]">
                    <CustomDropdown
                      label="Maskapai"
                      className="!mb-0"
                      value={formData.maskapai_id}
                      onChange={(val) => handleMaskapaiChange(val)}
                      required
                      placeholder="-- Pilih Maskapai --"
                      options={airlineOptions.map(a => ({ value: a.id, label: a.code ? `${a.name} (${a.code})` : a.name }))}
                      prefixIcon={
                        <svg className="w-4 h-4 text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                        </svg>
                      }
                    />
                  </div>

                  <div className="flex flex-wrap items-center gap-6 pb-1">
                    <div className="flex items-center gap-2">
                      <Toggle
                        id="is_direct_flight"
                        name="is_direct_flight"
                        checked={formData.is_direct_flight}
                        onChange={handleChange}
                      />
                      <label htmlFor="is_direct_flight" className="text-sm font-medium text-neutral-700 font-body cursor-pointer select-none">
                        Penerbangan Direct
                      </label>
                    </div>

                    <div className="flex items-center gap-2">
                      <Toggle
                        id="is_ticket_confirmed"
                        name="is_ticket_confirmed"
                        checked={formData.is_ticket_confirmed}
                        onChange={handleChange}
                      />
                      <label htmlFor="is_ticket_confirmed" className="text-sm font-medium text-neutral-700 font-body cursor-pointer select-none">
                        Tiket Confirmed
                      </label>
                    </div>
                  </div>
                </div>

                {/* Divider */}
                <div className="border-t border-neutral-200" />

                {/* Bagian 2: Penerbangan Berangkat */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-indigo-700 text-xs font-bold font-heading uppercase tracking-wider">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
                    </svg>
                    <span>Penerbangan Berangkat</span>
                  </div>

                  {/* Baris 1: Waktu & Kode Penerbangan */}
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Input
                      label="Tanggal Berangkat"
                      className="!mb-0"
                      type="date"
                      name="berangkat_tanggal"
                      value={formData.berangkat_tanggal}
                      onChange={handleChange}
                      required
                    />
                    <Input
                      label="Jam Berangkat"
                      className="!mb-0"
                      type="text"
                      name="berangkat_jam"
                      value={formData.berangkat_jam}
                      onChange={handleTimeChange}
                      placeholder="mis. 09:00"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                      }
                    />
                    {!berangkatIsCustom && currentAirlineCode ? (
                      <div className="space-y-1">
                        <label className="block text-sm font-medium text-neutral-700 font-body">
                          Kode Penerbangan <span className="text-danger-600">*</span>
                        </label>
                        <div className="flex rounded-md shadow-sm">
                          <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-neutral-300 bg-neutral-100 text-neutral-700 font-mono font-bold text-sm select-none">
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
                            className="flex-1 min-w-0 block w-full px-3 py-2 rounded-none rounded-r-md border border-neutral-300 focus:ring-primary-500 focus:border-primary-500 text-sm font-mono text-neutral-900 bg-white placeholder-neutral-400"
                            required
                          />
                        </div>
                      </div>
                    ) : (
                      <Input
                        label="Kode Penerbangan"
                        className="!mb-0"
                        name="berangkat_kode_penerbangan"
                        value={formData.berangkat_kode_penerbangan}
                        onChange={handleChange}
                        placeholder="mis. SV 822"
                        required
                        prefixIcon={
                          <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                          </svg>
                        }
                      />
                    )}
                  </div>

                  {/* Baris 2: Bandara Asal & Tujuan */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <Input
                      label="Bandara Asal"
                      className="!mb-0"
                      name="berangkat_bandara_asal"
                      value={formData.berangkat_bandara_asal}
                      onChange={handleChange}
                      placeholder="mis. CGK (Jakarta)"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                      }
                    />
                    <Input
                      label="Bandara Tujuan Akhir"
                      className="!mb-0"
                      name="berangkat_bandara_tujuan"
                      value={formData.berangkat_bandara_tujuan}
                      onChange={handleChange}
                      placeholder="mis. JED (Jeddah)"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                      }
                    />
                  </div>

                  {/* Transit Keberangkatan */}
                  {!formData.is_direct_flight && (
                    <div className="pt-2 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-amber-800 font-heading flex items-center gap-1.5 uppercase tracking-wider">
                          <svg className="w-3.5 h-3.5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
                          </svg>
                          <span>Transit Keberangkatan</span>
                        </span>
                      </div>

                      <div className="space-y-3">
                        {(formData.transit_items_berangkat || [{ bandara: '', durasi: '' }]).map((item, idx) => (
                          <div key={idx} className="flex flex-col sm:flex-row items-start sm:items-end gap-3 pb-2.5 border-b border-neutral-100 last:border-b-0">
                            <div className="flex-1 w-full grid grid-cols-1 sm:grid-cols-3 gap-3">
                              <div className="sm:col-span-2">
                                <Input
                                  label={`Bandara Transit ${formData.transit_items_berangkat.length > 1 ? `#${idx + 1}` : ''}`}
                                  className="!mb-0"
                                  value={item.bandara}
                                  onChange={(e) => handleTransitBerangkatChange(idx, 'bandara', e.target.value)}
                                  placeholder="mis. KUL atau DOH"
                                  prefixIcon={
                                    <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                                    </svg>
                                  }
                                />
                              </div>
                              <div>
                                <Input
                                  label="Durasi Transit"
                                  className="!mb-0"
                                  value={item.durasi}
                                  onChange={(e) => handleTransitBerangkatChange(idx, 'durasi', e.target.value)}
                                  placeholder="mis. 2 Jam"
                                  prefixIcon={
                                    <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                                    </svg>
                                  }
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
                  <div className="flex items-center gap-2 text-emerald-700 text-xs font-bold font-heading uppercase tracking-wider">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
                    </svg>
                    <span>Penerbangan Pulang</span>
                  </div>

                  {/* Baris 1: Waktu & Kode Penerbangan */}
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Input
                      label="Tanggal Pulang"
                      className="!mb-0"
                      type="date"
                      name="pulang_tanggal"
                      value={formData.pulang_tanggal}
                      onChange={handleChange}
                      required
                    />
                    <Input
                      label="Jam Pulang"
                      className="!mb-0"
                      type="text"
                      name="pulang_jam"
                      value={formData.pulang_jam}
                      onChange={handleTimeChange}
                      placeholder="mis. 15:30"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                      }
                    />
                    {!pulangIsCustom && currentAirlineCode ? (
                      <div className="space-y-1">
                        <label className="block text-sm font-medium text-neutral-700 font-body">
                          Kode Penerbangan <span className="text-danger-600">*</span>
                        </label>
                        <div className="flex rounded-md shadow-sm">
                          <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-neutral-300 bg-neutral-100 text-neutral-700 font-mono font-bold text-sm select-none">
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
                            className="flex-1 min-w-0 block w-full px-3 py-2 rounded-none rounded-r-md border border-neutral-300 focus:ring-primary-500 focus:border-primary-500 text-sm font-mono text-neutral-900 bg-white placeholder-neutral-400"
                            required
                          />
                        </div>
                      </div>
                    ) : (
                      <Input
                        label="Kode Penerbangan"
                        className="!mb-0"
                        name="pulang_kode_penerbangan"
                        value={formData.pulang_kode_penerbangan}
                        onChange={handleChange}
                        placeholder="mis. SV 823"
                        required
                        prefixIcon={
                          <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                          </svg>
                        }
                      />
                    )}
                  </div>

                  {/* Baris 2: Bandara Asal & Tujuan */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <Input
                      label="Bandara Asal"
                      className="!mb-0"
                      name="pulang_bandara_asal"
                      value={formData.pulang_bandara_asal}
                      onChange={handleChange}
                      placeholder="mis. MED (Madinah)"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                      }
                    />
                    <Input
                      label="Bandara Tujuan Akhir"
                      className="!mb-0"
                      name="pulang_bandara_tujuan"
                      value={formData.pulang_bandara_tujuan}
                      onChange={handleChange}
                      placeholder="mis. CGK (Jakarta)"
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                      }
                    />
                  </div>

                  {/* Transit Kepulangan */}
                  {!formData.is_direct_flight && (
                    <div className="pt-2 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-amber-800 font-heading flex items-center gap-1.5 uppercase tracking-wider">
                          <svg className="w-3.5 h-3.5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
                          </svg>
                          <span>Transit Kepulangan</span>
                        </span>
                      </div>

                      <div className="space-y-3">
                        {(formData.transit_items_pulang || [{ bandara: '', durasi: '' }]).map((item, idx) => (
                          <div key={idx} className="flex flex-col sm:flex-row items-start sm:items-end gap-3 pb-2.5 border-b border-neutral-100 last:border-b-0">
                            <div className="flex-1 w-full grid grid-cols-1 sm:grid-cols-3 gap-3">
                              <div className="sm:col-span-2">
                                <Input
                                  label={`Bandara Transit ${formData.transit_items_pulang.length > 1 ? `#${idx + 1}` : ''}`}
                                  className="!mb-0"
                                  value={item.bandara}
                                  onChange={(e) => handleTransitPulangChange(idx, 'bandara', e.target.value)}
                                  placeholder="mis. DOH atau DXB"
                                  prefixIcon={
                                    <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                                    </svg>
                                  }
                                />
                              </div>
                              <div>
                                <Input
                                  label="Durasi Transit"
                                  className="!mb-0"
                                  value={item.durasi}
                                  onChange={(e) => handleTransitPulangChange(idx, 'durasi', e.target.value)}
                                  placeholder="mis. 3 Jam"
                                  prefixIcon={
                                    <svg className="w-4 h-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                                    </svg>
                                  }
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
            </MetaBox>

            {/* 3. SECTION: AKOMODASI HOTEL */}
            <MetaBox
              title="Akomodasi Hotel"
              icon={
                <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                </svg>
              }
            >
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {/* Hotel Mekkah */}
                  <div>
                    <CustomDropdown
                      label="Hotel Mekkah"
                      className="!mb-0"
                      value={formData.hotel_mekkah_id}
                      onChange={(val) => handleChange({ target: { name: 'hotel_mekkah_id', value: val } })}
                      required
                      placeholder="-- Pilih Hotel --"
                      options={mekkahHotels.map(h => ({ value: h.id, label: `${h.name} (${h.star_rating ? `${h.star_rating} Bintang` : 'Hotel'})` }))}
                      prefixIcon={
                        <svg className="w-4 h-4 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                      }
                    />
                  </div>

                  {/* Hotel Madinah */}
                  <div>
                    <CustomDropdown
                      label="Hotel Madinah"
                      className="!mb-0"
                      value={formData.hotel_madinah_id}
                      onChange={(val) => handleChange({ target: { name: 'hotel_madinah_id', value: val } })}
                      required
                      placeholder="-- Pilih Hotel --"
                      options={madinahHotels.map(h => ({ value: h.id, label: `${h.name} (${h.star_rating ? `${h.star_rating} Bintang` : 'Hotel'})` }))}
                      prefixIcon={
                        <svg className="w-4 h-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                      }
                    />
                  </div>
                </div>

                {/* Hotel Transit (opsional) */}
                <div className="border-t border-neutral-200 pt-3 space-y-3">
                  <label className="block text-xs font-semibold text-neutral-700 font-heading">
                    Hotel Transit (Opsional)
                  </label>

                  {/* Selector Dropdown */}
                  <div className="w-full sm:w-2/3">
                    <CustomDropdown
                      className="!mb-0"
                      value=""
                      onChange={(val) => handleAddTransitHotel(val)}
                      placeholder="Tambah Hotel Transit..."
                      options={hotelOptions
                        .filter(h => 
                          h.id !== parseInt(formData.hotel_mekkah_id, 10) &&
                          h.id !== parseInt(formData.hotel_madinah_id, 10) &&
                          !formData.transit_hotel_ids.includes(h.id)
                        )
                        .map(h => ({ 
                          value: h.id, 
                          label: `${h.name} — ${h.city || 'Lainnya'} (${h.star_rating ? `${h.star_rating}★` : 'Hotel'})` 
                        }))
                      }
                      prefixIcon={
                        <svg className="w-4 h-4 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
                        </svg>
                      }
                    />
                  </div>

                  {/* Chips list of selected transit hotels */}
                  {formData.transit_hotel_ids.length > 0 && (
                    <div className="flex flex-wrap gap-2 pt-1">
                      {formData.transit_hotel_ids.map((thId, idx) => {
                        const hotel = hotelOptions.find(h => h.id === thId);
                        const hotelName = hotel ? hotel.name : `Hotel #${thId}`;
                        const hotelCity = hotel ? hotel.city : '';
                        const hotelStar = hotel?.star_rating ? `${hotel.star_rating}★` : null;

                        return (
                          <span
                            key={thId}
                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-neutral-100 border border-neutral-200 text-xs font-medium text-neutral-800 shadow-2xs"
                          >
                            <span className="w-4 h-4 rounded-full bg-neutral-200 text-[10px] font-bold flex items-center justify-center text-neutral-600">
                              {idx + 1}
                            </span>
                            <span className="font-semibold">{hotelName}</span>
                            {hotelCity && (
                              <span className="text-neutral-500">({hotelCity})</span>
                            )}
                            {hotelStar && (
                              <span className="text-amber-600 font-semibold">{hotelStar}</span>
                            )}
                            <button
                              type="button"
                              onClick={() => handleRemoveTransitHotel(thId)}
                              className="ml-1 text-neutral-400 hover:text-danger-600 transition-colors cursor-pointer p-0.5 rounded"
                              title="Hapus hotel transit"
                            >
                              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                              </svg>
                            </button>
                          </span>
                        );
                      })}
                    </div>
                  )}
                </div>
              </div>
            </MetaBox>

            {/* 4. SECTION: HARGA PAKET & PROMO */}
            <MetaBox
              title="Harga Paket & Promo"
              icon={
                <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              }
            >
              <div className="space-y-4">
                {/* 4 Kolom Tipe Kamar & Infant */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <Input
                    label="Harga Quad (Ber-4)"
                    className="!mb-0"
                    type="text"
                    name="harga_quad"
                    value={formatCurrency(formData.harga_quad)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="mis. 25.000.000"
                    prefixIcon={<span className="text-xs font-bold text-neutral-500">Rp</span>}
                  />
                  <Input
                    label="Harga Triple (Ber-3)"
                    className="!mb-0"
                    type="text"
                    name="harga_triple"
                    value={formatCurrency(formData.harga_triple)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="mis. 26.000.000"
                    prefixIcon={<span className="text-xs font-bold text-neutral-500">Rp</span>}
                  />
                  <Input
                    label="Harga Double (Ber-2)"
                    className="!mb-0"
                    type="text"
                    name="harga_double"
                    value={formatCurrency(formData.harga_double)}
                    onChange={handleCurrencyChange}
                    required
                    placeholder="mis. 28.000.000"
                    prefixIcon={<span className="text-xs font-bold text-neutral-500">Rp</span>}
                  />
                  <Input
                    label="Harga Infant (Bayi)"
                    className="!mb-0"
                    type="text"
                    name="harga_infant"
                    value={formatCurrency(formData.harga_infant)}
                    onChange={handleCurrencyChange}
                    placeholder="mis. 5.000.000"
                    prefixIcon={<span className="text-xs font-bold text-neutral-500">Rp</span>}
                  />
                </div>

                {/* Promo Divider & Row */}
                <div className="border-t border-neutral-200 pt-3">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                      <Toggle
                        id="is_promo"
                        name="is_promo"
                        checked={formData.is_promo}
                        onChange={handleChange}
                      />
                      <label htmlFor="is_promo" className="text-sm font-medium text-neutral-800 font-body cursor-pointer select-none">
                        Paket Promo
                      </label>
                    </div>

                    {formData.is_promo && (
                      <div className="w-full sm:w-64">
                        <Input
                          label="Harga Coret"
                          className="!mb-0"
                          type="text"
                          name="harga_coret"
                          value={formatCurrency(formData.harga_coret)}
                          onChange={handleCurrencyChange}
                          placeholder="mis. 27.500.000"
                          prefixIcon={<span className="text-xs font-bold text-amber-700">Rp</span>}
                        />
                      </div>
                    )}
                  </div>

                  {formData.is_promo && formData.harga_coret && parseCurrency(formData.harga_coret) <= parseCurrency(formData.harga_quad) && (
                    <p className="text-xs text-danger-600 font-medium mt-2">Harga coret harus lebih tinggi dari Harga Quad.</p>
                  )}
                </div>
              </div>
            </MetaBox>

            {/* 5. SECTION: FASILITAS INCLUDE & EXCLUDE */}
            <MetaBox
              title="Fasilitas Paket"
              subtitle="Tuliskan 1 item per baris"
              icon={
                <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                </svg>
              }
            >
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
            </MetaBox>

            {/* 6. SECTION: ITINERARY & ADD-ON */}
            <MetaBox
              title="Itinerary & Add-On"
              icon={
                <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
                </svg>
              }
            >
              <div className="space-y-4">
                {/* Itinerary Dropdown */}
                <div className="w-full sm:w-1/2">
                  <CustomDropdown
                    label="Master Itinerary"
                    className="!mb-0"
                    value={formData.itinerary_id}
                    onChange={(val) => handleChange({ target: { name: 'itinerary_id', value: val } })}
                    placeholder="-- Pilih Itinerary --"
                    options={itineraryOptions.map(it => ({ value: it.id, label: `${it.title} (${it.duration_days || '-'} Hari)` }))}
                    prefixIcon={
                      <svg className="w-4 h-4 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
                      </svg>
                    }
                  />
                </div>

                {/* Divider */}
                <div className="border-t border-neutral-200 pt-2" />

                {/* Add-ons Checkbox List */}
                <div className="space-y-2">
                  <label className="block text-xs font-semibold text-neutral-700 font-heading">
                    Layanan Tambahan (Add-On)
                  </label>
                  {addOnOptions.length > 0 ? (
                    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
                      {addOnOptions.map(addon => {
                        const isChecked = formData.add_on_ids.includes(addon.id);
                        return (
                          <label
                            key={addon.id}
                            className="flex items-center gap-2.5 cursor-pointer select-none py-1 group"
                          >
                            <input
                              type="checkbox"
                              checked={isChecked}
                              onChange={(e) => handleAddOnCheckboxChange(addon.id, e.target.checked)}
                              className="w-4 h-4 text-primary-600 border-neutral-300 rounded focus:ring-primary-500 cursor-pointer"
                            />
                            <span className="text-sm text-neutral-700 font-body group-hover:text-neutral-900 transition-colors">
                              {addon.name}
                            </span>
                          </label>
                        );
                      })}
                    </div>
                  ) : (
                    <p className="text-xs text-neutral-500 italic">Belum ada data add-on.</p>
                  )}
                </div>
              </div>
            </MetaBox>

            {formError && (
              <Alert variant="error">{formError}</Alert>
            )}

          </div>

          {/* Kolom KANAN (Sidebar Form) */}
          <div className="flex flex-col gap-6">

            {/* MetaBox: Brand & Kategori */}
            <div>
              <MetaBox
                title="Brand & Kategori"
                icon={
                  <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                  </svg>
                }
              >
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
                      prefixIcon={
                        <svg className="w-4 h-4 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                        </svg>
                      }
                    />
                  </div>

                  {/* Kategori */}
                  <div>
                    <CustomDropdown
                      label="Kategori"
                      className="!mb-0"
                      value={formData.category_id}
                      onChange={(val) => handleChange({ target: { name: 'category_id', value: val } })}
                      placeholder="-- Pilih Kategori --"
                      options={filteredCategories.map(c => ({ value: c.id, label: c.name }))}
                      prefixIcon={
                        <svg className="w-4 h-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
                        </svg>
                      }
                    />
                  </div>
                </div>
              </MetaBox>
            </div>

            {/* MetaBox: Brosur Paket (Image) */}
            <div>
              <MetaBox
                title="Brosur Paket"
                icon={
                  <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                }
              >
                {formData.brosur_url ? (
                  <div className="space-y-3">
                    <div className="border border-neutral-200 rounded-md p-1 bg-neutral-50 overflow-hidden flex justify-center shadow-2xs">
                      <img
                        src={formData.brosur_url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${formData.brosur_url}` : formData.brosur_url}
                        alt="Preview Brosur"
                        className="max-w-full h-auto max-h-52 object-contain rounded"
                      />
                    </div>
                    <label className="flex items-center justify-center gap-2 px-4 py-2 w-full bg-white border border-neutral-300 rounded-md text-xs font-semibold text-neutral-700 hover:bg-neutral-50 cursor-pointer shadow-2xs transition-colors text-center">
                      <svg className="w-4 h-4 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                      </svg>
                      <span>{isUploadingBrosur ? 'Mengupload...' : 'Ganti Gambar'}</span>
                      <input type="file" className="hidden" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                    </label>
                  </div>
                ) : (
                  <div className="flex justify-center px-4 pt-6 pb-6 border-2 border-neutral-300 border-dashed rounded-md hover:border-primary-400 transition-colors bg-neutral-50">
                    <div className="space-y-2 text-center">
                      <div className="w-10 h-10 mx-auto rounded-full bg-white border border-neutral-200 flex items-center justify-center text-neutral-400 shadow-2xs">
                        <svg className="h-5 w-5" stroke="currentColor" fill="none" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                        </svg>
                      </div>
                      <div className="text-xs text-neutral-600">
                        <label htmlFor="brosur-upload" className="cursor-pointer font-semibold text-primary-600 hover:text-primary-700">
                          <span>{isUploadingBrosur ? 'Mengupload...' : 'Upload Brosur'}</span>
                          <input id="brosur-upload" name="brosur-upload" type="file" className="sr-only" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                        </label>
                        <p className="text-[11px] text-neutral-400 mt-0.5">JPG, PNG, WebP</p>
                      </div>
                    </div>
                  </div>
                )}
              </MetaBox>
            </div>

            {/* MetaBox: Publikasikan */}
            <div>
              <MetaBox
                title="Publikasikan"
                icon={
                  <svg className="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
                  </svg>
                }
              >
                <div className="space-y-4">
                  <CustomDropdown
                    label="Status"
                    value={formData.status}
                    onChange={(val) => handleChange({ target: { name: 'status', value: val } })}
                    options={[
                      { value: 'draft', label: 'Draft' },
                      { value: 'published', label: 'Published' },
                      { value: 'archived', label: 'Archived' }
                    ]}
                  />

                  <Button
                    type="submit"
                    variant="primary"
                    isLoading={isSubmitting}
                    disabled={isUploadingBrosur}
                    className="w-full justify-center shadow-xs"
                  >
                    {isEditMode ? 'Simpan Perubahan' : 'Simpan Paket'}
                  </Button>
                </div>
              </MetaBox>
            </div>

          </div>
        </div>
      </form>
    </div>
  );
};

export default ScheduleFormPage;

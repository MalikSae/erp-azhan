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
  const [itineraryOptions, setItineraryOptions] = useState([]);
  const [brandOptions, setBrandOptions] = useState([]);
  const [addOnOptions, setAddOnOptions] = useState([]);

  const [includeItemsText, setIncludeItemsText] = useState('');
  const [excludeItemsText, setExcludeItemsText] = useState('');

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
    pulang_tanggal: '',
    pulang_jam: '',
    pulang_kode_penerbangan: '',
    hotel_mekkah_id: '',
    hotel_madinah_id: '',
    harga_quad: '',
    harga_triple: '',
    harga_double: '',
    harga_coret: '',
    itinerary_id: '',
    include_items: [''],
    exclude_items: [''],
    add_on_ids: [],
    brosur_url: '',
    brosur_thumb_url: '',
    status: 'draft',
    brand_id: ''
  });

  useEffect(() => {
    const fetchInitialData = async () => {
      setIsLoadingInitial(true);
      setFormError(null);
      try {
        const promises = [
          listHotels(),
          listAirlines(),
          listItineraries(),
          listBrands(),
          listAddOns()
        ];

        const results = await Promise.all(promises);
        const hotels = results[0];
        const airlines = results[1];
        const itineraries = results[2];
        const brands = results[3] || [];
        const addons = results[4] || [];
        
        setHotelOptions(hotels || []);
        setAirlineOptions(airlines || []);
        setItineraryOptions(itineraries || []);
        setBrandOptions(brands || []);
        setAddOnOptions(addons || []);

        if (isEditMode) {
          const scheduleData = await getSchedule(id);
          const safeArray = (arr) => (Array.isArray(arr) && arr.length > 0) ? arr : [''];
          
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
            berangkat_kode_penerbangan: scheduleData.berangkat_kode_penerbangan || '',
            pulang_tanggal: scheduleData.pulang_tanggal ? scheduleData.pulang_tanggal.split('T')[0] : '',
            pulang_jam: scheduleData.pulang_jam || '',
            pulang_kode_penerbangan: scheduleData.pulang_kode_penerbangan || '',
            hotel_mekkah_id: scheduleData.hotel_mekkah?.id || scheduleData.hotel_mekkah_id || '',
            hotel_madinah_id: scheduleData.hotel_madinah?.id || scheduleData.hotel_madinah_id || '',
            harga_quad: scheduleData.harga_quad || '',
            harga_triple: scheduleData.harga_triple || '',
            harga_double: scheduleData.harga_double || '',
            harga_coret: scheduleData.harga_coret || '',
            itinerary_id: scheduleData.itinerary?.id || scheduleData.itinerary_id || '',
            include_items: safeArray(scheduleData.include_items),
            exclude_items: safeArray(scheduleData.exclude_items),
            add_on_ids: scheduleData.add_ons ? scheduleData.add_ons.map(a => a.id) : [],
            brosur_url: scheduleData.brosur_url || '',
            brosur_thumb_url: scheduleData.brosur_thumb_url || '',
            status: scheduleData.status || 'draft',
            brand_id: scheduleData.brand_id || ''
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
    if (!val) return '';
    const num = val.toString().replace(/\D/g, '');
    return num.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  };

  const handleCurrencyChange = (e) => {
    const { name, value } = e.target;
    const rawValue = value.replace(/\D/g, '');
    setFormData(prev => ({
      ...prev,
      [name]: value
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

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    setFormError(null);

    try {
      const payload = {
        ...formData,
        is_promo: !!formData.is_promo,
        is_ticket_confirmed: !!formData.is_ticket_confirmed,
        is_direct_flight: !!formData.is_direct_flight,
        seat_total: parseInt(formData.seat_total, 10),
        seat_sisa: parseInt(formData.seat_total, 10) - parseInt(formData.kuota_terisi || 0, 10),
        maskapai_id: parseInt(formData.maskapai_id, 10),
        hotel_mekkah_id: parseInt(formData.hotel_mekkah_id, 10),
        hotel_madinah_id: parseInt(formData.hotel_madinah_id, 10),
        harga_quad: parseFloat(formData.harga_quad),
        harga_triple: parseFloat(formData.harga_triple),
        harga_double: parseFloat(formData.harga_double),
        harga_coret: formData.is_promo && formData.harga_coret ? parseFloat(formData.harga_coret) : null,
        status: formData.status,
        itinerary_id: formData.itinerary_id ? parseInt(formData.itinerary_id, 10) : null,
        include_items: includeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        exclude_items: excludeItemsText.split('\n').map(item => item.trim()).filter(item => item !== ''),
        add_on_ids: formData.add_on_ids,
        brand_id: parseInt(formData.brand_id, 10)
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

  let mekkahHotels = hotelOptions.filter(h => h.city?.toLowerCase().includes('mekkah') || h.city?.toLowerCase().includes('makkah'));
  if (mekkahHotels.length === 0) mekkahHotels = hotelOptions;

  let madinahHotels = hotelOptions.filter(h => h.city?.toLowerCase().includes('madinah'));
  if (madinahHotels.length === 0) madinahHotels = hotelOptions;

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
            
            {/* Judul Besar (Tanpa MetaBox) */}
            <div>
              <Input 
                label="Nama Paket"
                type="text" 
                name="jadwal_nama"
                value={formData.jadwal_nama}
                onChange={handleChange}
                required
                placeholder="Tambahkan nama paket..."
              />
            </div>

            <MetaBox title="Info Dasar">
              <div className="mb-4">
                <CustomDropdown
                  label="Brand"
                  value={formData.brand_id}
                  onChange={(val) => handleChange({ target: { name: 'brand_id', value: val } })}
                  required
                  placeholder="-- Pilih Brand --"
                  options={brandOptions.map(b => ({ value: b.id, label: b.name }))}
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Input 
                  label="Total Kuota Kursi" 
                  className="!mb-0"
                  type="number"
                  name="seat_total"
                  value={formData.seat_total}
                  onChange={handleChange}
                  required
                  min="1"
                  placeholder="mis. 45"
                />
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
                  />
                  {parseInt(formData.kuota_terisi || 0) > parseInt(formData.seat_total || 0) && (
                    <p className="text-sm text-danger-600 mt-1">Kuota terisi tidak boleh melebihi total kuota!</p>
                  )}
                </div>
                <Input 
                  label="Sisa Kuota" 
                  className="!mb-0"
                  type="number"
                  value={formData.seat_total ? Math.max(0, parseInt(formData.seat_total) - (parseInt(formData.kuota_terisi) || 0)) : ''}
                  readOnly
                  placeholder="Otomatis"
                />
              </div>
            </MetaBox>

            <MetaBox title="Penerbangan">
              <div className="space-y-4">
                <div className="flex items-end gap-4 mb-4">
                  <div className="flex-1">
                    <CustomDropdown
                      label="Maskapai"
                      className="!mb-0"
                      value={formData.maskapai_id}
                      onChange={(val) => handleChange({ target: { name: 'maskapai_id', value: val } })}
                      required
                      placeholder="-- Pilih Maskapai --"
                      options={airlineOptions.map(a => ({ value: a.id, label: a.name }))}
                    />
                  </div>
                  <div className="flex items-center gap-6">
                    <div className="flex items-center gap-3 h-[38px]">
                      <Toggle
                        id="is_direct_flight"
                        name="is_direct_flight"
                        checked={formData.is_direct_flight}
                        onChange={handleChange}
                      />
                      <label htmlFor="is_direct_flight" className="text-sm text-neutral-700 font-body cursor-pointer whitespace-nowrap">
                        Penerbangan Direct
                      </label>
                    </div>

                    <div className="flex items-center gap-3 h-[38px]">
                      <Toggle
                        id="is_ticket_confirmed"
                        name="is_ticket_confirmed"
                        checked={formData.is_ticket_confirmed}
                        onChange={handleChange}
                      />
                      <label htmlFor="is_ticket_confirmed" className="text-sm text-neutral-700 font-body cursor-pointer whitespace-nowrap">
                        Tiket Sudah Confirm
                      </label>
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <Input label="Tanggal Berangkat" className="!mb-0" type="date" name="berangkat_tanggal" value={formData.berangkat_tanggal} onChange={handleChange} required />
                  <Input label="Jam Berangkat" className="text-center !mb-0" type="text" name="berangkat_jam" value={formData.berangkat_jam} onChange={handleTimeChange} placeholder="mis. 09:00" />
                  <Input label="Kode Penerbangan Berangkat" className="!mb-0" name="berangkat_kode_penerbangan" value={formData.berangkat_kode_penerbangan} onChange={handleChange} placeholder="mis. SV 821" />
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <Input label="Tanggal Pulang" className="!mb-0" type="date" name="pulang_tanggal" value={formData.pulang_tanggal} onChange={handleChange} required />
                  <Input label="Jam Pulang" className="text-center !mb-0" type="text" name="pulang_jam" value={formData.pulang_jam} onChange={handleTimeChange} placeholder="mis. 15:30 +1" />
                  <Input label="Kode Penerbangan Pulang" className="!mb-0" name="pulang_kode_penerbangan" value={formData.pulang_kode_penerbangan} onChange={handleChange} placeholder="mis. SV 822" />
                </div>
              </div>
            </MetaBox>

            <MetaBox title="Hotel">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <CustomDropdown
                    label="Hotel Mekkah"
                    className="!mb-0"
                    value={formData.hotel_mekkah_id}
                    onChange={(val) => handleChange({ target: { name: 'hotel_mekkah_id', value: val } })}
                    required
                    placeholder="-- Pilih Hotel Mekkah --"
                    options={mekkahHotels.map(h => ({ value: h.id, label: h.name }))}
                  />
                  {hotelOptions.length > 0 && mekkahHotels.length === hotelOptions.length && (
                    <p className="text-xs text-neutral-500 mt-1">Menampilkan semua hotel karena tidak ada yang berlokasi di Mekkah.</p>
                  )}
                </div>

                <div>
                  <CustomDropdown
                    label="Hotel Madinah"
                    className="!mb-0"
                    value={formData.hotel_madinah_id}
                    onChange={(val) => handleChange({ target: { name: 'hotel_madinah_id', value: val } })}
                    required
                    placeholder="-- Pilih Hotel Madinah --"
                    options={madinahHotels.map(h => ({ value: h.id, label: h.name }))}
                  />
                  {hotelOptions.length > 0 && madinahHotels.length === hotelOptions.length && (
                    <p className="text-xs text-neutral-500 mt-1">Menampilkan semua hotel karena tidak ada yang berlokasi di Madinah.</p>
                  )}
                </div>
              </div>
            </MetaBox>

            <MetaBox title="Harga">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Input label="Harga Quad" className="!mb-0" type="text" name="harga_quad" value={formatCurrency(formData.harga_quad)} onChange={handleCurrencyChange} required placeholder="mis. 25.000.000" />
                <Input label="Harga Triple" className="!mb-0" type="text" name="harga_triple" value={formatCurrency(formData.harga_triple)} onChange={handleCurrencyChange} required placeholder="mis. 26.000.000" />
                <Input label="Harga Double" className="!mb-0" type="text" name="harga_double" value={formatCurrency(formData.harga_double)} onChange={handleCurrencyChange} required placeholder="mis. 28.000.000" />

                <div className="md:col-span-3 mt-2 flex flex-col gap-1 border-t border-neutral-200 pt-4">
                  <div className="flex items-end gap-4">
                    <div className="flex items-center gap-3 h-[38px]">
                      <Toggle
                        id="is_promo"
                        name="is_promo"
                        checked={formData.is_promo}
                        onChange={handleChange}
                      />
                      <label htmlFor="is_promo" className="text-sm text-neutral-700 font-body cursor-pointer whitespace-nowrap">
                        Tandai Promo
                      </label>
                    </div>
                    
                    {formData.is_promo && (
                      <div className="flex-1 md:flex-none md:w-1/3">
                        <Input label="Harga Coret" className="!mb-0" type="text" name="harga_coret" value={formatCurrency(formData.harga_coret)} onChange={handleCurrencyChange} placeholder="mis. 27.000.000" />
                      </div>
                    )}
                  </div>
                  
                  {formData.is_promo && (
                    <div className="pl-[148px]">
                      {formData.harga_coret && parseInt(formData.harga_coret.toString().replace(/\D/g, '')) <= parseInt(formData.harga_quad.toString().replace(/\D/g, '')) && (
                        <p className="text-xs text-danger-600 mt-1">Harga coret harus lebih besar!</p>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </MetaBox>

            <MetaBox title="Include / Exclude">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                {/* Include */}
                <div>
                  <Textarea
                    label="Include Items"
                    value={includeItemsText}
                    onChange={(e) => setIncludeItemsText(e.target.value)}
                    placeholder="1 baris = 1 item, contoh:&#10;Tiket Pesawat PP&#10;Visa Umroh&#10;Hotel Bintang 4"
                    rows={6}
                    className="!mb-0"
                  />
                </div>

                {/* Exclude */}
                <div>
                  <Textarea
                    label="Exclude Items"
                    value={excludeItemsText}
                    onChange={(e) => setExcludeItemsText(e.target.value)}
                    placeholder="1 baris = 1 item, contoh:&#10;Pengeluaran Pribadi&#10;Suntik Meningitis"
                    rows={6}
                    className="!mb-0"
                  />
                </div>
              </div>
            </MetaBox>

            <MetaBox title="Itinerary">
              <div className="w-full md:w-1/2">
                <CustomDropdown
                  label="Pilih Itinerary"
                  value={formData.itinerary_id}
                  onChange={(val) => handleChange({ target: { name: 'itinerary_id', value: val } })}
                  placeholder="-- Tidak Ada --"
                  options={itineraryOptions.map(it => ({ value: it.id, label: it.title }))}
                />
              </div>
            </MetaBox>

            <MetaBox title="Layanan Tambahan (Add-On)">
              {addOnOptions.length > 0 ? (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {addOnOptions.map(addon => (
                    <label key={addon.id} className="flex items-center gap-3 cursor-pointer group">
                      <input 
                        type="checkbox" 
                        checked={formData.add_on_ids.includes(addon.id)}
                        onChange={(e) => handleAddOnCheckboxChange(addon.id, e.target.checked)}
                        className="w-4 h-4 text-primary-600 border-neutral-300 rounded focus:ring-primary-500 cursor-pointer transition-colors"
                      />
                      <span className="text-sm text-neutral-700 font-body group-hover:text-neutral-900 transition-colors">
                        {addon.name}
                      </span>
                    </label>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-neutral-500 italic">Belum ada add-on yang tersedia.</p>
              )}
            </MetaBox>

            {formError && (
              <Alert variant="error">{formError}</Alert>
            )}
            
          </div>

          {/* Kolom KANAN (Sidebar Form) */}
          <div className="sticky top-6 flex flex-col gap-6">
            
            <div className="order-3 md:order-1">
              <MetaBox title="Publikasikan">
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
                  className="w-full justify-center"
                >
                  {isEditMode ? 'Simpan Perubahan' : 'Simpan'}
                </Button>
              </div>
              </MetaBox>
            </div>


            <div className="order-2 md:order-3">
              <MetaBox title="Brosur Paket">
              {formData.brosur_url ? (
                <div className="space-y-4">
                  <div className="border border-neutral-200 rounded-md p-1 bg-white overflow-hidden flex justify-center shadow-sm">
                    <img src={formData.brosur_url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${formData.brosur_url}` : formData.brosur_url} alt="Preview Brosur" className="max-w-full h-auto max-h-48 object-contain rounded" />
                  </div>
                  <label className="flex items-center justify-center gap-2 px-4 py-2 w-full bg-white border border-neutral-300 rounded-md text-sm font-medium text-neutral-700 hover:bg-neutral-50 cursor-pointer shadow-sm transition-colors text-center">
                    {isUploadingBrosur ? 'Mengupload...' : 'Ganti Brosur'}
                    <input type="file" className="hidden" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                  </label>
                </div>
              ) : (
                <div className="flex justify-center px-4 pt-5 pb-6 border-2 border-neutral-300 border-dashed rounded-md hover:border-primary-400 transition-colors bg-neutral-50">
                  <div className="space-y-1 text-center">
                    <svg className="mx-auto h-8 w-8 text-neutral-400" stroke="currentColor" fill="none" viewBox="0 0 48 48" aria-hidden="true">
                      <path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                    <div className="flex text-sm text-neutral-600 justify-center mt-2">
                      <label htmlFor="brosur-upload" className="relative cursor-pointer rounded-md font-medium text-primary-600 hover:text-primary-500 focus-within:outline-none">
                        <span>{isUploadingBrosur ? 'Mengupload...' : 'Upload Gambar'}</span>
                        <input id="brosur-upload" name="brosur-upload" type="file" className="sr-only" accept="image/*" onChange={handleBrosurUpload} disabled={isUploadingBrosur} />
                      </label>
                    </div>
                  </div>
                </div>
              )}
              </MetaBox>
            </div>
          </div>
        </div>
      </form>
    </div>
  );
};

export default ScheduleFormPage;

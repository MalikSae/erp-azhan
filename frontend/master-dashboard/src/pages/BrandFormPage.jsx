import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import { FormTabs, FormSection } from 'shared/ui/FormLayout';
import BrandBankAccounts from '../components/BrandBankAccounts';
import FormField from 'shared/ui/FormField';
import Input from 'shared/ui/Input';
import FileInput from 'shared/ui/FileInput';
import Textarea from 'shared/ui/Textarea';
import AutocompleteInput from 'shared/ui/AutocompleteInput';
import CustomDropdown from 'shared/ui/CustomDropdown';
import { INDONESIAN_REGIONS, INDONESIAN_PROVINCES } from 'shared/data/indonesianRegions';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { getBrand, createBrand, updateBrand } from '../api/brands';
import { uploadMedia } from '../api/media';

const SECTION_LINKS = [
  { id: 'sec-identitas', label: 'Identitas Brand' },
  { id: 'sec-media', label: 'Media & Visual' },
  { id: 'sec-kontak', label: 'Kontak & Alamat' },
  { id: 'sec-seo', label: 'SEO' },
  { id: 'sec-legal', label: 'Legalitas' },
  { id: 'sec-sosial', label: 'Media Sosial' },
  { id: 'sec-sk', label: 'S&K Booking' },
  { id: 'sec-agen', label: 'Aturan Agen' },
  { id: 'sec-rekening', label: 'Rekening Bank' },
];

const initialFormData = {
  kode_brand: '',
  name: '',
  domain: '',
  primary_color: '#FED853',
  logo_url: '',
  icon_url: '',
  og_image_url: '',
  whatsapp_number: '',
  email: '',
  phone: '',
  address: '',
  city: '',
  province: '',
  gmaps_url: '',
  ppiu_number: '',
  pihk_number: '',
  bank_name: '',
  bank_account_number: '',
  bank_account_holder: '',
  social_facebook: '',
  social_instagram: '',
  social_tiktok: '',
  social_youtube: '',
  meta_title: '',
  meta_description: '',
  google_verification_code: '',
  biaya_pendaftaran_agen: '',
  no_wa_admin_travel: '',
  minimal_dp: '',
  batas_pelunasan_hari: '45',
  ketentuan_pembatalan: '',
  cashback_alumni: '',
  biaya_mode: 'gratis',
  ujroh_closing: '',
  ujroh_pembinaan: '',
  ujroh_repeat_booking: '',
  syarat_ketentuan_agen: ''
};

const BrandFormPage = () => {
  const { id } = useParams();
  const isEditMode = Boolean(id);
  const navigate = useNavigate();

  const [formData, setFormData] = useState(initialFormData);
  const [isLoadingDetail, setIsLoadingDetail] = useState(isEditMode);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formErrors, setFormErrors] = useState(null);
  const [notFound, setNotFound] = useState(false);

  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const [isUploadingIcon, setIsUploadingIcon] = useState(false);
  const [isUploadingOgImage, setIsUploadingOgImage] = useState(false);
  const [localLogoPreview, setLocalLogoPreview] = useState(null);
  const [localIconPreview, setLocalIconPreview] = useState(null);
  const [localOgImagePreview, setLocalOgImagePreview] = useState(null);

  useEffect(() => {
    if (isEditMode) {
      const fetchDetail = async () => {
        setIsLoadingDetail(true);
        setFormErrors(null);
        try {
          const data = await getBrand(id);
          setFormData({
            kode_brand: data.kode_brand || '',
            name: data.name || '',
            domain: data.domain || '',
            primary_color: data.primary_color || '#FED853',
            logo_url: data.logo_url || '',
            icon_url: data.icon_url || '',
            og_image_url: data.og_image_url || '',
            whatsapp_number: data.whatsapp_number || '',
            email: data.email || '',
            phone: data.phone || '',
            address: data.address || '',
            city: data.city || '',
            province: data.province || '',
            gmaps_url: data.gmaps_url || '',
            ppiu_number: data.ppiu_number || '',
            pihk_number: data.pihk_number || '',
            bank_name: data.bank_name || '',
            bank_account_number: data.bank_account_number || '',
            bank_account_holder: data.bank_account_holder || '',
            social_facebook: data.social_facebook || '',
            social_instagram: data.social_instagram || '',
            social_tiktok: data.social_tiktok || '',
            social_youtube: data.social_youtube || '',
            meta_title: data.meta_title || '',
            meta_description: data.meta_description || '',
            google_verification_code: data.google_verification_code || '',
            biaya_pendaftaran_agen: data.biaya_pendaftaran_agen ? String(data.biaya_pendaftaran_agen) : '',
            no_wa_admin_travel: data.no_wa_admin_travel || '',
            minimal_dp: data.minimal_dp ? String(data.minimal_dp) : '',
            batas_pelunasan_hari: data.batas_pelunasan_hari ? String(data.batas_pelunasan_hari) : '45',
            ketentuan_pembatalan: data.ketentuan_pembatalan || '',
            cashback_alumni: data.cashback_alumni ? String(data.cashback_alumni) : '',
            biaya_mode: Number(data.biaya_pendaftaran_agen) > 0 ? 'berbayar' : 'gratis',
            ujroh_closing: data.ujroh_closing ? String(data.ujroh_closing) : '',
            ujroh_pembinaan: data.ujroh_pembinaan ? String(data.ujroh_pembinaan) : '',
            ujroh_repeat_booking: data.ujroh_repeat_booking ? String(data.ujroh_repeat_booking) : '',
            syarat_ketentuan_agen: data.syarat_ketentuan_agen || ''
          });
        } catch (err) {
          if (err.response?.status === 404) {
            setNotFound(true);
          } else {
            setFormErrors("Gagal mengambil data detail brand.");
          }
        } finally {
          setIsLoadingDetail(false);
        }
      };
      fetchDetail();
    }
  }, [id, isEditMode]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    if (name === 'kode_brand') {
      setFormData(prev => ({
        ...prev,
        [name]: value.slice(0, 2).toUpperCase()
      }));
      return;
    }
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleLogoUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setLocalLogoPreview(URL.createObjectURL(file));
    setIsUploadingLogo(true);
    setFormErrors(null);

    try {
      const url = await uploadMedia(file, 'brand-logos');
      setFormData(prev => ({ ...prev, logo_url: url }));
    } catch (err) {
      console.error(err);
      setFormErrors('Gagal mengunggah logo brand.');
    } finally {
      setIsUploadingLogo(false);
    }
  };

  const handleIconUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setLocalIconPreview(URL.createObjectURL(file));
    setIsUploadingIcon(true);
    setFormErrors(null);

    try {
      const url = await uploadMedia(file, 'brand-icons');
      setFormData(prev => ({ ...prev, icon_url: url }));
    } catch (err) {
      console.error(err);
      setFormErrors('Gagal mengunggah ikon brand.');
    } finally {
      setIsUploadingIcon(false);
    }
  };

  const handleOgImageUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setLocalOgImagePreview(URL.createObjectURL(file));
    setIsUploadingOgImage(true);
    setFormErrors(null);

    try {
      const url = await uploadMedia(file, 'brand-og');
      setFormData(prev => ({ ...prev, og_image_url: url }));
    } catch (err) {
      console.error(err);
      setFormErrors('Gagal mengunggah banner social share.');
    } finally {
      setIsUploadingOgImage(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);

    if (!formData.name.trim()) {
      setFormErrors("Nama Brand wajib diisi.");
      return;
    }

    if (formData.biaya_mode === 'berbayar' && !Number(String(formData.biaya_pendaftaran_agen).replace(/\D/g, ''))) {
      setFormErrors("Isi nominal Biaya Pendaftaran, atau pilih Gratis.");
      return;
    }

    const pelunasanHari = Number(String(formData.batas_pelunasan_hari).replace(/\D/g, ''));
    if (!pelunasanHari || pelunasanHari > 365) {
      setFormErrors("Batas Pelunasan harus antara 1 dan 365 hari sebelum keberangkatan.");
      return;
    }

    if (formData.primary_color && !/^#[0-9A-Fa-f]{6}$/.test(formData.primary_color)) {
      setFormErrors("Warna Utama Brand harus berformat Hex yang valid, contoh #1B3A6B.");
      return;
    }

    setIsSubmitting(true);

    try {
      const payload = {
        ...formData,
        kode_brand: formData.kode_brand ? formData.kode_brand.toUpperCase() : null,
        domain: formData.domain ? formData.domain.toLowerCase().trim() : null,
        whatsapp_number: formData.whatsapp_number ? formData.whatsapp_number.trim() : null,
        email: formData.email ? formData.email.trim() : null,
        phone: formData.phone ? formData.phone.trim() : null,
        address: formData.address ? formData.address.trim() : null,
        city: formData.city ? formData.city.trim() : null,
        province: formData.province ? formData.province.trim() : null,
        gmaps_url: formData.gmaps_url ? formData.gmaps_url.trim() : null,
        ppiu_number: formData.ppiu_number ? formData.ppiu_number.trim() : null,
        pihk_number: formData.pihk_number ? formData.pihk_number.trim() : null,
        bank_name: formData.bank_name ? formData.bank_name.trim() : null,
        bank_account_number: formData.bank_account_number ? formData.bank_account_number.trim() : null,
        bank_account_holder: formData.bank_account_holder ? formData.bank_account_holder.trim() : null,
        social_facebook: formData.social_facebook ? formData.social_facebook.trim() : null,
        social_instagram: formData.social_instagram ? formData.social_instagram.trim() : null,
        social_tiktok: formData.social_tiktok ? formData.social_tiktok.trim() : null,
        social_youtube: formData.social_youtube ? formData.social_youtube.trim() : null,
        logo_url: formData.logo_url || null,
        icon_url: formData.icon_url || null,
        og_image_url: formData.og_image_url || null,
        meta_title: formData.meta_title ? formData.meta_title.trim() : null,
        meta_description: formData.meta_description ? formData.meta_description.trim() : null,
        google_verification_code: formData.google_verification_code ? formData.google_verification_code.trim() : null,
        // Kosong = Rp0 (brand tidak memungut biaya pendaftaran agen).
        biaya_pendaftaran_agen: formData.biaya_mode === 'berbayar'
          ? Number(String(formData.biaya_pendaftaran_agen).replace(/\D/g, '')) || 0
          : 0,
        ujroh_closing: Number(String(formData.ujroh_closing).replace(/\D/g, '')) || 0,
        ujroh_pembinaan: Number(String(formData.ujroh_pembinaan).replace(/\D/g, '')) || 0,
        ujroh_repeat_booking: Number(String(formData.ujroh_repeat_booking).replace(/\D/g, '')) || 0,
        syarat_ketentuan_agen: formData.syarat_ketentuan_agen ? formData.syarat_ketentuan_agen.trim() : null,
        no_wa_admin_travel: formData.no_wa_admin_travel ? formData.no_wa_admin_travel.trim() : null,
        // Wajib dikirim: backend menyimpan minimal_dp apa adanya, jadi field
        // yang tidak terkirim akan me-reset nilainya ke 0.
        minimal_dp: Number(String(formData.minimal_dp).replace(/\D/g, '')) || 0,
        batas_pelunasan_hari: Number(String(formData.batas_pelunasan_hari).replace(/\D/g, '')) || 45,
        ketentuan_pembatalan: formData.ketentuan_pembatalan ? formData.ketentuan_pembatalan.trim() : null,
        cashback_alumni: Number(String(formData.cashback_alumni).replace(/\D/g, '')) || 0,
      };

      delete payload.biaya_mode;

      if (isEditMode) {
        await updateBrand(id, payload);
      } else {
        await createBrand(payload);
      }

      navigate('/brands', {
        state: { successMessage: `Brand "${payload.name}" berhasil ${isEditMode ? 'diperbarui' : 'ditambahkan'}.` },
      });
    } catch (err) {
      if (err.response?.data?.error) {
        setFormErrors(err.response.data.error);
      } else {
        setFormErrors("Terjadi kesalahan saat menyimpan data brand. Silakan coba lagi.");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoadingDetail) {
    return (
      <div className="flex justify-center items-center min-h-[400px]">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (notFound) {
    return (
      <div className="space-y-6">
        <PageHeader 
          title="Brand Tidak Ditemukan"
          onBack={() => navigate('/brands')}
        />
        <Alert variant="error">
          Data brand yang Anda cari tidak ditemukan atau telah dihapus.
        </Alert>
      </div>
    );
  }

  // Nilai lama yang tidak ada di daftar resmi tetap ditampilkan agar tidak hilang diam-diam.
  const withCurrent = (list, current) => (current && !list.includes(current) ? [current, ...list] : list);
  const provinceOptions = withCurrent(INDONESIAN_PROVINCES, formData.province);
  const cityOptions = withCurrent(INDONESIAN_REGIONS[formData.province] || [], formData.city);

  const handleProvinceChange = (val) => {
    setFormData(prev => ({
      ...prev,
      province: val,
      // Kota hanya valid di provinsinya; kosongkan bila tak ada di provinsi baru.
      city: (INDONESIAN_REGIONS[val] || []).includes(prev.city) ? prev.city : '',
    }));
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title={isEditMode ? "Edit Profil Brand" : "Tambah Brand Baru"}
        onBack={() => navigate('/brands')}
      >
        <Button
          type="submit"
          form="brand-form"
          variant="primary"
          isLoading={isSubmitting}
          disabled={isSubmitting || isUploadingLogo || isUploadingIcon || isUploadingOgImage}
        >
          {isSubmitting ? 'Menyimpan...' : (isEditMode ? 'Perbarui Data Brand' : 'Simpan Brand Baru')}
        </Button>
      </PageHeader>

      {formErrors && (
        <Alert variant="error">
          {formErrors}
        </Alert>
      )}

      <form id="brand-form" onSubmit={handleSubmit}>
        <FormTabs tabs={SECTION_LINKS}>
<FormSection id="sec-identitas" title="Identitas Brand">
              <div className="space-y-4">
                <Input
                  label="Nama Brand"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Contoh: Alsha Tours, Hana Travel"
                  required
                  className="!mb-0"
                />

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <Input
                    label="Kode Brand"
                    name="kode_brand"
                    value={formData.kode_brand}
                    onChange={handleChange}
                    placeholder="AS"
                    maxLength={2}
                    className="!mb-0"
                  />

                  <Input
                    label="Domain"
                    name="domain"
                    value={formData.domain}
                    onChange={handleChange}
                    placeholder="alsha.azhan.test"
                    className="!mb-0"
                  />
                </div>

                <FormField label="Warna Utama" className="!mb-0">
                  <div className="flex items-center gap-2">
                    <input 
                      type="color" 
                      name="primary_color"
                      value={formData.primary_color || '#FED853'}
                      onChange={handleChange}
                      aria-label="Pilih warna utama brand"
                      className="brand-color-picker w-11 h-11 shrink-0 p-2 rounded-xl border border-neutral-200 cursor-pointer bg-white"
                    />
                    <Input 
                      name="primary_color"
                      value={formData.primary_color}
                      onChange={handleChange}
                      placeholder="#FED853"
                      className="!mb-0 min-w-0 flex-1"
                      aria-label="Kode hex warna utama brand"
                    />
                  </div>
                </FormField>
              </div>
            </FormSection>
            <FormSection id="sec-media" title="Media & Visual Brand">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <FileInput 
                  label="Logo Brand" 
                  value={formData.logo_url}
                  previewUrl={localLogoPreview}
                  onChange={handleLogoUpload}
                  onRemove={() => {
                    setLocalLogoPreview(null);
                    setFormData(prev => ({ ...prev, logo_url: '' }));
                  }}
                  isUploading={isUploadingLogo}
                  uploadingText="Mengunggah logo..."
                  placeholder="Pilih file..."
                />

                <FileInput 
                  label="Ikon Brand (1:1)" 
                  value={formData.icon_url}
                  previewUrl={localIconPreview}
                  onChange={handleIconUpload}
                  onRemove={() => {
                    setLocalIconPreview(null);
                    setFormData(prev => ({ ...prev, icon_url: '' }));
                  }}
                  isUploading={isUploadingIcon}
                  uploadingText="Mengunggah ikon..."
                  placeholder="Pilih file..."
                />

                <div className="md:col-span-2">
                  <FileInput 
                    label="Banner Sharing (1200×630)" 
                    value={formData.og_image_url}
                    previewUrl={localOgImagePreview}
                    onChange={handleOgImageUpload}
                    onRemove={() => {
                      setLocalOgImagePreview(null);
                      setFormData(prev => ({ ...prev, og_image_url: '' }));
                    }}
                    isUploading={isUploadingOgImage}
                    uploadingText="Mengunggah banner..."
                    placeholder="Pilih file..."
                  />
                </div>
              </div>
            </FormSection>
            <FormSection id="sec-kontak" title="Kontak & Alamat">
              <div className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <Input
                    label="WhatsApp CS"
                    name="whatsapp_number"
                    value={formData.whatsapp_number}
                    onChange={handleChange}
                    placeholder="Contoh: 62812345678"
                    className="!mb-0"
                  />

                  <Input
                    label="Email"
                    type="email"
                    name="email"
                    value={formData.email}
                    onChange={handleChange}
                    placeholder="cs@travel.com"
                    className="!mb-0"
                  />
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <AutocompleteInput
                    label="Provinsi"
                    name="province"
                    value={formData.province}
                    onChange={(e) => handleProvinceChange(e.target.value)}
                    placeholder="Ketik / cari provinsi..."
                    options={provinceOptions}
                    className="!mb-0"
                  />

                  <AutocompleteInput
                    label="Kota / Kabupaten"
                    name="city"
                    value={formData.city}
                    onChange={(e) => setFormData(prev => ({ ...prev, city: e.target.value }))}
                    placeholder={formData.province ? 'Ketik / cari kota atau kabupaten...' : 'Pilih provinsi dulu'}
                    options={cityOptions}
                    disabled={!formData.province && !formData.city}
                    className="!mb-0"
                  />
                </div>

                <Input
                  label="Google Maps"
                  name="gmaps_url"
                  value={formData.gmaps_url}
                  onChange={handleChange}
                  placeholder="https://maps.google.com/..."
                  className="!mb-0"
                />

                <Textarea
                  label="Alamat"
                  name="address"
                  value={formData.address}
                  onChange={handleChange}
                  placeholder="Jl. Utama No. 123, Jakarta"
                  rows={3}
                  className="!mb-0"
                />
              </div>
            </FormSection>
            <FormSection id="sec-seo" title="SEO">
              <div className="space-y-4">
                <Input
                  label="Meta Title"
                  name="meta_title"
                  value={formData.meta_title}
                  onChange={handleChange}
                  placeholder="Alsha Travel - Paket Umroh & Haji Khusus Resmi Berizin PPIU"
                  maxLength={150}
                  className="!mb-0"
                />

                <Textarea
                  label="Meta Description"
                  name="meta_description"
                  value={formData.meta_description}
                  onChange={handleChange}
                  placeholder="Biro perjalanan umroh resmi terpercaya dengan jadwal keberangkatan pasti, hotel bintang 5 dekat masjid, dan bimbingan ibadah sesuai Sunnah..."
                  rows={2}
                  maxLength={255}
                  className="!mb-0"
                />

                <Input
                  label="Kode Verifikasi Google"
                  name="google_verification_code"
                  value={formData.google_verification_code}
                  onChange={handleChange}
                  placeholder="google-site-verification=xxxxxxxxxxxx"
                  className="!mb-0"
                />
              </div>
            </FormSection>
            <FormSection id="sec-legal" title="Legalitas">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <Input
                  label="PPIU"
                  name="ppiu_number"
                  value={formData.ppiu_number}
                  onChange={handleChange}
                  placeholder="cth. No. 484/2020"
                  className="!mb-0"
                />
                <Input
                  label="PIHK"
                  name="pihk_number"
                  value={formData.pihk_number}
                  onChange={handleChange}
                  placeholder="cth. No. 395/2021"
                  className="!mb-0"
                />
              </div>
            </FormSection>
            <FormSection id="sec-sosial" title="Media Sosial">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <Input 
                  label="Facebook"
                  className="!mb-0"
                  name="social_facebook"
                  value={formData.social_facebook}
                  onChange={handleChange}
                  placeholder="https://facebook.com/..."
                />

                <Input 
                  label="Instagram"
                  className="!mb-0"
                  name="social_instagram"
                  value={formData.social_instagram}
                  onChange={handleChange}
                  placeholder="https://instagram.com/..."
                />

                <Input 
                  label="TikTok"
                  className="!mb-0"
                  name="social_tiktok"
                  value={formData.social_tiktok}
                  onChange={handleChange}
                  placeholder="https://tiktok.com/@..."
                />

                <Input 
                  label="YouTube"
                  className="!mb-0"
                  name="social_youtube"
                  value={formData.social_youtube}
                  onChange={handleChange}
                  placeholder="https://youtube.com/@..."
                />
              </div>
            </FormSection>

          <FormSection id="sec-sk" title="S&K Booking">
            <div className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <Input
                  label="Minimal DP (Rp)"
                  name="minimal_dp"
                  value={formData.minimal_dp ? Number(formData.minimal_dp).toLocaleString('id-ID') : ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, minimal_dp: e.target.value.replace(/\D/g, '') }))}
                  placeholder="mis. 5.000.000"
                  inputMode="numeric"
                  className="!mb-0"
                />
                <Input
                  label="Cashback Alumni (Rp)"
                  name="cashback_alumni"
                  value={formData.cashback_alumni ? Number(formData.cashback_alumni).toLocaleString('id-ID') : ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, cashback_alumni: e.target.value.replace(/\D/g, '') }))}
                  placeholder="mis. 500.000"
                  inputMode="numeric"
                  className="!mb-0"
                />
                <Input
                    label="Batas Pelunasan (H-)"
                    name="batas_pelunasan_hari"
                    value={formData.batas_pelunasan_hari}
                    onChange={(e) => setFormData(prev => ({ ...prev, batas_pelunasan_hari: e.target.value.replace(/\D/g, '').slice(0, 3) }))}
                    placeholder="mis. 45"
                    inputMode="numeric"
                    className="!mb-0"
                  />
              </div>

              <Textarea
                label="Ketentuan Pembatalan"
                name="ketentuan_pembatalan"
                value={formData.ketentuan_pembatalan}
                onChange={handleChange}
                placeholder={'Contoh:\n- Pembatalan H-60 s.d. H-31: potongan 25% dari total biaya\n- Pembatalan H-30 s.d. H-15: potongan 50%\n- Pembatalan di bawah H-14: tidak ada pengembalian dana'}
                rows={8}
                className="!mb-0"
              />
            </div>
          </FormSection>

          <FormSection id="sec-agen" title="Aturan Agen">
            <div className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <CustomDropdown
                  label="Biaya Pendaftaran"
                  value={formData.biaya_mode}
                  onChange={(val) => setFormData(prev => ({ ...prev, biaya_mode: val }))}
                  options={[
                    { value: 'gratis', label: 'Gratis' },
                    { value: 'berbayar', label: 'Berbayar' },
                  ]}
                  className="!mb-0"
                />
                {formData.biaya_mode === 'berbayar' && (
                  <Input
                    label="Nominal Biaya Pendaftaran (Rp)"
                    name="biaya_pendaftaran_agen"
                    value={formData.biaya_pendaftaran_agen ? Number(formData.biaya_pendaftaran_agen).toLocaleString('id-ID') : ''}
                    onChange={(e) => setFormData(prev => ({ ...prev, biaya_pendaftaran_agen: e.target.value.replace(/\D/g, '') }))}
                    placeholder="mis. 150.000"
                    inputMode="numeric"
                    className="!mb-0"
                  />
                )}
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <Input
                  label="Ujroh Closing (Rp/jamaah)"
                  name="ujroh_closing"
                  value={formData.ujroh_closing ? Number(formData.ujroh_closing).toLocaleString('id-ID') : ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, ujroh_closing: e.target.value.replace(/\D/g, '') }))}
                  placeholder="mis. 500.000"
                  inputMode="numeric"
                  className="!mb-0"
                />
                <Input
                  label="Ujroh Pembinaan (Rp/jamaah)"
                  name="ujroh_pembinaan"
                  value={formData.ujroh_pembinaan ? Number(formData.ujroh_pembinaan).toLocaleString('id-ID') : ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, ujroh_pembinaan: e.target.value.replace(/\D/g, '') }))}
                  placeholder="mis. 200.000"
                  inputMode="numeric"
                  className="!mb-0"
                />
                <Input
                  label="Ujroh Repeat Booking (Rp/jamaah)"
                  name="ujroh_repeat_booking"
                  value={formData.ujroh_repeat_booking ? Number(formData.ujroh_repeat_booking).toLocaleString('id-ID') : ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, ujroh_repeat_booking: e.target.value.replace(/\D/g, '') }))}
                  placeholder="mis. 250.000"
                  inputMode="numeric"
                  className="!mb-0"
                />
              </div>

              <Textarea
                label="Syarat & Ketentuan"
                name="syarat_ketentuan_agen"
                value={formData.syarat_ketentuan_agen}
                onChange={handleChange}
                placeholder="Syarat dan ketentuan yang disetujui agen saat mendaftar..."
                rows={8}
                className="!mb-0"
              />
            </div>
          </FormSection>

          <FormSection id="sec-rekening" title="Rekening Bank">
            <BrandBankAccounts brandId={id} />
          </FormSection>
        </FormTabs>
      </form>
    </div>
  );
};

export default BrandFormPage;

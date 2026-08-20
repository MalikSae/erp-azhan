import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import MetaBox from '../components/ui/MetaBox';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import Textarea from '../components/ui/Textarea';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { getBrand, createBrand, updateBrand } from '../api/brands';
import { uploadMedia } from '../api/media';

const initialFormData = {
  kode_brand: '',
  name: '',
  domain: '',
  primary_color: '#CC904A',
  logo_url: '',
  icon_url: '',
  whatsapp_number: '',
  address: '',
  gmaps_url: '',
  legalitas: '',
  bank_name: '',
  bank_account_number: '',
  bank_account_holder: '',
  social_facebook: '',
  social_instagram: '',
  social_tiktok: '',
  social_youtube: ''
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
  const [localLogoPreview, setLocalLogoPreview] = useState(null);
  const [localIconPreview, setLocalIconPreview] = useState(null);

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
            primary_color: data.primary_color || '#CC904A',
            logo_url: data.logo_url || '',
            icon_url: data.icon_url || '',
            whatsapp_number: data.whatsapp_number || '',
            address: data.address || '',
            gmaps_url: data.gmaps_url || '',
            legalitas: data.legalitas || '',
            bank_name: data.bank_name || '',
            bank_account_number: data.bank_account_number || '',
            bank_account_holder: data.bank_account_holder || '',
            social_facebook: data.social_facebook || '',
            social_instagram: data.social_instagram || '',
            social_tiktok: data.social_tiktok || '',
            social_youtube: data.social_youtube || ''
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

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);
    setIsSubmitting(true);

    const payload = {
      kode_brand: formData.kode_brand.trim() || null,
      name: formData.name.trim(),
      domain: formData.domain.trim() || null,
      primary_color: formData.primary_color.trim() || null,
      logo_url: formData.logo_url.trim() || null,
      icon_url: formData.icon_url.trim() || null,
      whatsapp_number: formData.whatsapp_number.trim() || null,
      address: formData.address.trim() || null,
      gmaps_url: formData.gmaps_url.trim() || null,
      legalitas: formData.legalitas.trim() || null,
      bank_name: formData.bank_name.trim() || null,
      bank_account_number: formData.bank_account_number.trim() || null,
      bank_account_holder: formData.bank_account_holder.trim() || null,
      social_facebook: formData.social_facebook.trim() || null,
      social_instagram: formData.social_instagram.trim() || null,
      social_tiktok: formData.social_tiktok.trim() || null,
      social_youtube: formData.social_youtube.trim() || null
    };

    try {
      if (isEditMode) {
        await updateBrand(id, payload);
      } else {
        await createBrand(payload);
      }
      navigate('/brands');
    } catch (err) {
      const msg = err.response?.data?.error || "Gagal menyimpan data brand.";
      setFormErrors(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const getMediaUrl = (path) => {
    if (!path) return '';
    return path.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL}${path}` : path;
  };

  if (isLoadingDetail) {
    return (
      <div className="flex justify-center p-12 bg-pure-white rounded-lg border border-neutral-200 shadow-sm">
        <LoadingSpinner />
      </div>
    );
  }

  if (notFound) {
    return (
      <div className="space-y-6">
        <PageHeader title="Brand Tidak Ditemukan" onBack={() => navigate('/brands')} />
        <Alert variant="error">Data brand yang dicari tidak ditemukan atau telah dihapus.</Alert>
      </div>
    );
  }

  return (
    <div className="space-y-6 pb-20">
      <PageHeader 
        title={isEditMode ? "Edit Brand" : "Tambah Brand"} 
        onBack={() => navigate("/brands")}
      />

      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 md:grid-cols-[1fr_320px] gap-6 md:items-start">
          {/* Kolom KIRI (Konten Utama) */}
          <div className="space-y-6">
            
            {/* MetaBox Identitas */}
            <MetaBox title="Identitas Brand">
              <FormField label="Nama Brand" required>
                <Input 
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  required
                  placeholder="Contoh: Azhan Travel"
                />
              </FormField>

              <FormField 
                label="Kode Brand" 
                helperText="2 huruf unik, dipakai untuk format ID Jamaah (mis. HN-2608000031)"
              >
                <Input 
                  name="kode_brand"
                  value={formData.kode_brand}
                  onChange={handleChange}
                  maxLength={2}
                  placeholder="mis. HN"
                  className="font-mono text-sm uppercase"
                />
              </FormField>

              <FormField label="Domain" helperText="Domain unik untuk me-resolve microsite brand">
                <Input 
                  name="domain"
                  value={formData.domain}
                  onChange={handleChange}
                  placeholder="alsha.azhan.test"
                  className="font-mono text-sm"
                />
              </FormField>

              <FormField label="Warna Utama Brand">
                <div className="flex items-center gap-3">
                  <input
                    type="color"
                    name="primary_color"
                    value={formData.primary_color || '#CC904A'}
                    onChange={handleChange}
                    className="w-10 h-10 p-1 bg-pure-white border border-neutral-200 rounded cursor-pointer"
                  />
                  <Input
                    name="primary_color"
                    value={formData.primary_color}
                    onChange={handleChange}
                    placeholder="#CC904A"
                    className="font-mono uppercase"
                  />
                </div>
              </FormField>

              <FormField 
                label="Ikon Brand (Square 1:1)" 
                helperText="Gunakan gambar persegi (1:1), contoh 512x512px — dipakai untuk favicon dan ikon PWA"
              >
                {(localIconPreview || formData.icon_url) && (
                  <div className="mb-3 relative inline-block">
                    <img 
                      src={localIconPreview || getMediaUrl(formData.icon_url)}
                      alt="Preview Ikon"
                      className="w-16 h-16 aspect-square object-cover rounded bg-pure-white border border-neutral-200"
                    />
                    <button
                      type="button"
                      onClick={() => {
                        setLocalIconPreview(null);
                        setFormData(prev => ({ ...prev, icon_url: '' }));
                      }}
                      className="absolute -top-2 -right-2 bg-danger-600 text-pure-white rounded-full w-5 h-5 flex items-center justify-center shadow-sm text-xs"
                      title="Hapus Ikon"
                    >
                      ×
                    </button>
                  </div>
                )}
                <Input 
                  type="file"
                  accept="image/*"
                  onChange={handleIconUpload}
                  disabled={isUploadingIcon}
                />
                {isUploadingIcon && <p className="text-xs text-neutral-500 mt-1">Mengunggah ikon...</p>}
              </FormField>

              <FormField 
                label="Logo Brand (Landscape)" 
                helperText="Gunakan gambar horizontal (landscape), untuk header microsite"
              >
                {(localLogoPreview || formData.logo_url) && (
                  <div className="mb-3 relative inline-block">
                    <img 
                      src={localLogoPreview || getMediaUrl(formData.logo_url)}
                      alt="Preview Logo"
                      className="h-16 w-auto object-contain rounded bg-pure-white border border-neutral-200 p-1"
                    />
                    <button
                      type="button"
                      onClick={() => {
                        setLocalLogoPreview(null);
                        setFormData(prev => ({ ...prev, logo_url: '' }));
                      }}
                      className="absolute -top-2 -right-2 bg-danger-600 text-pure-white rounded-full w-5 h-5 flex items-center justify-center shadow-sm text-xs"
                      title="Hapus Logo"
                    >
                      ×
                    </button>
                  </div>
                )}
                <Input 
                  type="file"
                  accept="image/*"
                  onChange={handleLogoUpload}
                  disabled={isUploadingLogo}
                />
                {isUploadingLogo && <p className="text-xs text-neutral-500 mt-1">Mengunggah logo...</p>}
              </FormField>
            </MetaBox>

            {/* MetaBox Kontak */}
            <MetaBox title="Kontak & Lokasi">
              <FormField label="Nomor WhatsApp">
                <Input 
                  name="whatsapp_number"
                  value={formData.whatsapp_number}
                  onChange={handleChange}
                  placeholder="Contoh: 62812345678"
                />
              </FormField>

              <FormField label="Alamat Kantor">
                <Textarea 
                  name="address"
                  value={formData.address}
                  onChange={handleChange}
                  placeholder="Jl. Utama No. 123, Jakarta"
                  rows={3}
                />
              </FormField>

              <FormField label="URL Google Maps">
                <Input 
                  name="gmaps_url"
                  value={formData.gmaps_url}
                  onChange={handleChange}
                  placeholder="https://maps.google.com/..."
                />
              </FormField>
            </MetaBox>

            {/* MetaBox Legalitas */}
            <MetaBox title="Legalitas">
              <FormField label="Keterangan Legalitas">
                <Textarea 
                  name="legalitas"
                  value={formData.legalitas}
                  onChange={handleChange}
                  placeholder="mis. Nomor izin PPIU: ..., PIHK: ..."
                  rows={3}
                />
              </FormField>
            </MetaBox>

            {/* MetaBox Info Rekening */}
            <MetaBox title="Info Rekening Bank">
              <div className="rounded-lg border border-primary-200 bg-primary-50 p-4">
                <p className="text-sm font-semibold text-neutral-900">Rekening dikelola terpusat</p>
                <p className="mt-1 text-sm leading-6 text-neutral-600">Satu brand dapat memiliki beberapa rekening aktif. Rekening lama sudah dipindahkan otomatis dan tetap tersedia.</p>
                <Button type="button" variant="secondary" className="mt-3" onClick={() => navigate('/bank-accounts')}>Kelola Rekening Bank</Button>
              </div>
            </MetaBox>

            {/* MetaBox Media Sosial */}
            <MetaBox title="Media Sosial">
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
            </MetaBox>
          </div>

          {/* Kolom KANAN (Sidebar Form) */}
          <div className="sticky top-6 flex flex-col gap-6">
            <MetaBox title="Simpan Data">
              <p className="text-sm text-neutral-500">
                {isEditMode 
                  ? "Perubahan data brand akan langsung diperbarui di sistem ERP dan microsite."
                  : "Brand baru yang dibuat akan langsung dapat dipilih untuk scopenya."}
              </p>

              {formErrors && (
                <Alert variant="error">{formErrors}</Alert>
              )}

              <div className="flex flex-col gap-3 pt-2">
                <Button 
                  type="submit" 
                  variant="primary" 
                  isLoading={isSubmitting}
                  disabled={isSubmitting || isUploadingLogo || isUploadingIcon}
                  className="w-full justify-center"
                >
                  {isEditMode ? "Simpan Perubahan" : "Simpan"}
                </Button>

                <Button 
                  type="button" 
                  variant="ghost" 
                  onClick={() => navigate('/brands')}
                  disabled={isSubmitting}
                  className="w-full justify-center"
                >
                  Batal
                </Button>
              </div>
            </MetaBox>
          </div>
        </div>
      </form>
    </div>
  );
};

export default BrandFormPage;

import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import MetaBox from '../components/ui/MetaBox';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import FileInput from '../components/ui/FileInput';
import Textarea from '../components/ui/Textarea';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { getBrand, createBrand, updateBrand } from '../api/brands';
import { uploadMedia } from '../api/media';
import { Building2, Upload, Phone, CreditCard, Globe, Save, CheckCircle2, X } from 'lucide-react';

const initialFormData = {
  kode_brand: '',
  name: '',
  domain: '',
  primary_color: '#FED853',
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
            primary_color: data.primary_color || '#FED853',
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
    <div className="space-y-6">
      <PageHeader 
        title={isEditMode ? "Edit Profil Brand" : "Tambah Brand Baru"} 
        subtitle={isEditMode ? `Perbarui profil, konfigurasi, dan identitas visual brand ${formData.name || ""}` : "Pendaftaran biro travel / brand baru ke dalam ekosistem Azhan"}
        onBack={() => navigate("/brands")}
      />

      {formErrors && (
        <Alert variant="error">{formErrors}</Alert>
      )}

      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
          {/* Kolom KIRI (Konten Utama) */}
          <div className="lg:col-span-2 space-y-6">
            
            {/* MetaBox Identitas */}
            <MetaBox 
              title="Identitas Brand" 
              subtitle="Nama, kode, domain, dan warna tema brand"
              icon={<Building2 size={18} className="text-neutral-700" />}
            >
              <div className="space-y-4">
                <FormField label="Nama Brand" required>
                  <Input 
                    name="name"
                    value={formData.name}
                    onChange={handleChange}
                    required
                    placeholder="Contoh: Azhan Travel"
                  />
                </FormField>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <FormField 
                    label="Kode Brand" 
                    helperText="2 huruf unik untuk format ID Jamaah (mis. HN)"
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

                  <FormField label="Domain Microsite" helperText="Domain untuk me-resolve microsite">
                    <Input 
                      name="domain"
                      value={formData.domain}
                      onChange={handleChange}
                      placeholder="alsha.azhan.test"
                      className="font-mono text-sm"
                    />
                  </FormField>
                </div>

                <FormField label="Warna Utama Brand">
                  <div className="flex items-center gap-3">
                    <input
                      type="color"
                      name="primary_color"
                      value={formData.primary_color || '#FED853'}
                      onChange={handleChange}
                      className="w-10 h-10 p-1 bg-white border border-neutral-200 rounded-xl cursor-pointer"
                    />
                    <Input
                      name="primary_color"
                      value={formData.primary_color}
                      onChange={handleChange}
                      placeholder="#FED853"
                      className="font-mono uppercase"
                    />
                  </div>
                </FormField>
              </div>
            </MetaBox>

            {/* MetaBox Media & Visual Brand */}
            <MetaBox 
              title="Media & Visual Brand" 
              subtitle="Logo dan ikon/favicon resmi brand"
              icon={<Upload size={18} className="text-neutral-700" />}
            >
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <FileInput 
                  label="Logo Brand" 
                  helperText="Format PNG transparan atau SVG direkomendasikan"
                  value={formData.logo_url}
                  previewUrl={localLogoPreview}
                  onChange={handleLogoUpload}
                  onRemove={() => {
                    setLocalLogoPreview(null);
                    setFormData(prev => ({ ...prev, logo_url: '' }));
                  }}
                  isUploading={isUploadingLogo}
                  uploadingText="Mengunggah logo..."
                  placeholder="Pilih file logo brand..."
                />

                <FileInput 
                  label="Ikon Brand (Square 1:1)" 
                  helperText="Format persegi (512x512px) untuk favicon & PWA"
                  value={formData.icon_url}
                  previewUrl={localIconPreview}
                  onChange={handleIconUpload}
                  onRemove={() => {
                    setLocalIconPreview(null);
                    setFormData(prev => ({ ...prev, icon_url: '' }));
                  }}
                  isUploading={isUploadingIcon}
                  uploadingText="Mengunggah ikon..."
                  placeholder="Pilih file ikon brand..."
                />
              </div>
            </MetaBox>

            {/* MetaBox Kontak & Lokasi */}
            <MetaBox 
              title="Kontak & Alamat Kantor" 
              subtitle="WhatsApp, alamat kantor fisik, dan Google Maps"
              icon={<Phone size={18} className="text-neutral-700" />}
            >
              <div className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <FormField label="Nomor WhatsApp CS">
                    <Input 
                      name="whatsapp_number"
                      value={formData.whatsapp_number}
                      onChange={handleChange}
                      placeholder="Contoh: 62812345678"
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
                </div>

                <FormField label="Alamat Kantor Fisik">
                  <Textarea 
                    name="address"
                    value={formData.address}
                    onChange={handleChange}
                    placeholder="Jl. Utama No. 123, Jakarta"
                    rows={3}
                  />
                </FormField>
              </div>
            </MetaBox>

            {/* MetaBox Legalitas */}
            <MetaBox 
              title="Legalitas & Izin Usaha" 
              subtitle="Nomor SK Kemenag PPIU / PIHK resmi"
              icon={<CreditCard size={18} className="text-neutral-700" />}
            >
              <FormField label="Keterangan Legalitas Resmi">
                <Textarea 
                  name="legalitas"
                  value={formData.legalitas}
                  onChange={handleChange}
                  placeholder="mis. Nomor izin PPIU: No. 123 Tahun 2024, Izin PIHK: No. 456..."
                  rows={3}
                />
              </FormField>
            </MetaBox>

            {/* MetaBox Media Sosial */}
            <MetaBox 
              title="Akun Media Sosial" 
              subtitle="Tautan channel sosial media resmi brand"
              icon={<Globe size={18} className="text-neutral-700" />}
            >
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

          {/* Kolom KANAN (Sidebar Aksi & Simpan) */}
          <div className="space-y-6 lg:sticky lg:top-6">
            <MetaBox 
              title="Aksi & Simpan" 
              subtitle="Penyimpanan data brand"
              icon={<Save size={18} className="text-neutral-700" />}
            >
              <div className="space-y-3.5">
                <div className="p-3 bg-neutral-50 rounded-xl border border-neutral-200/80 text-xs text-neutral-600 space-y-1.5 font-body">
                  <div className="flex items-center gap-1.5 text-neutral-900 font-semibold">
                    <CheckCircle2 size={14} className="text-success-600" />
                    <span>Sinkronisasi Otomatis</span>
                  </div>
                  <p>
                    {isEditMode 
                      ? "Perubahan data brand akan langsung diperbarui di seluruh ekosistem ERP dan microsite."
                      : "Brand baru yang dibuat akan langsung aktif dan siap digunakan untuk paket & booking."}
                  </p>
                </div>

                <div className="space-y-2 pt-1">
                  <Button 
                    type="submit" 
                    variant="primary" 
                    isLoading={isSubmitting}
                    disabled={isSubmitting || isUploadingLogo || isUploadingIcon}
                    className="w-full justify-center shadow-sm"
                  >
                    {isSubmitting ? "Menyimpan..." : (isEditMode ? "Perbarui Data Brand" : "Simpan Brand Baru")}
                  </Button>

                  <Button 
                    type="button" 
                    variant="ghost" 
                    onClick={() => navigate('/brands')}
                    disabled={isSubmitting}
                    className="w-full justify-center text-neutral-600 hover:text-neutral-900"
                  >
                    Batal
                  </Button>
                </div>
              </div>
            </MetaBox>
          </div>
        </div>
      </form>
    </div>
  );
};

export default BrandFormPage;

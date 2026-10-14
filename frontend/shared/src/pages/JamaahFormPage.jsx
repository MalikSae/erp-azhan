import React, { useState, useEffect, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { getJamaah, createJamaah, updateJamaah } from "../api/jamaah";
import { listBrands } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import FormLayout, { FormSection, SidebarSection, FormAside } from "../components/ui/FormLayout";
import Input from "../components/ui/Input";
import AutocompleteInput from "../components/ui/AutocompleteInput";
import CustomDropdown from "../components/ui/CustomDropdown";
import Textarea from "../components/ui/Textarea";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import { INDONESIAN_CITIES } from "../data/indonesianCities";

const SECTION_LINKS = [
  { id: "sec-identitas", label: "Identitas" },
  { id: "sec-paspor", label: "Paspor" },
  { id: "sec-kontak", label: "Kontak" },
  { id: "sec-darurat", label: "Kontak Darurat" },
];

export const JamaahFormPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = Boolean(id);

  const [loading, setLoading] = useState(isEdit);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);
  const [notFound, setNotFound] = useState(false);
  // Konten dashboard menggulir di container dalam, bukan window; gulir ke pesan error langsung.
  const errorRef = useRef(null);
  useEffect(() => {
    if (error) errorRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [error]);

  // Brands State (Khusus Office Dashboard / Super Admin)
  const [brands, setBrands] = useState([]);
  const [brandError, setBrandError] = useState(null);

  const [formData, setFormData] = useState({
    brand_id: "",
    nama_lengkap: "",
    jenis_kelamin: "",
    nama_ayah_kandung: "",
    nik: "",
    tempat_lahir: "",
    tanggal_lahir: "",
    no_paspor: "",
    tempat_paspor_keluar: "",
    tanggal_paspor_keluar: "",
    paspor_berlaku_sampai: "",
    no_hp: "",
    email: "",
    alamat: "",
    kota: "",
    pekerjaan: "",
    pendidikan_terakhir: "",
    penjamin_kesehatan: "",
    no_asuransi_bpjs: "",
    emergency_nama: "",
    emergency_nik: "",
    emergency_hp: "",
    emergency_hubungan: "",
    emergency_alamat: "",
    catatan: ""
  });

  // Fetch data jamaah saat mode Edit
  useEffect(() => {
    if (isEdit) {
      getJamaah(id)
        .then((jamaahData) => {
          setFormData(prev => ({
            ...prev,
            ...jamaahData,
            brand_id: jamaahData.brand_id || prev.brand_id || ""
          }));
          setLoading(false);
        })
        .catch((err) => {
          if (err.response?.status === 404) setNotFound(true);
          else setError("Gagal memuat data jamaah");
          setLoading(false);
        });
    }
  }, [id, isEdit]);

  // Fetch daftar Brands jika diakses dari Office Dashboard
  useEffect(() => {
    if (showBrandColumn) {
      listBrands()
        .then(res => {
          const brandData = res || [];
          setBrands(brandData);
        })
        .catch(err => {
          console.error("Gagal memuat daftar brand", err);
        });
    }
  }, [showBrandColumn, isEdit]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleBrandChange = (val) => {
    const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
    setFormData(prev => ({ ...prev, brand_id: actualVal ? Number(actualVal) : "" }));
    if (brandError) setBrandError(null);
  };

  const sanitizePayload = (data) => {
    const sanitized = { ...data };
    for (const [key, value] of Object.entries(sanitized)) {
      if (typeof value === 'string' && value.trim() === '') {
        sanitized[key] = null;
      }
    }
    return sanitized;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);

    // Validasi Brand jika Super Admin di Office Dashboard
    if (showBrandColumn && !formData.brand_id) {
      setBrandError("Pilih biro travel / brand terlebih dahulu");
      setError("Silakan pilih biro travel / brand untuk jamaah ini.");
      return;
    }

    setSubmitting(true);

    try {
      const payloadData = { ...formData };
      if (showBrandColumn && formData.brand_id) {
        payloadData.brand_id = Number(formData.brand_id);
      } else if (!showBrandColumn) {
        delete payloadData.brand_id;
      }

      const payload = sanitizePayload(payloadData);
      
      if (isEdit) {
        await updateJamaah(id, payload);
        navigate(`/jamaah/${id}`, { state: { success: `Data jamaah "${payload.nama_lengkap}" berhasil diperbarui.` } });
      } else {
        const res = await createJamaah({ ...payload, kaitan_agen: { mode: "tanpa_agen" } });
        const newJamaahId = res?.id;
        navigate(newJamaahId ? `/jamaah/${newJamaahId}` : "/jamaah", { state: { success: `Jamaah "${payload.nama_lengkap}" berhasil ditambahkan.` } });
      }
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menyimpan data jamaah");
    } finally {
      setSubmitting(false);
    }
  };

  const handleBack = () => {
    navigate(-1);
  };

  if (loading) return <div className="p-8 flex justify-center"><LoadingSpinner /></div>;

  if (notFound) {
    return (
      <div className="space-y-6">
        <PageHeader title="Jamaah Tidak Ditemukan" onBack={() => navigate("/jamaah")} />
        <Alert variant="error">Data jamaah yang Anda cari tidak ada atau sudah dihapus.</Alert>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader 
        title={isEdit ? "Edit Data Jamaah" : "Tambah Data Jamaah"}
        onBack={handleBack}
      />

      <div ref={errorRef}>
        {error && <Alert variant="error">{error}</Alert>}
      </div>

      <form onSubmit={handleSubmit}>
        {/* Responsive 2-Column Grid Layout: Konten Utama (Kiri) & Sidebar Aksi/Meta (Kanan) */}
        <FormLayout tabs={SECTION_LINKS}>

            {/* 1. IDENTITAS & DATA PRIBADI */}
            <FormSection id="sec-identitas" title="Identitas & Data Pribadi">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Baris 1: Nama Lengkap & Jenis Kelamin */}
                <Input 
                  label="Nama Lengkap (Sesuai KTP / Paspor)" 
                  name="nama_lengkap" 
                  value={formData.nama_lengkap} 
                  onChange={handleChange} 
                  required 
                  placeholder="Contoh: Muhammad Abdullah" 
                />

                <CustomDropdown
                  label="Jenis Kelamin"
                  name="jenis_kelamin"
                  value={formData.jenis_kelamin || ""}
                  onChange={(val) => {
                    const actual = (val && typeof val === "object" && "target" in val) ? val.target.value : val;
                    setFormData(prev => ({ ...prev, jenis_kelamin: actual }));
                  }}
                  placeholder="Pilih jenis kelamin"
                  options={[
                    { value: "L", label: "Laki-laki" },
                    { value: "P", label: "Perempuan" },
                  ]}
                />

                {/* Baris 2: NIK & Nama Ayah Kandung */}
                <Input 
                  label="NIK (Nomor Induk Kependudukan)" 
                  name="nik" 
                  value={formData.nik || ''} 
                  onChange={handleChange} 
                  placeholder="16 digit NIK KTP" 
                  maxLength={16} 
                />
                <Input 
                  label="Nama Ayah Kandung" 
                  name="nama_ayah_kandung" 
                  value={formData.nama_ayah_kandung} 
                  onChange={handleChange} 
                  placeholder="Nama ayah kandung" 
                />

                {/* Baris 3: Tempat & Tanggal Lahir */}
                <AutocompleteInput 
                  label="Tempat Lahir" 
                  name="tempat_lahir" 
                  value={formData.tempat_lahir || ''} 
                  onChange={handleChange} 
                  options={INDONESIAN_CITIES}
                  placeholder="Ketik nama kota/kabupaten..."
                />
                <Input 
                  type="date" 
                  label="Tanggal Lahir" 
                  name="tanggal_lahir" 
                  value={formData.tanggal_lahir || ''} 
                  onChange={handleChange} 
                />

                {/* Baris 4: Pekerjaan & Pendidikan Terakhir */}
                <Input 
                  label="Pekerjaan" 
                  name="pekerjaan" 
                  value={formData.pekerjaan || ''} 
                  onChange={handleChange} 
                  placeholder="Contoh: Pegawai Swasta, Wiraswasta" 
                />
                <Input 
                  label="Pendidikan Terakhir" 
                  name="pendidikan_terakhir" 
                  value={formData.pendidikan_terakhir || ''} 
                  onChange={handleChange} 
                  placeholder="Contoh: S1, SMA, S2" 
                />

                {/* Baris 5: Jaminan Kesehatan & No Asuransi */}
                <Input 
                  label="Penjamin Kesehatan (mis. BPJS)" 
                  name="penjamin_kesehatan" 
                  value={formData.penjamin_kesehatan || ''} 
                  onChange={handleChange} 
                  placeholder="Contoh: BPJS Kesehatan, Mandiri Inhealth" 
                />
                <Input 
                  label="Nomor Asuransi / BPJS" 
                  name="no_asuransi_bpjs" 
                  value={formData.no_asuransi_bpjs || ''} 
                  onChange={handleChange} 
                  placeholder="Nomor kepesertaan asuransi" 
                />
              </div>
            </FormSection>

            {/* 2. DOKUMEN PASPOR */}
            <FormSection id="sec-paspor" title="Dokumen Paspor">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <Input 
                  label="Nomor Paspor" 
                  name="no_paspor" 
                  value={formData.no_paspor || ''} 
                  onChange={handleChange} 
                  placeholder="Contoh: A1234567"
                />
                <AutocompleteInput 
                  label="Kota Penerbit Paspor" 
                  name="tempat_paspor_keluar" 
                  value={formData.tempat_paspor_keluar || ''} 
                  onChange={handleChange} 
                  options={INDONESIAN_CITIES}
                  placeholder="Ketik kota imigrasi penerbit..."
                />
                <Input 
                  type="date" 
                  label="Tanggal Dikeluarkan (Issued Date)" 
                  name="tanggal_paspor_keluar" 
                  value={formData.tanggal_paspor_keluar || ''} 
                  onChange={handleChange} 
                />
                <Input 
                  type="date" 
                  label="Paspor Berlaku Sampai (Expiry Date)" 
                  name="paspor_berlaku_sampai" 
                  value={formData.paspor_berlaku_sampai || ''} 
                  onChange={handleChange} 
                />
              </div>
            </FormSection>

            {/* 3. KONTAK & ALAMAT DOMISILI */}
            <FormSection id="sec-kontak" title="Kontak & Alamat Domisili">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input 
                  label="No HP / WhatsApp" 
                  name="no_hp" 
                  value={formData.no_hp || ''} 
                  onChange={handleChange} 
                  placeholder="Contoh: 081234567890" 
                />
                <Input 
                  type="email" 
                  label="Email" 
                  name="email" 
                  value={formData.email || ''} 
                  onChange={handleChange} 
                  placeholder="nama@email.com" 
                />
                <div className="md:col-span-2">
                  <AutocompleteInput 
                    label="Kota / Kabupaten Domisili" 
                    name="kota" 
                    value={formData.kota || ''} 
                    onChange={handleChange} 
                    options={INDONESIAN_CITIES}
                    placeholder="Ketik nama kota/kabupaten domisili..."
                  />
                </div>
                <div className="md:col-span-2">
                  <Textarea 
                    label="Alamat Lengkap" 
                    name="alamat" 
                    value={formData.alamat || ''} 
                    onChange={handleChange} 
                    rows={3} 
                    placeholder="Nama jalan, nomor rumah, RT/RW, kelurahan, kecamatan..." 
                  />
                </div>
              </div>
            </FormSection>

            {/* 4. KONTAK DARURAT */}
            <FormSection id="sec-darurat" title="Kontak Darurat">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input 
                  label="Nama Kontak Darurat" 
                  name="emergency_nama" 
                  value={formData.emergency_nama || ''} 
                  onChange={handleChange} 
                  placeholder="Nama lengkap kerabat"
                />
                <Input 
                  label="Hubungan Kerabat" 
                  name="emergency_hubungan" 
                  value={formData.emergency_hubungan || ''} 
                  onChange={handleChange} 
                  placeholder="Contoh: Saudara Kandung, Anak"
                />
                <Input 
                  label="No HP Kontak Darurat" 
                  name="emergency_hp" 
                  value={formData.emergency_hp || ''} 
                  onChange={handleChange} 
                  placeholder="Contoh: 081234567890"
                />
                <Input 
                  label="NIK Kontak Darurat" 
                  name="emergency_nik" 
                  value={formData.emergency_nik || ''} 
                  onChange={handleChange} 
                  placeholder="16 digit NIK"
                  maxLength={16}
                />
                <div className="md:col-span-2">
                  <Textarea 
                    label="Alamat Kontak Darurat" 
                    name="emergency_alamat" 
                    value={formData.emergency_alamat || ''} 
                    onChange={handleChange} 
                    rows={2} 
                    placeholder="Alamat domisili kerabat darurat..."
                  />
                </div>
              </div>
            </FormSection>

          <FormAside>
            <SidebarSection title="Simpan">
              <div className="space-y-3">
                <p className="text-xs text-neutral-500 font-body">
                  Pastikan nama dan nomor dokumen identitas sesuai dengan KTP dan paspor fisik jamaah.
                </p>
                <Button
                  type="submit"
                  variant="primary"
                  disabled={submitting}
                  className="w-full justify-center"
                >
                  {submitting ? "Menyimpan..." : (isEdit ? "Perbarui Data Jamaah" : "Simpan Data Jamaah")}
                </Button>
              </div>
            </SidebarSection>

            {/* Panel Afiliasi Biro Travel (Hanya Super Admin di Office Dashboard) */}
            {showBrandColumn && (
              <SidebarSection title={<>Brand <span className="text-danger-500" aria-hidden="true">*</span></>}>
                <div className="space-y-2">
                  <CustomDropdown
                    name="brand_id"
                    className="!mb-0"
                    options={brands.map(b => ({ value: b.id, label: b.name }))}
                    value={formData.brand_id || ""}
                    onChange={handleBrandChange}
                    placeholder="Pilih Brand Travel..."
                    required
                    error={brandError}
                    disabled={isEdit}
                  />
                  <p className="text-[11px] text-neutral-400 font-body">
                    Data jamaah akan terdaftar dan dikelola di bawah unit biro travel yang dipilih.
                  </p>
                </div>
              </SidebarSection>
            )}

            {/* Panel Catatan Khusus Internal */}
            <SidebarSection title="Catatan Internal">
              <div>
                <Textarea 
                  name="catatan" 
                  className="!mb-0"
                  value={formData.catatan || ''} 
                  onChange={handleChange} 
                  rows={4} 
                  placeholder="Catatan riwayat medis, penyakit kronis, kebutuhan kursi roda, atau permintaan khusus kamar..." 
                />
                <p className="text-[11px] text-neutral-400 font-body mt-1.5">
                  Hanya dapat dilihat oleh petugas internal biro travel.
                </p>
              </div>
            </SidebarSection>

          </FormAside>
        </FormLayout>
      </form>
    </div>
  );
};

export default JamaahFormPage;

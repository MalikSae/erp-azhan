import React, { useState, useEffect, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { getJamaah, createJamaah, updateJamaah } from "../api/jamaah";
import { listBrandOptions } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import FormLayout, { FormSection, SidebarSection, FormAside } from "../components/ui/FormLayout";
import Input from "../components/ui/Input";
import AutocompleteInput from "../components/ui/AutocompleteInput";
import CustomDropdown from "../components/ui/CustomDropdown";
import Textarea from "../components/ui/Textarea";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import KaitanAgenPicker, { EMPTY_KAITAN, validateKaitan, toKaitanPayload } from "../components/KaitanAgenPicker";
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
  // Gagal memuat data saat edit: form tidak boleh tampil, karena menyimpan form kosong menimpa data jamaah.
  const [loadFailed, setLoadFailed] = useState(false);
  const [loadAttempt, setLoadAttempt] = useState(0);
  // Kaitan agen jamaah baru: wajib dipilih eksplisit (agen aktif atau tanpa agen).
  const [kaitan, setKaitan] = useState(EMPTY_KAITAN);
  const [kaitanError, setKaitanError] = useState(null);
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
    nama_paspor: "",
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
    if (!isEdit) return;
    let batal = false;
    setLoading(true);
    setLoadFailed(false);
    getJamaah(id)
      .then((jamaahData) => {
        if (batal) return;
        setFormData(prev => ({
          ...prev,
          ...jamaahData,
          brand_id: jamaahData.brand_id || prev.brand_id || ""
        }));
        setLoading(false);
      })
      .catch((err) => {
        if (batal) return;
        if (err.response?.status === 404) setNotFound(true);
        else setLoadFailed(true);
        setLoading(false);
      });
    return () => { batal = true; };
  }, [id, isEdit, loadAttempt]);

  // Fetch daftar Brands jika diakses dari Office Dashboard
  useEffect(() => {
    if (showBrandColumn) {
      listBrandOptions()
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
    // Agen terikat pada brand; ganti brand mengosongkan agen yang sudah dipilih.
    setKaitan(prev => (prev.mode === 'agen' && prev.agen ? { mode: 'agen', agen: null } : prev));
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
      setBrandError("Pilih brand.");
      setError("Pilih brand.");
      return;
    }

    if (!isEdit) {
      const kaitanErr = validateKaitan(kaitan);
      if (kaitanErr) {
        setKaitanError(kaitanErr);
        setError(kaitanErr);
        return;
      }
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
        const res = await createJamaah({ ...payload, kaitan_agen: toKaitanPayload(kaitan) });
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

  if (loadFailed) {
    return (
      <div className="space-y-6">
        <PageHeader title="Edit Jamaah" onBack={handleBack} />
        <Alert variant="error">Gagal memuat data jamaah. Form tidak ditampilkan agar data yang tersimpan tidak tertimpa.</Alert>
        <Button type="button" onClick={() => setLoadAttempt((n) => n + 1)}>Coba Lagi</Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={isEdit ? "Edit Jamaah" : "Tambah Jamaah"}
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
                  label="Nama lengkap"
                  name="nama_lengkap"
                  value={formData.nama_lengkap}
                  onChange={handleChange}
                  required
                  placeholder="Muhammad Abdullah"
                />

                <CustomDropdown
                  label="Jenis kelamin"
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
                  label="NIK"
                  name="nik"
                  value={formData.nik || ''}
                  onChange={handleChange}
                  placeholder="16 digit"
                  maxLength={16}
                />
                <Input
                  label="Nama ayah kandung"
                  name="nama_ayah_kandung"
                  value={formData.nama_ayah_kandung}
                  onChange={handleChange}
                />

                {/* Baris 3: Tempat & Tanggal Lahir */}
                <AutocompleteInput
                  label="Tempat lahir"
                  name="tempat_lahir"
                  value={formData.tempat_lahir || ''}
                  onChange={handleChange}
                  options={INDONESIAN_CITIES}
                  placeholder="Cari kota"
                />
                <Input
                  type="date"
                  label="Tanggal lahir"
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
                  placeholder="Pegawai swasta"
                />
                <Input
                  label="Pendidikan terakhir"
                  name="pendidikan_terakhir"
                  value={formData.pendidikan_terakhir || ''}
                  onChange={handleChange}
                  placeholder="S1"
                />

                {/* Baris 5: Jaminan Kesehatan & No Asuransi */}
                <Input
                  label="Penjamin kesehatan"
                  name="penjamin_kesehatan"
                  value={formData.penjamin_kesehatan || ''}
                  onChange={handleChange}
                  placeholder="BPJS Kesehatan"
                />
                <Input
                  label="Nomor asuransi"
                  name="no_asuransi_bpjs"
                  value={formData.no_asuransi_bpjs || ''}
                  onChange={handleChange}
                />
              </div>
            </FormSection>

            {/* 2. DOKUMEN PASPOR */}
            <FormSection id="sec-paspor" title="Dokumen Paspor">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="sm:col-span-2">
                  <Input
                    label="Nama di paspor"
                    name="nama_paspor"
                    value={formData.nama_paspor || ''}
                    onChange={handleChange}
                    placeholder="SITI AMINAH BINTI ABDULLAH"
                  />
                </div>
                <Input
                  label="Nomor paspor"
                  name="no_paspor"
                  value={formData.no_paspor || ''}
                  onChange={handleChange}
                  placeholder="A1234567"
                />
                <AutocompleteInput
                  label="Kota penerbit"
                  name="tempat_paspor_keluar"
                  value={formData.tempat_paspor_keluar || ''}
                  onChange={handleChange}
                  options={INDONESIAN_CITIES}
                  placeholder="Cari kota"
                />
                <Input
                  type="date"
                  label="Tanggal terbit"
                  name="tanggal_paspor_keluar" 
                  value={formData.tanggal_paspor_keluar || ''} 
                  onChange={handleChange} 
                />
                <Input 
                  type="date" 
                  label="Masa berlaku"
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
                  placeholder="081234567890"
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
                    label="Kota / kabupaten domisili"
                    name="kota" 
                    value={formData.kota || ''} 
                    onChange={handleChange} 
                    options={INDONESIAN_CITIES}
                    placeholder="Cari kota"
                  />
                </div>
                <div className="md:col-span-2">
                  <Textarea 
                    label="Alamat lengkap"
                    name="alamat" 
                    value={formData.alamat || ''} 
                    onChange={handleChange} 
                    rows={3} 
                    placeholder="Jalan, RT/RW, kelurahan, kecamatan"
                  />
                </div>
              </div>
            </FormSection>

            {/* 4. KONTAK DARURAT */}
            <FormSection id="sec-darurat" title="Kontak Darurat">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="Nama kontak darurat"
                  name="emergency_nama"
                  value={formData.emergency_nama || ''}
                  onChange={handleChange}
                />
                <Input
                  label="Hubungan kerabat"
                  name="emergency_hubungan"
                  value={formData.emergency_hubungan || ''}
                  onChange={handleChange}
                  placeholder="Saudara kandung"
                />
                <Input
                  label="No HP kontak darurat"
                  name="emergency_hp"
                  value={formData.emergency_hp || ''}
                  onChange={handleChange}
                  placeholder="081234567890"
                />
                <Input
                  label="NIK kontak darurat"
                  name="emergency_nik"
                  value={formData.emergency_nik || ''}
                  onChange={handleChange}
                  placeholder="16 digit"
                  maxLength={16}
                />
                <div className="md:col-span-2">
                  <Textarea
                    label="Alamat kontak darurat"
                    name="emergency_alamat"
                    value={formData.emergency_alamat || ''}
                    onChange={handleChange}
                    rows={2}
                  />
                </div>
              </div>
            </FormSection>

          <FormAside>
            {/* Panel Afiliasi Biro Travel (Hanya Super Admin di Office Dashboard) */}
            {showBrandColumn && (
              <SidebarSection title={<>Brand <span className="text-danger-500" aria-hidden="true">*</span></>}>
                <CustomDropdown
                  name="brand_id"
                  className="!mb-0"
                  options={brands.map(b => ({ value: b.id, label: b.name }))}
                  value={formData.brand_id || ""}
                  onChange={handleBrandChange}
                  placeholder="Pilih brand"
                  required
                  error={brandError}
                  disabled={isEdit}
                />
              </SidebarSection>
            )}

            {/* Kaitan agen: wajib dipilih saat membuat jamaah. Sesudah itu hanya bisa diganti lewat override admin. */}
            {!isEdit && (
              <SidebarSection>
                <KaitanAgenPicker
                  value={kaitan}
                  onChange={(v) => { setKaitan(v); setKaitanError(null); }}
                  brandId={formData.brand_id || undefined}
                  brandRequired={showBrandColumn}
                  error={kaitanError}
                />
              </SidebarSection>
            )}

            {/* Panel Catatan Khusus Internal */}
            <SidebarSection title="Catatan Internal">
              <Textarea
                name="catatan"
                className="!mb-0"
                value={formData.catatan || ''}
                onChange={handleChange}
                rows={4}
              />
            </SidebarSection>

            <SidebarSection>
              <Button
                type="submit"
                variant="primary"
                disabled={submitting}
                className="w-full justify-center"
              >
                {submitting ? "Menyimpan..." : "Simpan"}
              </Button>
            </SidebarSection>

          </FormAside>
        </FormLayout>
      </form>
    </div>
  );
};

export default JamaahFormPage;

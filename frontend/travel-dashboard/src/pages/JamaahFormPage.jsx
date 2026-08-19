import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { getJamaah, createJamaah, updateJamaah } from "../api/jamaah";
import PageHeader from "../components/ui/PageHeader";
import MetaBox from "../components/ui/MetaBox";
import Input from "../components/ui/Input";
import AutocompleteInput from "../components/ui/AutocompleteInput";
import Textarea from "../components/ui/Textarea";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import { INDONESIAN_CITIES } from "../data/indonesianCities";

const JamaahFormPage = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = Boolean(id);

  const [loading, setLoading] = useState(isEdit);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  const [formData, setFormData] = useState({
    nama_lengkap: "",
    nama_ayah_kandung: "",
    nik: "",
    tempat_lahir: "",
    tanggal_lahir: "",
    no_paspor: "",
    tempat_paspor_keluar: "",
    paspor_berlaku_sampai: "",
    no_hp: "",
    email: "",
    alamat: "",
    pekerjaan: "",
    pendidikan_terakhir: "",
    penjamin_kesehatan: "",
    no_asuransi_bpjs: "",
    emergency_nama: "",
    emergency_nik: "",
    emergency_hp: "",
    emergency_hubungan: "",
    emergency_alamat: ""
  });

  useEffect(() => {
    if (isEdit) {
      getJamaah(id)
        .then(data => {
          setFormData(prev => ({ ...prev, ...data }));
          setLoading(false);
        })
        .catch(err => {
          setError("Gagal memuat data jamaah");
          setLoading(false);
        });
    }
  }, [id, isEdit]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    // Filter empty strings to null for optional fields if needed, 
    // but the backend should handle it or we pass as is.
    try {
      if (isEdit) {
        await updateJamaah(id, formData);
        navigate(`/jamaah/${id}`);
      } else {
        const res = await createJamaah(formData);
        navigate(res?.id ? `/jamaah/${res.id}` : "/jamaah");
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

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <PageHeader 
        title={isEdit ? "Edit Data Jamaah" : "Tambah Data Jamaah"} 
        onBack={handleBack}
      />

      {error && <Alert variant="error" message={error} onClose={() => setError(null)} />}

      <form onSubmit={handleSubmit} className="space-y-6">
        <MetaBox title="Identitas Jamaah">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input label="Nama Lengkap *" name="nama_lengkap" value={formData.nama_lengkap} onChange={handleChange} required />
            <Input label="Nama Ayah Kandung" name="nama_ayah_kandung" value={formData.nama_ayah_kandung} onChange={handleChange} />
            <Input label="NIK" name="nik" value={formData.nik || ''} onChange={handleChange} />
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <AutocompleteInput 
                label="Tempat Lahir" 
                name="tempat_lahir" 
                value={formData.tempat_lahir || ''} 
                onChange={handleChange} 
                options={INDONESIAN_CITIES}
                placeholder="Ketik nama kota/kab..."
              />
              <Input type="date" label="Tanggal Lahir" name="tanggal_lahir" value={formData.tanggal_lahir || ''} onChange={handleChange} />
            </div>
            <Input label="No Paspor" name="no_paspor" value={formData.no_paspor || ''} onChange={handleChange} />
            <AutocompleteInput 
              label="Tempat Paspor Keluar" 
              name="tempat_paspor_keluar" 
              value={formData.tempat_paspor_keluar || ''} 
              onChange={handleChange} 
              options={INDONESIAN_CITIES}
              placeholder="Ketik nama kota..."
            />
            <Input type="date" label="Paspor Berlaku Sampai" name="paspor_berlaku_sampai" value={formData.paspor_berlaku_sampai || ''} onChange={handleChange} />
          </div>
        </MetaBox>

        <MetaBox title="Informasi Kontak & Pekerjaan">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input label="No HP" name="no_hp" value={formData.no_hp || ''} onChange={handleChange} />
            <Input type="email" label="Email" name="email" value={formData.email || ''} onChange={handleChange} />
            <div className="md:col-span-2">
              <Textarea label="Alamat Lengkap" name="alamat" value={formData.alamat || ''} onChange={handleChange} rows={3} />
            </div>
            <Input label="Pekerjaan" name="pekerjaan" value={formData.pekerjaan || ''} onChange={handleChange} />
            <Input label="Pendidikan Terakhir" name="pendidikan_terakhir" value={formData.pendidikan_terakhir || ''} onChange={handleChange} />
            <Input label="Penjamin Kesehatan (mis. BPJS)" name="penjamin_kesehatan" value={formData.penjamin_kesehatan || ''} onChange={handleChange} />
            <Input label="No Asuransi/BPJS" name="no_asuransi_bpjs" value={formData.no_asuransi_bpjs || ''} onChange={handleChange} />
          </div>
        </MetaBox>

        <MetaBox title="Kontak Darurat">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input label="Nama Kontak Darurat" name="emergency_nama" value={formData.emergency_nama || ''} onChange={handleChange} />
            <Input label="Hubungan" name="emergency_hubungan" value={formData.emergency_hubungan || ''} onChange={handleChange} />
            <Input label="No HP Kontak Darurat" name="emergency_hp" value={formData.emergency_hp || ''} onChange={handleChange} />
            <Input label="NIK Kontak Darurat" name="emergency_nik" value={formData.emergency_nik || ''} onChange={handleChange} />
            <div className="md:col-span-2">
              <Textarea label="Alamat Kontak Darurat" name="emergency_alamat" value={formData.emergency_alamat || ''} onChange={handleChange} rows={2} />
            </div>
          </div>
        </MetaBox>

        <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
          <Button type="button" variant="ghost" onClick={handleBack}>Batal</Button>
          <Button type="submit" variant="primary" disabled={submitting}>
            {submitting ? "Menyimpan..." : "Simpan Jamaah"}
          </Button>
        </div>
      </form>
    </div>
  );
};

export default JamaahFormPage;

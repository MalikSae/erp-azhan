import React, { useState, useEffect, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { 
  getJamaah, 
  createJamaah, 
  updateJamaah, 
  listJamaah, 
  listRelasi, 
  createRelasi, 
  updateRelasi, 
  deleteRelasi 
} from "../api/jamaah";
import { listBrands } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import MetaBox from "../components/ui/MetaBox";
import Input from "../components/ui/Input";
import AutocompleteInput from "../components/ui/AutocompleteInput";
import CustomDropdown from "../components/ui/CustomDropdown";
import Textarea from "../components/ui/Textarea";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import { INDONESIAN_CITIES } from "../data/indonesianCities";
import { 
  Building2, 
  User, 
  FileText, 
  MapPin, 
  PhoneCall, 
  ClipboardList, 
  Users, 
  Plus, 
  X, 
  Save,
  CheckCircle2
} from "lucide-react";

const HUBUNGAN_OPTIONS = [
  { value: "Pasangan", label: "Pasangan" },
  { value: "Orang Tua", label: "Orang Tua" },
  { value: "Anak", label: "Anak" },
  { value: "Saudara Kandung", label: "Saudara Kandung" },
  { value: "Mahram", label: "Mahram" },
  { value: "Kerabat Lain", label: "Kerabat Lain" },
];

export const JamaahFormPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = Boolean(id);

  const [loading, setLoading] = useState(isEdit);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  // Brands State (Khusus Master Dashboard / Super Admin)
  const [brands, setBrands] = useState([]);
  const [brandError, setBrandError] = useState(null);

  const [formData, setFormData] = useState({
    brand_id: "",
    nama_lengkap: "",
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

  // Relasi Kekerabatan Repeater State
  const [candidateList, setCandidateList] = useState([]);
  const [relasiList, setRelasiList] = useState([]);
  const [initialRelasiList, setInitialRelasiList] = useState([]);
  const [relasiErrors, setRelasiErrors] = useState({});

  // Fetch data jamaah & relasi saat mode Edit
  useEffect(() => {
    if (isEdit) {
      Promise.all([getJamaah(id), listRelasi(id)])
        .then(([jamaahData, relasiData]) => {
          setFormData(prev => ({ 
            ...prev, 
            ...jamaahData,
            brand_id: jamaahData.brand_id || prev.brand_id || "" 
          }));
          const mappedRelasi = (relasiData || []).map(r => ({
            id: r.id,
            db_relasi_id: r.id,
            relasi_jamaah_id: r.relasi_jamaah_id,
            searchText: `${r.nama_relasi} (${r.id_jamaah_relasi || r.nik_relasi || `ID: ${r.relasi_jamaah_id}`})`,
            hubungan: r.hubungan,
            keterangan: r.keterangan || "",
          }));
          setRelasiList(mappedRelasi);
          setInitialRelasiList(mappedRelasi);
          setLoading(false);
        })
        .catch(() => {
          setError("Gagal memuat data jamaah");
          setLoading(false);
        });
    }
  }, [id, isEdit]);

  // Fetch daftar Brands jika diakses dari Master Dashboard
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

  // Fetch candidate list untuk Autocomplete Relasi
  useEffect(() => {
    const fetchCandidates = async () => {
      try {
        const data = await listJamaah({ status: "aktif" });
        setCandidateList(data || []);
      } catch (err) {
        console.error("Gagal memuat daftar jamaah", err);
      }
    };
    fetchCandidates();
  }, []);

  const candidateOptions = useMemo(() => {
    return candidateList
      .filter((j) => {
        if (isEdit && String(j.id) === String(id)) return false;
        if (showBrandColumn && formData.brand_id && j.brand_id && Number(j.brand_id) !== Number(formData.brand_id)) {
          return false;
        }
        return true;
      })
      .map((j) => ({
        value: j.id,
        label: `${j.nama_lengkap} (${j.id_jamaah || j.nik || `ID: ${j.id}`})`,
        item: j,
      }));
  }, [candidateList, isEdit, id, showBrandColumn, formData.brand_id]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleBrandChange = (val) => {
    const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
    setFormData(prev => ({ ...prev, brand_id: actualVal ? Number(actualVal) : "" }));
    if (brandError) setBrandError(null);
  };

  // Repeater Relasi Handlers
  const handleAddRelasi = () => {
    setRelasiList((prev) => [
      ...prev,
      {
        id: Date.now(),
        relasi_jamaah_id: "",
        searchText: "",
        hubungan: "Pasangan",
      },
    ]);
  };

  const handleRemoveRelasi = (rowId) => {
    setRelasiList((prev) => prev.filter((r) => r.id !== rowId));
    setRelasiErrors((prev) => {
      const next = { ...prev };
      delete next[`row_${rowId}_jamaah`];
      delete next[`row_${rowId}_hubungan`];
      return next;
    });
  };

  const handleRelasiSearchChange = (rowId, value) => {
    const textVal = typeof value === 'string' ? value : (value?.target ? value.target.value : String(value ?? ''));
    setRelasiList(prev => prev.map(row => {
      if (row.id === rowId) {
        return {
          ...row,
          searchText: textVal,
          relasi_jamaah_id: textVal ? row.relasi_jamaah_id : "",
        };
      }
      return row;
    }));

    if (relasiErrors[`row_${rowId}_jamaah`]) {
      setRelasiErrors(prev => {
        const next = { ...prev };
        delete next[`row_${rowId}_jamaah`];
        return next;
      });
    }
  };

  const handleRelasiSelectJamaah = (rowId, selectedItem) => {
    setRelasiList(prev => prev.map(row => {
      if (row.id === rowId) {
        if (!selectedItem) {
          return {
            ...row,
            relasi_jamaah_id: "",
            searchText: "",
          };
        }
        const itemObj = selectedItem?.item;
        const candidateId = selectedItem?.value || itemObj?.id;
        const label = selectedItem?.label || (typeof selectedItem === 'string' ? selectedItem : '');
        return {
          ...row,
          relasi_jamaah_id: candidateId || "",
          searchText: label,
        };
      }
      return row;
    }));

    if (relasiErrors[`row_${rowId}_jamaah`]) {
      setRelasiErrors(prev => {
        const next = { ...prev };
        delete next[`row_${rowId}_jamaah`];
        return next;
      });
    }
  };

  const handleRelasiFieldChange = (rowId, field, value) => {
    setRelasiList((prev) =>
      prev.map((r) => (r.id === rowId ? { ...r, [field]: value } : r))
    );
    if (field === "hubungan") {
      setRelasiErrors((prev) => {
        const next = { ...prev };
        delete next[`row_${rowId}_hubungan`];
        return next;
      });
    }
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

    // Validasi Brand jika Super Admin di Master Dashboard
    if (showBrandColumn && !formData.brand_id) {
      setBrandError("Pilih biro travel / brand terlebih dahulu");
      setError("Silakan pilih biro travel / brand untuk jamaah ini.");
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    // Validasi relasi jika ada baris relasi (berlaku di mode create dan edit)
    if (relasiList.length > 0) {
      const rErrors = {};
      relasiList.forEach((r, idx) => {
        if (!r.relasi_jamaah_id) {
          rErrors[`row_${r.id}_jamaah`] = `Pilih jamaah untuk Relasi #${idx + 1}`;
        }
        if (!r.hubungan) {
          rErrors[`row_${r.id}_hubungan`] = `Pilih hubungan untuk Relasi #${idx + 1}`;
        }
      });

      if (Object.keys(rErrors).length > 0) {
        setRelasiErrors(rErrors);
        setError("Silakan lengkapi baris relasi yang belum valid atau hapus baris jika tidak digunakan.");
        window.scrollTo({ top: 0, behavior: "smooth" });
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

        // Rekonsiliasi sinkronisasi relasi di mode Edit
        const failedRelasi = [];

        // 1. DELETE: Baris yang ada di initialRelasiList tapi sudah dihapus dari UI
        const currentDbIds = new Set(relasiList.filter(r => r.db_relasi_id).map(r => r.db_relasi_id));
        const toDelete = initialRelasiList.filter(r => !currentDbIds.has(r.db_relasi_id));
        for (const delItem of toDelete) {
          try {
            await deleteRelasi(id, delItem.db_relasi_id);
          } catch (delErr) {
            const errMsg = delErr.response?.data?.error || delErr.message || "Gagal menghapus relasi";
            failedRelasi.push(`Hapus relasi (${delItem.searchText || 'Kerabat'}): ${errMsg}`);
          }
        }

        // 2. UPDATE (PUT) & CREATE (POST)
        for (let i = 0; i < relasiList.length; i++) {
          const row = relasiList[i];
          if (row.db_relasi_id) {
            // Baris existing: cek apakah hubungan / keterangan berubah
            const initialItem = initialRelasiList.find(r => r.db_relasi_id === row.db_relasi_id);
            const isHubunganChanged = initialItem && initialItem.hubungan !== row.hubungan;
            const isKeteranganChanged = initialItem && (initialItem.keterangan || "") !== (row.keterangan || "").trim();

            if (isHubunganChanged || isKeteranganChanged) {
              try {
                await updateRelasi(id, row.db_relasi_id, {
                  hubungan: row.hubungan,
                  keterangan: row.keterangan ? row.keterangan.trim() : null,
                });
              } catch (updErr) {
                const errMsg = updErr.response?.data?.error || updErr.message || "Gagal memperbarui relasi";
                failedRelasi.push(`Update Relasi #${i + 1} (${row.searchText || 'Kerabat'}): ${errMsg}`);
              }
            }
          } else {
            // Baris baru tanpa db_relasi_id: POST
            try {
              await createRelasi(id, {
                relasi_jamaah_id: Number(row.relasi_jamaah_id),
                hubungan: row.hubungan,
                keterangan: row.keterangan ? row.keterangan.trim() : null,
              });
            } catch (addErr) {
              const errMsg = addErr.response?.data?.error || addErr.message || "Gagal menambahkan relasi baru";
              failedRelasi.push(`Tambah Relasi #${i + 1} (${row.searchText || 'Kerabat'}): ${errMsg}`);
            }
          }
        }

        if (failedRelasi.length > 0) {
          const warningMsg = `Data jamaah berhasil diperbarui, tapi ada ${failedRelasi.length} relasi yang gagal disinkronkan: ${failedRelasi.join("; ")}`;
          navigate(`/jamaah/${id}`, { state: { error: warningMsg } });
          return;
        }

        navigate(`/jamaah/${id}`);
      } else {
        const res = await createJamaah(payload);
        const newJamaahId = res?.id;

        if (newJamaahId && relasiList.length > 0) {
          const failedRelasi = [];
          for (let i = 0; i < relasiList.length; i++) {
            const row = relasiList[i];
            try {
              await createRelasi(newJamaahId, {
                relasi_jamaah_id: Number(row.relasi_jamaah_id),
                hubungan: row.hubungan,
                keterangan: row.keterangan ? row.keterangan.trim() : null,
              });
            } catch (relErr) {
              const errMsg = relErr.response?.data?.error || relErr.message || "Gagal menambahkan relasi";
              failedRelasi.push(`Relasi #${i + 1} (${row.searchText || 'Jamaah'}): ${errMsg}`);
            }
          }

          if (failedRelasi.length > 0) {
            const warningMsg = `Data jamaah berhasil disimpan, tapi ada ${failedRelasi.length} relasi yang gagal ditambahkan: ${failedRelasi.join("; ")}`;
            navigate(`/jamaah/${newJamaahId}`, { state: { error: warningMsg } });
            return;
          }
        }

        navigate(newJamaahId ? `/jamaah/${newJamaahId}` : "/jamaah");
      }
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menyimpan data jamaah");
      window.scrollTo({ top: 0, behavior: "smooth" });
    } finally {
      setSubmitting(false);
    }
  };

  const handleBack = () => {
    navigate(-1);
  };

  if (loading) return <div className="p-8 flex justify-center"><LoadingSpinner /></div>;

  return (
    <div className="max-w-6xl mx-auto space-y-6 pb-20">
      <PageHeader 
        title={isEdit ? "Edit Data Jamaah" : "Tambah Data Jamaah"} 
        subtitle={isEdit ? "Perbarui informasi lengkap identitas dan dokumen jamaah" : "Daftarkan jamaah baru ke dalam sistem administrasi"}
        onBack={handleBack}
      />

      {error && <Alert variant="error">{error}</Alert>}

      <form onSubmit={handleSubmit}>
        {/* Responsive 2-Column Grid Layout: Konten Utama (Kiri) & Sidebar Aksi/Meta (Kanan) */}
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_340px] gap-6 items-start">
          
          {/* KOLOM KIRI: Data Utama Jamaah */}
          <div className="space-y-6">

            {/* 1. IDENTITAS & DATA PRIBADI */}
            <MetaBox 
              title="Identitas & Data Pribadi" 
              subtitle="Data identitas diri, kependudukan, dan latar belakang jamaah"
              icon={<User size={18} className="text-neutral-700" />}
            >
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Baris 1: Nama Lengkap (Full Width untuk penekanan nama) */}
                <div className="md:col-span-2">
                  <Input 
                    label="Nama Lengkap (Sesuai KTP / Paspor)" 
                    name="nama_lengkap" 
                    value={formData.nama_lengkap} 
                    onChange={handleChange} 
                    required 
                    placeholder="Contoh: Muhammad Abdullah" 
                  />
                </div>

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
            </MetaBox>

            {/* 2. DOKUMEN PASPOR */}
            <MetaBox 
              title="Dokumen Paspor" 
              subtitle="Kelengkapan dokumen paspor jamaah untuk manifest & visa"
              icon={<FileText size={18} className="text-neutral-700" />}
            >
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
            </MetaBox>

            {/* 3. KONTAK & ALAMAT DOMISILI */}
            <MetaBox 
              title="Kontak & Alamat Domisili" 
              subtitle="Informasi nomor kontak aktif dan alamat tempat tinggal"
              icon={<MapPin size={18} className="text-neutral-700" />}
            >
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
            </MetaBox>

            {/* 4. KONTAK DARURAT */}
            <MetaBox 
              title="Kontak Darurat" 
              subtitle="Keluarga atau kerabat yang dapat dihubungi dalam situasi darurat"
              icon={<PhoneCall size={18} className="text-neutral-700" />}
            >
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
            </MetaBox>

            {/* 5. HUBUNGAN KEKERABATAN (OPSIONAL) */}
            <MetaBox
              title="Hubungan Kekerabatan (Opsional)"
              subtitle={isEdit ? "Hubungan dengan anggota keluarga atau mahram yang sudah terdaftar" : "Hubungkan jamaah ini dengan keluarga atau mahram yang sudah terdaftar"}
              icon={<Users size={18} className="text-neutral-700" />}
              headerActions={
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={handleAddRelasi}
                  className="flex items-center gap-1.5 text-xs"
                >
                  <Plus size={14} />
                  <span>Tambah Relasi</span>
                </Button>
              }
            >
              <p className="text-xs text-neutral-500 font-body -mt-1 mb-4">
                Cari jamaah yang sudah terdaftar, lalu tentukan jenis hubungannya.
              </p>

              {relasiList.length === 0 ? (
                <div className="text-center py-6 border-2 border-dashed border-neutral-200 rounded-xl bg-neutral-50/50">
                  <p className="text-xs md:text-sm text-neutral-500 font-body mb-3">
                    Belum ada relasi kekerabatan yang ditambahkan.
                  </p>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={handleAddRelasi}
                    className="inline-flex items-center gap-1.5 text-xs"
                  >
                    <Plus size={14} />
                    <span>+ Tambah Relasi</span>
                  </Button>
                </div>
              ) : (
                <div className="space-y-4">
                  {relasiList.map((row, idx) => {
                    const jamaahError = relasiErrors[`row_${row.id}_jamaah`];
                    const hubunganError = relasiErrors[`row_${row.id}_hubungan`];

                    return (
                      <div
                        key={row.id}
                        className="p-4 rounded-xl border border-neutral-200 bg-white space-y-3 relative hover:border-neutral-300 transition-colors"
                      >
                        <div className="flex items-center justify-between pb-2 border-b border-neutral-100">
                          <span className="font-heading font-semibold text-xs md:text-sm text-neutral-900 flex items-center gap-2">
                            <span>Relasi #{idx + 1}</span>
                            {row.db_relasi_id && (
                              <span className="text-[10px] font-medium text-neutral-500 bg-neutral-100 px-2 py-0.5 rounded-full">
                                Tersimpan
                              </span>
                            )}
                          </span>
                          <button
                            type="button"
                            onClick={() => handleRemoveRelasi(row.id)}
                            className="w-8 h-8 flex items-center justify-center rounded-lg text-neutral-400 hover:text-danger-600 hover:bg-danger-50 transition-colors cursor-pointer"
                            title="Hapus baris relasi"
                            aria-label={`Hapus Relasi #${idx + 1}`}
                          >
                            <X size={16} />
                          </button>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-start">
                          {/* Autocomplete Jamaah */}
                          <div>
                            <AutocompleteInput
                              label="Pilih Jamaah"
                              name={`relasi_jamaah_${row.id}`}
                              value={row.searchText || ""}
                              onChange={(e) => handleRelasiSearchChange(row.id, e.target.value)}
                              onSelect={(opt) => handleRelasiSelectJamaah(row.id, opt)}
                              options={candidateOptions}
                              placeholder="Cari nama atau NIK jamaah..."
                              error={jamaahError}
                              disabled={Boolean(row.db_relasi_id)}
                              minChars={0}
                              required
                            />
                          </div>

                          {/* Dropdown Hubungan */}
                          <div>
                            <CustomDropdown
                              label="Hubungan"
                              name={`hubungan_${row.id}`}
                              options={HUBUNGAN_OPTIONS}
                              value={row.hubungan}
                              onChange={(val) => {
                                const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
                                handleRelasiFieldChange(row.id, "hubungan", actualVal);
                              }}
                              error={hubunganError}
                              required
                            />
                          </div>
                        </div>
                      </div>
                    );
                  })}

                  <div className="mt-4 flex justify-center">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={handleAddRelasi}
                      className="w-full sm:w-auto flex items-center justify-center gap-1.5 text-xs"
                    >
                      <Plus size={14} />
                      <span>Tambah Relasi Lagi</span>
                    </Button>
                  </div>
                </div>
              )}
            </MetaBox>

          </div>

          {/* KOLOM KANAN: Sidebar Aksi, Brand Afiliasi, & Catatan Khusus */}
          <div className="space-y-6 lg:sticky lg:top-6">

            {/* Panel Aksi & Simpan */}
            <MetaBox 
              title="Aksi & Simpan" 
              subtitle="Penyimpanan data pendaftaran"
              icon={<Save size={18} className="text-neutral-700" />}
            >
              <div className="space-y-3.5">
                <div className="p-3 bg-neutral-50 rounded-xl border border-neutral-200/80 text-xs text-neutral-600 space-y-1.5 font-body">
                  <div className="flex items-center gap-1.5 text-neutral-900 font-semibold">
                    <CheckCircle2 size={14} className="text-success-600" />
                    <span>Verifikasi Data</span>
                  </div>
                  <p>Pastikan nama dan nomor dokumen identitas sesuai dengan KTP dan Paspor fisik jamaah.</p>
                </div>

                <div className="space-y-2 pt-1">
                  <Button 
                    type="submit" 
                    variant="primary" 
                    disabled={submitting} 
                    className="w-full justify-center shadow-sm"
                  >
                    {submitting ? "Menyimpan..." : (isEdit ? "Perbarui Data Jamaah" : "Simpan Data Jamaah")}
                  </Button>
                  <Button 
                    type="button" 
                    variant="ghost" 
                    onClick={handleBack} 
                    className="w-full justify-center text-neutral-600 hover:text-neutral-900"
                  >
                    Batal
                  </Button>
                </div>
              </div>
            </MetaBox>

            {/* Panel Afiliasi Biro Travel (Hanya Super Admin di Master Dashboard) */}
            {showBrandColumn && (
              <MetaBox 
                title="Afiliasi Biro Travel" 
                subtitle="Penetapan biro travel pengelola"
                icon={<Building2 size={18} className="text-neutral-700" />}
              >
                <div className="space-y-2">
                  <CustomDropdown
                    label="Biro Travel / Brand"
                    name="brand_id"
                    options={brands.map(b => ({ value: b.id, label: b.name }))}
                    value={formData.brand_id || ""}
                    onChange={handleBrandChange}
                    placeholder="Pilih Brand Travel..."
                    required
                    error={brandError}
                    disabled={isEdit}
                  />
                  <p className="text-[11px] text-neutral-500 font-body">
                    Data jamaah akan terdaftar dan dikelola di bawah unit biro travel yang dipilih.
                  </p>
                </div>
              </MetaBox>
            )}

            {/* Panel Catatan Khusus Internal */}
            <MetaBox 
              title="Catatan Khusus (Internal)" 
              subtitle="Kondisi fisik, alergi, atau preferensi khusus"
              icon={<ClipboardList size={18} className="text-neutral-700" />}
            >
              <div>
                <Textarea 
                  label="Catatan Internal" 
                  name="catatan" 
                  value={formData.catatan || ''} 
                  onChange={handleChange} 
                  rows={4} 
                  placeholder="Catatan riwayat medis, penyakit kronis, kebutuhan kursi roda, atau permintaan khusus kamar..." 
                />
                <p className="text-[11px] text-neutral-400 font-body mt-1.5">
                  Hanya dapat dilihat oleh petugas internal biro travel.
                </p>
              </div>
            </MetaBox>

          </div>

        </div>
      </form>
    </div>
  );
};

export default JamaahFormPage;

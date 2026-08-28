import React, { useState, useEffect, useMemo } from "react";
import { useNavigate, useParams, Link } from "react-router-dom";
import { getJamaah, createJamaah, updateJamaah, listJamaah, createRelasi } from "../api/jamaah";
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
import { Plus, X } from "lucide-react";

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
  const [relasiErrors, setRelasiErrors] = useState({});

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

  useEffect(() => {
    const fetchCandidates = async () => {
      try {
        const data = await listJamaah();
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
        if (formData.brand_id && j.brand_id && j.brand_id !== formData.brand_id) return false;
        return true;
      })
      .map((j) => ({
        value: j.id,
        label: `${j.nama_lengkap} (${j.id_jamaah || j.nik || `ID: ${j.id}`})`,
        item: j,
      }));
  }, [candidateList, isEdit, id, formData.brand_id]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
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

    // Validasi relasi di mode create jika ada baris relasi
    if (!isEdit && relasiList.length > 0) {
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
        return;
      }
    }

    setSubmitting(true);

    try {
      const payload = sanitizePayload(formData);
      if (isEdit) {
        await updateJamaah(id, payload);
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
    } finally {
      setSubmitting(false);
    }
  };

  const handleBack = () => {
    navigate(-1);
  };

  if (loading) return <div className="p-8 flex justify-center"><LoadingSpinner /></div>;

  return (
    <div className="max-w-3xl mx-auto space-y-6 pb-12">
      <PageHeader 
        title={isEdit ? "Edit Data Jamaah" : "Tambah Data Jamaah"} 
        onBack={handleBack}
      />

      {error && <Alert variant="error">{error}</Alert>}

      <form onSubmit={handleSubmit} className="space-y-6">
        <MetaBox title="Identitas Jamaah">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input label="Nama Lengkap" name="nama_lengkap" value={formData.nama_lengkap} onChange={handleChange} required />
            <Input label="Nama Ayah Kandung" name="nama_ayah_kandung" value={formData.nama_ayah_kandung} onChange={handleChange} />
            <Input label="NIK (Nomor Induk Kependudukan)" name="nik" value={formData.nik || ''} onChange={handleChange} />
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
          </div>
        </MetaBox>

        <MetaBox title="Data Paspor">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Input 
              label="Nomor Paspor" 
              name="no_paspor" 
              value={formData.no_paspor || ''} 
              onChange={handleChange} 
              placeholder="Contoh: A1234567"
            />
            <AutocompleteInput 
              label="Tempat Paspor Diterbitkan" 
              name="tempat_paspor_keluar" 
              value={formData.tempat_paspor_keluar || ''} 
              onChange={handleChange} 
              options={INDONESIAN_CITIES}
              placeholder="Ketik kota penerbit..."
            />
            <Input 
              type="date" 
              label="Paspor Berlaku Sampai" 
              name="paspor_berlaku_sampai" 
              value={formData.paspor_berlaku_sampai || ''} 
              onChange={handleChange} 
            />
          </div>
        </MetaBox>

        <MetaBox title="Kontak & Alamat Domisili">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input label="No HP / WhatsApp" name="no_hp" value={formData.no_hp || ''} onChange={handleChange} placeholder="Contoh: 081234567890" />
            <Input type="email" label="Email" name="email" value={formData.email || ''} onChange={handleChange} placeholder="nama@email.com" />
            <div className="md:col-span-2">
              <Textarea label="Alamat Lengkap" name="alamat" value={formData.alamat || ''} onChange={handleChange} rows={3} placeholder="Nama jalan, RT/RW, kelurahan, kecamatan..." />
            </div>
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
          </div>
        </MetaBox>

        <MetaBox title="Pekerjaan & Jaminan Kesehatan">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input label="Pekerjaan" name="pekerjaan" value={formData.pekerjaan || ''} onChange={handleChange} placeholder="Contoh: Pegawai Swasta, Wiraswasta" />
            <Input label="Pendidikan Terakhir" name="pendidikan_terakhir" value={formData.pendidikan_terakhir || ''} onChange={handleChange} placeholder="Contoh: S1, SMA" />
            <Input label="Penjamin Kesehatan (mis. BPJS)" name="penjamin_kesehatan" value={formData.penjamin_kesehatan || ''} onChange={handleChange} placeholder="Contoh: BPJS Kesehatan, Mandiri Inhealth" />
            <Input label="Nomor Asuransi / BPJS" name="no_asuransi_bpjs" value={formData.no_asuransi_bpjs || ''} onChange={handleChange} placeholder="Nomor kepesertaan asuransi" />
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

        <MetaBox title="Catatan Khusus (Internal)">
          <div>
            <Textarea 
              label="Catatan Khusus Jamaah" 
              name="catatan" 
              value={formData.catatan || ''} 
              onChange={handleChange} 
              rows={3} 
              placeholder="Catatan tambahan mengenai kondisi fisik, riwayat penyakit, atau preferensi jamaah..." 
            />
          </div>
        </MetaBox>

        {/* Section Hubungan Kekerabatan (Opsional) */}
        {isEdit ? (
          <MetaBox title="Hubungan Kekerabatan (Opsional)">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 py-1">
              <p className="text-xs md:text-sm text-neutral-600 font-body">
                Relasi kekerabatan dikelola langsung melalui halaman detail jamaah.
              </p>
              <Link
                to={`/jamaah/${id}`}
                className="text-xs md:text-sm font-semibold text-primary-700 hover:text-primary-800 hover:underline inline-flex items-center gap-1 shrink-0"
              >
                Kelola relasi kekerabatan di halaman detail →
              </Link>
            </div>
          </MetaBox>
        ) : (
          <MetaBox
            title={
              <div className="flex items-center justify-between w-full">
                <span>Hubungan Kekerabatan (Opsional)</span>
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
              </div>
            }
          >
            <p className="text-xs text-neutral-500 font-body -mt-1 mb-4">
              Cari jamaah, lalu pilih hubungannya.
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
                        <span className="font-heading font-semibold text-xs md:text-sm text-neutral-900">
                          Relasi #{idx + 1}
                        </span>
                        <button
                          type="button"
                          onClick={() => handleRemoveRelasi(row.id)}
                          className="p-1.5 rounded text-neutral-400 hover:text-danger-600 hover:bg-danger-50 transition-colors cursor-pointer"
                          title="Hapus baris relasi"
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
                            placeholder="Cari jamaah"
                            error={jamaahError}
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
        )}

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

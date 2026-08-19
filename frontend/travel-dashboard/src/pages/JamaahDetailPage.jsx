import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getJamaah } from "../api/jamaah";
import { listDokumen, upsertDokumen } from "../api/dokumen";
import { uploadMedia } from "../api/media";
import { listBookings } from "../api/bookings";
import PageHeader from "../components/ui/PageHeader";
import MetaBox from "../components/ui/MetaBox";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import Table from "../components/ui/Table";
import { Edit2, Eye, Upload, ExternalLink, RefreshCw } from "lucide-react";

const DOKUMEN_TYPES = [
  { key: "pas_foto", label: "Pas Foto" },
  { key: "paspor", label: "Paspor" },
  { key: "ktp", label: "KTP" },
  { key: "kk", label: "Kartu Keluarga" },
  { key: "buku_nikah", label: "Buku Nikah" },
  { key: "akte_lahir", label: "Akte Lahir" },
];

const formatRupiah = (angka) => {
  if (!angka && angka !== 0) return "-";
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", minimumFractionDigits: 0 }).format(angka);
};

const formatTanggal = (dateStr) => {
  if (!dateStr) return "-";
  try {
    const raw = String(dateStr).split("T")[0];
    const parts = raw.split("-");
    if (parts.length === 3 && parts[0].length === 4) {
      return `${parts[2]}-${parts[1]}-${parts[0]}`;
    }
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    const day = String(d.getDate()).padStart(2, "0");
    const month = String(d.getMonth() + 1).padStart(2, "0");
    const year = d.getFullYear();
    return `${day}-${month}-${year}`;
  } catch (e) {
    return dateStr;
  }
};

const formatBulanTahun = (dateStr) => {
  if (!dateStr) return "-";
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("id-ID", { month: "long", year: "numeric" });
  } catch (e) {
    return dateStr;
  }
};

const InfoItem = ({ label, value, colSpan = false }) => (
  <div className={colSpan ? "md:col-span-2" : ""}>
    <span className="block text-xs text-neutral-500 font-body mb-0.5">{label}</span>
    <p className="text-sm font-medium font-body text-neutral-900 break-words">{value || "-"}</p>
  </div>
);

const JamaahDetailPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const fileInputRef = useRef(null);

  const [jamaah, setJamaah] = useState(null);
  const [dokumenList, setDokumenList] = useState([]);
  const [bookings, setBookings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);

  const [activeUploadJenis, setActiveUploadJenis] = useState(null);
  const [uploadingJenis, setUploadingJenis] = useState(null);

  const fetchAllData = async () => {
    try {
      setLoading(true);
      setError(null);
      const [jamaahRes, dokRes, bookRes] = await Promise.all([
        getJamaah(id),
        listDokumen(id),
        listBookings({ jamaahId: id }),
      ]);
      setJamaah(jamaahRes);
      setDokumenList(dokRes || []);
      setBookings(bookRes || []);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memuat detail data jamaah");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAllData();
  }, [id]);

  const handleTriggerUpload = (jenis) => {
    setActiveUploadJenis(jenis);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
      fileInputRef.current.click();
    }
  };

  const handleFileChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file || !activeUploadJenis) return;

    try {
      setUploadingJenis(activeUploadJenis);
      setError(null);
      setSuccess(null);

      const fileUrl = await uploadMedia(file, "dokumen-jamaah");
      await upsertDokumen(id, { jenis: activeUploadJenis, file_url: fileUrl });

      setSuccess(`Dokumen ${activeUploadJenis.replace('_', ' ')} berhasil diunggah.`);
      const updatedDocs = await listDokumen(id);
      setDokumenList(updatedDocs || []);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal mengunggah dokumen");
    } finally {
      setUploadingJenis(null);
      setActiveUploadJenis(null);
    }
  };

  if (loading) {
    return (
      <div className="p-12 flex justify-center items-center">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="w-full space-y-6">
      {/* Hidden file input for document upload */}
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        className="hidden"
        accept="image/*,application/pdf"
      />

      <PageHeader 
        title={jamaah ? jamaah.nama_lengkap : "Detail Jamaah"} 
        onBack={() => navigate(-1)}
      />

      {error && <Alert variant="error" message={error} onClose={() => setError(null)} />}
      {success && <Alert variant="success" message={success} onClose={() => setSuccess(null)} />}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* ─── KOLOM KIRI: DATA JAMAAH (READ-ONLY) ─── */}
        <div className="lg:col-span-7 space-y-6">
          <MetaBox 
            title="Identitas Jamaah"
            action={
              <Button 
                variant="secondary" 
                size="sm" 
                onClick={() => navigate(`/jamaah/${id}/edit`)}
                className="flex items-center gap-1.5"
              >
                <Edit2 size={14} />
                <span>Edit Data</span>
              </Button>
            }
          >
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <InfoItem label="ID Jamaah" value={jamaah?.id_jamaah} />
              <InfoItem label="Nama Lengkap" value={jamaah?.nama_lengkap} />
              <InfoItem label="Nama Ayah Kandung" value={jamaah?.nama_ayah_kandung} />
              <InfoItem label="NIK" value={jamaah?.nik} />
              <InfoItem label="Tempat Lahir" value={jamaah?.tempat_lahir} />
              <InfoItem label="Tanggal Lahir" value={formatTanggal(jamaah?.tanggal_lahir)} />
              <InfoItem label="No Paspor" value={jamaah?.no_paspor} />
              <InfoItem label="Tempat Paspor Keluar" value={jamaah?.tempat_paspor_keluar} />
              <InfoItem label="Paspor Berlaku Sampai" value={formatTanggal(jamaah?.paspor_berlaku_sampai)} />
            </div>
          </MetaBox>

          <MetaBox title="Informasi Kontak & Pekerjaan">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <InfoItem label="No HP" value={jamaah?.no_hp} />
              <InfoItem label="Email" value={jamaah?.email} />
              <InfoItem label="Alamat Lengkap" value={jamaah?.alamat} colSpan />
              <InfoItem label="Pekerjaan" value={jamaah?.pekerjaan} />
              <InfoItem label="Pendidikan Terakhir" value={jamaah?.pendidikan_terakhir} />
              <InfoItem label="Penjamin Kesehatan" value={jamaah?.penjamin_kesehatan} />
              <InfoItem label="No Asuransi/BPJS" value={jamaah?.no_asuransi_bpjs} />
            </div>
          </MetaBox>

          <MetaBox title="Kontak Darurat">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <InfoItem label="Nama Kontak Darurat" value={jamaah?.emergency_nama} />
              <InfoItem label="Hubungan" value={jamaah?.emergency_hubungan} />
              <InfoItem label="No HP Kontak Darurat" value={jamaah?.emergency_hp} />
              <InfoItem label="NIK Kontak Darurat" value={jamaah?.emergency_nik} />
              <InfoItem label="Alamat Kontak Darurat" value={jamaah?.emergency_alamat} colSpan />
            </div>
          </MetaBox>
        </div>

        {/* ─── KOLOM KANAN: DOKUMEN & RIWAYAT BOOKING ─── */}
        <div className="lg:col-span-5 space-y-6">
          {/* SECTION: KELOLA DOKUMEN */}
          <MetaBox title="Kelola Dokumen">
            <div className="divide-y divide-neutral-100">
              {DOKUMEN_TYPES.map((type) => {
                const doc = dokumenList.find((d) => d.jenis === type.key);
                const isUploading = uploadingJenis === type.key;

                return (
                  <div 
                    key={type.key} 
                    className="py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 first:pt-0 last:pb-0"
                  >
                    <div className="space-y-1">
                      <span className="text-sm font-medium font-heading text-neutral-900">
                        {type.label}
                      </span>
                      {doc?.catatan && (
                        <p className="text-xs text-danger-600 font-body">
                          Catatan: {doc.catatan}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-2 self-start sm:self-auto shrink-0">
                      {doc && doc.file_url && (
                        <a
                          href={
                            doc.file_url.startsWith("http")
                              ? doc.file_url
                              : `${import.meta.env.VITE_API_BASE_URL || "http://localhost:9090"}${doc.file_url}`
                          }
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-md text-xs font-medium font-body text-neutral-700 bg-neutral-100 hover:bg-neutral-200 transition-colors"
                        >
                          <ExternalLink size={12} />
                          <span>Lihat</span>
                        </a>
                      )}

                      <Button
                        variant={doc ? "ghost" : "primary"}
                        size="sm"
                        onClick={() => handleTriggerUpload(type.key)}
                        disabled={isUploading}
                        className="flex items-center gap-1 text-xs !py-1.5"
                      >
                        {isUploading ? (
                          <>
                            <RefreshCw size={12} className="animate-spin" />
                            <span>Unggah...</span>
                          </>
                        ) : doc ? (
                          <>
                            <Upload size={12} />
                            <span>Ganti</span>
                          </>
                        ) : (
                          <>
                            <Upload size={12} />
                            <span>Upload</span>
                          </>
                        )}
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          </MetaBox>

          {/* SECTION: RIWAYAT BOOKING */}
          <MetaBox title="Riwayat Booking">
            {bookings.length === 0 ? (
              <p className="text-sm text-neutral-500 font-body py-1">
                Jamaah ini belum punya riwayat booking.
              </p>
            ) : (
              <div className="divide-y divide-neutral-100">
                {bookings.map((b) => (
                  <div
                    key={b.id}
                    onClick={() => navigate(`/bookings/${b.id}`)}
                    className="py-3 flex items-center justify-between gap-3 hover:bg-neutral-50 rounded-md px-2 -mx-2 cursor-pointer transition-colors group"
                  >
                    <div className="space-y-1 min-w-0 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-sm font-medium font-heading text-neutral-900 group-hover:text-primary-600 transition-colors">
                          {b.jadwal_nama}
                        </span>
                      </div>
                      <p className="text-xs text-neutral-500 font-body">
                        Bulan Berangkat: <span className="font-medium text-neutral-700">{formatBulanTahun(b.berangkat_tanggal)}</span>
                      </p>
                    </div>
                    <div className="shrink-0 text-neutral-400 group-hover:text-primary-600 transition-colors">
                      <Eye size={16} />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </MetaBox>
        </div>
      </div>
    </div>
  );
};

export default JamaahDetailPage;

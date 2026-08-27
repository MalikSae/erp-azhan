import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getJamaah } from "../api/jamaah";
import { listDokumen, upsertDokumen } from "../api/dokumen";
import { uploadMedia } from "../api/media";
import { listBookings } from "../api/bookings";
import { listBrands } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import MetaBox from "../components/ui/MetaBox";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import Badge from "../components/ui/Badge";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import Table from "../components/ui/Table";
import BrandCell from "../components/BrandCell";
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

export const JamaahDetailPage = ({ showBrandColumn = false }) => {
  const { id } = useParams();
  const navigate = useNavigate();
  const fileInputRef = useRef(null);

  const [jamaah, setJamaah] = useState(null);
  const [dokumenList, setDokumenList] = useState([]);
  const [bookings, setBookings] = useState([]);
  const [brand, setBrand] = useState(null);
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

      if (showBrandColumn && jamaahRes?.brand_id) {
        try {
          const brandsData = await listBrands();
          const found = (brandsData || []).find(b => b.id === jamaahRes.brand_id);
          setBrand(found || null);
        } catch (e) {
          // ignore brand fetch error
        }
      }
    } catch (err) {
      setError(err.response?.data?.error || "Gagal memuat detail data jamaah");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAllData();
  }, [id, showBrandColumn]);

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

  if (!jamaah) {
    return (
      <div className="space-y-4">
        <Alert variant="error">Data jamaah tidak ditemukan</Alert>
        <Button variant="ghost" onClick={() => navigate("/jamaah")}>← Kembali ke Daftar Jamaah</Button>
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
        accept=".jpg,.jpeg,.png,.pdf,.webp"
      />

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-3">
          <PageHeader
            title={jamaah.nama_lengkap}
            onBack={() => navigate("/jamaah")}
          />
          {showBrandColumn && (brand || jamaah.brand_id) && (
            <div className="mt-1">
              <BrandCell brand={brand} brandId={jamaah.brand_id} showText={true} />
            </div>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => navigate(`/jamaah/${id}/edit`)}
            className="flex items-center gap-1.5"
          >
            <Edit2 size={14} />
            <span>Edit Data</span>
          </Button>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}
      {success && <Alert variant="success">{success}</Alert>}

      {/* Main Grid: Data Identitas & Info Lainnya */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom Kiri: Identitas Utama (2 span) */}
        <div className="lg:col-span-2 space-y-6">
          <MetaBox title="Identitas Pribadi">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="ID Jamaah" value={jamaah.id_jamaah} />
              <InfoItem label="Nama Ayah Kandung" value={jamaah.nama_ayah_kandung} />
              <InfoItem label="NIK (Nomor Induk Kependudukan)" value={jamaah.nik} />
              <InfoItem
                label="Tempat, Tanggal Lahir"
                value={
                  jamaah.tempat_lahir || jamaah.tanggal_lahir
                    ? `${jamaah.tempat_lahir || "-"}, ${formatTanggal(jamaah.tanggal_lahir)}`
                    : "-"
                }
              />
              <InfoItem label="No HP / WhatsApp" value={jamaah.no_hp} />
              <InfoItem label="Email" value={jamaah.email} />
              <InfoItem label="Alamat Lengkap" value={jamaah.alamat} colSpan={true} />
            </div>
          </MetaBox>

          <MetaBox title="Pendidikan & Pekerjaan">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Pekerjaan" value={jamaah.pekerjaan} />
              <InfoItem label="Pendidikan Terakhir" value={jamaah.pendidikan_terakhir} />
              <InfoItem label="Penjamin Kesehatan" value={jamaah.penjamin_kesehatan} />
              <InfoItem label="No. BPJS / Asuransi" value={jamaah.no_asuransi_bpjs} />
            </div>
          </MetaBox>

          <MetaBox title="Kontak Darurat (Emergency Contact)">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Nama Kontak Darurat" value={jamaah.emergency_nama} />
              <InfoItem label="Hubungan Keluarga" value={jamaah.emergency_hubungan} />
              <InfoItem label="No. HP Darurat" value={jamaah.emergency_hp} />
              <InfoItem label="NIK Kontak Darurat" value={jamaah.emergency_nik} />
              <InfoItem label="Alamat Kontak Darurat" value={jamaah.emergency_alamat} colSpan={true} />
            </div>
          </MetaBox>
        </div>

        {/* Kolom Kanan: Paspor & Dokumen Digital (1 span) */}
        <div className="space-y-6">
          <MetaBox title="Dokumen Paspor">
            <div className="space-y-4">
              <InfoItem label="Nomor Paspor" value={jamaah.no_paspor} />
              <InfoItem label="Kantor Imigrasi Penerbit" value={jamaah.tempat_paspor_keluar} />
              <InfoItem label="Masa Berlaku Paspor" value={formatTanggal(jamaah.paspor_berlaku_sampai)} />
            </div>
          </MetaBox>

          <MetaBox title="Dokumen Digital Jamaah">
            <div className="divide-y divide-neutral-100">
              {DOKUMEN_TYPES.map((docType) => {
                const existing = dokumenList.find((d) => d.jenis === docType.key);
                const hasFile = Boolean(existing && existing.file_url);
                const isUploading = uploadingJenis === docType.key;

                return (
                  <div key={docType.key} className="py-3 first:pt-0 last:pb-0 flex items-center justify-between gap-2">
                    <div>
                      <span className="text-sm font-medium font-body text-neutral-800 block">
                        {docType.label}
                      </span>
                      <span className="text-xs font-body text-neutral-500 block">
                        {hasFile ? "Sudah Diunggah" : "Belum Diunggah"}
                      </span>
                    </div>

                    <div className="flex items-center gap-1.5 shrink-0">
                      {hasFile && (
                        <a
                          href={existing.file_url.startsWith('http') ? existing.file_url : `${import.meta.env.VITE_API_BASE_URL}${existing.file_url}`}
                          target="_blank"
                          rel="noreferrer"
                          className="p-1.5 text-neutral-500 hover:text-primary-600 rounded transition-colors"
                          title="Lihat Dokumen"
                        >
                          <ExternalLink size={16} />
                        </a>
                      )}

                      <Button
                        type="button"
                        variant={hasFile ? "ghost" : "secondary"}
                        size="sm"
                        disabled={isUploading}
                        onClick={() => handleTriggerUpload(docType.key)}
                        className="text-xs px-2.5 py-1 flex items-center gap-1"
                      >
                        {isUploading ? (
                          <RefreshCw size={12} className="animate-spin" />
                        ) : (
                          <Upload size={12} />
                        )}
                        <span>{hasFile ? "Ganti" : "Upload"}</span>
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          </MetaBox>
        </div>
      </div>

      {/* Riwayat Booking */}
      <MetaBox title="Riwayat Booking & Keberangkatan">
        {bookings.length === 0 ? (
          <p className="text-sm text-neutral-500 font-body">Belum ada riwayat booking untuk jamaah ini.</p>
        ) : (
          <Table
            columns={[
              { header: "ID Booking", key: "id_booking" },
              { header: "Paket Umroh", key: "jadwal_nama" },
              { header: "Tgl Berangkat", key: "berangkat_tanggal" },
              { header: "Kamar", key: "room_type" },
              { header: "Total Biaya", key: "total_harga" },
              { header: "Status", key: "status" },
              { header: "Aksi", key: "aksi" },
            ]}
            data={bookings}
            renderCell={(row, key) => {
              if (key === "id_booking") {
                return (
                  <span className="font-mono text-xs font-bold text-neutral-900 bg-neutral-100 px-2 py-0.5 rounded border border-neutral-200">
                    {row.id_booking || `ID: ${row.id}`}
                  </span>
                );
              }
              if (key === "berangkat_tanggal") return formatBulanTahun(row.berangkat_tanggal);
              if (key === "total_harga") return formatRupiah(row.total_harga);
              if (key === "status") {
                const mapVariant = {
                  baru: "neutral",
                  dp: "warning",
                  lunas: "success",
                  batal: "danger",
                };
                return (
                  <Badge variant={mapVariant[row.status] || "neutral"} hideIcon={true}>
                    {row.status?.toUpperCase()}
                  </Badge>
                );
              }
              if (key === "aksi") {
                return (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => navigate(`/bookings/${row.id}`)}
                    title="Lihat Detail Booking"
                  >
                    <Eye size={16} className="text-primary-600" />
                  </Button>
                );
              }
              return row[key];
            }}
          />
        )}
      </MetaBox>
    </div>
  );
};

export default JamaahDetailPage;

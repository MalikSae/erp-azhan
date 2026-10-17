import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate, useLocation, Link } from "react-router-dom";
import { getJamaah, updateCatatan } from "../api/jamaah";
import { listDokumen, upsertDokumen } from "../api/dokumen";
import { uploadMedia, openProtectedMedia } from "../api/media";
import { listBookings } from "../api/bookings";
import { listBrands } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import { Surface, Section, Field, SummaryItem, PILL_STRONG, BrandBadge, TableCard } from "../components/ui/DetailLayout";
import Badge from "../components/ui/Badge";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import ActionMenu from "../components/ui/ActionMenu";
import LoadingSpinner from "../components/ui/LoadingSpinner";
import Table from "../components/ui/Table";
import CustomDropdown from "../components/ui/CustomDropdown";
import Textarea from "../components/ui/Textarea";
import DocumentReview from "../components/DocumentReview";
import { Eye, Upload, ExternalLink, RefreshCw, Save, Printer } from "lucide-react";
import { STATUS_AGEN } from "../utils/agen";
import { printJamaah } from "../utils/printJamaah";

const DOKUMEN_TYPES = [
  { key: "pas_foto", label: "Pas Foto" },
  { key: "paspor", label: "Paspor" },
  { key: "ktp", label: "KTP" },
  { key: "kk", label: "Kartu Keluarga" },
  { key: "buku_nikah", label: "Buku Nikah" },
  { key: "akte_lahir", label: "Akte Lahir" },
  { key: "vaksin_meningitis", label: "Vaksin Meningitis" },
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

const formatTanggalWaktu = (dateStr) => {
  if (!dateStr) return "-";
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleString("id-ID", {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch (e) {
    return dateStr;
  }
};

const InfoItem = ({ label, value, colSpan = false }) => (
  <Field label={label} className={colSpan ? "md:col-span-2" : ""}>
    {value || "-"}
  </Field>
);

// Status = titik berwarna kecil + teks (bukan pil berwarna).
const StatusDot = ({ label, text, dot }) => (
  <span className={`inline-flex items-center gap-1.5 text-[12px] font-body whitespace-nowrap ${text}`}>
    <span className={`w-2 h-2 rounded-full ${dot}`} />
    {label}
  </span>
);

const BOOKING_STATUS = {
  baru: { text: "text-info-700", dot: "bg-info-500" },
  dp: { text: "text-warning-700", dot: "bg-warning-500" },
  lunas: { text: "text-success-700", dot: "bg-success-500" },
  batal: { text: "text-danger-700", dot: "bg-danger-500" },
};
const NEUTRAL_STATUS = { text: "text-neutral-500", dot: "bg-neutral-300" };
const AGEN_STATUS_STYLE = {
  success: { text: "text-success-700", dot: "bg-success-500" },
  neutral: NEUTRAL_STATUS,
};

export const JamaahDetailPage = ({ showBrandColumn = false }) => {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const fileInputRef = useRef(null);

  const [jamaah, setJamaah] = useState(null);
  const [dokumenList, setDokumenList] = useState([]);
  const [bookings, setBookings] = useState([]);
  const [brand, setBrand] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);

  // Catatan
  const [catatanText, setCatatanText] = useState("");
  const [savingCatatan, setSavingCatatan] = useState(false);
  const [catatanSuccess, setCatatanSuccess] = useState(null);



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
      setCatatanText(jamaahRes.catatan || "");
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

      if (location.state?.error) {
        setError(location.state.error);
      } else if (location.state?.success) {
        setSuccess(location.state.success);
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

  // ─── Catatan Actions ──────────────────────────────────────────────────────────

  const handleSaveCatatan = async () => {
    try {
      setSavingCatatan(true);
      setCatatanSuccess(null);
      setError(null);
      await updateCatatan(id, catatanText ? catatanText.trim() : null);
      setCatatanSuccess("Catatan jamaah berhasil diperbarui.");
      setTimeout(() => setCatatanSuccess(null), 3000);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menyimpan catatan jamaah");
    } finally {
      setSavingCatatan(false);
    }
  };

  // ─── Cetak / PDF ──────────────────────────────────────────────────────────────

  const assetSrc = (url) => {
    if (!url) return "";
    if (url.startsWith("http")) return url;
    return `${import.meta.env.VITE_API_BASE_URL || "http://localhost:9090"}${url.startsWith("/") ? "" : "/"}${url}`;
  };

  const handlePrint = () => {
    const gender = jamaah.jenis_kelamin === "L" ? "Laki-laki" : jamaah.jenis_kelamin === "P" ? "Perempuan" : "";
    const ttl = jamaah.tempat_lahir || jamaah.tanggal_lahir
      ? `${jamaah.tempat_lahir || "-"}, ${formatTanggal(jamaah.tanggal_lahir)}`
      : "";
    printJamaah({
      nama: jamaah.nama_lengkap || jamaah.id_jamaah || `ID: ${jamaah.id}`,
      idJamaah: jamaah.id_jamaah || jamaah.kode_jamaah || "",
      noHp: jamaah.no_hp || "",
      brandName: brand?.name || "",
      accentColor: brand?.primary_color || "",
      logoUrl: brand?.logo_url ? assetSrc(brand.logo_url) : (brand?.icon_url ? assetSrc(brand.icon_url) : ""),
      identitas: [
        ["Nama Lengkap", jamaah.nama_lengkap],
        ["Jenis Kelamin", gender],
        ["Nama Ayah Kandung", jamaah.nama_ayah_kandung],
        ["NIK", jamaah.nik],
        ["Tempat, Tanggal Lahir", ttl],
        ["No HP / WhatsApp", jamaah.no_hp],
        ["Email", jamaah.email],
        ["Kota / Kabupaten", jamaah.kota],
        ["Alamat Lengkap", jamaah.alamat],
      ],
      pekerjaan: [
        ["Pekerjaan", jamaah.pekerjaan],
        ["Pendidikan Terakhir", jamaah.pendidikan_terakhir],
        ["Penjamin Kesehatan", jamaah.penjamin_kesehatan],
        ["No BPJS / Asuransi", jamaah.no_asuransi_bpjs],
      ],
      paspor: [
        ["Nomor Paspor", jamaah.no_paspor],
        ["Kantor Imigrasi Penerbit", jamaah.tempat_paspor_keluar],
        ["Tanggal Dikeluarkan", jamaah.tanggal_paspor_keluar ? formatTanggal(jamaah.tanggal_paspor_keluar) : ""],
        ["Masa Berlaku", jamaah.paspor_berlaku_sampai ? formatTanggal(jamaah.paspor_berlaku_sampai) : ""],
      ],
      darurat: [
        ["Nama Kontak Darurat", jamaah.emergency_nama],
        ["Hubungan", jamaah.emergency_hubungan],
        ["No HP Kontak Darurat", jamaah.emergency_hp],
        ["NIK Kontak Darurat", jamaah.emergency_nik],
        ["Alamat Kontak Darurat", jamaah.emergency_alamat],
      ],
      printedAt: formatTanggalWaktu(new Date().toISOString()),
    });
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
        <Button variant="secondary" onClick={() => navigate("/jamaah")}>
          Kembali ke Daftar Jamaah
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Hidden File Input for Document Upload */}
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        className="hidden"
        accept=".jpg,.jpeg,.png,.pdf,.webp"
      />

      {/* Header */}
      <PageHeader
        title={jamaah.nama_lengkap || jamaah.id_jamaah || `ID: ${jamaah.id}`}
        breadcrumb={false}
        subtitle={
          <span className="inline-flex flex-wrap items-center gap-2">
            <Badge variant="neutral" hideIcon className={`${PILL_STRONG} !font-mono`}>
              {jamaah.id_jamaah || jamaah.kode_jamaah || `ID: ${jamaah.id}`}
            </Badge>
            {showBrandColumn && (brand || jamaah.brand_id) && (
              <BrandBadge name={brand?.name || `Brand #${jamaah.brand_id}`} color={brand?.primary_color} />
            )}
          </span>
        }
        onBack={() => navigate("/jamaah")}
        actionLabel="Edit Data"
        onAction={() => navigate(`/jamaah/${id}/edit`)}
      >
        <Button variant="secondary" onClick={handlePrint}>
          <Printer size={16} strokeWidth={1.75} />
          <span>Cetak / PDF</span>
        </Button>
      </PageHeader>

      {error && <Alert variant="error">{error}</Alert>}
      {success && <Alert variant="success">{success}</Alert>}

      {/* Ringkasan: satu strip */}
      <Surface className="grid grid-cols-3 divide-x divide-neutral-200">
        <SummaryItem
          label="Booking"
          value={bookings.length}
          hint={bookings.length > 0 ? "Lihat riwayat di bawah" : "Belum ada booking"}
        />
        <SummaryItem
          label="Dokumen terunggah"
          value={`${dokumenList.filter((d) => d.file_url).length}/${DOKUMEN_TYPES.length}`}
        />
        <SummaryItem
          label="Akun portal"
          value={jamaah.portal_aktif ? "Aktif" : "Belum aktif"}
        />
      </Surface>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* Kolom kiri: satu card, section dipisah garis tipis */}
        <Surface className="lg:col-span-2 divide-y divide-neutral-200/80">
          <Section title="Identitas Pribadi">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Nama Lengkap" value={jamaah.nama_lengkap} />
              <InfoItem label="Jenis Kelamin" value={jamaah.jenis_kelamin === 'L' ? 'Laki-laki' : jamaah.jenis_kelamin === 'P' ? 'Perempuan' : '-'} />
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
              <InfoItem label="Kota / Kabupaten" value={jamaah.kota} />
              <InfoItem label="Alamat Lengkap" value={jamaah.alamat} colSpan={true} />
            </div>
          </Section>

          <Section title="Pendidikan & Pekerjaan">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Pekerjaan" value={jamaah.pekerjaan} />
              <InfoItem label="Pendidikan Terakhir" value={jamaah.pendidikan_terakhir} />
              <InfoItem label="Penjamin Kesehatan" value={jamaah.penjamin_kesehatan} />
              <InfoItem label="No BPJS / Asuransi" value={jamaah.no_asuransi_bpjs} />
            </div>
          </Section>

          <Section title="Dokumen Paspor">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Nomor Paspor" value={jamaah.no_paspor} />
              <InfoItem label="Kantor Imigrasi Penerbit" value={jamaah.tempat_paspor_keluar} />
              <InfoItem label="Tanggal Dikeluarkan (Issued Date)" value={formatTanggal(jamaah.tanggal_paspor_keluar)} />
              <InfoItem label="Masa Berlaku Paspor (Expiry Date)" value={formatTanggal(jamaah.paspor_berlaku_sampai)} />
            </div>
          </Section>

          <Section title="Kontak Darurat">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Nama Kontak Darurat" value={jamaah.emergency_nama} />
              <InfoItem label="Hubungan" value={jamaah.emergency_hubungan} />
              <InfoItem label="No HP Kontak Darurat" value={jamaah.emergency_hp} />
              <InfoItem label="NIK Kontak Darurat" value={jamaah.emergency_nik} />
              <InfoItem label="Alamat Kontak Darurat" value={jamaah.emergency_alamat} colSpan={true} />
            </div>
          </Section>
        </Surface>

        {/* Kolom kanan: satu card */}
        <Surface className="divide-y divide-neutral-200/80">
          <Section title="Akun Pelanggan">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-neutral-500 font-body">Status</span>
              <StatusDot
                label={jamaah.portal_aktif ? "Punya akun" : "Belum punya akun"}
                text={jamaah.portal_aktif ? "text-success-700" : "text-warning-700"}
                dot={jamaah.portal_aktif ? "bg-success-500" : "bg-warning-500"}
              />
            </div>
          </Section>
          <Section title="Agen Syiar">
            <div className="space-y-3">
              <div className="flex items-center justify-between gap-3">
                <span className="text-xs font-medium text-neutral-500 font-body">Agen yang mengajak</span>
                {jamaah.kaitan_status === "terikat_agen" && jamaah.direkrut_oleh_jamaah_id ? (
                  <Link
                    to={`/agen/${jamaah.direkrut_oleh_jamaah_id}`}
                    className="text-sm font-semibold text-neutral-900 underline underline-offset-2 text-right"
                  >
                    {jamaah.direkrut_oleh_nama || `#${jamaah.direkrut_oleh_jamaah_id}`}
                  </Link>
                ) : (
                  <span className="text-sm font-medium text-neutral-700">
                    {jamaah.kaitan_status === "tanpa_agen" ? "Tanpa agen" : "Belum ditentukan"}
                  </span>
                )}
              </div>
              {STATUS_AGEN[jamaah.status_agen] && (
                <div className="flex items-center justify-between gap-3 border-t border-neutral-100 pt-3">
                  <span className="text-xs font-medium text-neutral-500 font-body">Jamaah ini juga agen</span>
                  <Link to={`/agen/${jamaah.id}`}>
                    <StatusDot
                      label={STATUS_AGEN[jamaah.status_agen].label}
                      {...(AGEN_STATUS_STYLE[STATUS_AGEN[jamaah.status_agen].variant] || NEUTRAL_STATUS)}
                    />
                  </Link>
                </div>
              )}
            </div>
          </Section>

          <Section title="Dokumen Digital">
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
                        {hasFile ? existing.status : "Belum Diunggah"}
                      </span>
                    </div>

                    <div className="flex items-center gap-1.5 shrink-0">
                      {hasFile && <DocumentReview document={existing} onChanged={updated => setDokumenList(previous => previous.map(doc => doc.id === updated.id ? updated : doc))} />}
                      {hasFile && (
                        <button
                          type="button"
                          onClick={() => openProtectedMedia(existing.file_url)}
                          className="p-1.5 text-neutral-500 hover:text-primary-600 rounded transition-colors"
                          title="Lihat Dokumen"
                        >
                          <ExternalLink size={16} />
                        </button>
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
          </Section>

          <Section title="Catatan Internal">
            <div className="space-y-3">
              <Textarea
                name="catatan"
                value={catatanText}
                onChange={(e) => setCatatanText(e.target.value)}
                placeholder="Tulis catatan internal mengenai jamaah ini (mis. riwayat penyakit, preferensi khusus)..."
                rows={4}
              />
              {catatanSuccess && (
                <p className="text-xs text-success-700 font-body font-medium">
                  ✓ {catatanSuccess}
                </p>
              )}
              <div className="flex justify-end">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  disabled={savingCatatan}
                  onClick={handleSaveCatatan}
                  className="flex items-center gap-1.5 text-xs"
                >
                  <Save size={13} />
                  <span>{savingCatatan ? "Menyimpan..." : "Simpan Catatan"}</span>
                </Button>
              </div>
            </div>
          </Section>
        </Surface>
      </div>

      {/* Riwayat Booking: judul di luar kartu, kartu hanya berisi tabel (aturan tabel) */}
      <TableCard title="Riwayat Booking & Keberangkatan">
        {bookings.length === 0 ? (
          <p className="p-5 text-sm text-neutral-500 font-body">Belum ada riwayat booking untuk jamaah ini.</p>
        ) : (
          <div className="overflow-x-auto">
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
                  <span className="font-mono text-xs font-semibold text-neutral-900">
                    {row.id_booking || `ID: ${row.id}`}
                  </span>
                );
              }
              if (key === "berangkat_tanggal") return formatBulanTahun(row.berangkat_tanggal);
              if (key === "total_harga") return formatRupiah(row.total_harga);
              if (key === "status") {
                const s = BOOKING_STATUS[row.status] || NEUTRAL_STATUS;
                return <StatusDot label={row.status?.toUpperCase()} text={s.text} dot={s.dot} />;
              }
              if (key === "aksi") {
                return (
                  <ActionMenu
                    items={[
                      {
                        label: "Lihat Detail",
                        icon: Eye,
                        tooltip: "Lihat Detail Booking",
                        onClick: () => navigate(`/bookings/${row.id}`),
                      },
                    ]}
                  />
                );
              }
              return row[key];
            }}
          />
          </div>
        )}
      </TableCard>


    </div>
  );
};

export default JamaahDetailPage;

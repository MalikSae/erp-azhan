import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate, useLocation, Link } from "react-router-dom";
import { getJamaah, listRelasi, createRelasi, deleteRelasi, updateCatatan, listJamaah } from "../api/jamaah";
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
import Modal from "../components/ui/Modal";
import CustomDropdown from "../components/ui/CustomDropdown";
import Textarea from "../components/ui/Textarea";
import BrandCell from "../components/BrandCell";
import { Edit2, Eye, Upload, ExternalLink, RefreshCw, Users, Plus, Save, ArrowRight, Building2 } from "lucide-react";

const DOKUMEN_TYPES = [
  { key: "pas_foto", label: "Pas Foto" },
  { key: "paspor", label: "Paspor" },
  { key: "ktp", label: "KTP" },
  { key: "kk", label: "Kartu Keluarga" },
  { key: "buku_nikah", label: "Buku Nikah" },
  { key: "akte_lahir", label: "Akte Lahir" },
];

const HUBUNGAN_OPTIONS = [
  { value: "Pasangan", label: "Pasangan" },
  { value: "Orang Tua", label: "Orang Tua" },
  { value: "Anak", label: "Anak" },
  { value: "Saudara Kandung", label: "Saudara Kandung" },
  { value: "Mahram", label: "Mahram" },
  { value: "Kerabat Lain", label: "Kerabat Lain" },
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
  const location = useLocation();
  const fileInputRef = useRef(null);

  const [jamaah, setJamaah] = useState(null);
  const [dokumenList, setDokumenList] = useState([]);
  const [bookings, setBookings] = useState([]);
  const [brand, setBrand] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);

  // Relasi Kekerabatan
  const [relasiList, setRelasiList] = useState([]);
  const [isRelasiModalOpen, setIsRelasiModalOpen] = useState(false);
  const [candidateList, setCandidateList] = useState([]);
  const [loadingCandidates, setLoadingCandidates] = useState(false);
  const [relasiForm, setRelasiForm] = useState({
    relasi_jamaah_id: "",
    hubungan: "Pasangan",
    keterangan: "",
  });
  const [savingRelasi, setSavingRelasi] = useState(false);
  const [relasiModalError, setRelasiModalError] = useState(null);

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
      const [jamaahRes, dokRes, bookRes, relasiRes] = await Promise.all([
        getJamaah(id),
        listDokumen(id),
        listBookings({ jamaahId: id }),
        listRelasi(id),
      ]);
      setJamaah(jamaahRes);
      setCatatanText(jamaahRes.catatan || "");
      setDokumenList(dokRes || []);
      setBookings(bookRes || []);
      setRelasiList(relasiRes || []);

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

  // ─── Relasi Actions ──────────────────────────────────────────────────────────

  const handleOpenRelasiModal = async () => {
    setRelasiModalError(null);
    setRelasiForm({
      relasi_jamaah_id: "",
      hubungan: "Pasangan",
      keterangan: "",
    });
    setIsRelasiModalOpen(true);

    try {
      setLoadingCandidates(true);
      const allJamaah = await listJamaah({ status: "aktif" });
      // Filter: brand sama & exclude diri sendiri
      const currentBrandId = jamaah?.brand_id;
      const currentId = Number(id);
      const filtered = (allJamaah || []).filter(
        (j) => j.id !== currentId && (!currentBrandId || j.brand_id === currentBrandId)
      );
      setCandidateList(filtered);
    } catch (err) {
      setRelasiModalError("Gagal mengambil daftar jamaah untuk pilihan relasi");
    } finally {
      setLoadingCandidates(false);
    }
  };

  const handleSaveRelasi = async (e) => {
    e.preventDefault();
    if (!relasiForm.relasi_jamaah_id) {
      setRelasiModalError("Pilih jamaah target relasi");
      return;
    }

    try {
      setSavingRelasi(true);
      setRelasiModalError(null);
      await createRelasi(id, {
        relasi_jamaah_id: Number(relasiForm.relasi_jamaah_id),
        hubungan: relasiForm.hubungan,
        keterangan: relasiForm.keterangan ? relasiForm.keterangan.trim() : null,
      });

      setSuccess("Hubungan kekerabatan berhasil ditambahkan.");
      setIsRelasiModalOpen(false);
      const updatedRelasi = await listRelasi(id);
      setRelasiList(updatedRelasi || []);
    } catch (err) {
      setRelasiModalError(err.response?.data?.error || "Gagal menambahkan relasi");
    } finally {
      setSavingRelasi(false);
    }
  };

  const handleDeleteRelasi = async (relasiId, targetNama) => {
    if (!window.confirm(`Apakah Anda yakin ingin menghapus relasi dengan ${targetNama}?`)) {
      return;
    }

    try {
      setError(null);
      setSuccess(null);
      await deleteRelasi(id, relasiId);
      setSuccess("Relasi kekerabatan berhasil dihapus.");
      const updatedRelasi = await listRelasi(id);
      setRelasiList(updatedRelasi || []);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal menghapus relasi");
    }
  };

  // ─── Catatan Actions ─────────────────────────────────────────────────────────

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

  const candidateOptions = candidateList.map((j) => ({
    value: j.id,
    label: `${j.nama_lengkap} (${j.id_jamaah || j.nik || `ID: ${j.id}`})`,
  }));

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
        title={jamaah.id_jamaah || jamaah.kode_jamaah || `ID: ${jamaah.id}`}
        onBack={() => navigate("/jamaah")}
      >
        <Button
          variant="secondary"
          size="sm"
          onClick={() => navigate(`/jamaah/${id}/edit`)}
          className="flex items-center gap-1.5 shadow-2xs"
        >
          <Edit2 size={14} />
          <span>Edit Data</span>
        </Button>
      </PageHeader>

      {error && <Alert variant="error">{error}</Alert>}
      {success && <Alert variant="success">{success}</Alert>}

      {/* Main Grid: Data Identitas & Info Lainnya */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom Kiri: Identitas Utama (2 span) */}
        <div className="lg:col-span-2 space-y-6">
          <MetaBox title="Identitas Pribadi">
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
              <InfoItem label="Alamat Lengkap" value={jamaah.alamat} />
            </div>
          </MetaBox>

          <MetaBox title="Pendidikan & Pekerjaan">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Pekerjaan" value={jamaah.pekerjaan} />
              <InfoItem label="Pendidikan Terakhir" value={jamaah.pendidikan_terakhir} />
              <InfoItem label="Penjamin Kesehatan" value={jamaah.penjamin_kesehatan} />
              <InfoItem label="No BPJS / Asuransi" value={jamaah.no_asuransi_bpjs} />
            </div>
          </MetaBox>

          <MetaBox title="Kontak Darurat (Emergency Contact)">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-6">
              <InfoItem label="Nama Kontak Darurat" value={jamaah.emergency_nama} />
              <InfoItem label="Hubungan" value={jamaah.emergency_hubungan} />
              <InfoItem label="No HP Kontak Darurat" value={jamaah.emergency_hp} />
              <InfoItem label="NIK Kontak Darurat" value={jamaah.emergency_nik} />
              <InfoItem label="Alamat Kontak Darurat" value={jamaah.emergency_alamat} colSpan={true} />
            </div>
          </MetaBox>

          {/* Hubungan Kekerabatan & Mahram */}
          <MetaBox 
            title="Hubungan Kekerabatan & Mahram"
            headerActions={
              <Button
                variant="secondary"
                size="sm"
                onClick={handleOpenRelasiModal}
                className="flex items-center gap-1.5 text-xs"
              >
                <Plus size={14} />
                <span>Tambah Kerabat</span>
              </Button>
            }
          >
            {relasiList.length === 0 ? (
              <p className="text-sm text-neutral-500 font-body">
                Belum ada relasi kekerabatan / mahram yang terdaftar untuk jamaah ini.
              </p>
            ) : (
              <div className="divide-y divide-neutral-100">
                {relasiList.map((rel) => (
                  <div
                    key={rel.id}
                    className="py-3 first:pt-0 last:pb-0 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                  >
                    <div className="space-y-0.5">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-semibold font-heading text-neutral-900">
                          {rel.nama_relasi}
                        </span>
                        <span className="text-xs font-bold font-body px-2 py-0.5 rounded-full bg-neutral-100 text-neutral-800 border border-neutral-200">
                          {rel.hubungan}
                        </span>
                      </div>
                      {rel.id_jamaah_relasi && (
                        <p className="text-xs text-neutral-500 font-mono font-medium">
                          {rel.id_jamaah_relasi}
                        </p>
                      )}
                      {rel.keterangan && (
                        <p className="text-xs text-neutral-500 font-body">
                          {rel.keterangan}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/jamaah/${rel.relasi_jamaah_id}`)}
                        className="text-xs flex items-center gap-1 text-neutral-600 hover:text-neutral-900"
                        title="Buka Profil Kerabat"
                      >
                        <span>Lihat Profil</span>
                        <ArrowRight size={13} />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </MetaBox>
        </div>

        {/* Kolom Kanan: Afiliasi Biro Travel, Paspor, Dokumen Digital, Catatan (1 span) */}
        <div className="space-y-6">
          {/* Afiliasi Biro Travel (Khusus Super Admin / Master Dashboard) */}
          {showBrandColumn && (brand || jamaah.brand_id) && (
            <MetaBox 
              title="Afiliasi Biro Travel"
              icon={<Building2 size={18} className="text-neutral-700" />}
            >
              <div className="py-0.5">
                <BrandCell brand={brand} brandId={jamaah.brand_id} showText={true} />
              </div>
            </MetaBox>
          )}

          <MetaBox title="Dokumen Paspor">
            <div className="space-y-4">
              <InfoItem label="Nomor Paspor" value={jamaah.no_paspor} />
              <InfoItem label="Kantor Imigrasi Penerbit" value={jamaah.tempat_paspor_keluar} />
              <InfoItem label="Tanggal Dikeluarkan (Issued Date)" value={formatTanggal(jamaah.tanggal_paspor_keluar)} />
              <InfoItem label="Masa Berlaku Paspor (Expiry Date)" value={formatTanggal(jamaah.paspor_berlaku_sampai)} />
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

          {/* Catatan Jamaah */}
          <MetaBox title="Catatan Khusus Jamaah">
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

      {/* Modal Tambah Relasi Kekerabatan */}
      <Modal
        isOpen={isRelasiModalOpen}
        onClose={() => setIsRelasiModalOpen(false)}
        title="Tambah Hubungan Kekerabatan / Mahram"
        maxWidth="max-w-lg"
      >
        <form onSubmit={handleSaveRelasi} className="space-y-4">
          {relasiModalError && <Alert variant="error">{relasiModalError}</Alert>}

          <CustomDropdown
            label="Pilih Jamaah Kerabat"
            options={candidateOptions}
            value={relasiForm.relasi_jamaah_id}
            onChange={(val) => setRelasiForm((prev) => ({ ...prev, relasi_jamaah_id: val }))}
            placeholder={loadingCandidates ? "Memuat daftar jamaah..." : "Pilih jamaah..."}
            disabled={loadingCandidates}
          />

          <CustomDropdown
            label="Hubungan Kekerabatan"
            options={HUBUNGAN_OPTIONS}
            value={relasiForm.hubungan}
            onChange={(val) => setRelasiForm((prev) => ({ ...prev, hubungan: val }))}
          />

          <Textarea
            label="Keterangan (Opsional)"
            name="keterangan"
            value={relasiForm.keterangan}
            onChange={(e) => setRelasiForm((prev) => ({ ...prev, keterangan: e.target.value }))}
            placeholder="Catatan tambahan (mis. No. Kartu Keluarga sama, Mahram resmi)..."
            rows={2}
          />

          <div className="flex justify-end gap-2 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setIsRelasiModalOpen(false)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              size="sm"
              disabled={savingRelasi || loadingCandidates}
            >
              {savingRelasi ? "Menyimpan..." : "Simpan Hubungan"}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default JamaahDetailPage;

import { TableCard } from "../ui/DetailLayout";
import Badge from "../ui/Badge";
import EmptyState from "../ui/EmptyState";
import Table from "../ui/Table";
import Toggle from "../ui/Toggle";

// Status tahap yang diisi di tempat lain, dengan label yang menyebut keadaannya (bukan "Diproses").
// Perlengkapan diserahkan di Distribusi Perlengkapan; tidak berlaku untuk infant ("-").
const StatusPerlengkapan = ({ tahap }) => {
  if (!tahap) return <span className="text-xs text-neutral-400">-</span>;
  return tahap.status === "selesai"
    ? <Badge variant="success">Diserahkan</Badge>
    : <Badge variant="neutral">Belum diserahkan</Badge>;
};

// Paspor dari dokumen jamaah: disetujui, ditolak, terunggah menunggu dicek, atau belum diunggah.
const StatusPaspor = ({ tahap, terunggah }) => {
  if (tahap?.status === "selesai") return <Badge variant="success">Disetujui</Badge>;
  if (tahap?.status === "perbaiki") return <Badge variant="danger">Ditolak</Badge>;
  if (terunggah) return <Badge variant="warning">Perlu dicek</Badge>;
  return <Badge variant="neutral">Belum diunggah</Badge>;
};

const CentangTahap = ({ checked, onChange, label }) => (
  <input
    type="checkbox"
    className="w-4 h-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-600 focus:ring-offset-0 transition-colors cursor-pointer"
    checked={checked}
    onChange={onChange}
    aria-label={label}
  />
);

const KOLOM = [
  { header: "Nama Jamaah", key: "nama" },
  { header: "Perlengkapan", key: "perlengkapan", align: "center" },
  { header: "Paspor", key: "paspor", align: "center" },
  { header: "SISKOPATUH", key: "siskopatuh", align: "center" },
  { header: "Visa", key: "visa", align: "center" },
];

// Tahapan umroh di Detail Booking: urutan dan status sama dengan yang dilihat jamaah dan agen. Siap berangkat
// dihitung server (booking.siap_berangkat). onTogglePax(paxId, kunci, nilaiSekarang); onToggleTiket(nilaiSekarang).
const TahapanUmrohCard = ({ booking, paxList, onTogglePax, onToggleTiket }) => (
  <TableCard
    title="Tahapan Umroh"
    aside={booking.siap_berangkat
      ? <Badge variant="success">Siap Berangkat</Badge>
      : <Badge variant="warning">Persiapan Berjalan</Badge>}
  >
    {paxList.length === 0 ? (
      <EmptyState variant="section" message="Tidak ada jamaah aktif." />
    ) : (
      <>
        <Table
          columns={KOLOM}
          data={paxList}
          renderCell={(pax, key) => {
            const tahap = Object.fromEntries((pax.tahapan || []).map((t) => [t.kunci, t]));
            if (key === "nama") {
              return (
                <span className="flex flex-col">
                  <span className="font-heading font-semibold text-neutral-900">{pax.nama_jamaah || `Jamaah #${pax.jamaah_id}`}</span>
                  {pax.pax_type === "infant" && <span className="text-xs text-neutral-400 font-body">Infant</span>}
                </span>
              );
            }
            if (key === "perlengkapan") return <StatusPerlengkapan tahap={tahap.perlengkapan} />;
            if (key === "paspor") return <StatusPaspor tahap={tahap.paspor} terunggah={Boolean(pax.progress_paspor)} />;
            if (key === "siskopatuh" || key === "visa") {
              const nilai = Boolean(pax[`progress_${key}`]);
              return (
                <CentangTahap
                  checked={nilai}
                  onChange={() => onTogglePax(pax.id, key, nilai)}
                  label={key === "visa" ? "Visa" : "SISKOPATUH"}
                />
              );
            }
            return null;
          }}
        />
        <div className="flex items-center justify-between gap-3 px-4 py-3 border-t border-neutral-200">
          <span className="text-sm font-medium font-body text-neutral-800">Tiket PP</span>
          <Toggle
            id="toggle-tiket-detail"
            checked={Boolean(booking.progress_tiket)}
            onChange={() => onToggleTiket(Boolean(booking.progress_tiket))}
          />
        </div>
      </>
    )}
  </TableCard>
);

export default TahapanUmrohCard;

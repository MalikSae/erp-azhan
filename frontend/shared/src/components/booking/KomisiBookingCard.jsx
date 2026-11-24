import { Link } from "react-router-dom";
import { TableCard } from "../ui/DetailLayout";
import Badge from "../ui/Badge";
import EmptyState from "../ui/EmptyState";
import Table from "../ui/Table";
import { JENIS_KOMISI } from "../../utils/agen";
import AgenNama, { LINK_TEKS } from "./AgenNama";

// Status rincian komisi booking (GET /bookings/{id}/agen-komisi).
const STATUS_KOMISI_BADGE = {
  estimasi: { label: "Estimasi", variant: "warning" },
  tercatat: { label: "Tercatat", variant: "success" },
  batal: { label: "Batal", variant: "danger" },
};

const JENIS_KOMISI_SINGKAT = { langsung: "Langsung", pembinaan: "Pembinaan" };

const formatRupiah = (angka) =>
  new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", minimumFractionDigits: 0 }).format(angka || 0);

// Kartu Komisi di Detail Booking: rincian per penerima, Batal dicoret, total hanya yang aktif.
// Nominal dan tautan Riwayat Komisi hanya untuk izin komisi.view (agenKomisi.lihat_nominal).
const KomisiBookingCard = ({ agenKomisi, kodeBooking, bolehLihatAgen }) => {
  const rincian = agenKomisi?.komisi || [];
  const lihatNominal = Boolean(agenKomisi?.lihat_nominal);
  return (
    <TableCard
      title="Komisi"
      aside={lihatNominal && kodeBooking ? (
        <Link to={`/agen/komisi?cari=${encodeURIComponent(kodeBooking)}`} className={`text-xs font-body ${LINK_TEKS}`}>
          Riwayat komisi
        </Link>
      ) : null}
    >
      {rincian.length === 0 ? (
        <EmptyState variant="section" message="Belum ada komisi." />
      ) : (
        <>
          <Table
            columns={[
              { header: "Penerima", key: "penerima" },
              { header: "Jamaah", key: "jamaah" },
              { header: "Jenis", key: "jenis" },
              { header: "Status", key: "status" },
              ...(lihatNominal ? [{ header: "Nominal", key: "nominal", align: "right" }] : []),
            ]}
            data={rincian}
            renderCell={(k, key) => {
              const batal = k.status === "batal";
              const coret = batal ? "line-through text-neutral-400" : "";
              if (key === "penerima") {
                return (
                  <span className={`text-sm font-body whitespace-nowrap ${coret}`}>
                    {batal ? k.penerima.nama_lengkap : <AgenNama agen={k.penerima} bolehLihat={bolehLihatAgen} />}
                  </span>
                );
              }
              if (key === "jamaah") {
                return <span className={`text-sm font-body text-neutral-800 whitespace-nowrap ${coret}`}>{k.nama_jamaah || "-"}</span>;
              }
              if (key === "jenis") {
                return (
                  <span className={`text-sm font-body text-neutral-600 whitespace-nowrap ${coret}`}>
                    {JENIS_KOMISI_SINGKAT[k.jenis] || JENIS_KOMISI[k.jenis] || k.jenis}
                  </span>
                );
              }
              if (key === "status") {
                const s = STATUS_KOMISI_BADGE[k.status];
                return <Badge variant={s?.variant || "neutral"}>{s?.label || k.status}</Badge>;
              }
              if (key === "nominal") {
                return (
                  <span className={`text-sm font-body font-medium text-neutral-900 tabular-nums whitespace-nowrap ${coret}`}>
                    {k.nominal == null ? "-" : formatRupiah(k.nominal)}
                  </span>
                );
              }
              return k[key];
            }}
          />
          {lihatNominal && agenKomisi.total != null && (
            <div className="px-4 py-3 border-t border-neutral-200 flex justify-between text-sm font-body">
              <span className="text-neutral-500">Total</span>
              <span className="font-semibold text-neutral-900 tabular-nums">{formatRupiah(agenKomisi.total)}</span>
            </div>
          )}
        </>
      )}
    </TableCard>
  );
};

export default KomisiBookingCard;

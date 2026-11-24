import { Link } from "react-router-dom";

export const LINK_TEKS = "font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900";

// Nama agen (tautan ke Detail Agen bila boleh) dengan kode referral.
const AgenNama = ({ agen, bolehLihat, denganKode = false, className = "" }) => (
  <span className={className}>
    {bolehLihat ? (
      <Link to={`/agen/${agen.id}`} className={LINK_TEKS}>{agen.nama_lengkap}</Link>
    ) : (
      <span className="font-medium text-neutral-900">{agen.nama_lengkap}</span>
    )}
    {denganKode && agen.kode_referral && (
      <span className="font-mono text-xs text-neutral-500"> ({agen.kode_referral})</span>
    )}
  </span>
);

export default AgenNama;

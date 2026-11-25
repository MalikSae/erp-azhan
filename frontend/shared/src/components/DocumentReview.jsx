import { useEffect, useRef, useState } from 'react';
import Alert from './ui/Alert';
import Badge from './ui/Badge';
import Button from './ui/Button';
import Modal from './ui/Modal';
import Textarea from './ui/Textarea';
import LoadingSpinner from './ui/LoadingSpinner';
import { getRiwayatDokumen, updateDokumenStatus } from '../api/dokumen';
import { loadProtectedMedia, protectedMediaErrorMessage } from '../api/media';

// Label status dokumen jamaah, sama di admin, web, dan app (nilai API: submitted | approved | rejected).
export const STATUS_DOKUMEN = {
  submitted: { label: 'Menunggu verifikasi', tone: 'warning' },
  approved: { label: 'Disetujui', tone: 'success' },
  rejected: { label: 'Ditolak', tone: 'danger' },
};

export const StatusDokumen = ({ status }) => {
  const s = STATUS_DOKUMEN[status];
  return s ? <Badge variant={s.tone}>{s.label}</Badge> : <Badge variant="neutral">Belum diunggah</Badge>;
};

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

// "24 Nov 10.12"; tahun ditambahkan bila bukan tahun ini.
export function waktuSingkat(v) {
  const d = new Date(v);
  if (!v || Number.isNaN(d.getTime())) return '';
  const jam = `${String(d.getHours()).padStart(2, '0')}.${String(d.getMinutes()).padStart(2, '0')}`;
  const tahun = d.getFullYear() === new Date().getFullYear() ? '' : ` ${d.getFullYear()}`;
  return `${d.getDate()} ${BULAN[d.getMonth()]}${tahun} ${jam}`;
}

// "Disetujui oleh X, 24 Nov 10.12". Status submitted dengan peninjau berarti persetujuannya dibatalkan.
export function jejakPeninjau(doc) {
  if (!doc?.verified_at) return '';
  const aksi = { approved: 'Disetujui', rejected: 'Ditolak', submitted: 'Persetujuan dibatalkan' }[doc.status];
  if (!aksi) return '';
  const oleh = doc.verified_by_nama ? ` oleh ${doc.verified_by_nama}` : '';
  return `${aksi}${oleh}, ${waktuSingkat(doc.verified_at)}`;
}

const linkClass = 'text-xs font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900';

function Pratinjau({ preview }) {
  const { objectUrl, type, name } = preview;
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-center overflow-auto rounded-xl border border-neutral-200 bg-neutral-50 p-3">
        {type.startsWith('image/') ? <img src={objectUrl} alt="Berkas dokumen" className="max-h-96 max-w-full rounded object-contain" />
          : type === 'application/pdf' ? <iframe src={objectUrl} title="Berkas dokumen" className="h-80 w-full rounded border-0 bg-white md:h-96" />
          : <p className="text-sm text-neutral-500">Pratinjau tidak tersedia.</p>}
      </div>
      <a href={objectUrl} download={name} className={linkClass}>Unduh berkas</a>
    </div>
  );
}

// Dialog satu dokumen: pratinjau berkas, peninjau terakhir, berkas sebelumnya, lalu Setujui/Tolak.
// document null = tertutup. bolehUbah = izin dokumen.edit.
export default function DocumentReview({ document, label, bolehUbah = false, onClose, onChanged }) {
  const [preview, setPreview] = useState(null);
  const [versiDilihat, setVersiDilihat] = useState(null);
  const [riwayat, setRiwayat] = useState([]);
  const [menolak, setMenolak] = useState(false);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const docId = document?.id;
  const docVersion = document?.version;

  // Nomor permintaan: hasil muat yang datang setelah dokumen/berkas lain dibuka dibuang.
  const reqRef = useRef(0);

  const muat = async (url, versi) => {
    const req = ++reqRef.current;
    setError('');
    setPreview(prev => { if (prev) URL.revokeObjectURL(prev.objectUrl); return null; });
    try {
      const next = await loadProtectedMedia(url);
      if (req !== reqRef.current) { URL.revokeObjectURL(next.objectUrl); return; }
      setPreview(next);
      setVersiDilihat(versi);
    } catch (e) {
      if (req === reqRef.current) setError(protectedMediaErrorMessage(e));
    }
  };

  useEffect(() => {
    if (!document) return undefined;
    let aktif = true;
    setMenolak(false); setReason(''); setError(''); setRiwayat([]); setVersiDilihat(null);
    muat(document.file_url, null);
    getRiwayatDokumen(document.id).then(items => { if (aktif) setRiwayat(items || []); }).catch(() => { if (aktif) setRiwayat([]); });
    return () => {
      aktif = false;
      reqRef.current += 1;
      setPreview(prev => { if (prev) URL.revokeObjectURL(prev.objectUrl); return null; });
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [docId, docVersion]);

  const tutup = () => { if (!busy) onClose(); };

  async function save(status) {
    if (status === 'rejected' && !reason.trim()) { setError('Isi alasan penolakan.'); return; }
    setBusy(true); setError('');
    try {
      const result = await updateDokumenStatus(document.id, status, document.version, status === 'rejected' ? reason.trim() : '');
      onChanged(result);
      if (status !== 'submitted') onClose();
    } catch (e) {
      setError(e.response?.data?.error || 'Verifikasi gagal disimpan.');
    } finally {
      setBusy(false);
    }
  }

  if (!document) return null;
  const melihatLama = versiDilihat !== null;
  const bisaTinjau = bolehUbah && document.status === 'submitted' && !melihatLama;
  const jejak = jejakPeninjau(document);

  // Satu aksi utama (Setujui); Tolak sekunder membuka kolom alasan.
  let footer = null;
  if (bisaTinjau) {
    footer = menolak ? (
      <>
        <Button variant="secondary" onClick={() => { setMenolak(false); setError(''); }} disabled={busy}>Batal</Button>
        <Button variant="danger" onClick={() => save('rejected')} isLoading={busy}>Tolak</Button>
      </>
    ) : (
      <>
        <Button variant="secondary" onClick={() => { setMenolak(true); setError(''); }} disabled={busy || !preview}>Tolak</Button>
        <Button onClick={() => save('approved')} isLoading={busy} disabled={!preview}>Setujui</Button>
      </>
    );
  }

  return (
    <Modal isOpen onClose={tutup} title={label || 'Dokumen'} size="2xl" footer={footer}>
      <div className="space-y-4">
        {error && <Alert variant="error">{error}</Alert>}

        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <StatusDokumen status={document.status} />
            {jejak && <span className="text-xs text-neutral-500">{jejak}</span>}
          </div>
          {document.status === 'rejected' && document.rejection_reason && (
            <p className="text-sm text-neutral-700 break-words">Alasan: {document.rejection_reason}</p>
          )}
          {bolehUbah && document.status === 'approved' && (
            <button type="button" className={linkClass} disabled={busy} onClick={() => save('submitted')}>Batalkan persetujuan</button>
          )}
        </div>

        {melihatLama && (
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2">
            <span className="text-xs text-neutral-700">Menampilkan berkas sebelumnya</span>
            <button type="button" className={linkClass} onClick={() => muat(document.file_url, null)}>Lihat berkas terbaru</button>
          </div>
        )}

        {preview ? <Pratinjau preview={preview} /> : !error && <LoadingSpinner />}

        {menolak && bisaTinjau && (
          <Textarea id={'doc-reason-' + document.id} label="Alasan penolakan" value={reason} maxLength={500} rows={2} onChange={e => setReason(e.target.value)} />
        )}

        {riwayat.length > 0 && (
          <div className="space-y-2 border-t border-neutral-200 pt-4">
            <p className="text-sm font-semibold font-heading text-neutral-900">Berkas sebelumnya</p>
            <ul className="divide-y divide-neutral-100">
              {riwayat.map(v => (
                <li key={v.version} className="flex items-start justify-between gap-3 py-2">
                  <div className="min-w-0 space-y-0.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <StatusDokumen status={v.status} />
                      <span className="text-xs text-neutral-500">{jejakPeninjau(v) || `Diganti ${waktuSingkat(v.recorded_at)}`}</span>
                    </div>
                    {v.status === 'rejected' && v.rejection_reason && <p className="text-xs text-neutral-600 break-words">Alasan: {v.rejection_reason}</p>}
                  </div>
                  {v.file_url && versiDilihat !== v.version && (
                    <button type="button" className={`${linkClass} shrink-0`} onClick={() => muat(v.file_url, v.version)}>Lihat</button>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </Modal>
  );
}

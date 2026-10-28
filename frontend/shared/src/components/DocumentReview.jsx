import { useState } from 'react';
import Button from './ui/Button';
import Modal from './ui/Modal';
import Textarea from './ui/Textarea';
import { updateDokumenStatus } from '../api/dokumen';
import { loadProtectedMedia } from '../api/media';

// Satu dialog verifikasi: berkas ditampilkan di dalam dialog yang sama (bukan lapisan pratinjau terpisah),
// lalu admin menyetujui atau menolak dengan alasan.
function Pratinjau({ preview }) {
  if (!preview) return null;
  const { objectUrl, type, name } = preview;
  return (
    <div className="space-y-2">
      <div className="flex max-h-[60vh] items-center justify-center overflow-auto rounded-lg bg-neutral-50 p-3">
        {type.startsWith('image/') ? <img src={objectUrl} alt="Berkas dokumen" className="max-h-[56vh] max-w-full rounded object-contain" />
          : type === 'application/pdf' ? <iframe src={objectUrl} title="Berkas dokumen" className="h-[56vh] w-full rounded border-0 bg-white" />
          : <p className="text-sm text-neutral-500">Pratinjau tidak tersedia untuk jenis berkas ini. Gunakan tombol Download.</p>}
      </div>
      <a href={objectUrl} download={name} className="inline-block text-sm font-medium text-primary-600 hover:underline">Download berkas</a>
    </div>
  );
}

export default function DocumentReview({ document, onChanged }) {
  const [reviewed, setReviewed] = useState(null);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const tutup = () => {
    if (reviewed?.preview) URL.revokeObjectURL(reviewed.preview.objectUrl);
    setReviewed(null);
  };
  async function review() {
    setError('');
    try { const preview = await loadProtectedMedia(document.file_url); setReviewed({ ...document, preview }); setReason(''); }
    catch { setError('Berkas belum dapat dibuka. Coba lagi.'); }
  }
  async function save(status) {
    if (status === 'rejected' && !reason.trim()) { setError('Isi alasan penolakan.'); return; }
    setBusy(true); setError('');
    try { const result = await updateDokumenStatus(reviewed.id, status, reviewed.version, reason.trim()); onChanged(result); tutup(); }
    catch (e) { setError(e.response?.data?.error || 'Verifikasi gagal disimpan.'); }
    finally { setBusy(false); }
  }
  return <div className="space-y-2"><Button size="sm" variant="secondary" onClick={review}>Tinjau v{document.version}</Button>
    {error && !reviewed && <p role="alert" className="text-xs text-danger-600">{error}</p>}
    <Modal isOpen={!!reviewed} onClose={() => { if (!busy) tutup(); }} title="Verifikasi dokumen" size="2xl">
      <div className="space-y-4"><p className="text-sm">Periksa berkas versi {reviewed?.version} di bawah sebelum menyetujui. Perubahan berkas oleh jamaah akan membatalkan persetujuan versi lama.</p>
        <Pratinjau preview={reviewed?.preview} />
        <Textarea id={'doc-reason-' + document.id} label="Alasan jika ditolak" value={reason} maxLength={500} onChange={e => setReason(e.target.value)} />
        {error && <p role="alert" className="text-sm text-danger-600">{error}</p>}
        <div className="flex gap-2"><Button disabled={busy} onClick={() => save('approved')}>Setujui versi ini</Button><Button variant="secondary" disabled={busy} onClick={() => save('rejected')}>Tolak dengan alasan</Button></div>
      </div>
    </Modal>
  </div>;
}

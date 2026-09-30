import { useState } from 'react';
import Button from './ui/Button';
import Modal from './ui/Modal';
import Textarea from './ui/Textarea';
import { updateDokumenStatus } from '../api/dokumen';
import { openProtectedMedia } from '../api/media';

export default function DocumentReview({ document, onChanged }) {
  const [reviewed, setReviewed] = useState(null);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function review() {
    setError('');
    try { await openProtectedMedia(document.file_url); setReviewed({ ...document }); setReason(''); }
    catch { setError('Berkas belum dapat dibuka. Coba lagi.'); }
  }
  async function save(status) {
    if (status === 'rejected' && !reason.trim()) { setError('Isi alasan penolakan.'); return; }
    setBusy(true); setError('');
    try { const result = await updateDokumenStatus(reviewed.id, status, reviewed.version, reason.trim()); onChanged(result); setReviewed(null); }
    catch (e) { setError(e.response?.data?.error || 'Verifikasi gagal disimpan.'); }
    finally { setBusy(false); }
  }
  return <div className="space-y-2"><Button size="sm" variant="secondary" onClick={review}>Tinjau v{document.version}</Button>
    {error && <p role="alert" className="text-xs text-danger-600">{error}</p>}
    <Modal isOpen={!!reviewed} onClose={() => { if (!busy) setReviewed(null); }} title="Verifikasi dokumen">
      <div className="space-y-4"><p className="text-sm">Periksa berkas versi {reviewed?.version} yang dibuka sebelum menyetujui. Perubahan berkas oleh jamaah akan membatalkan persetujuan versi lama.</p>
        <Textarea id={'doc-reason-' + document.id} label="Alasan jika ditolak" value={reason} maxLength={500} onChange={e => setReason(e.target.value)} />
        {error && <p role="alert" className="text-sm text-danger-600">{error}</p>}
        <div className="flex gap-2"><Button disabled={busy} onClick={() => save('approved')}>Setujui versi ini</Button><Button variant="secondary" disabled={busy} onClick={() => save('rejected')}>Tolak dengan alasan</Button></div>
      </div>
    </Modal>
  </div>;
}

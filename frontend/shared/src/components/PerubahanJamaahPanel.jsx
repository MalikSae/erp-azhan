import { useEffect, useState } from 'react';
import { Eye } from 'lucide-react';
import { listPerubahanJamaah, getPerubahanJamaah, setujuiPerubahanJamaah, tolakPerubahanJamaah } from '../api/perubahanJamaah';
import Alert from './ui/Alert';
import ActionMenu from './ui/ActionMenu';
import Button from './ui/Button';
import DataTable from './ui/DataTable';
import Modal from './ui/Modal';

const waktu = (value) => !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(value));

const STATUS = {
  menunggu: { label: 'Menunggu persetujuan', dot: 'bg-warning-500' },
  disetujui: { label: 'Disetujui', dot: 'bg-success-500' },
  ditolak: { label: 'Ditolak', dot: 'bg-danger-500' },
  diganti: { label: 'Digantikan pengajuan baru', dot: 'bg-neutral-400' },
  langsung: { label: 'Langsung tersimpan', dot: 'bg-info-500' },
};

const StatusDot = ({ status }) => {
  const meta = STATUS[status] || { label: status, dot: 'bg-neutral-400' };
  return (
    <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-neutral-700">
      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${meta.dot}`} />
      {meta.label}
    </span>
  );
};

const pelaku = (row) => (row.sumber === 'admin' ? `Admin ${row.pelaku || ''}`.trim() : row.akun_nama || 'Pelanggan');

// Nama event yang dikirim setelah pengajuan diputuskan, supaya penghitung "menunggu" di tempat lain
// (sidebar, tab) ikut diperbarui tanpa menunggu pindah halaman.
export const EVENT_PERUBAHAN_JAMAAH = 'perubahan-jamaah-diputuskan';

// Daftar perubahan data jamaah beserta modal tinjau, setujui, dan tolak. Butuh izin jamaah.perubahan.approve.
// Pelanggan mengisi isian kosong langsung, tetapi mengubah isian yang sudah terisi menunggu keputusan di sini.
// Menyetujui menerapkan seluruh perubahan ke semua baris jamaah orang itu; menolak wajib memakai alasan yang
// dibaca pelanggan.
//
// Dua pemakaian:
// - tab "Menunggu persetujuan" di Kelola Jamaah: semua jamaah, hanya yang menunggu;
// - detail jamaah (jamaahId diisi): seluruh riwayat satu jamaah, tanpa kolom nama dan tanpa pencarian.
export default function PerubahanJamaahPanel({ jamaahId, toolbarLeading, onChanged }) {
  const perJamaah = Boolean(jamaahId);
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [menolak, setMenolak] = useState(false);
  const [catatan, setCatatan] = useState('');
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      setItems(await listPerubahanJamaah({ status: perJamaah ? 'semua' : 'menunggu', jamaahId }));
    } catch (err) {
      setError(err.response?.status === 403
        ? 'Anda tidak memiliki izin untuk melihat perubahan data jamaah.'
        : 'Daftar perubahan gagal dimuat. Silakan coba kembali.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [jamaahId]); // eslint-disable-line react-hooks/exhaustive-deps

  const open = async (row) => {
    setActionError('');
    setMenolak(false);
    setCatatan('');
    try {
      setSelected(await getPerubahanJamaah(row.id));
    } catch {
      setError('Detail perubahan gagal dimuat.');
    }
  };

  const close = () => {
    if (!saving) setSelected(null);
  };

  const putuskan = async (aksi) => {
    if (aksi === 'tolak' && catatan.trim().length < 3) {
      setActionError('Alasan penolakan wajib diisi (minimal 3 karakter).');
      return;
    }
    setSaving(true);
    setActionError('');
    try {
      if (aksi === 'setujui') await setujuiPerubahanJamaah(selected.id);
      else await tolakPerubahanJamaah(selected.id, catatan.trim());
      setSelected(null);
      await load();
      onChanged?.(aksi);
      window.dispatchEvent(new Event(EVENT_PERUBAHAN_JAMAAH));
    } catch (err) {
      setActionError(err.response?.data?.error || 'Keputusan gagal disimpan.');
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    !perJamaah && { header: 'Jamaah', key: 'jamaah_nama', sortable: true },
    { header: perJamaah ? 'Diubah oleh' : 'Diajukan oleh', key: 'akun_nama' },
    { header: 'Isian', key: 'jumlah_item' },
    { header: 'Waktu', key: 'dibuat_at', sortable: true },
    { header: 'Status', key: 'status' },
    { header: 'Aksi', key: 'actions' },
  ].filter(Boolean);

  const renderCell = (row, key) => {
    if (key === 'jamaah_nama') {
      return (
        <button type="button" onClick={(e) => { e.stopPropagation(); open(row); }} className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors cursor-pointer">
          {row.jamaah_nama}
        </button>
      );
    }
    if (key === 'akun_nama') return pelaku(row);
    if (key === 'jumlah_item') return `${row.jumlah_item} isian`;
    if (key === 'dibuat_at') return waktu(row.dibuat_at);
    if (key === 'status') return <StatusDot status={row.status} />;
    if (key === 'actions') return <ActionMenu items={[{ label: 'Tinjau', icon: Eye, tooltip: 'Tinjau perubahan', onClick: () => open(row) }]} align="right" />;
    return row[key] ?? '-';
  };

  const menunggu = selected?.status === 'menunggu';
  const pesanKosong = loading
    ? 'Memuat perubahan...'
    : perJamaah
      ? 'Belum ada perubahan data yang tercatat untuk jamaah ini'
      : 'Tidak ada perubahan yang menunggu persetujuan';

  return (
    <div className="space-y-4">
      {error && <Alert variant="error" message={error} />}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        itemsPerPage={perJamaah ? 5 : 10}
        onRowClick={open}
        searchable={!perJamaah}
        searchPlaceholder="Cari nama jamaah atau pelanggan..."
        emptyMessage={pesanKosong}
        toolbarLeading={toolbarLeading}
      />

      <Modal
        isOpen={Boolean(selected)}
        onClose={close}
        title="Tinjau Perubahan Data Jamaah"
        size="2xl"
        footer={
          <div className="flex flex-col-reverse sm:flex-row sm:items-center sm:justify-between gap-2 w-full">
            <Button variant="secondary" onClick={close} disabled={saving}>Tutup</Button>
            {menunggu && (
              <div className="flex flex-col-reverse sm:flex-row gap-2">
                {menolak ? (
                  <Button variant="danger" onClick={() => putuskan('tolak')} isLoading={saving}>Simpan penolakan</Button>
                ) : (
                  <Button variant="secondary" onClick={() => { setMenolak(true); setActionError(''); }} disabled={saving}>Tolak</Button>
                )}
                <Button onClick={() => putuskan('setujui')} isLoading={saving} disabled={menolak}>Setujui dan terapkan</Button>
              </div>
            )}
          </div>
        }
      >
        {selected && (
          <div className="space-y-4">
            {actionError && <Alert variant="error" message={actionError} />}

            <div className="p-3.5 rounded-xl border border-neutral-200/70 space-y-2 text-xs">
              {[
                ['Jamaah', selected.jamaah_nama],
                ['Diajukan oleh', pelaku(selected)],
                ['Waktu', waktu(selected.dibuat_at)],
              ].map(([label, nilai]) => (
                <div key={label} className="flex justify-between gap-2 border-b border-neutral-100 pb-1.5 last:border-0 last:pb-0">
                  <span className="text-neutral-500">{label}</span>
                  <span className="font-semibold text-neutral-900 text-right">{nilai}</span>
                </div>
              ))}
              <div className="flex justify-between gap-2">
                <span className="text-neutral-500">Status</span>
                <StatusDot status={selected.status} />
              </div>
              {selected.diputus_at && (
                <div className="flex justify-between gap-2 border-t border-neutral-100 pt-1.5">
                  <span className="text-neutral-500">Diputuskan</span>
                  <span className="text-neutral-800 text-right">{waktu(selected.diputus_at)}{selected.diputus_oleh ? ` oleh ${selected.diputus_oleh}` : ''}</span>
                </div>
              )}
              {selected.catatan && (
                <div className="flex justify-between gap-2 border-t border-neutral-100 pt-1.5">
                  <span className="text-neutral-500">Catatan</span>
                  <span className="text-neutral-800 text-right">{selected.catatan}</span>
                </div>
              )}
            </div>

            <div className="rounded-xl border border-neutral-200/70 overflow-hidden">
              <table className="w-full text-xs">
                <thead className="bg-neutral-50 text-neutral-500">
                  <tr><th className="text-left font-semibold px-3 py-2">Isian</th><th className="text-left font-semibold px-3 py-2">Sebelumnya</th><th className="text-left font-semibold px-3 py-2">Menjadi</th></tr>
                </thead>
                <tbody>
                  {selected.perubahan.map((p) => (
                    <tr key={p.kolom} className="border-t border-neutral-100">
                      <td className="px-3 py-2 font-semibold text-neutral-800">{p.label}</td>
                      <td className="px-3 py-2 text-neutral-500 break-words">{p.lama || '(kosong)'}</td>
                      <td className="px-3 py-2 text-neutral-900 font-semibold break-words">{p.baru || '(dikosongkan)'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {!menunggu && selected.perubahan.some((p) => ['no_paspor', 'no_asuransi_bpjs', 'nik'].includes(p.kolom)) && (
              <p className="text-[11px] text-neutral-400">Nomor paspor, NIK, dan nomor asuransi disamarkan pada riwayat yang sudah diputuskan.</p>
            )}

            {menunggu && menolak && (
              <div className="space-y-1.5">
                <label htmlFor="catatan-tolak" className="block text-xs font-bold text-neutral-700">Alasan penolakan <span className="text-danger-600">*</span></label>
                <textarea
                  id="catatan-tolak"
                  value={catatan}
                  onChange={(e) => setCatatan(e.target.value)}
                  rows={3}
                  placeholder="Alasan yang dibaca pelanggan, mis. foto paspor belum cocok"
                  className="w-full rounded-lg border border-neutral-300 bg-white p-2.5 text-xs text-neutral-800 placeholder-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
                  autoFocus
                />
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

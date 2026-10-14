import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Network, Users, Layers, Plus, Pencil, Power, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import StatTile from 'shared/ui/StatTile';
import ActionMenu from 'shared/ui/ActionMenu';
import { listOrgUnits, createOrgUnit, updateOrgUnit, setOrgUnitStatus, deleteOrgUnit } from '../api/orgUnits';
import { errMsg, StatusChip } from '../components/penggunaUi';

// Struktur Organisasi (SPEK-PENGGUNA-ORGANISASI §6): kelola divisi & tim.
// Tidak ada urusan akun di sini; anggota diatur dari form Pengguna.
const StrukturOrganisasiPage = () => {
  const navigate = useNavigate();
  const [tree, setTree] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);
  const [showInactive, setShowInactive] = useState(false);

  // editor: { mode: 'create'|'edit', jenis, unit?, parentId? }
  const [editor, setEditor] = useState(null);
  const [form, setForm] = useState({ nama: '', urutan: 0, parent_id: '' });
  const [editorError, setEditorError] = useState(null);
  const [isSaving, setIsSaving] = useState(false);

  // confirm: { kind: 'status'|'delete', unit }
  const [confirm, setConfirm] = useState(null);
  const [confirmError, setConfirmError] = useState(null);
  const [isConfirming, setIsConfirming] = useState(false);

  const fetchData = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      setTree((await listOrgUnits()) || []);
    } catch (error) {
      setErrorMessage(errMsg(error, 'Gagal memuat struktur organisasi.'));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const stats = useMemo(() => {
    const divisi = tree.filter(d => d.is_active);
    const tim = tree.flatMap(d => d.tim || []).filter(t => t.is_active);
    const anggota = tree.reduce((s, d) => s + d.jumlah_anggota + (d.tim || []).reduce((a, t) => a + t.jumlah_anggota, 0), 0);
    return { divisi: divisi.length, tim: tim.length, penugasan: anggota };
  }, [tree]);

  const visibleTree = useMemo(() => tree
    .filter(d => showInactive || d.is_active)
    .map(d => ({ ...d, tim: (d.tim || []).filter(t => showInactive || t.is_active) })), [tree, showInactive]);

  const activeDivisiOptions = (currentParentId) => tree
    .filter(d => d.is_active || d.id === currentParentId)
    .map(d => ({ value: String(d.id), label: `${d.nama}${d.is_active ? '' : ' (nonaktif)'}` }));

  // ─── Editor (tambah/ubah) ────────────────────────────────────────────────
  const openCreate = (jenis, parentId = null) => {
    setForm({ nama: '', urutan: 0, parent_id: parentId ? String(parentId) : '' });
    setEditorError(null);
    setEditor({ mode: 'create', jenis });
  };
  const openEdit = (unit) => {
    setForm({ nama: unit.nama, urutan: unit.urutan, parent_id: unit.parent_id ? String(unit.parent_id) : '' });
    setEditorError(null);
    setEditor({ mode: 'edit', jenis: unit.jenis, unit });
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setEditorError(null);
    if (editor.jenis === 'tim' && !form.parent_id) { setEditorError('Pilih divisi induk.'); return; }
    const payload = {
      nama: form.nama.trim(),
      urutan: Number(form.urutan) || 0,
      parent_id: editor.jenis === 'tim' ? Number(form.parent_id) : null,
    };
    setIsSaving(true);
    try {
      if (editor.mode === 'create') {
        await createOrgUnit({ ...payload, jenis: editor.jenis });
        setSuccessMessage(`${editor.jenis === 'divisi' ? 'Divisi' : 'Tim'} "${payload.nama}" ditambahkan.`);
      } else {
        await updateOrgUnit(editor.unit.id, payload);
        setSuccessMessage(`"${payload.nama}" diperbarui.`);
      }
      setEditor(null);
      fetchData();
    } catch (error) {
      setEditorError(errMsg(error, 'Gagal menyimpan unit.'));
    } finally {
      setIsSaving(false);
    }
  };

  // ─── Status & hapus ──────────────────────────────────────────────────────
  const handleConfirm = async () => {
    setIsConfirming(true);
    setConfirmError(null);
    const { kind, unit } = confirm;
    try {
      if (kind === 'status') {
        await setOrgUnitStatus(unit.id, !unit.is_active);
        setSuccessMessage(`"${unit.nama}" ${unit.is_active ? 'dinonaktifkan' : 'diaktifkan kembali'}.`);
      } else {
        await deleteOrgUnit(unit.id);
        setSuccessMessage(`"${unit.nama}" dihapus.`);
      }
      setConfirm(null);
      fetchData();
    } catch (error) {
      setConfirmError(errMsg(error, 'Aksi gagal.'));
    } finally {
      setIsConfirming(false);
    }
  };

  const PemimpinList = ({ unit }) => (unit.pemimpin || []).length === 0
    ? <span className="text-xs text-neutral-400 font-body">—</span>
    : (
      <div className="flex flex-wrap gap-x-2 gap-y-1">
        {unit.pemimpin.map(p => (
          <button key={p.id} type="button" onClick={() => navigate(`/users/${p.id}`)} className="text-[13px] font-body text-neutral-800 hover:text-primary-600">
            {p.display_name}
          </button>
        ))}
      </div>
    );

  const UnitActions = ({ unit }) => (
    <div className="flex items-center justify-end">
      <ActionMenu
        align="right"
        items={[
          { label: 'Ubah', icon: Pencil, tooltip: 'Ubah unit', onClick: () => openEdit(unit) },
          {
            label: unit.is_active ? 'Nonaktifkan' : 'Aktifkan',
            icon: Power,
            tooltip: unit.is_active ? 'Nonaktifkan' : 'Aktifkan kembali',
            onClick: () => { setConfirm({ kind: 'status', unit }); setConfirmError(null); },
          },
          {
            label: 'Hapus',
            icon: Trash2,
            danger: true,
            tooltip: 'Hapus (hanya unit kosong)',
            onClick: () => { setConfirm({ kind: 'delete', unit }); setConfirmError(null); },
          },
        ]}
      />
    </div>
  );

  return (
    <div className="space-y-6">
      <PageHeader title="Struktur Organisasi" actionLabel="+ Tambah Divisi" onAction={() => openCreate('divisi')} />

      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}

      {!isLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <StatTile icon={Network} variant="primary" value={stats.divisi} label="Divisi Aktif" />
          <StatTile icon={Layers} variant="info" value={stats.tim} label="Tim Aktif" />
          <StatTile icon={Users} variant="success" value={stats.penugasan} label="Penugasan (pengguna aktif)" />
        </div>
      )}

      <div className="flex items-center justify-between gap-3">
        <p className="text-xs font-body text-neutral-500">Anggota & pemimpin diatur dari form Pengguna (bagian Penugasan). Unit yang bisa dipilih mengikuti divisi asal role.</p>
        <label className="inline-flex items-center gap-2 text-xs font-body text-neutral-600 cursor-pointer select-none">
          <input type="checkbox" className="h-4 w-4 rounded border-neutral-300 accent-primary-500" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} />
          Tampilkan nonaktif
        </label>
      </div>

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : visibleTree.length === 0 ? (
        <div className="p-8 text-center text-sm font-body text-neutral-500 bg-white rounded-2xl border border-neutral-200 shadow-card">Belum ada divisi.</div>
      ) : (
        <div className="space-y-4">
          {visibleTree.map(d => (
            <section key={d.id} className={`bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden ${d.is_active ? '' : 'bg-neutral-50'}`}>
              <div className="px-5 py-4 border-b border-neutral-100 flex flex-wrap items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className={`text-[15px] font-heading font-bold ${d.is_active ? 'text-neutral-900' : 'text-neutral-500'}`}>{d.nama}</h3>
                    <span className="text-[11px] font-body text-neutral-400">Divisi</span>
                    {!d.is_active && <StatusChip active={false} />}
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs font-body text-neutral-500">
                    <span>{d.jumlah_anggota} anggota langsung</span>
                    <span className="flex items-center gap-1.5">Pemimpin: <PemimpinList unit={d} /></span>
                    <span>Role: {(d.role || []).length ? d.role.join(', ') : '—'}</span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button type="button" variant="secondary" size="sm" disabled={!d.is_active} onClick={() => openCreate('tim', d.id)}>
                    <Plus className="w-3.5 h-3.5" /> Tim
                  </Button>
                  <UnitActions unit={d} />
                </div>
              </div>

              {(d.tim || []).length === 0 ? (
                <p className="px-5 py-4 text-xs font-body text-neutral-400">Belum ada tim di divisi ini.</p>
              ) : (
                <table className="w-full text-left text-[13px] font-body text-neutral-700">
                  <thead className="bg-neutral-50 text-neutral-800 font-heading font-semibold border-b border-neutral-200">
                    <tr>
                      <th className="px-5 py-2.5">Tim</th>
                      <th className="px-4 py-2.5">Pemimpin</th>
                      <th className="px-4 py-2.5">Anggota</th>
                      <th className="px-4 py-2.5">Status</th>
                      <th className="px-5 py-2.5 text-right">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-100">
                    {d.tim.map(t => (
                      <tr key={t.id} className={t.is_active ? '' : 'bg-neutral-50/60 text-neutral-400'}>
                        <td className={`px-5 py-3 font-heading font-medium ${t.is_active ? 'text-neutral-900' : 'text-neutral-500'}`}>{t.nama}</td>
                        <td className="px-4 py-3"><PemimpinList unit={t} /></td>
                        <td className="px-4 py-3">{t.jumlah_anggota}</td>
                        <td className="px-4 py-3"><StatusChip active={t.is_active} /></td>
                        <td className="px-5 py-3"><UnitActions unit={t} /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          ))}
        </div>
      )}

      {/* Tambah/ubah unit */}
      <Modal
        isOpen={!!editor}
        onClose={() => { if (!isSaving) setEditor(null); }}
        title={editor ? `${editor.mode === 'create' ? 'Tambah' : 'Ubah'} ${editor.jenis === 'divisi' ? 'Divisi' : 'Tim'}` : ''}
      >
        <form onSubmit={handleSave} className="space-y-1">
          {editorError && <Alert variant="error" className="mb-4">{editorError}</Alert>}
          <Input label="Nama" required value={form.nama} autoFocus onChange={(e) => setForm(p => ({ ...p, nama: e.target.value }))} />
          {editor?.jenis === 'tim' && (
            <CustomDropdown
              label="Divisi induk"
              required
              value={form.parent_id}
              onChange={(val) => setForm(p => ({ ...p, parent_id: val }))}
              placeholder="Pilih divisi..."
              options={activeDivisiOptions(editor?.unit?.parent_id)}
            />
          )}
          <Input label="Urutan tampil" type="number" value={form.urutan}
            onChange={(e) => setForm(p => ({ ...p, urutan: e.target.value }))} />
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setEditor(null)} disabled={isSaving}>Batal</Button>
            <Button type="submit" variant="primary" disabled={isSaving}>{isSaving ? 'Menyimpan...' : 'Simpan'}</Button>
          </div>
        </form>
      </Modal>

      {/* Konfirmasi status / hapus */}
      <Modal
        isOpen={!!confirm}
        onClose={() => { if (!isConfirming) setConfirm(null); }}
        title={confirm?.kind === 'delete' ? 'Hapus Unit' : (confirm?.unit?.is_active ? 'Nonaktifkan Unit' : 'Aktifkan Kembali Unit')}
      >
        <div className="space-y-4">
          {confirmError && <Alert variant="error">{confirmError}</Alert>}
          <p className="text-sm font-body text-neutral-700">
            {confirm?.kind === 'delete'
              ? <>Hapus <strong className="font-heading text-neutral-900">{confirm?.unit?.nama}</strong>? Hanya bisa bila unit kosong (tanpa tim & tanpa anggota). Unit yang berisi, nonaktifkan saja.</>
              : confirm?.unit?.is_active
                ? <>Nonaktifkan <strong className="font-heading text-neutral-900">{confirm?.unit?.nama}</strong>? Keanggotaan lama tetap tercatat, tetapi unit ini tidak bisa dipilih untuk penugasan baru.</>
                : <>Aktifkan kembali <strong className="font-heading text-neutral-900">{confirm?.unit?.nama}</strong>?</>}
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setConfirm(null)} disabled={isConfirming}>Batal</Button>
            <Button type="button" variant={confirm?.kind === 'delete' || confirm?.unit?.is_active ? 'danger' : 'primary'} onClick={handleConfirm} disabled={isConfirming}>
              {isConfirming ? 'Memproses...' : confirm?.kind === 'delete' ? 'Hapus' : (confirm?.unit?.is_active ? 'Nonaktifkan' : 'Aktifkan')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default StrukturOrganisasiPage;

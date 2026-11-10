import { useContext, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { JamaahPage, PerubahanJamaahPanel, EVENT_PERUBAHAN_JAMAAH, listPerubahanJamaah } from 'shared';
import PageHeader from 'shared/ui/PageHeader';
import SegmentedTabs from 'shared/ui/SegmentedTabs';
import { AuthContext } from '../context/AuthContext';

// Kelola Jamaah: daftar jamaah dan antrean persetujuan perubahan data dalam satu halaman.
// tab "daftar" = /jamaah, tab "persetujuan" = /jamaah/perubahan (URL lama tetap berlaku).
// Tab persetujuan hanya tampil untuk pemegang izin jamaah.perubahan.approve.
export default function KelolaJamaahPage({ tab = 'daftar', showBrandColumn = false }) {
  const navigate = useNavigate();
  const { hasPermission } = useContext(AuthContext);
  const bolehSetujui = hasPermission('jamaah.perubahan.approve');
  const bolehLihatDaftar = hasPermission('jamaah.view');

  const [menunggu, setMenunggu] = useState(null);
  useEffect(() => {
    if (!bolehSetujui) return undefined;
    let batal = false;
    const hitung = () => listPerubahanJamaah().then((d) => { if (!batal) setMenunggu(d.length); }).catch(() => {});
    hitung();
    window.addEventListener(EVENT_PERUBAHAN_JAMAAH, hitung);
    return () => {
      batal = true;
      window.removeEventListener(EVENT_PERUBAHAN_JAMAAH, hitung);
    };
  }, [bolehSetujui]);

  const tabs = [
    bolehLihatDaftar && { id: 'daftar', label: 'Daftar jamaah' },
    bolehSetujui && { id: 'persetujuan', label: 'Menunggu persetujuan', count: menunggu ?? undefined },
  ].filter(Boolean);
  // Satu tab saja tidak perlu pemilih tab.
  const pemilihTab = tabs.length > 1
    ? <SegmentedTabs tabs={tabs} value={tab} onChange={(id) => navigate(id === 'persetujuan' ? '/jamaah/perubahan' : '/jamaah')} />
    : undefined;

  if (tab === 'persetujuan') {
    return (
      <div className="space-y-6">
        <PageHeader title="Kelola Jamaah" actionLabel="+ Tambah Jamaah" onAction={() => navigate('/jamaah/new')} />
        <PerubahanJamaahPanel toolbarLeading={pemilihTab} />
      </div>
    );
  }
  return <JamaahPage showBrandColumn={showBrandColumn} toolbarLeading={pemilihTab} />;
}

import { useContext, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { RiwayatKomisiPage, listPencairanAdmin, EVENT_PENCAIRAN_KOMISI } from 'shared';
import SegmentedTabs from 'shared/ui/SegmentedTabs';
import { AuthContext } from '../context/AuthContext';
import KomisiReferralPage from './KomisiReferralPage';

const ALAMAT_TAB = { pencairan: '/komisi', riwayat: '/agen/komisi' };

// Menu Komisi: antrean pencairan dan buku komisi dalam satu halaman bertab.
// tab "pencairan" = /komisi (izin agen.payout.approve), tab "riwayat" = /agen/komisi (izin komisi.view).
// Alamat lama tetap berlaku. Tab ditaruh di bawah judul karena kedua tab punya kartu ringkasan di atas tabel.
export default function KelolaKomisiPage({ tab = 'pencairan', showBrandColumn = false }) {
  const navigate = useNavigate();
  const { hasPermission } = useContext(AuthContext);
  const bolehCairkan = hasPermission('agen.payout.approve');
  const bolehRiwayat = hasPermission('komisi.view');

  const [menunggu, setMenunggu] = useState(null);
  useEffect(() => {
    if (!bolehCairkan) return undefined;
    let batal = false;
    const hitung = () => listPencairanAdmin({ status: 'pending' }).then((d) => { if (!batal) setMenunggu((d || []).length); }).catch(() => {});
    hitung();
    window.addEventListener(EVENT_PENCAIRAN_KOMISI, hitung);
    return () => {
      batal = true;
      window.removeEventListener(EVENT_PENCAIRAN_KOMISI, hitung);
    };
  }, [bolehCairkan]);

  const tabs = [
    bolehCairkan && { id: 'pencairan', label: 'Pencairan', count: menunggu ?? undefined },
    bolehRiwayat && { id: 'riwayat', label: 'Riwayat' },
  ].filter(Boolean);
  // Satu tab saja tidak perlu pemilih tab.
  const subnav = tabs.length > 1
    ? <SegmentedTabs tabs={tabs} value={tab} onChange={(id) => navigate(ALAMAT_TAB[id])} />
    : undefined;

  if (tab === 'riwayat') return <RiwayatKomisiPage showBrandColumn={showBrandColumn} judul="Komisi" subnav={subnav} />;
  return <KomisiReferralPage judul="Komisi" subnav={subnav} />;
}

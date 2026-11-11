import { useContext, useState } from 'react';
import { useParams } from 'react-router-dom';
import { DetailAgenPage } from 'shared';
import { AuthContext } from '../context/AuthContext';
import TinjauPengajuanAgen from '../components/TinjauPengajuanAgen';

// Halaman satu agen di dashboard (/agen/:id). Untuk akun yang ada di antrean pengajuan, kartu keputusan
// tampil paling atas; setelah keputusan tersimpan, data agen di bawahnya dimuat ulang (pengajuan yang
// disetujui langsung berubah jadi halaman agen penuh).
export default function HalamanAgenPage({ showBrandColumn = false }) {
  const { id } = useParams();
  const { hasPermission } = useContext(AuthContext);
  const bolehSetujui = hasPermission('agen.approve');
  // undefined = antrean belum dimuat, null = akun ini tidak di antrean.
  const [antre, setAntre] = useState(bolehSetujui ? undefined : null);
  const [muatUlang, setMuatUlang] = useState(0);

  return (
    <DetailAgenPage
      key={id}
      showBrandColumn={showBrandColumn}
      calon={antre}
      muatUlang={muatUlang}
      izin={{ lihatAkun: hasPermission('akun.view'), ubahNik: hasPermission('agen.kaitan.edit') }}
      atas={bolehSetujui ? (
        <TinjauPengajuanAgen
          akunId={id}
          onAda={(ada, row) => setAntre(ada ? row : null)}
          onBerubah={() => setMuatUlang((n) => n + 1)}
        />
      ) : null}
    />
  );
}

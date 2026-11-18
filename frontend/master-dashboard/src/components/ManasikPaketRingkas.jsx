import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import EmptyState from 'shared/ui/EmptyState';
import { getScheduleManasik } from '../api/schedules';
import { tanggalPendek } from './PilihPaketManasik';

const jam = (hhmm) => (hhmm || '').replace(':', '.');

// Ringkasan manasik paket di halaman detail paket. Manasik dikelola di menu Manasik; di sini hanya tautan ke detailnya.
const ManasikPaketRingkas = ({ scheduleId, bolehLihat }) => {
  const [manasik, setManasik] = useState(undefined);

  useEffect(() => {
    let batal = false;
    getScheduleManasik(scheduleId)
      .then((d) => { if (!batal) setManasik(d.manasik); })
      .catch(() => { if (!batal) setManasik(null); });
    return () => { batal = true; };
  }, [scheduleId]);

  if (manasik === undefined) return null;
  if (!manasik) {
    return (
      <p className="font-body text-sm text-neutral-500">
        Belum ada manasik.
      </p>
    );
  }
  const isi = `${manasik.judul} · ${tanggalPendek(manasik.tanggal)}, ${jam(manasik.jam_mulai)}–${jam(manasik.jam_selesai)} WIB`;
  return bolehLihat ? (
    <Link to={`/manasik/${manasik.id}`} className="font-body text-sm font-medium text-neutral-900 underline decoration-neutral-300 underline-offset-2 hover:decoration-neutral-900">
      {isi}
    </Link>
  ) : (
    <EmptyState variant="inline" message={isi} />
  );
};

export default ManasikPaketRingkas;

import React, { useEffect, useMemo, useRef, useState } from 'react';
import Modal from 'shared/ui/Modal';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import { LEBAR, potongGambar } from '../utils/gambarBanner';

const LEBAR_TAMPIL = 560;

// Pemotong gambar banner: bingkai 2:1 yang bisa digeser dan diperbesar di atas gambar asli. Lebar bingkai minimal 1200 px
// gambar asli supaya hasil 1200 × 600 tidak buram.
const PotongGambarModal = ({ img, file, onBatal, onSelesai }) => {
  const W = img.naturalWidth;
  const H = img.naturalHeight;
  // Bingkai terbesar yang muat: penuh tinggi bila gambar lebih lebar dari 2:1, penuh lebar bila lebih tinggi.
  const maksW = Math.min(W, H * 2);
  const skala = Math.min(1, LEBAR_TAMPIL / W);
  const [cropW, setCropW] = useState(maksW);
  const [pos, setPos] = useState({ x: (W - maksW) / 2, y: (H - maksW / 2) / 2 });
  const [memproses, setMemproses] = useState(false);
  const [galat, setGalat] = useState(null);
  const tarik = useRef(null);
  const src = useMemo(() => img.src, [img]);

  const batasi = (x, y, w) => ({ x: Math.min(Math.max(0, x), W - w), y: Math.min(Math.max(0, y), H - w / 2) });

  useEffect(() => {
    setPos((p) => batasi(p.x, p.y, cropW));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cropW]);

  const mulaiTarik = (e) => {
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    tarik.current = { sx: e.clientX, sy: e.clientY, x: pos.x, y: pos.y };
  };
  const geser = (e) => {
    if (!tarik.current) return;
    const t = tarik.current;
    setPos(batasi(t.x + (e.clientX - t.sx) / skala, t.y + (e.clientY - t.sy) / skala, cropW));
  };
  const selesaiTarik = () => { tarik.current = null; };

  // Perbesar: bingkai mengecil (dari terbesar sampai 1200 px), titik tengahnya tetap.
  const ubahZoom = (e) => {
    const w = Number(e.target.value);
    const cx = pos.x + cropW / 2;
    const cy = pos.y + cropW / 4;
    setCropW(w);
    setPos(batasi(cx - w / 2, cy - w / 4, w));
  };

  const pakai = async () => {
    setMemproses(true);
    setGalat(null);
    try {
      onSelesai(await potongGambar(img, file.name, { x: pos.x, y: pos.y, w: cropW }));
    } catch (err) {
      setGalat(err.message);
      setMemproses(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={() => !memproses && onBatal()}
      title="Potong Gambar"
      size="lg"
      footer={(
        <>
          <Button variant="secondary" onClick={onBatal} disabled={memproses}>Batal</Button>
          <Button onClick={pakai} isLoading={memproses}>Pakai</Button>
        </>
      )}
    >
      {galat && <div className="mb-3"><Alert variant="error">{galat}</Alert></div>}
      <div className="relative mx-auto select-none overflow-hidden rounded-xl bg-neutral-100" style={{ width: W * skala, height: H * skala }}>
        <img src={src} alt="" draggable={false} className="pointer-events-none block h-full w-full" />
        {/* Area di luar bingkai digelapkan dengan empat panel; posisinya dihitung dari bingkai. */}
        <div className="pointer-events-none absolute inset-x-0 top-0 bg-neutral-900/60" style={{ height: pos.y * skala }} />
        <div className="pointer-events-none absolute inset-x-0 bottom-0 bg-neutral-900/60" style={{ top: (pos.y + cropW / 2) * skala }} />
        <div className="pointer-events-none absolute left-0 bg-neutral-900/60" style={{ top: pos.y * skala, width: pos.x * skala, height: (cropW / 2) * skala }} />
        <div className="pointer-events-none absolute right-0 bg-neutral-900/60" style={{ top: pos.y * skala, left: (pos.x + cropW) * skala, height: (cropW / 2) * skala }} />
        <div
          role="presentation"
          onPointerDown={mulaiTarik}
          onPointerMove={geser}
          onPointerUp={selesaiTarik}
          onPointerCancel={selesaiTarik}
          className="absolute cursor-move touch-none rounded-sm border-2 border-white"
          style={{ left: pos.x * skala, top: pos.y * skala, width: cropW * skala, height: (cropW / 2) * skala }}
        />
      </div>
      {maksW > LEBAR && (
        <label className="mt-4 flex items-center gap-3 font-body text-sm text-neutral-700">
          Perbesar
          <input type="range" min={LEBAR} max={maksW} step={1} value={maksW - cropW + LEBAR} onChange={(e) => ubahZoom({ target: { value: maksW - Number(e.target.value) + LEBAR } })} className="flex-1 accent-primary-500" />
        </label>
      )}
    </Modal>
  );
};

export default PotongGambarModal;

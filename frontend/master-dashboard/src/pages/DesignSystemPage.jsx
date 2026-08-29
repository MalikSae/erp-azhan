import React, { useState } from 'react';
import PageHeader from '../components/ui/PageHeader';
import Button from '../components/ui/Button';
import Badge from '../components/ui/Badge';
import Pill from '../components/ui/Pill';
import Alert from '../components/ui/Alert';
import Card from '../components/ui/Card';
import MetaBox from '../components/ui/MetaBox';
import Input from '../components/ui/Input';
import Label from '../components/ui/Label';
import Textarea from '../components/ui/Textarea';
import FormField from '../components/ui/FormField';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import EmptyState from '../components/ui/EmptyState';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import Modal from '../components/ui/Modal';
import Toggle from '../components/ui/Toggle';
import KaabaIcon from '../../../shared/src/components/icons/KaabaIcon';
import {
  Plane,
  Building2,
  Hotel,
  Calendar,
  ArrowRight,
  AlertCircle,
  CheckCircle2,
  Users,
  Search,
  Plus,
  ShieldCheck,
  CreditCard,
  Layers,
  Sparkles,
  Palette,
  Type,
  LayoutGrid,
  Check,
  X
} from 'lucide-react';

const ColorSwatch = ({ name, hex, bgClass, textClass = 'text-white', borderClass = 'border-transparent' }) => (
  <div className="flex flex-col rounded-xl overflow-hidden border border-neutral-200/80 shadow-2xs">
    <div className={`h-14 ${bgClass} ${borderClass} flex items-end justify-start p-2`}>
      <span className={`text-[10px] font-mono font-bold ${textClass}`}>{hex}</span>
    </div>
    <div className="bg-white p-2.5">
      <p className="text-xs font-heading font-bold text-neutral-900 leading-none">{name}</p>
      <p className="text-[10px] font-mono text-neutral-400 mt-1">{bgClass}</p>
    </div>
  </div>
);

const DesignSystemPage = () => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [toggleState, setToggleState] = useState(true);
  const [selectedDropdown, setSelectedDropdown] = useState('');

  return (
    <div className="p-4 md:p-8 max-w-7xl mx-auto space-y-12 font-body">
      {/* Header */}
      <div className="bg-white border border-neutral-200/80 rounded-2xl p-6 md:p-8 shadow-card flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="inline-flex items-center gap-1.5 bg-sidebar-bg text-primary-500 px-3 py-1 rounded-full text-xs font-heading font-bold shadow-xs">
              <KaabaIcon className="w-3.5 h-3.5" />
              Azhan Design System
            </span>
            <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200/60 px-2.5 py-0.5 rounded-full text-xs font-semibold">
              <Sparkles className="w-3 h-3 text-emerald-500" />
              v2.0 Altezza Style
            </span>
          </div>
          <h1 className="text-2xl md:text-3xl font-heading font-extrabold text-neutral-900 tracking-tight">
            Design Tokens & UI Component Kit
          </h1>
          <p className="mt-1 text-sm text-neutral-500 max-w-2xl font-body">
            Dokumentasi lengkap standar palet warna, tipografi, kartu metrik, form controls, badge status, dan tabel untuk seluruh antarmuka Master Dashboard.
          </p>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <Button variant="secondary" onClick={() => setIsModalOpen(true)} className="flex items-center gap-1.5">
            <span>Uji Modal</span>
          </Button>
          <Button variant="primary" onClick={() => window.scrollTo({ top: 800, behavior: 'smooth' })}>
            <span>Jelajahi Komponen</span>
          </Button>
        </div>
      </div>

      {/* 1. Color Palette */}
      <section className="space-y-6">
        <div className="flex items-center gap-2.5 pb-3 border-b border-neutral-200">
          <Palette className="w-5 h-5 text-primary-600" />
          <h2 className="text-xl font-heading font-bold text-neutral-900">1. Palet Warna (Color Palette)</h2>
        </div>

        {/* Sidebar Dark Tokens */}
        <div className="space-y-3">
          <h3 className="text-sm font-heading font-bold text-neutral-700 uppercase tracking-wider">
            A. Dark Sidebar & Charcoal Palette
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3">
            <ColorSwatch name="Sidebar Base" hex="#181C1F" bgClass="bg-sidebar-bg" />
            <ColorSwatch name="Sidebar Surface" hex="#22272B" bgClass="bg-sidebar-surface" />
            <ColorSwatch name="Sidebar Hover" hex="#292F34" bgClass="bg-sidebar-hover" />
            <ColorSwatch name="Sidebar Border" hex="#2A3036" bgClass="bg-sidebar-border" />
            <ColorSwatch name="Sidebar Muted" hex="#8C95A0" bgClass="bg-sidebar-muted" />
          </div>
        </div>

        {/* Warm Amber Gold Brand Palette */}
        <div className="space-y-3">
          <h3 className="text-sm font-heading font-bold text-neutral-700 uppercase tracking-wider">
            B. Warm Amber Gold Palette (Brand Basis & Aksen Emas)
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
            <ColorSwatch name="Primary Gold (Basis)" hex="#FED853" bgClass="bg-primary-500" textClass="text-brand-dark" />
            <ColorSwatch name="Gold Hover" hex="#F5CD3E" bgClass="bg-primary-600" textClass="text-brand-dark" />
            <ColorSwatch name="Gold Light Tint" hex="#FEF7D6" bgClass="bg-primary-100" textClass="text-neutral-800" borderClass="border-amber-200" />
            <ColorSwatch name="Soft Cream 50" hex="#FEFDF0" bgClass="bg-primary-50" textClass="text-neutral-800" borderClass="border-neutral-200" />
          </div>
        </div>

        {/* Semantic Status Colors */}
        <div className="space-y-3">
          <h3 className="text-sm font-heading font-bold text-neutral-700 uppercase tracking-wider">
            C. Semantic Status Scales (Pastel Tint + Bold Text)
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div className="p-3.5 rounded-xl bg-success-50 border border-success-200 text-success-800 space-y-1">
              <span className="text-[10px] font-bold uppercase tracking-wider bg-success-200/60 text-success-900 px-2 py-0.5 rounded-full">Success / Terbit</span>
              <p className="text-xs font-semibold mt-2">#16A34A / #EAF8F0</p>
            </div>
            <div className="p-3.5 rounded-xl bg-warning-50 border border-warning-200 text-warning-800 space-y-1">
              <span className="text-[10px] font-bold uppercase tracking-wider bg-warning-200/60 text-warning-900 px-2 py-0.5 rounded-full">Warning / Review</span>
              <p className="text-xs font-semibold mt-2">#D97706 / #FEF3C7</p>
            </div>
            <div className="p-3.5 rounded-xl bg-danger-50 border border-danger-200 text-danger-800 space-y-1">
              <span className="text-[10px] font-bold uppercase tracking-wider bg-danger-200/60 text-danger-900 px-2 py-0.5 rounded-full">Danger / Batal</span>
              <p className="text-xs font-semibold mt-2">#DC2626 / #FEF2F2</p>
            </div>
            <div className="p-3.5 rounded-xl bg-sky-50 border border-sky-200 text-sky-800 space-y-1">
              <span className="text-[10px] font-bold uppercase tracking-wider bg-sky-200/60 text-sky-900 px-2 py-0.5 rounded-full">Info / DP Paid</span>
              <p className="text-xs font-semibold mt-2">#0284C7 / #EBF5FF</p>
            </div>
          </div>
        </div>
      </section>

      {/* 2. Typography Hierarchy */}
      <section className="space-y-6">
        <div className="flex items-center gap-2.5 pb-3 border-b border-neutral-200">
          <Type className="w-5 h-5 text-primary-600" />
          <h2 className="text-xl font-heading font-bold text-neutral-900">2. Tipografi (DM Sans)</h2>
        </div>

        <div className="bg-white border border-neutral-200/80 rounded-2xl p-6 shadow-card space-y-5">
          <div className="flex items-baseline justify-between border-b border-neutral-100 pb-3">
            <span className="text-3xl font-heading font-extrabold text-neutral-900">Display / Hero (30px)</span>
            <span className="text-xs font-mono text-neutral-400 font-medium">text-3xl font-extrabold</span>
          </div>
          <div className="flex items-baseline justify-between border-b border-neutral-100 pb-3">
            <span className="text-2xl font-heading font-bold text-neutral-900">Page Header Title (24px)</span>
            <span className="text-xs font-mono text-neutral-400 font-medium">text-2xl font-bold</span>
          </div>
          <div className="flex items-baseline justify-between border-b border-neutral-100 pb-3">
            <span className="text-lg font-heading font-bold text-neutral-800">Card Header / Subjudul (18px)</span>
            <span className="text-xs font-mono text-neutral-400 font-medium">text-lg font-bold</span>
          </div>
          <div className="flex items-baseline justify-between border-b border-neutral-100 pb-3">
            <span className="text-sm font-body text-neutral-700">Body Text Reguler — Menggunakan font DM Sans untuk keterbacaan tinggi di seluruh tabel dan formulir.</span>
            <span className="text-xs font-mono text-neutral-400 font-medium">text-sm font-body</span>
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-xs font-mono font-semibold bg-neutral-100 text-neutral-800 px-2 py-1 rounded">TRX-202608-0026 (Monospace Code/ID)</span>
            <span className="text-xs font-mono text-neutral-400 font-medium">text-xs font-mono</span>
          </div>
        </div>
      </section>

      {/* 3. Buttons & Actions */}
      <section className="space-y-6">
        <div className="flex items-center gap-2.5 pb-3 border-b border-neutral-200">
          <Sparkles className="w-5 h-5 text-primary-600" />
          <h2 className="text-xl font-heading font-bold text-neutral-900">3. Tombol & Aksi (Buttons)</h2>
        </div>

        <div className="bg-white border border-neutral-200/80 rounded-2xl p-6 shadow-card space-y-6">
          <div className="space-y-2">
            <p className="text-xs font-heading font-bold text-neutral-500 uppercase tracking-wider">Varian Standar</p>
            <div className="flex flex-wrap gap-3 items-center">
              <Button variant="primary" icon={<Plus className="w-4 h-4" />}>
                Primary Gold
              </Button>
              <Button variant="dark">Dark Charcoal</Button>
              <Button variant="secondary">Secondary Outline</Button>
              <Button variant="danger">Danger Action</Button>
              <Button variant="ghost">Ghost Button</Button>
            </div>
          </div>

          <div className="space-y-2 pt-4 border-t border-neutral-100">
            <p className="text-xs font-heading font-bold text-neutral-500 uppercase tracking-wider">Ukuran & Status</p>
            <div className="flex flex-wrap gap-3 items-center">
              <Button variant="primary" size="sm">Small (sm)</Button>
              <Button variant="primary" size="md">Medium (md)</Button>
              <Button variant="primary" disabled>Disabled State</Button>
              <Button variant="primary" isLoading>Memproses...</Button>
              <Button variant="secondary" isLoading>Menyimpan...</Button>
            </div>
          </div>
        </div>
      </section>

      {/* 4. Badges & Status Pills */}
      <section className="space-y-6">
        <div className="flex items-center gap-2.5 pb-3 border-b border-neutral-200">
          <Layers className="w-5 h-5 text-primary-600" />
          <h2 className="text-xl font-heading font-bold text-neutral-900">4. Status Badges (Altezza Pill Style)</h2>
        </div>

        <div className="bg-white border border-neutral-200/80 rounded-2xl p-6 shadow-card space-y-6">
          <div className="space-y-3">
            <p className="text-xs font-heading font-bold text-neutral-500 uppercase tracking-wider">Badge Status Paket & Transaksi</p>
            <div className="flex flex-wrap gap-3 items-center">
              <Badge variant="published">Terbit (Published)</Badge>
              <Badge variant="approved">Terkonfirmasi (Approved)</Badge>
              <Badge variant="warning">Draft / Review</Badge>
              <Badge variant="danger">Batal (Cancelled)</Badge>
              <Badge variant="archived">Diarsipkan</Badge>
              <Badge variant="primary">PIC Jamaah</Badge>
              <Badge variant="promo">Promo Khusus</Badge>
            </div>
          </div>

          <div className="space-y-3 pt-4 border-t border-neutral-100">
            <p className="text-xs font-heading font-bold text-neutral-500 uppercase tracking-wider">Filter Tag Pills</p>
            <div className="flex flex-wrap gap-3 items-center">
              <Pill label="Semua Brand (4)" />
              <Pill label="Hana Tours" variant="primary" onRemove={() => {}} />
              <Pill label="Keberangkatan Aktif" variant="success" onRemove={() => {}} />
              <Pill label="Filter Khusus" variant="warning" onRemove={() => {}} />
            </div>
          </div>
        </div>
      </section>

      {/* 5. Attention Cards Demonstration */}
      <section className="space-y-6">
        <div className="flex items-center gap-2.5 pb-3 border-b border-neutral-200">
          <LayoutGrid className="w-5 h-5 text-primary-600" />
          <h2 className="text-xl font-heading font-bold text-neutral-900">5. Kartu Metrik Operasional (Attention Cards)</h2>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-white border border-neutral-200/80 rounded-2xl p-5 shadow-card flex flex-col justify-between group">
            <div>
              <div className="flex items-start justify-between gap-3 mb-3">
                <div className="w-10 h-10 rounded-xl bg-neutral-100 text-neutral-700 flex items-center justify-center">
                  <Plane className="w-5 h-5" />
                </div>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold bg-warning-50 text-warning-700 border border-warning-200/80">
                  <AlertCircle className="w-3 h-3 text-warning-600" />
                  <span>Kesiapan Tiket</span>
                </span>
              </div>
              <span className="text-3xl font-heading font-extrabold text-neutral-900">1,302</span>
              <p className="text-xs font-semibold text-neutral-500 font-heading mt-1 uppercase tracking-wider">Arrival & Departure</p>
            </div>
            <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between">
              <span className="text-xs font-semibold text-neutral-700 group-hover:text-primary-700 flex items-center gap-1.5">
                Konfirmasi Jadwal <ArrowRight className="w-3.5 h-3.5" />
              </span>
            </div>
          </div>

          <div className="bg-white border border-neutral-200/80 rounded-2xl p-5 shadow-card flex flex-col justify-between group">
            <div>
              <div className="flex items-start justify-between gap-3 mb-3">
                <div className="w-10 h-10 rounded-xl bg-neutral-100 text-neutral-700 flex items-center justify-center">
                  <Users className="w-5 h-5" />
                </div>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold bg-success-50 text-success-700 border border-success-200/80">
                  <CheckCircle2 className="w-3 h-3 text-success-600" />
                  <span>Sisa Kuota</span>
                </span>
              </div>
              <span className="text-3xl font-heading font-extrabold text-neutral-900">4,272</span>
              <p className="text-xs font-semibold text-neutral-500 font-heading mt-1 uppercase tracking-wider">Kapasitas Kursi Aktif</p>
            </div>
            <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between">
              <span className="text-xs font-semibold text-neutral-700 group-hover:text-primary-700 flex items-center gap-1.5">
                Kelola Kuota <ArrowRight className="w-3.5 h-3.5" />
              </span>
            </div>
          </div>

          <div className="bg-white border border-neutral-200/80 rounded-2xl p-5 shadow-card flex flex-col justify-between group">
            <div>
              <div className="flex items-start justify-between gap-3 mb-3">
                <div className="w-10 h-10 rounded-xl bg-neutral-100 text-neutral-700 flex items-center justify-center">
                  <CreditCard className="w-5 h-5" />
                </div>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold bg-warning-50 text-warning-700 border border-warning-200/80">
                  <AlertCircle className="w-3 h-3 text-warning-600" />
                  <span>Verifikasi Bank</span>
                </span>
              </div>
              <span className="text-3xl font-heading font-extrabold text-neutral-900">809</span>
              <p className="text-xs font-semibold text-neutral-500 font-heading mt-1 uppercase tracking-wider">Pembayaran Pending</p>
            </div>
            <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between">
              <span className="text-xs font-semibold text-neutral-700 group-hover:text-primary-700 flex items-center gap-1.5">
                Lihat Pembayaran <ArrowRight className="w-3.5 h-3.5" />
              </span>
            </div>
          </div>

          <div className="bg-white border border-neutral-200/80 rounded-2xl p-5 shadow-card flex flex-col justify-between group">
            <div>
              <div className="flex items-start justify-between gap-3 mb-3">
                <div className="w-10 h-10 rounded-xl bg-neutral-100 text-neutral-700 flex items-center justify-center">
                  <Hotel className="w-5 h-5" />
                </div>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold bg-success-50 text-success-700 border border-success-200/80">
                  <CheckCircle2 className="w-3 h-3 text-success-600" />
                  <span>Tervalidasi</span>
                </span>
              </div>
              <span className="text-3xl font-heading font-extrabold text-neutral-900">55</span>
              <p className="text-xs font-semibold text-neutral-500 font-heading mt-1 uppercase tracking-wider">Mitra Hotel & Maskapai</p>
            </div>
            <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between">
              <span className="text-xs font-semibold text-neutral-700 group-hover:text-primary-700 flex items-center gap-1.5">
                Daftar Vendor <ArrowRight className="w-3.5 h-3.5" />
              </span>
            </div>
          </div>
        </div>
      </section>

      {/* 6. Form Inputs & Interactive Controls */}
      <section className="space-y-6">
        <div className="flex items-center gap-2.5 pb-3 border-b border-neutral-200">
          <Layers className="w-5 h-5 text-primary-600" />
          <h2 className="text-xl font-heading font-bold text-neutral-900">6. Form Controls & Interactivity</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-white border border-neutral-200/80 rounded-2xl p-6 shadow-card space-y-4">
            <h3 className="text-sm font-heading font-bold text-neutral-900">Input Text & Select</h3>
            
            <FormField label="Nama Paket Perjalanan">
              <Input placeholder="Contoh: Umroh Syawal 9 Hari Bintang 5" />
            </FormField>

            <FormField label="Pilih Kategori Paket">
              <CustomDropdown
                options={[
                  { label: 'Umroh Reguler (9 Hari)', value: 'reguler_9' },
                  { label: 'Umroh VIP Bintang 5 (12 Hari)', value: 'vip_12' },
                  { label: 'Umroh Plus Turki (14 Hari)', value: 'turki_14' }
                ]}
                value={selectedDropdown}
                onChange={setSelectedDropdown}
                placeholder="-- Pilih Jenis Paket --"
              />
            </FormField>

            <FormField label="Catatan Tambahan">
              <Textarea placeholder="Tuliskan keterangan detail di sini..." rows={3} />
            </FormField>
          </div>

          <div className="bg-white border border-neutral-200/80 rounded-2xl p-6 shadow-card space-y-5">
            <h3 className="text-sm font-heading font-bold text-neutral-900">Status Controls & Toggles</h3>
            
            <div className="flex items-center justify-between p-3.5 bg-neutral-50 rounded-xl border border-neutral-200/80">
              <div>
                <p className="text-xs font-bold text-neutral-900 font-heading">Kunci Kuota Kursi Otomatis</p>
                <p className="text-[11px] text-neutral-500 font-body">Kurangi sisa kursi begitu booking masuk status DP.</p>
              </div>
              <Toggle id="toggle-seat" checked={toggleState} onChange={setToggleState} />
            </div>

            <FormField label="Input Status Error" error="Nominal harga tidak boleh bernilai negatif">
              <Input defaultValue="-500000" className="border-danger-500" />
            </FormField>

            <div className="space-y-2">
              <p className="text-xs font-bold text-neutral-700 font-heading">Notifikasi / Alerts</p>
              <Alert variant="success">Data paket berhasil disimpan ke database.</Alert>
              <Alert variant="error">Terjadi kesalahan pada koneksi API backend.</Alert>
            </div>
          </div>
        </div>
      </section>

      {/* 7. Modern Data Table */}
      <section className="space-y-6">
        <div className="flex items-center gap-2.5 pb-3 border-b border-neutral-200">
          <LayoutGrid className="w-5 h-5 text-primary-600" />
          <h2 className="text-xl font-heading font-bold text-neutral-900">7. Modern Data Table (Altezza Card-Row)</h2>
        </div>

        <DataTable
          columns={[
            { header: 'ID Booking', key: 'id' },
            { header: 'Nama Jamaah (PIC)', key: 'pic' },
            { header: 'Paket & Tanggal', key: 'package' },
            { header: 'Pax / Kamar', key: 'pax' },
            { header: 'Tagihan / Sisa', key: 'finance' },
            { header: 'Status', key: 'status' }
          ]}
          data={[
            { id: 'AZH-2026-001', pic: 'Darlene Robertson', package: 'Umroh 9 Hari · 12 Nov', pax: '3 Pax (Quad)', finance: 'Rp 28.500.000 (Lunas)', status: 'published' },
            { id: 'AZH-2026-002', pic: 'Diane Russel', package: 'Umroh Reguler · 18 Nov', pax: '2 Pax (Double)', finance: 'Rp 15.000.000 (DP)', status: 'approved' },
            { id: 'AZH-2026-003', pic: 'Robert Fox', package: 'VIP Bintang 5 · 22 Nov', pax: '4 Pax (Quad)', finance: 'Rp 0 (Draft)', status: 'draft' },
            { id: 'AZH-2026-004', pic: 'Theresa Webb', package: 'Umroh Plus · 25 Nov', pax: '1 Pax (Single)', finance: 'Rp 0 (Batal)', status: 'danger' },
          ]}
          itemsPerPage={5}
          renderCell={(row, key) => {
            if (key === 'status') {
              return <Badge variant={row.status}>{row.status === 'published' ? 'Lunas' : row.status === 'approved' ? 'DP Terverifikasi' : row.status === 'draft' ? 'Draft' : 'Batal'}</Badge>;
            }
            if (key === 'id') {
              return <span className="font-mono text-xs font-bold text-neutral-900">{row.id}</span>;
            }
            if (key === 'pic') {
              return (
                <div className="flex items-center gap-2">
                  <div className="w-7 h-7 rounded-full bg-primary-100 text-primary-800 flex items-center justify-center text-xs font-bold">
                    {row.pic.charAt(0)}
                  </div>
                  <span className="font-semibold text-neutral-900">{row.pic}</span>
                </div>
              );
            }
            return row[key];
          }}
        />
      </section>

      {/* Modal Demo */}
      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title="Uji Dialog Konfirmasi"
        footer={
          <>
            <Button variant="ghost" onClick={() => setIsModalOpen(false)}>Batal</Button>
            <Button variant="primary" onClick={() => setIsModalOpen(false)}>Konfirmasi Simpan</Button>
          </>
        }
      >
        <p className="text-sm text-neutral-600 font-body leading-relaxed">
          Ini adalah contoh tampilan modal dialog dengan latar belakang blur halus, header tebal yang kontras, dan tombol aksi terstruktur.
        </p>
      </Modal>
    </div>
  );
};

export default DesignSystemPage;

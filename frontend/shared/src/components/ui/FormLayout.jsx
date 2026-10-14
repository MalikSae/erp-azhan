import React, { Children, isValidElement, useEffect, useRef, useState } from 'react';

// Standar form tambah/edit (acuan: Tambah/Edit Paket).
//
//   <FormLayout tabs={[{ id: 'sec-dasar', label: 'Dasar' }, ...]}>
//     <FormSection id="sec-dasar" title="Informasi Dasar">…field…</FormSection>
//     <FormSection id="sec-…" title="…">…</FormSection>
//     <FormAside>
//       <SidebarSection title="Publikasi">…tombol simpan…</SidebarSection>
//       <SidebarSection title="…">…</SidebarSection>
//     </FormAside>
//   </FormLayout>
//
// Kolom kiri: satu box putih, section dipisah garis tipis, judul di atas isi, tanpa kotak bersarang.
// Kolom kanan (320 px, menempel saat scroll): satu box berisi aksi simpan dan data pendukung.
// Tab navigasi (opsional) mengikuti scroll; di layar kecil sidebar turun ke bawah kolom utama.

// Satu section di kolom utama.
export const FormSection = ({ id, title, aside, children }) => (
  <section id={id} className="scroll-mt-16 px-5 py-6 sm:px-6">
    <div className="mb-5 flex items-center justify-between gap-3">
      <h3 className="font-heading text-base font-bold text-neutral-900">{title}</h3>
      {aside}
    </div>
    <div className="min-w-0">{children}</div>
  </section>
);

// Satu bagian di sidebar.
export const SidebarSection = ({ title, children }) => (
  <section className="p-4">
    {title && <h3 className="mb-3 font-heading text-[13px] font-bold text-neutral-900">{title}</h3>}
    {children}
  </section>
);

// Penanda isi sidebar; FormLayout memindahkannya ke kolom kanan.
export const FormAside = ({ children }) => <>{children}</>;

const FormLayout = ({ tabs = [], children }) => {
  const [activeTab, setActiveTab] = useState(tabs[0]?.id || null);
  // Setelah tab diklik, abaikan pemantau scroll sebentar agar tab yang diklik tetap aktif.
  const lockUntilRef = useRef(0);

  const tabKey = tabs.map(tab => tab.id).join('|');
  useEffect(() => {
    if (!tabs.length) return undefined;
    const main = document.querySelector('main');
    if (!main) return undefined;
    const onScroll = () => {
      if (Date.now() < lockUntilRef.current) return;
      const top = main.getBoundingClientRect().top;
      // Di dasar halaman section terakhir tak bisa naik ke atas, jadi ambang dilonggarkan.
      const atBottom = main.scrollTop + main.clientHeight >= main.scrollHeight - 4;
      const threshold = atBottom ? main.clientHeight * 0.55 : 140;
      let current = tabs[0].id;
      tabs.forEach(tab => {
        const el = document.getElementById(tab.id);
        if (el && el.getBoundingClientRect().top - top <= threshold) current = tab.id;
      });
      setActiveTab(current);
    };
    main.addEventListener('scroll', onScroll, { passive: true });
    onScroll();
    return () => main.removeEventListener('scroll', onScroll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tabKey]);

  const nodes = Children.toArray(children);
  const asideNodes = nodes.filter(node => isValidElement(node) && node.type === FormAside);
  const mainNodes = nodes.filter(node => !(isValidElement(node) && node.type === FormAside));

  return (
    <div className="mx-auto grid w-full max-w-7xl grid-cols-1 items-start gap-8 lg:grid-cols-[minmax(0,1fr)_320px] [&_input]:scroll-mt-28 [&_textarea]:scroll-mt-28 [&_select]:scroll-mt-28">
      <div className="min-w-0">
        {tabs.length > 0 && (
          <nav className="sticky -top-6 z-10 -mx-1 bg-page-bg px-1 pb-3 pt-2.5 md:-top-8">
            <div className="inline-flex max-w-full flex-wrap gap-1 rounded-xl bg-neutral-200/60 p-1">
              {tabs.map(tab => (
                <button
                  key={tab.id}
                  type="button"
                  onClick={() => {
                    lockUntilRef.current = Date.now() + 900;
                    setActiveTab(tab.id);
                    document.getElementById(tab.id)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
                  }}
                  className={`cursor-pointer rounded-lg px-3 py-1.5 font-body text-xs font-semibold transition-colors ${
                    activeTab === tab.id
                      ? 'bg-white text-primary-600 shadow-sm'
                      : 'text-neutral-600 hover:text-neutral-900'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </nav>
        )}

        <div className="divide-y divide-neutral-200 rounded-xl border border-neutral-200 bg-white">
          {mainNodes}
        </div>
      </div>

      {asideNodes.length > 0 && (
        <aside className="lg:sticky lg:top-4">
          <div className="divide-y divide-neutral-200 rounded-xl border border-neutral-200 bg-white">
            {asideNodes}
          </div>
        </aside>
      )}
    </div>
  );
};

// Varian tab vertikal: hanya satu section tampil sekaligus; navigasi di kolom kiri.
// Cocok untuk form besar dengan banyak kelompok data (mis. profil brand).
//
//   <FormTabs tabs={[{ id: 'sec-a', label: 'A' }, ...]}>
//     <FormSection id="sec-a" title="A">…</FormSection>
//     <FormAside>…</FormAside>
//   </FormTabs>
export const FormTabs = ({ tabs = [], children }) => {
  const [activeTab, setActiveTab] = useState(tabs[0]?.id || null);

  const nodes = Children.toArray(children);
  const asideNodes = nodes.filter(node => isValidElement(node) && node.type === FormAside);
  const mainNodes = nodes.filter(node => !(isValidElement(node) && node.type === FormAside));
  const activeNode = mainNodes.find(node => isValidElement(node) && node.props.id === activeTab) || mainNodes[0];

  return (
    <div className={`mx-auto grid w-full max-w-7xl grid-cols-1 items-start gap-6 ${asideNodes.length > 0 ? 'lg:grid-cols-[190px_minmax(0,1fr)_320px]' : 'lg:grid-cols-[190px_minmax(0,1fr)]'}`}>
      <nav className="lg:sticky lg:top-4" aria-label="Bagian form">
        <div className="flex gap-1 overflow-x-auto lg:flex-col lg:overflow-visible">
          {tabs.map(tab => (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              aria-current={activeTab === tab.id ? 'page' : undefined}
              className={`shrink-0 cursor-pointer rounded-lg px-3 py-2 text-left font-body text-[13px] font-semibold transition-colors ${
                activeTab === tab.id
                  ? 'bg-white text-primary-600 shadow-sm ring-1 ring-neutral-200'
                  : 'text-neutral-600 hover:bg-neutral-200/50 hover:text-neutral-900'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </nav>

      <div className="min-w-0 rounded-xl border border-neutral-200 bg-white">
        {activeNode}
      </div>

      {asideNodes.length > 0 && (
        <aside className="lg:sticky lg:top-4">
          <div className="divide-y divide-neutral-200 rounded-xl border border-neutral-200 bg-white">
            {asideNodes}
          </div>
        </aside>
      )}
    </div>
  );
};

export default FormLayout;

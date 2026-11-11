import React from 'react';

// Tab segmented (gaya tab navigasi form Tambah/Edit Paket): grup abu datar (gaya Cloudflare),
// tab aktif putih bertulisan gelap. Dipakai untuk berpindah antar daftar
// pada halaman yang sama, mis. Maskapai | Bandara.
//
//   <SegmentedTabs
//     tabs={[{ id: 'a', label: 'Maskapai', count: 17 }, { id: 'b', label: 'Bandara', count: 27 }]}
//     value={activeTab}
//     onChange={setActiveTab}
//   />
const SegmentedTabs = ({ tabs = [], value, onChange, className = '' }) => (
  <div className={`inline-flex max-w-full flex-wrap gap-1 rounded-md border border-neutral-200 bg-neutral-100 p-0.5 ${className}`} role="tablist">
    {tabs.map((tab) => {
      const isActive = value === tab.id;
      return (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={isActive}
          onClick={() => onChange?.(tab.id)}
          className={`inline-flex cursor-pointer items-center gap-1.5 rounded-sm px-3 py-1.5 font-body text-xs font-medium transition-colors ${
            isActive ? 'bg-white text-neutral-900 shadow-popover' : 'text-neutral-600 hover:text-neutral-900'
          }`}
        >
          <span>{tab.label}</span>
          {tab.count !== undefined && (
            <span className={`text-xs font-medium ${isActive ? 'text-neutral-600' : 'text-neutral-500'}`}>
              {tab.count}
            </span>
          )}
        </button>
      );
    })}
  </div>
);

export default SegmentedTabs;

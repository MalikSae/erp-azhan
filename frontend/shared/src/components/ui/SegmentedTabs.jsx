import React from 'react';

// Tab segmented (gaya tab navigasi form Tambah/Edit Paket): grup pil abu-abu,
// tab aktif putih bertulisan primary. Dipakai untuk berpindah antar daftar
// pada halaman yang sama, mis. Maskapai | Bandara.
//
//   <SegmentedTabs
//     tabs={[{ id: 'a', label: 'Maskapai', count: 17 }, { id: 'b', label: 'Bandara', count: 27 }]}
//     value={activeTab}
//     onChange={setActiveTab}
//   />
const SegmentedTabs = ({ tabs = [], value, onChange, className = '' }) => (
  <div className={`inline-flex max-w-full flex-wrap gap-1 rounded-xl bg-neutral-200/60 p-1 ${className}`} role="tablist">
    {tabs.map((tab) => {
      const isActive = value === tab.id;
      return (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={isActive}
          onClick={() => onChange?.(tab.id)}
          className={`inline-flex cursor-pointer items-center gap-1.5 rounded-lg px-3 py-1.5 font-body text-xs font-semibold transition-colors ${
            isActive ? 'bg-white text-primary-600 shadow-sm' : 'text-neutral-600 hover:text-neutral-900'
          }`}
        >
          <span>{tab.label}</span>
          {tab.count !== undefined && (
            <span className={`text-[11px] font-medium ${isActive ? 'text-primary-400' : 'text-neutral-400'}`}>
              {tab.count}
            </span>
          )}
        </button>
      );
    })}
  </div>
);

export default SegmentedTabs;

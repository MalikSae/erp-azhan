import React from 'react';
import { Plus, Trash2 } from 'lucide-react';
import Input from 'shared/ui/Input';
import CustomDropdown from 'shared/ui/CustomDropdown';

export const UNTUK_OPTIONS = [
  { value: 'semua', label: 'Semua' },
  { value: 'pria', label: 'Pria' },
  { value: 'wanita', label: 'Wanita' },
];

const untukLabel = (v) => UNTUK_OPTIONS.find((o) => o.value === v)?.label || 'Semua';
const valueOf = (v) => (v?.target ? v.target.value : v);

// Daftar baca-saja isi perlengkapan (untuk menampilkan set bawaan).
export const PerlengkapanSetList = ({ items = [], emptyText = 'Belum ada isi perlengkapan.' }) => {
  if (!items.length) {
    return <p className="text-xs font-body text-neutral-500">{emptyText}</p>;
  }
  return (
    <ul className="divide-y divide-neutral-200/80 text-[13px] font-body">
      {items.map((it) => (
        <li key={it.perlengkapan_item_id} className="flex items-center justify-between gap-3 py-2">
          <span className="font-medium text-neutral-800">{it.nama}</span>
          <span className="shrink-0 text-xs text-neutral-500">
            {it.qty} pcs · {untukLabel(it.untuk)}
          </span>
        </li>
      ))}
    </ul>
  );
};

// Editor isi perlengkapan: item dari katalog + jumlah per jamaah + berlaku untuk siapa.
//   value: [{ perlengkapan_item_id, qty, untuk }]
const PerlengkapanSetEditor = ({ catalog = [], value = [], onChange }) => {
  const used = new Set(value.map((r) => String(r.perlengkapan_item_id)));

  const update = (index, patch) => {
    onChange(value.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  };
  const remove = (index) => onChange(value.filter((_, i) => i !== index));
  const add = () => {
    const next = catalog.find((c) => !used.has(String(c.id)));
    if (!next) return;
    onChange([...value, { perlengkapan_item_id: next.id, qty: 1, untuk: 'semua' }]);
  };

  const canAdd = catalog.some((c) => !used.has(String(c.id)));

  return (
    <div className="space-y-2">
      {value.length === 0 && (
        <p className="text-xs font-body text-neutral-500">Belum ada item. Tambahkan item perlengkapan untuk jamaah.</p>
      )}

      {value.map((row, index) => {
        const options = catalog
          .filter((c) => String(c.id) === String(row.perlengkapan_item_id) || !used.has(String(c.id)))
          .map((c) => ({ value: String(c.id), label: c.nama }));
        return (
          <div key={`${row.perlengkapan_item_id}-${index}`} className="grid grid-cols-[minmax(0,1fr)_72px_132px_36px] items-center gap-2">
            <CustomDropdown
              value={String(row.perlengkapan_item_id)}
              onChange={(v) => update(index, { perlengkapan_item_id: Number(valueOf(v)) })}
              options={options}
              className="!mb-0"
            />
            <Input
              type="number"
              min="1"
              step="1"
              value={row.qty}
              onChange={(e) => update(index, { qty: e.target.value === '' ? '' : Math.max(1, parseInt(e.target.value, 10) || 1) })}
              className="!mb-0"
              aria-label="Jumlah per jamaah"
            />
            <CustomDropdown
              value={row.untuk || 'semua'}
              onChange={(v) => update(index, { untuk: valueOf(v) })}
              options={UNTUK_OPTIONS}
              className="!mb-0"
            />
            <button
              type="button"
              onClick={() => remove(index)}
              title="Hapus item"
              className="flex h-10 w-9 cursor-pointer items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-500 transition-colors hover:bg-neutral-50 hover:text-danger-600"
            >
              <Trash2 size={15} strokeWidth={1.75} />
            </button>
          </div>
        );
      })}

      <button
        type="button"
        onClick={add}
        disabled={!canAdd}
        className="inline-flex cursor-pointer items-center gap-1.5 text-xs font-semibold text-primary-600 hover:text-primary-700 disabled:cursor-not-allowed disabled:text-neutral-400"
      >
        <Plus size={14} strokeWidth={2} />
        Tambah item
      </button>
    </div>
  );
};

export default PerlengkapanSetEditor;

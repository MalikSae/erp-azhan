import React from 'react';
import Button from 'shared/ui/Button';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';

// Format mengikuti dokumen operasional: per hari ada rute dan poin kegiatan (tanpa jam).
export const emptyActivity = () => ({ time: '', text: '' });
export const emptyDay = () => ({ title: '', location: '', activities: [emptyActivity()] });

// Data server -> bentuk editor. Format lama memakai jam per kegiatan; kini jam (jika ada) menjadi bagian teks.
export const normalizeDays = (days) => (days || []).map(d => ({
  title: d.title || '',
  location: d.location || d.title || '',
  activities: (d.activities || []).map(act => ({
    time: '',
    text: [act.time, act.text].filter(Boolean).join(' ').trim()
  }))
}));

// Bentuk editor -> payload server.
export const daysToPayload = (days) => (days || []).map(day => ({
  title: day.title || '',
  location: (day.location || '').trim(),
  activities: day.activities
    .map(act => ({ time: '', text: (act.text || '').trim() }))
    .filter(act => act.text !== '')
}));

// Pesan galat pertama, atau '' bila valid.
export const validateDays = (days) => {
  if (!days || days.length === 0) return 'Itinerary khusus minimal satu hari.';
  const idx = days.findIndex(d => !(d.location || '').trim());
  if (idx >= 0) return `Lokasi hari ke-${idx + 1} wajib diisi.`;
  return '';
};

const AddButton = ({ onClick, children }) => (
  <Button variant="dashed" className="w-full"
    onClick={onClick}
  >
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
    </svg>
    <span>{children}</span>
  </Button>
);

const RemoveButton = ({ onClick, title }) => (
  <Button variant="danger-outline" className="shrink-0"
    onClick={onClick}
    title={title}
    aria-label={title}
  >
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
    </svg>
  </Button>
);

// Penyunting hari dan poin kegiatan. `days` dikendalikan induk lewat onChange(daysBaru).
const ItineraryDaysEditor = ({ days, onChange }) => {
  const updateDay = (dayIndex, updater) => {
    const next = [...days];
    next[dayIndex] = updater(next[dayIndex]);
    onChange(next);
  };

  const handleAddDay = () => onChange([...days, emptyDay()]);
  const handleRemoveDay = (dayIndex) => onChange(days.filter((_, i) => i !== dayIndex));
  const handleDayChange = (dayIndex, field, value) => updateDay(dayIndex, day => ({ ...day, [field]: value }));
  const handleAddActivity = (dayIndex) => updateDay(dayIndex, day => ({ ...day, activities: [...day.activities, emptyActivity()] }));
  const handleRemoveActivity = (dayIndex, actIndex) => {
    updateDay(dayIndex, day => ({ ...day, activities: day.activities.filter((_, i) => i !== actIndex) }));
  };
  const handleActivityChange = (dayIndex, actIndex, field, value) => {
    updateDay(dayIndex, day => ({
      ...day,
      activities: day.activities.map((act, i) => (i === actIndex ? { ...act, [field]: value } : act))
    }));
  };

  return (
    <div className="space-y-6">
      {days.length === 0 && (
        <p className="text-sm text-neutral-500 font-body">
          Belum ada hari.
        </p>
      )}

      {days.map((day, dayIndex) => (
        <div key={dayIndex} className="space-y-4 border-t border-neutral-200 pt-5 first:border-t-0 first:pt-0">
          <div className="flex items-center justify-between">
            <h4 className="font-heading text-sm font-semibold text-neutral-900">Hari ke-{dayIndex + 1}</h4>
            <button
              type="button"
              onClick={() => handleRemoveDay(dayIndex)}
              className="text-xs font-medium text-neutral-500 hover:text-danger-600 transition-colors cursor-pointer"
            >
              Hapus hari
            </button>
          </div>

          <Input
            label="Lokasi"
            value={day.location}
            onChange={(e) => handleDayChange(dayIndex, 'location', e.target.value)}
            placeholder="Makkah – Madinah"
            required
            className="!mb-0"
          />

          <div className="space-y-2">
            <span className="block min-h-5 text-sm font-medium leading-5 text-neutral-800 font-body">Kegiatan</span>
            {day.activities.map((act, actIndex) => (
              <div key={actIndex} className="flex items-start gap-3">
                <div className="min-w-0 flex-1">
                  <Textarea
                    value={act.text}
                    onChange={(e) => handleActivityChange(dayIndex, actIndex, 'text', e.target.value)}
                    placeholder="Sarapan pagi lalu berkumpul di lobby"
                    rows={2}
                    className="!mb-0"
                  />
                </div>
                <RemoveButton
                  title="Hapus poin kegiatan"
                  onClick={() => handleRemoveActivity(dayIndex, actIndex)}
                />
              </div>
            ))}
            <AddButton onClick={() => handleAddActivity(dayIndex)}>Tambah poin kegiatan</AddButton>
            <p className="text-xs text-neutral-500 font-body">
              Satu poin per kotak. Tebalkan kata dengan **tanda bintang dua**.
            </p>
          </div>
        </div>
      ))}

      <AddButton onClick={handleAddDay}>
        {days.length === 0 ? 'Tambah hari pertama' : 'Tambah hari'}
      </AddButton>
    </div>
  );
};

export default ItineraryDaysEditor;

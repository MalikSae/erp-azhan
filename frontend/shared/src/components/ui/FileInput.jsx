import React, { useRef } from 'react';
import { Upload, Image as ImageIcon, X, RefreshCw, Loader2 } from 'lucide-react';
import FormField from './FormField';

const FileInput = ({
  label,
  name,
  accept = 'image/*',
  value,
  previewUrl,
  onChange,
  onRemove,
  isUploading = false,
  uploadingText = 'Mengunggah file...',
  helperText,
  placeholder = 'Pilih file gambar (JPG, PNG, WebP)...',
  error,
  required,
  disabled,
  className = '',
  showPreview = true,
}) => {
  const fileInputRef = useRef(null);

  const displayPreview = previewUrl || (value && typeof value === 'string' ? value : null);

  const handleTriggerClick = () => {
    if (fileInputRef.current && !disabled && !isUploading) {
      fileInputRef.current.click();
    }
  };

  const handleFileChange = (e) => {
    if (onChange) {
      onChange(e);
    }
  };

  const handleRemove = (e) => {
    e.stopPropagation();
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
    if (onRemove) {
      onRemove();
    }
  };

  return (
    <FormField label={label} error={error} required={required} helperText={helperText} className={className}>
      <input
        ref={fileInputRef}
        type="file"
        name={name}
        accept={accept}
        onChange={handleFileChange}
        disabled={disabled || isUploading}
        className="hidden"
        tabIndex={-1}
      />

      {displayPreview && showPreview ? (
        <div className="flex items-center gap-3 p-2 bg-neutral-50/80 border border-neutral-200/90 rounded-xl shadow-2xs">
          <div className="relative shrink-0 rounded-lg overflow-hidden bg-white border border-neutral-200 w-14 h-14 flex items-center justify-center">
            <img
              src={
                displayPreview.startsWith('/') && !displayPreview.startsWith('blob:') && !displayPreview.startsWith('http')
                  ? `${import.meta.env.VITE_API_BASE_URL || ''}${displayPreview}`
                  : displayPreview
              }
              alt="Preview"
              className="w-full h-full object-contain p-1"
            />
            {isUploading && (
              <div className="absolute inset-0 bg-white/80 backdrop-blur-xs flex items-center justify-center">
                <Loader2 size={16} className="animate-spin text-primary-600" />
              </div>
            )}
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-1.5 text-xs font-semibold text-neutral-800 truncate">
              <ImageIcon size={14} className="text-neutral-500 shrink-0" />
              <span className="truncate">File terpilih</span>
            </div>
            <p className="text-[11px] text-neutral-400 truncate mt-0.5">
              {isUploading ? uploadingText : 'Klik tombol untuk mengganti atau menghapus'}
            </p>
          </div>

          <div className="flex items-center gap-1.5 shrink-0">
            <button
              type="button"
              onClick={handleTriggerClick}
              disabled={disabled || isUploading}
              className="px-2.5 py-1.5 text-xs font-medium text-neutral-700 bg-white border border-neutral-200 rounded-lg hover:bg-neutral-50 hover:text-neutral-900 transition-colors shadow-2xs cursor-pointer flex items-center gap-1"
              title="Ganti file"
            >
              <RefreshCw size={12} className={isUploading ? "animate-spin" : ""} />
              <span>Ganti</span>
            </button>
            {onRemove && (
              <button
                type="button"
                onClick={handleRemove}
                disabled={disabled || isUploading}
                className="p-1.5 text-neutral-400 hover:text-danger-600 hover:bg-danger-50 rounded-lg transition-colors cursor-pointer"
                title="Hapus file"
              >
                <X size={16} />
              </button>
            )}
          </div>
        </div>
      ) : (
        <div
          onClick={handleTriggerClick}
          className={`flex items-center justify-between gap-3 px-3.5 py-2.5 bg-white border rounded-xl shadow-2xs transition-all cursor-pointer group ${
            error
              ? 'border-danger-500 ring-2 ring-danger-500/20'
              : 'border-neutral-200/90 hover:border-neutral-300 hover:bg-neutral-50/50'
          } ${disabled || isUploading ? 'opacity-60 cursor-not-allowed pointer-events-none' : ''}`}
        >
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-7 h-7 rounded-lg bg-neutral-100 border border-neutral-200/80 flex items-center justify-center text-neutral-500 group-hover:text-neutral-900 group-hover:bg-primary-500 transition-colors shrink-0">
              {isUploading ? (
                <Loader2 size={14} className="animate-spin text-neutral-700" />
              ) : (
                <Upload size={14} />
              )}
            </div>
            <div className="min-w-0">
              <span className="text-xs md:text-sm text-neutral-500 group-hover:text-neutral-900 transition-colors block truncate">
                {isUploading ? uploadingText : placeholder}
              </span>
            </div>
          </div>

          <div className="shrink-0">
            <span className="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold rounded-lg bg-neutral-100 text-neutral-700 border border-neutral-200 group-hover:bg-primary-500 group-hover:text-neutral-900 group-hover:border-primary-500 transition-all shadow-2xs">
              <Upload size={12} />
              <span>Pilih File</span>
            </span>
          </div>
        </div>
      )}
    </FormField>
  );
};

export default FileInput;

import { useEffect } from 'react';
import { EditorContent, useEditor } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import { Bold, Heading2, Heading3, Italic, List, ListOrdered, Underline as IkonUnderline, Undo2, Redo2 } from 'lucide-react';
import Label from 'shared/ui/Label';

// Editor teks berformat terbatas untuk konten yang tampil di app dan web (S&K agen, MoU agen). Hanya format yang
// diterima server (lihat shared.SanitasiHTML di backend): judul, tebal, miring, garis bawah, daftar butir dan
// bernomor. Nilai berupa HTML; teks lama tanpa tag dimuat sebagai paragraf.
//   value   : HTML tersimpan
//   onChange: (html) => void; editor kosong mengirim ''
export default function EditorTeks({ label, value, onChange, placeholder, helpText, minTinggi = 'min-h-48' }) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [2, 3] },
        // Hanya format yang lolos sanitasi server.
        code: false,
        codeBlock: false,
        horizontalRule: false,
        strike: false,
        link: false,
      }),
    ],
    content: keHTML(value),
    editorProps: {
      attributes: {
        class: `prose-editor ${minTinggi} px-3 py-2 font-body text-sm text-neutral-900 focus:outline-none`,
        'aria-label': label,
      },
    },
    onUpdate: ({ editor: e }) => onChange(e.isEmpty ? '' : e.getHTML()),
  });

  // Nilai dari luar berubah (mis. data brand selesai dimuat): isi ulang editor tanpa memicu onChange.
  useEffect(() => {
    if (!editor) return;
    const html = keHTML(value);
    if (html !== editor.getHTML() && !(editor.isEmpty && !html)) {
      editor.commands.setContent(html, { emitUpdate: false });
    }
  }, [editor, value]);

  const tombol = [
    { ikon: Heading2, label: 'Judul', aktif: () => editor?.isActive('heading', { level: 2 }), jalankan: () => editor.chain().focus().toggleHeading({ level: 2 }).run() },
    { ikon: Heading3, label: 'Subjudul', aktif: () => editor?.isActive('heading', { level: 3 }), jalankan: () => editor.chain().focus().toggleHeading({ level: 3 }).run() },
    { ikon: Bold, label: 'Tebal', aktif: () => editor?.isActive('bold'), jalankan: () => editor.chain().focus().toggleBold().run() },
    { ikon: Italic, label: 'Miring', aktif: () => editor?.isActive('italic'), jalankan: () => editor.chain().focus().toggleItalic().run() },
    { ikon: IkonUnderline, label: 'Garis bawah', aktif: () => editor?.isActive('underline'), jalankan: () => editor.chain().focus().toggleUnderline().run() },
    { ikon: List, label: 'Daftar butir', aktif: () => editor?.isActive('bulletList'), jalankan: () => editor.chain().focus().toggleBulletList().run() },
    { ikon: ListOrdered, label: 'Daftar bernomor', aktif: () => editor?.isActive('orderedList'), jalankan: () => editor.chain().focus().toggleOrderedList().run() },
    { ikon: Undo2, label: 'Urungkan', aktif: () => false, jalankan: () => editor.chain().focus().undo().run() },
    { ikon: Redo2, label: 'Ulangi', aktif: () => false, jalankan: () => editor.chain().focus().redo().run() },
  ];

  return (
    <div>
      {label && <Label>{label}</Label>}
      <div className="rounded-xl border border-neutral-200 bg-white focus-within:border-neutral-400">
        <div role="toolbar" aria-label={`Format ${label || 'teks'}`} className="flex flex-wrap gap-1 border-b border-neutral-200 p-1.5">
          {tombol.map(({ ikon: Ikon, label: nama, aktif, jalankan }) => (
            <button
              key={nama}
              type="button"
              title={nama}
              aria-label={nama}
              aria-pressed={aktif()}
              disabled={!editor}
              onClick={jalankan}
              className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 ${
                aktif() ? 'bg-neutral-900 text-white' : 'text-neutral-600 hover:bg-neutral-100'
              }`}
            >
              <Ikon className="h-4 w-4" aria-hidden="true" />
            </button>
          ))}
        </div>
        <EditorContent editor={editor} />
        {placeholder && editor?.isEmpty && (
          <p className="pointer-events-none -mt-9 px-3 pb-2 font-body text-sm text-neutral-400">{placeholder}</p>
        )}
      </div>
      {helpText && <p className="mt-1 text-xs text-neutral-500 font-body">{helpText}</p>}
    </div>
  );
}

// Teks lama tanpa tag (S&K agen sebelum editor ada): baris jadi paragraf supaya urutan butir tetap terbaca.
function keHTML(nilai) {
  const s = String(nilai || '').trim();
  if (!s) return '';
  if (/<[a-z][\s\S]*>/i.test(s)) return s;
  const esc = (t) => t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  return s.split(/\n+/).map((baris) => `<p>${esc(baris)}</p>`).join('');
}

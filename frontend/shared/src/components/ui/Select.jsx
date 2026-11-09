// Dropdown standar proyek ini: Select adalah CustomDropdown (menu sendiri, bukan <select> bawaan browser), supaya tampilan seragam di semua halaman.
// Pemakaian: options={[{ value, label }]} atau anak <option>; onChange(nilai), atau onChange({ target: { name, value } }) bila name diberikan.
// Pilihan kosong (mis. "Semua") ditulis sebagai option dengan value ''.
export { default } from './CustomDropdown';

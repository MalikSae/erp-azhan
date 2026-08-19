import React, { useState } from 'react';
import PageHeader from '../components/ui/PageHeader';
import Button from '../components/ui/Button';
import Badge from '../components/ui/Badge';
import Pill from '../components/ui/Pill';
import Alert from '../components/ui/Alert';
import Card from '../components/ui/Card';
import Input from '../components/ui/Input';
import Label from '../components/ui/Label';
import Select from '../components/ui/Select';
import Textarea from '../components/ui/Textarea';
import FormField from '../components/ui/FormField';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import EmptyState from '../components/ui/EmptyState';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import Modal from '../components/ui/Modal';

const DesignSystemPage = () => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-12">
      <PageHeader 
        title="Design System" 
        subtitle="Daftar komponen UI yang tersedia di Master Dashboard."
      />

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">1. Typography & Colors</h2>
        <div className="space-y-2">
          <h1 className="text-3xl font-heading font-bold text-neutral-900">Heading 1</h1>
          <h2 className="text-2xl font-heading font-semibold text-neutral-800">Heading 2</h2>
          <h3 className="text-xl font-heading font-medium text-neutral-800">Heading 3</h3>
          <p className="font-body text-neutral-600">This is standard body text using the body font. Text is neutral-600.</p>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">2. Buttons</h2>
        <div className="flex gap-4 flex-wrap items-center">
          <Button variant="primary">Primary Button</Button>
          <Button variant="secondary">Secondary Button</Button>
          <Button variant="danger">Danger Button</Button>
          <Button variant="ghost">Ghost Button</Button>
          <Button variant="primary" size="sm">Small Primary</Button>
          <Button variant="primary" disabled>Disabled Button</Button>
          <Button variant="primary" isLoading>Loading...</Button>
          <Button variant="secondary" isLoading>Saving...</Button>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">3. Badges (Pastel Icon Style)</h2>
        <div className="flex gap-3 flex-wrap items-center">
          <Badge variant="published">Published</Badge>
          <Badge variant="approved">Approved</Badge>
          <Badge variant="warning">Pending</Badge>
          <Badge variant="danger">Failed</Badge>
          <Badge variant="draft">Draft</Badge>
          <Badge variant="archived">Archived</Badge>
          <Badge variant="promo">Promo</Badge>
          <Badge variant="neutral">Neutral</Badge>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">3.1 Pills (Tags/Filters)</h2>
        <div className="flex gap-4 flex-wrap">
          <Pill label="All Hotels" />
          <Pill label="Makkah Only" variant="primary" onRemove={() => {}} />
          <Pill label="Active" variant="success" onRemove={() => {}} />
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">4. Alerts</h2>
        <div className="space-y-4">
          <Alert variant="success">This is a success alert!</Alert>
          <Alert variant="error">This is an error alert!</Alert>
          <Alert variant="warning">This is a warning alert!</Alert>
          <Alert variant="info">This is an info alert!</Alert>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">5. Form Inputs</h2>
        <div className="max-w-md space-y-4">
          <FormField label="Standard Input">
            <Input placeholder="Enter something..." />
          </FormField>
          
          <FormField label="Input with Error" error="This field is required.">
            <Input placeholder="Error input" className="border-danger-500" />
          </FormField>
          
          <CustomDropdown 
            label="Custom Dropdown"
            options={[
              { label: 'Hotel Makkah (Bintang 5)', value: 'makkah_5' },
              { label: 'Hotel Madinah (Bintang 4)', value: 'madinah_4' },
              { label: 'Paket Reguler', value: 'paket_reguler' }
            ]}
            placeholder="Pilih opsi..."
          />

          <FormField label="Textarea">
            <Textarea placeholder="Type your message here..." rows={3} />
          </FormField>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">6. Data Table (with Search & Pagination)</h2>
        <DataTable 
          columns={[
            { header: 'ID', key: 'id' },
            { header: 'Nama', key: 'name' },
            { header: 'Kategori', key: 'category' },
            { header: 'Status', key: 'status' }
          ]}
          data={[
            { id: 'TRX-001', name: 'Paket Umrah Makkah', category: 'Layanan', status: 'Active' },
            { id: 'TRX-002', name: 'Hotel Pullman Zamzam', category: 'Akomodasi', status: 'Draft' },
            { id: 'TRX-003', name: 'Tiket Saudia Airlines', category: 'Transportasi', status: 'Active' },
            { id: 'TRX-004', name: 'Makan Malam Tambahan', category: 'Konsumsi', status: 'Archived' },
            { id: 'TRX-005', name: 'Visa Umrah Multiple', category: 'Dokumen', status: 'Active' },
            { id: 'TRX-006', name: 'Asuransi Perjalanan', category: 'Layanan', status: 'Active' },
            { id: 'TRX-007', name: 'Bus Saptco VIP', category: 'Transportasi', status: 'Draft' },
          ]}
          itemsPerPage={5}
          renderCell={(row, key) => {
            if (key === 'status') {
              return <Badge variant={row.status.toLowerCase()}>{row.status}</Badge>;
            }
            return row[key];
          }}
        />
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">7. Loading & Empty State</h2>
        <div className="space-y-8">
          <div className="flex justify-center p-8 border border-dashed rounded">
            <LoadingSpinner />
          </div>
          
          <div className="border border-dashed rounded">
            <EmptyState 
              title="No Data Found" 
              message="Get started by creating a new record." 
              action={<Button>Create New</Button>}
            />
          </div>
        </div>
      </section>

      <section className="space-y-4 pb-20">
        <h2 className="text-xl font-bold font-heading border-b pb-2">8. Modal / Dialog</h2>
        <div>
          <Button onClick={() => setIsModalOpen(true)}>Buka Modal</Button>
          
          <Modal 
            isOpen={isModalOpen} 
            onClose={() => setIsModalOpen(false)}
            title="Konfirmasi Hapus Data"
            footer={
              <>
                <Button variant="ghost" onClick={() => setIsModalOpen(false)}>Batal</Button>
                <Button variant="danger" onClick={() => setIsModalOpen(false)}>Ya, Hapus</Button>
              </>
            }
          >
            <p className="text-neutral-600 font-body">
              Apakah Anda yakin ingin menghapus data ini? Tindakan ini tidak dapat dibatalkan dan semua data terkait akan ikut terhapus dari sistem.
            </p>
          </Modal>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-bold font-heading border-b pb-2">7. Cards</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <div className="p-4">
              <h3 className="font-bold text-lg mb-2">Basic Card</h3>
              <p className="text-neutral-600">This is content inside a card component.</p>
            </div>
          </Card>
        </div>
      </section>
    </div>
  );
};

export default DesignSystemPage;

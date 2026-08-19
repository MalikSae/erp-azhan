import React from 'react';
import PageHeader from '../components/ui/PageHeader';
import Card from '../components/ui/Card';

const DashboardHome = () => {
  return (
    <div>
      <PageHeader title="Selamat datang" />
      <Card>
        <p className="text-neutral-600 font-body text-base">
          Pilih menu di samping untuk mengelola master data.
        </p>
      </Card>
    </div>
  );
};

export default DashboardHome;

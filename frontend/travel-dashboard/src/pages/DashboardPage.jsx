import PageHeader from "../components/ui/PageHeader";
import Card from "../components/ui/Card";

const DashboardPage = () => {
  return (
    <div className="space-y-6">
      <PageHeader title="Selamat Datang" />
      <Card className="p-6">
        <p className="text-neutral-600">
          Gunakan menu di samping untuk mengelola jamaah dan booking paket umroh Anda.
        </p>
      </Card>
    </div>
  );
};

export default DashboardPage;

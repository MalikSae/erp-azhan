import React from 'react';
import { useNavigate } from 'react-router-dom';

const UrgentPackagesBanner = ({ packages }) => {
  const navigate = useNavigate();

  if (!packages || packages.length === 0) {
    return null;
  }

  const displayedPackages = packages.slice(0, 5);
  const remainingCount = packages.length - 5;

  return (
    <div className="mb-6 space-y-3">
      {displayedPackages.map((pkg) => {
        return (
          <div 
            key={pkg.id} 
            className="flex items-center justify-between bg-danger-50 border-l-4 border-danger-500 p-4 rounded-r-md"
          >
            <div className="flex items-center gap-4 w-full">
              <div className="flex flex-col items-center justify-center bg-white border border-danger-200 rounded-lg min-w-[64px] h-14 shrink-0 shadow-sm px-1">
                <span className="text-danger-700 font-heading font-bold text-xl leading-none">
                  {pkg.daysRemaining}
                </span>
                <span className="text-danger-500 text-[9px] font-bold uppercase mt-1 text-center whitespace-nowrap">
                  HARI LAGI
                </span>
              </div>
              <div className="flex-1 min-w-0 pr-4">
                <h3 className="text-danger-800 font-heading font-bold text-base truncate">
                  {pkg.jadwal_nama} <span className="text-sm font-normal text-danger-600 ml-1">({pkg.brand_name})</span>
                </h3>
                
                {/* Progress Bar (SeatProgressBar Style) */}
                {pkg.seat_total > 0 && (
                  <div className="space-y-1.5 max-w-sm mt-1.5">
                    <div className="flex justify-between items-center text-xs font-medium">
                      <span className="text-neutral-600">{pkg.seat_total - pkg.seat_sisa} pax terisi</span>
                      <span className="px-2 py-0.5 rounded-full border text-[11px] font-semibold text-danger-800 bg-danger-50 border-danger-200">
                        Kurang {pkg.seat_sisa} pax lagi!
                      </span>
                    </div>
                    <div className="w-full h-2.5 bg-neutral-100 rounded-full overflow-hidden border border-neutral-200">
                      <div 
                        className="h-full bg-danger-500 transition-all duration-300 animate-progress-stripes" 
                        style={{ width: `${Math.max(0, Math.min(100, ((pkg.seat_total - pkg.seat_sisa) / pkg.seat_total) * 100))}%` }}
                      />
                    </div>
                  </div>
                )}
              </div>
            </div>
            
            <button 
              onClick={() => navigate(`/schedules/${pkg.id}/edit`)}
              className="shrink-0 ml-4 px-4 py-2 bg-white border border-danger-200 text-danger-700 font-semibold text-sm rounded-md hover:bg-danger-100 transition-colors"
            >
              Buka Paket
            </button>
          </div>
        );
      })}
      
      {remainingCount > 0 && (
        <div className="text-sm text-danger-700 italic mt-2 px-1">
          +{remainingCount} paket lainnya menunggu...
        </div>
      )}
    </div>
  );
};

export default UrgentPackagesBanner;

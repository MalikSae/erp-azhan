import React, { useState, useMemo } from 'react';
import Table from './Table';
import Input from './Input';
import Button from './Button';

const DataTable = ({ 
  columns, 
  data = [], 
  searchable = true,
  searchPlaceholder = "Cari data...",
  pagination = true,
  itemsPerPage = 10,
  emptyMessage = "Tidak ada data",
  renderCell,
  toolbarActions,
  onRowClick
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [sortConfig, setSortConfig] = useState(null);

  const handleSort = (key) => {
    let direction = 'asc';
    if (sortConfig && sortConfig.key === key && sortConfig.direction === 'asc') {
      direction = 'desc';
    }
    setSortConfig({ key, direction });
  };

  // Filter and sort data
  const filteredData = useMemo(() => {
    let result = data;
    
    if (searchQuery) {
      const lowerQuery = searchQuery.toLowerCase();
      result = result.filter(row => {
        return Object.values(row).some(value => {
          if (value === null || value === undefined) return false;
          return String(value).toLowerCase().includes(lowerQuery);
        });
      });
    }

    if (sortConfig) {
      const { key, direction } = sortConfig;
      const col = columns.find(c => c.key === key);
      if (col && col.sortable) {
        result = [...result].sort((a, b) => {
          let compareResult = 0;
          if (col.sortFn) {
            compareResult = col.sortFn(a, b);
          } else {
            const valA = a[key] ?? '';
            const valB = b[key] ?? '';
            if (valA < valB) compareResult = -1;
            if (valA > valB) compareResult = 1;
          }
          return direction === 'asc' ? compareResult : -compareResult;
        });
      }
    }
    
    return result;
  }, [data, searchQuery, sortConfig, columns]);

  // Reset to first page when search changes
  React.useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, data.length, itemsPerPage]);

  // Paginate data
  const totalPages = Math.ceil(filteredData.length / itemsPerPage);
  const paginatedData = useMemo(() => {
    if (!pagination) return filteredData;
    const startIndex = (currentPage - 1) * itemsPerPage;
    return filteredData.slice(startIndex, startIndex + itemsPerPage);
  }, [filteredData, pagination, currentPage, itemsPerPage]);

  return (
    <div className="w-full bg-white rounded-2xl border border-neutral-200/80 shadow-card overflow-hidden">
      {/* Header / Toolbar */}
      {(searchable || toolbarActions) && (
        <div className="p-4 border-b border-neutral-200/80 flex flex-col sm:flex-row justify-between items-start sm:items-center bg-white gap-4">
          <div className="w-full max-w-sm relative">
            {searchable && (
              <>
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <svg className="h-4 w-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </div>
                <input
                  type="text"
                  placeholder={searchPlaceholder}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full rounded-xl border border-neutral-200/90 pl-10 pr-3.5 py-2 text-xs md:text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 shadow-2xs font-body bg-white transition-all"
                />
              </>
            )}
          </div>
          {toolbarActions && (
            <div className="flex items-center gap-3 w-full sm:w-auto">
              {toolbarActions}
            </div>
          )}
        </div>
      )}

      {/* Table Content */}
      <Table 
        columns={columns} 
        data={paginatedData} 
        emptyMessage={searchQuery ? "Data tidak ditemukan" : emptyMessage} 
        renderCell={renderCell}
        sortConfig={sortConfig}
        onSort={handleSort}
        onRowClick={onRowClick}
      />

      {/* Footer / Pagination */}
      {pagination && filteredData.length > 0 && (
        <div className="p-4 border-t border-neutral-200/80 flex flex-col sm:flex-row items-center justify-between bg-white gap-4">
          <div className="text-xs md:text-sm text-neutral-500 font-body text-center sm:text-left">
            Menampilkan <span className="font-semibold text-neutral-900">{(currentPage - 1) * itemsPerPage + 1}</span> hingga <span className="font-semibold text-neutral-900">{Math.min(currentPage * itemsPerPage, filteredData.length)}</span> dari <span className="font-semibold text-neutral-900">{filteredData.length}</span> entri
          </div>
          <div className="flex gap-1.5 items-center">
            <Button 
              variant="secondary" 
              size="sm" 
              onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
              disabled={currentPage === 1}
            >
              Sebelumnya
            </Button>
            
            <div className="hidden sm:flex gap-1 mx-1 items-center">
              {(() => {
                const getPageNumbers = () => {
                  if (totalPages <= 7) {
                    return Array.from({ length: totalPages }, (_, i) => i + 1);
                  }
                  if (currentPage <= 3) {
                    return [1, 2, 3, '...', totalPages];
                  }
                  if (currentPage >= totalPages - 2) {
                    return [1, '...', totalPages - 2, totalPages - 1, totalPages];
                  }
                  return [1, '...', currentPage - 1, currentPage, currentPage + 1, '...', totalPages];
                };

                return getPageNumbers().map((page, index) => {
                  if (page === '...') {
                    return (
                      <span key={`ellipsis-${index}`} className="px-2 text-neutral-400 text-xs font-bold">
                        ...
                      </span>
                    );
                  }
                  return (
                    <Button
                      key={page}
                      variant={currentPage === page ? "dark" : "ghost"}
                      size="sm"
                      onClick={() => setCurrentPage(page)}
                      className={`w-8 h-8 p-0 text-xs ${currentPage === page ? 'shadow-2xs font-bold' : 'text-neutral-600 hover:bg-neutral-100'}`}
                    >
                      {page}
                    </Button>
                  );
                });
              })()}
            </div>

            <Button 
              variant="secondary" 
              size="sm" 
              onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
              disabled={currentPage === totalPages}
            >
              Selanjutnya
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

export default DataTable;

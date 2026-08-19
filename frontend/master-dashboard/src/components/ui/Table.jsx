import React from 'react';
import EmptyState from './EmptyState';

const Table = ({ columns, data = [], emptyMessage = "Tidak ada data", renderCell, sortConfig, onSort }) => {
  if (!data || data.length === 0) {
    return <EmptyState message={emptyMessage} />;
  }

  return (
    <div className="overflow-x-auto w-full">
      <table className="w-full text-left text-sm font-body text-neutral-900 border-collapse min-w-[800px]">
        <thead className="bg-neutral-50 text-neutral-600 font-medium border-b border-neutral-200">
          <tr>
            {columns.map((col, idx) => (
              <th 
                key={idx} 
                className={`px-4 py-3 whitespace-nowrap ${col.sortable ? 'cursor-pointer select-none hover:bg-neutral-100' : ''}`}
                onClick={() => col.sortable && onSort && onSort(col.key)}
              >
                <div className="flex items-center gap-1">
                  {col.header}
                  {col.sortable && (
                    <span className="inline-flex flex-col w-3">
                      <svg className={`w-3 h-3 -mb-1 ${sortConfig?.key === col.key && sortConfig.direction === 'asc' ? 'text-primary-600 font-bold' : 'text-neutral-300'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 15l7-7 7 7" />
                      </svg>
                      <svg className={`w-3 h-3 -mt-1 ${sortConfig?.key === col.key && sortConfig.direction === 'desc' ? 'text-primary-600 font-bold' : 'text-neutral-300'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </span>
                  )}
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((row, rowIndex) => (
            <tr key={rowIndex} className="border-b border-neutral-100 hover:bg-neutral-50 transition-colors">
              {columns.map((col, colIndex) => (
                <td key={colIndex} className="px-4 py-4 whitespace-nowrap">
                  {renderCell ? renderCell(row, col.key) : row[col.key]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default Table;

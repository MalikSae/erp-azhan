import React from 'react';
import EmptyState from './EmptyState';

const Table = ({ columns, data = [], emptyMessage = "Tidak ada data", renderCell, sortConfig, onSort, onRowClick }) => {
  if (!data || data.length === 0) {
    return <EmptyState message={emptyMessage} />;
  }

  const getCellContent = (row, col, rowIndex) => {
    if (renderCell) {
      return renderCell(row, col.key || col.accessor, rowIndex);
    }
    if (typeof col.accessor === 'function') {
      return col.accessor(row, rowIndex);
    }
    if (typeof col.accessor === 'string') {
      return row[col.accessor];
    }
    if (col.key) {
      return row[col.key];
    }
    return null;
  };

  return (
    <div className="overflow-x-auto w-full">
      <table className="w-full text-left text-xs md:text-sm font-body text-neutral-900 border-collapse min-w-full">
        <thead className="bg-neutral-50/80 text-neutral-500 font-heading text-[11px] font-bold uppercase tracking-wider border-b border-neutral-200/80">
          <tr>
            {columns.map((col, idx) => (
              <th 
                key={idx} 
                className={`px-4 py-3.5 whitespace-nowrap ${col.align === 'center' ? 'text-center' : col.align === 'right' ? 'text-right' : ''} ${col.sortable ? 'cursor-pointer select-none hover:bg-neutral-100/70' : ''}`}
                onClick={() => col.sortable && onSort && onSort(col.key)}
              >
                <div className={`flex items-center gap-1.5 ${col.align === 'center' ? 'justify-center' : col.align === 'right' ? 'justify-end' : ''}`}>
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
        <tbody className="divide-y divide-neutral-100">
          {data.map((row, rowIndex) => (
            <tr 
              key={rowIndex} 
              onClick={() => onRowClick && onRowClick(row, rowIndex)}
              className={`hover:bg-neutral-50/70 transition-colors ${onRowClick ? 'cursor-pointer' : ''}`}
            >
              {columns.map((col, colIndex) => (
                <td key={colIndex} className={`px-4 py-4 whitespace-nowrap ${col.align === 'center' ? 'text-center' : col.align === 'right' ? 'text-right' : ''}`}>
                  {getCellContent(row, col, rowIndex)}
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

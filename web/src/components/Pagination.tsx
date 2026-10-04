import React from 'react';

export interface PaginationProps {
    currentPage: number;
    totalPages: number;
    totalItems: number;
    pageSize: number;
    pageSizeOptions?: number[];
    onPageChange: (page: number) => void;
    onPageSizeChange: (size: number) => void;
    itemName?: string;
}

export const Pagination: React.FC<PaginationProps> = ({
    currentPage,
    totalPages,
    totalItems,
    pageSize,
    pageSizeOptions = [10, 25, 50, 100],
    onPageChange,
    onPageSizeChange,
    itemName = 'entries'
}) => {
    if (totalItems === 0) {
        return null;
    }

    const startItem = Math.min((currentPage - 1) * pageSize + 1, totalItems);
    const endItem = Math.min(currentPage * pageSize, totalItems);

    // Calculate page numbers to display with smart ellipsis
    const getPageNumbers = () => {
        const pages: (number | string)[] = [];
        if (totalPages <= 7) {
            for (let i = 1; i <= totalPages; i++) {
                pages.push(i);
            }
        } else {
            pages.push(1);
            if (currentPage > 3) {
                pages.push('...');
            }

            const start = Math.max(2, currentPage - 1);
            const end = Math.min(totalPages - 1, currentPage + 1);

            for (let i = start; i <= end; i++) {
                pages.push(i);
            }

            if (currentPage < totalPages - 2) {
                pages.push('...');
            }
            pages.push(totalPages);
        }
        return pages;
    };

    return (
        <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
            gap: '16px',
            padding: '14px 20px',
            background: '#ffffff',
            borderTop: '1px solid #e2e8f0',
            fontSize: '13px',
            color: '#64748b'
        }}>
            {/* Left: Page Size Selector & Count Info */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '14px', flexWrap: 'wrap' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span>Show</span>
                    <select
                        value={pageSize}
                        onChange={e => onPageSizeChange(Number(e.target.value))}
                        style={{
                            padding: '4px 8px',
                            borderRadius: '6px',
                            border: '1px solid #cbd5e1',
                            background: '#f8fafc',
                            fontSize: '13px',
                            fontWeight: 600,
                            color: '#0f172a',
                            outline: 'none',
                            cursor: 'pointer'
                        }}
                    >
                        {pageSizeOptions.map(opt => (
                            <option key={opt} value={opt}>
                                {opt}
                            </option>
                        ))}
                    </select>
                    <span>{itemName} per page</span>
                </div>

                <div style={{ color: '#94a3b8' }}>|</div>

                <div>
                    Showing <strong style={{ color: '#0f172a' }}>{startItem}</strong> to{' '}
                    <strong style={{ color: '#0f172a' }}>{endItem}</strong> of{' '}
                    <strong style={{ color: '#0f172a' }}>{totalItems.toLocaleString()}</strong> {itemName}
                </div>
            </div>

            {/* Right: Page Buttons */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                {/* Previous Button */}
                <button
                    type="button"
                    onClick={() => onPageChange(currentPage - 1)}
                    disabled={currentPage <= 1}
                    style={{
                        padding: '5px 12px',
                        borderRadius: '6px',
                        border: '1px solid #e2e8f0',
                        background: currentPage <= 1 ? '#f8fafc' : '#ffffff',
                        color: currentPage <= 1 ? '#94a3b8' : '#334155',
                        fontWeight: 600,
                        fontSize: '12.5px',
                        cursor: currentPage <= 1 ? 'not-allowed' : 'pointer',
                        transition: 'all 120ms'
                    }}
                >
                    &larr; Previous
                </button>

                {/* Page Number Buttons */}
                {getPageNumbers().map((p, idx) => {
                    if (p === '...') {
                        return (
                            <span key={`ellipsis-${idx}`} style={{ padding: '0 4px', color: '#94a3b8' }}>
                                &hellip;
                            </span>
                        );
                    }

                    const pageNum = p as number;
                    const isActive = pageNum === currentPage;

                    return (
                        <button
                            key={pageNum}
                            type="button"
                            onClick={() => onPageChange(pageNum)}
                            style={{
                                minWidth: '32px',
                                height: '32px',
                                padding: '0 8px',
                                borderRadius: '6px',
                                border: isActive ? '1px solid #2563eb' : '1px solid #e2e8f0',
                                background: isActive ? '#2563eb' : '#ffffff',
                                color: isActive ? '#ffffff' : '#334155',
                                fontWeight: isActive ? 700 : 500,
                                fontSize: '12.5px',
                                cursor: 'pointer',
                                transition: 'all 120ms'
                            }}
                        >
                            {pageNum}
                        </button>
                    );
                })}

                {/* Next Button */}
                <button
                    type="button"
                    onClick={() => onPageChange(currentPage + 1)}
                    disabled={currentPage >= totalPages}
                    style={{
                        padding: '5px 12px',
                        borderRadius: '6px',
                        border: '1px solid #e2e8f0',
                        background: currentPage >= totalPages ? '#f8fafc' : '#ffffff',
                        color: currentPage >= totalPages ? '#94a3b8' : '#334155',
                        fontWeight: 600,
                        fontSize: '12.5px',
                        cursor: currentPage >= totalPages ? 'not-allowed' : 'pointer',
                        transition: 'all 120ms'
                    }}
                >
                    Next &rarr;
                </button>
            </div>
        </div>
    );
};

export default Pagination;

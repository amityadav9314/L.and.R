import React from 'react';
import type { LucideIcon } from 'lucide-react';

interface PageHeaderProps {
    title: string;
    subtitle?: string;
    icon?: LucideIcon;
    actions?: React.ReactNode;
}

export const PageHeader: React.FC<PageHeaderProps> = ({ title, subtitle, icon: Icon, actions }) => {
    return (
        <div className="d-flex justify-content-between align-items-center mb-4">
            <div className="d-flex align-items-center gap-3">
                {Icon && (
                    <div className="text-primary">
                        <Icon size={32} />
                    </div>
                )}
                <div>
                    <h2 className="mb-0 fw-bold h3">{title}</h2>
                    {subtitle && <p className="text-muted mb-0">{subtitle}</p>}
                </div>
            </div>
            {actions && <div>{actions}</div>}
        </div>
    );
};

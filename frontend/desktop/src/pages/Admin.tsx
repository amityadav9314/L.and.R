import { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { Nav } from 'react-bootstrap';
import { Users, Settings } from 'lucide-react';
import { useAuthStore } from '../store/authStore.ts';
import { PageHeader } from '../components/PageHeader.tsx';

// Import the actual content components
import AdminUsersContent from './AdminUsers.tsx';
import AdminSettingsContent from './AdminSettings.tsx';

type TabKey = 'users' | 'settings';

const Admin = () => {
    const { user: currentUser } = useAuthStore();
    const [activeTab, setActiveTab] = useState<TabKey>('users');

    // Redirect if not admin
    if (!currentUser?.isAdmin) {
        return <Navigate to="/" replace />;
    }

    return (
        <div className="d-flex flex-column flex-grow-1">
            <PageHeader
                title="Admin Panel"
                subtitle="Manage users and application settings"
                icon={Settings}
            />

            <Nav variant="tabs" className="mb-4">
                <Nav.Item>
                    <Nav.Link
                        active={activeTab === 'users'}
                        onClick={() => setActiveTab('users')}
                        className="d-flex align-items-center gap-2"
                        style={{ cursor: 'pointer' }}
                    >
                        <Users size={18} />
                        Users
                    </Nav.Link>
                </Nav.Item>
                <Nav.Item>
                    <Nav.Link
                        active={activeTab === 'settings'}
                        onClick={() => setActiveTab('settings')}
                        className="d-flex align-items-center gap-2"
                        style={{ cursor: 'pointer' }}
                    >
                        <Settings size={18} />
                        Settings
                    </Nav.Link>
                </Nav.Item>
            </Nav>

            {activeTab === 'users' && <AdminUsersContent />}
            {activeTab === 'settings' && <AdminSettingsContent />}
        </div>
    );
};

export default Admin;

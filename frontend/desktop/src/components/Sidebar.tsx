import { useQuery } from '@tanstack/react-query';
import { Nav, Badge } from 'react-bootstrap';
import { NavLink } from 'react-router-dom';
import { Home, Database, Newspaper, Settings, LogOut, Shield } from 'lucide-react';
import { useAuthStore } from '../store/authStore.ts';
import { learningClient } from '../services/api.ts';

const Sidebar = () => {
    const { logout, user } = useAuthStore();

    // Fetch notification status for due count
    const { data: notificationStatus } = useQuery({
        queryKey: ['notificationStatus'],
        queryFn: () => learningClient.getNotificationStatus({}),
        refetchInterval: 60000 // Refresh every minute
    });

    const navItems = [
        { name: 'Due for Revise', path: '/', icon: Home, count: notificationStatus?.dueFlashcardsCount },
        { name: 'Vault', path: '/vault', icon: Database },
        { name: 'Daily Feed', path: '/feed', icon: Newspaper },
        { name: 'Settings', path: '/settings', icon: Settings },
        // Admin menu item - only shown if user is admin
        ...(user?.isAdmin ? [{ name: 'Admin', path: '/admin/users', icon: Shield }] : []),
    ];

    return (
        <div className="d-flex flex-column flex-shrink-0 p-3 bg-white border-end shadow-sm" style={{ width: '280px', height: '100vh', position: 'fixed' }}>
            <NavLink to="/" className="d-flex align-items-center mb-4 mb-md-0 me-md-auto link-dark text-decoration-none gap-2">
                <img src="/logo.png" alt="L.and.R" width="40" height="40" className="rounded" />
                <div className="d-flex flex-column">
                    <span className="fs-5 fw-bold text-primary">L.and.R</span>
                    <span className="text-secondary" style={{ fontSize: '0.7rem' }}>Desktop</span>
                </div>
            </NavLink>
            <hr />
            <Nav variant="pills" className="flex-column mb-auto">
                {navItems.map((item) => (
                    <Nav.Item key={item.name} className="mb-1">
                        <NavLink
                            to={item.path}
                            className={({ isActive }) =>
                                `d-flex align-items-center justify-content-between gap-3 py-2 px-3 rounded-3 text-decoration-none transition-all ${isActive ? 'bg-primary text-white shadow-sm' : 'text-dark hover-bg-light'}`
                            }
                        >
                            <div className="d-flex align-items-center gap-3">
                                <item.icon size={20} />
                                <span>{item.name}</span>
                            </div>
                            {item.count !== undefined && item.count > 0 && (
                                <Badge bg="danger" className="rounded-pill">{item.count}</Badge>
                            )}
                        </NavLink>
                    </Nav.Item>
                ))}
            </Nav>
            <hr />
            <div className="d-flex flex-column gap-2">
                <div className="d-flex align-items-center gap-2 px-2">
                    <img src={user?.picture || 'https://via.placeholder.com/32'} alt="" width="40" height="40" className="rounded-circle" />
                    <div className="d-flex flex-column text-truncate flex-grow-1">
                        <strong className="text-truncate small">{user?.name}</strong>
                        <span className="small text-muted text-truncate" style={{ fontSize: '0.75rem' }}>{user?.email}</span>
                    </div>
                </div>
                <button
                    className="btn btn-outline-danger btn-sm d-flex align-items-center justify-content-center gap-2 w-100"
                    onClick={() => logout()}
                >
                    <LogOut size={16} />
                    <span>Sign Out</span>
                </button>
            </div>
        </div>
    );
};

export default Sidebar;

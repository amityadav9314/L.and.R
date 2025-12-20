import { Nav } from 'react-bootstrap';
import { NavLink } from 'react-router-dom';
import { Home, Database, Newspaper, Settings, LogOut } from 'lucide-react';
import { useAuthStore } from '../store/authStore.ts';

const Sidebar = () => {
    const { logout, user } = useAuthStore();

    const navItems = [
        { name: 'Due for Revise', path: '/', icon: Home },
        { name: 'Vault', path: '/vault', icon: Database },
        { name: 'Daily Feed', path: '/feed', icon: Newspaper },
        { name: 'Settings', path: '/settings', icon: Settings },
    ];

    return (
        <div className="d-flex flex-column flex-shrink-0 p-3 bg-white border-end shadow-sm" style={{ width: '280px', height: '100vh', position: 'fixed' }}>
            <NavLink to="/" className="d-flex align-items-center mb-4 mb-md-0 me-md-auto link-dark text-decoration-none">
                <span className="fs-4 fw-bold text-primary">L.and.R <span className="text-secondary small">Desktop</span></span>
            </NavLink>
            <hr />
            <Nav variant="pills" className="flex-column mb-auto">
                {navItems.map((item) => (
                    <Nav.Item key={item.name} className="mb-1">
                        <NavLink
                            to={item.path}
                            className={({ isActive }) =>
                                `d-flex align-items-center gap-3 py-2 px-3 rounded-3 text-decoration-none transition-all ${isActive ? 'bg-primary text-white shadow-sm' : 'text-dark hover-bg-light'}`
                            }
                        >
                            <item.icon size={20} />
                            <span>{item.name}</span>
                        </NavLink>
                    </Nav.Item>
                ))}
            </Nav>
            <hr />
            <div className="dropdown">
                <div
                    className="d-flex align-items-center link-dark text-decoration-none dropdown-toggle cursor-pointer"
                    id="dropdownUser2"
                    data-bs-toggle="dropdown"
                    aria-expanded="false"
                >
                    <img src={user?.picture || 'https://via.placeholder.com/32'} alt="" width="32" height="32" className="rounded-circle me-2" />
                    <div className="d-flex flex-column text-truncate" style={{ maxWidth: '180px' }}>
                        <strong className="text-truncate">{user?.name}</strong>
                        <span className="small text-muted text-truncate">{user?.email}</span>
                    </div>
                </div>
                <ul className="dropdown-menu text-small shadow" aria-labelledby="dropdownUser2">
                    <li><a className="dropdown-item d-flex align-items-center gap-2" onClick={() => logout()} href="#"><LogOut size={16} /> Sign out</a></li>
                </ul>
            </div>
        </div>
    );
};

export default Sidebar;

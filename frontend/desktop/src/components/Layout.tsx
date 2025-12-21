import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar.tsx';

const Layout = () => {
    return (
        <div className="d-flex bg-light min-vh-100">
            <Sidebar />
            <main className="flex-grow-1 overflow-hidden" style={{ marginLeft: '280px', maxWidth: 'calc(100vw - 280px)' }}>
                <div className="container-fluid py-4 px-3 px-lg-4">
                    <Outlet />
                </div>
            </main>
        </div>
    );
};

export default Layout;

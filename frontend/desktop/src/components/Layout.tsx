import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar.tsx';

const Layout = () => {
    return (
        <div className="d-flex bg-light min-vh-100">
            <Sidebar />
            <main className="flex-grow-1" style={{ marginLeft: '280px' }}>
                <div className="container-fluid py-4 px-lg-5">
                    <Outlet />
                </div>
            </main>
        </div>
    );
};

export default Layout;

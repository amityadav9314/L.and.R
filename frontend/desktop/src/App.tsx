import { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from './store/authStore.ts';
import Login from './pages/Login.tsx';
import Dashboard from './pages/Dashboard.tsx';
import DailyFeed from './pages/DailyFeed.tsx';
import Settings from './pages/Settings.tsx';
import AdminUsers from './pages/AdminUsers.tsx';
import UpgradePage from './pages/UpgradePage.tsx';
import { TermsPage } from './pages/TermsPage.tsx';
import { PrivacyPage } from './pages/PrivacyPage.tsx';

import Layout from './components/Layout.tsx';

const queryClient = new QueryClient();

function App() {
  const { user, isLoading, restoreSession } = useAuthStore();

  useEffect(() => {
    restoreSession();
  }, [restoreSession]);

  if (isLoading) {
    return (
      <div className="d-flex justify-content-center align-items-center vh-100 bg-light">
        <div className="spinner-border text-primary" role="status">
          <span className="visually-hidden">Loading...</span>
        </div>
      </div>
    );
  }

  return (
    <QueryClientProvider client={queryClient}>
      <Router>
        <Routes>
          <Route path="/login" element={!user ? <Login /> : <Navigate to="/" />} />
          <Route path="/" element={user ? <Layout /> : <Navigate to="/login" />}>
            <Route index element={<Dashboard />} />
            <Route path="vault" element={<Dashboard />} />
            <Route path="feed" element={<DailyFeed />} />
            <Route path="settings" element={<Settings />} />
            <Route path="admin/users" element={<AdminUsers />} />
            <Route path="upgrade" element={<UpgradePage />} />
            <Route path="terms" element={<TermsPage />} />
            <Route path="privacy" element={<PrivacyPage />} />

          </Route>
        </Routes>
      </Router>
    </QueryClientProvider>
  );
}

export default App;

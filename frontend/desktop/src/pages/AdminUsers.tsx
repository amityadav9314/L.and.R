import { useState, useEffect, useCallback } from 'react';
import { Navigate } from 'react-router-dom';
import { Table, Card, Badge, Spinner, Alert, Button } from 'react-bootstrap';
import { Users, Shield, ShieldOff, ChevronLeft, ChevronRight } from 'lucide-react';
import { useAuthStore } from '../store/authStore.ts';
import { API_URL } from '../utils/config.ts';

interface User {
    id: string;
    email: string;
    name: string;
    picture: string;
    is_admin: boolean;
    created_at: string;
    material_count: number;
}

interface PaginatedResponse {
    users: User[];
    page: number;
    page_size: number;
    total_count: number;
    total_pages: number;
}

const PAGE_SIZE = 10;

const AdminUsers = () => {
    const { user, token } = useAuthStore();
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [page, setPage] = useState(1);
    const [totalCount, setTotalCount] = useState(0);
    const [totalPages, setTotalPages] = useState(0);

    const fetchUsers = useCallback(async (pageNum: number) => {
        if (!token) return;

        setLoading(true);
        try {
            const response = await fetch(`${API_URL}/api/admin/users?page=${pageNum}&page_size=${PAGE_SIZE}`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                },
            });

            if (response.status === 403) {
                setError('Access denied. Admin privileges required.');
                return;
            }

            if (!response.ok) {
                throw new Error('Failed to fetch users');
            }

            const data: PaginatedResponse = await response.json();
            setUsers(data.users || []);
            setPage(data.page);
            setTotalCount(data.total_count);
            setTotalPages(data.total_pages);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An error occurred');
        } finally {
            setLoading(false);
        }
    }, [token]);

    useEffect(() => {
        fetchUsers(1);
    }, [fetchUsers]);

    const handlePrev = () => {
        if (page > 1) {
            fetchUsers(page - 1);
        }
    };

    const handleNext = () => {
        if (page < totalPages) {
            fetchUsers(page + 1);
        }
    };

    // Redirect if not admin
    if (!user?.isAdmin) {
        return <Navigate to="/" replace />;
    }

    if (error) {
        return (
            <Alert variant="danger" className="m-4">
                <Alert.Heading>Error</Alert.Heading>
                <p>{error}</p>
            </Alert>
        );
    }

    return (
        <div className="container-lg">
            <div className="d-flex align-items-center gap-3 mb-4">
                <Users size={32} className="text-primary" />
                <div>
                    <h2 className="mb-0 fw-bold">User Management</h2>
                    <p className="text-muted mb-0">View and manage all registered users</p>
                </div>
            </div>

            <Card className="shadow-sm border-0">
                <Card.Header className="bg-white border-bottom py-3">
                    <div className="d-flex justify-content-between align-items-center">
                        <span className="fw-semibold">All Users</span>
                        <Badge bg="primary" className="rounded-pill px-3 py-2">
                            {totalCount} users
                        </Badge>
                    </div>
                </Card.Header>
                <Card.Body className="p-0">
                    {loading ? (
                        <div className="d-flex justify-content-center align-items-center py-5">
                            <Spinner animation="border" variant="primary" />
                        </div>
                    ) : (
                        <Table hover responsive className="mb-0">
                            <thead className="bg-light">
                                <tr>
                                    <th className="border-0 py-3 ps-4">User</th>
                                    <th className="border-0 py-3">Email</th>
                                    <th className="border-0 py-3">Joined</th>
                                    <th className="border-0 py-3 text-center">Materials</th>
                                    <th className="border-0 py-3 text-center">Role</th>
                                </tr>
                            </thead>
                            <tbody>
                                {users.map((u) => (
                                    <tr key={u.id}>
                                        <td className="py-3 ps-4">
                                            <div className="d-flex align-items-center gap-3">
                                                <img
                                                    src={u.picture || 'https://via.placeholder.com/40'}
                                                    alt={u.name}
                                                    width="40"
                                                    height="40"
                                                    className="rounded-circle"
                                                />
                                                <span className="fw-medium">{u.name}</span>
                                            </div>
                                        </td>
                                        <td className="py-3 text-muted">{u.email}</td>
                                        <td className="py-3 text-muted">
                                            {new Date(u.created_at).toLocaleDateString('en-IN', {
                                                year: 'numeric',
                                                month: 'short',
                                                day: 'numeric'
                                            })}
                                        </td>
                                        <td className="py-3 text-center">
                                            <Badge bg="info" className="rounded-pill px-2 py-1">
                                                {u.material_count}
                                            </Badge>
                                        </td>
                                        <td className="py-3 text-center">
                                            {u.is_admin ? (
                                                <Badge bg="success" className="d-inline-flex align-items-center gap-1 px-3 py-2">
                                                    <Shield size={14} />
                                                    Admin
                                                </Badge>
                                            ) : (
                                                <Badge bg="secondary" className="d-inline-flex align-items-center gap-1 px-3 py-2">
                                                    <ShieldOff size={14} />
                                                    User
                                                </Badge>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </Table>
                    )}
                </Card.Body>
                {totalPages > 1 && (
                    <Card.Footer className="bg-white border-top py-3">
                        <div className="d-flex justify-content-between align-items-center">
                            <span className="text-muted small">
                                Page {page} of {totalPages}
                            </span>
                            <div className="d-flex gap-2">
                                <Button
                                    variant="outline-primary"
                                    size="sm"
                                    onClick={handlePrev}
                                    disabled={page <= 1 || loading}
                                    className="d-flex align-items-center gap-1"
                                >
                                    <ChevronLeft size={16} />
                                    Previous
                                </Button>
                                <Button
                                    variant="outline-primary"
                                    size="sm"
                                    onClick={handleNext}
                                    disabled={page >= totalPages || loading}
                                    className="d-flex align-items-center gap-1"
                                >
                                    Next
                                    <ChevronRight size={16} />
                                </Button>
                            </div>
                        </div>
                    </Card.Footer>
                )}
            </Card>

            <Card className="shadow-sm border-0 mt-4">
                <Card.Header className="bg-white border-bottom py-3">
                    <span className="fw-semibold">Admin Commands</span>
                </Card.Header>
                <Card.Body>
                    <p className="text-muted mb-2">To toggle a user's admin status, use the following curl command:</p>
                    <pre className="bg-dark text-light p-3 rounded" style={{ fontSize: '0.85rem' }}>
                        {`# Make user an admin
curl -X POST "${API_URL}/api/admin/set-admin?email=USER_EMAIL&is_admin=true" \\
  -H "X-API-Key: YOUR_FEED_API_KEY"

# Remove admin privileges
curl -X POST "${API_URL}/api/admin/set-admin?email=USER_EMAIL&is_admin=false" \\
  -H "X-API-Key: YOUR_FEED_API_KEY"`}
                    </pre>
                </Card.Body>
            </Card>
        </div>
    );
};

export default AdminUsers;

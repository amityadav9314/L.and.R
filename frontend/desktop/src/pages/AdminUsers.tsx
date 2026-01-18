import { useState, useEffect, useCallback } from 'react';
import { Navigate } from 'react-router-dom';
import { Table, Card, Badge, Spinner, Alert, Button, Dropdown, Modal, Form, InputGroup } from 'react-bootstrap';
import { Users, Shield, ShieldOff, ChevronLeft, ChevronRight, MoreVertical, Ban, CheckCircle, Crown, Search, X } from 'lucide-react';
import { useAuthStore } from '../store/authStore.ts';

import { API_URL } from '../utils/config.ts';
import { PageHeader } from '../components/PageHeader.tsx';

interface User {
    id: string;
    email: string;
    name: string;
    picture: string;
    is_admin: boolean;
    is_pro: boolean;
    is_blocked: boolean;
    created_at: string;
    material_count: number;
    current_period_end?: string;
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
    const { user: currentUser, token, login } = useAuthStore(); // Rename user to currentUser to avoid conflict
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [page, setPage] = useState(1);
    const [totalCount, setTotalCount] = useState(0);
    const [totalPages, setTotalPages] = useState(0);
    const [filterEmail, setFilterEmail] = useState('');
    const [searchInput, setSearchInput] = useState('');

    const fetchUsers = useCallback(async (pageNum: number) => {
        if (!token) return;

        setLoading(true);
        try {
            const query = new URLSearchParams({
                page: pageNum.toString(),
                page_size: PAGE_SIZE.toString(),
                email: filterEmail
            });
            const response = await fetch(`${API_URL}/api/admin/users?${query.toString()}`, {
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
    }, [token, filterEmail]);

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

    const [showProModal, setShowProModal] = useState(false);
    const [proDays, setProDays] = useState(30);
    const [selectedUser, setSelectedUser] = useState<User | null>(null);

    // Action handler
    const updateUserStatus = async (targetUser: User, action: 'pro' | 'block' | 'admin', value: boolean) => {
        if (!token) return;

        // If setting Pro status to TRUE, show modal instead of immediate update
        if (action === 'pro' && value === true) {
            setSelectedUser(targetUser);
            setProDays(30); // Default
            setShowProModal(true);
            return;
        }

        performUpdate(targetUser, action, value);
    };

    const confirmProStatus = async () => {
        if (!selectedUser) return;
        performUpdate(selectedUser, 'pro', true, proDays);
        setShowProModal(false);
    };

    const performUpdate = async (targetUser: User, action: 'pro' | 'block' | 'admin', value: boolean, days?: number) => {
        if (!token) return;
        // Optimistic update
        const originalUsers = [...users];
        setUsers(users.map(u => {
            if (u.id === targetUser.id) {
                if (action === 'pro') {
                    const expiry = new Date();
                    expiry.setDate(expiry.getDate() + (days || 30));
                    return { ...u, is_pro: value, current_period_end: value ? expiry.toISOString() : undefined };
                }
                if (action === 'block') return { ...u, is_blocked: value };
                if (action === 'admin') return { ...u, is_admin: value };
            }
            return u;
        }));

        // If updating self, update global auth store
        if (currentUser && targetUser.id === currentUser.id) {
            const updatedProfile = { ...currentUser };
            if (action === 'pro') updatedProfile.isPro = value;
            if (action === 'admin') updatedProfile.isAdmin = value;
            // Blocking self would logout, but let's handle prop update
            if (action === 'block') updatedProfile.isBlocked = value;

            login(updatedProfile, token!);
        }

        let endpoint = '';
        let queryParams = '';

        if (action === 'pro') {
            endpoint = '/api/admin/set-pro';
            queryParams = `?user_id=${targetUser.id}&is_pro=${value}`;
            if (value && days) {
                queryParams += `&days=${days}`;
            }
        } else if (action === 'block') {
            endpoint = '/api/admin/set-block';
            queryParams = `?email=${targetUser.email}&is_blocked=${value}`;
        } else if (action === 'admin') {
            endpoint = '/api/admin/set-admin';
            queryParams = `?email=${targetUser.email}&is_admin=${value}`;
        }

        try {
            const response = await fetch(`${API_URL}${endpoint}${queryParams}`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                },
            });

            if (!response.ok) {
                throw new Error('Action failed');
            }
        } catch (err) {
            // Revert on error
            console.error(err);
            setUsers(originalUsers);

            // Revert auth store if it was self
            if (currentUser && targetUser.id === currentUser.id) {
                login(currentUser, token!);
            }

            alert('Failed to update user status');
        }
    };



    // Redirect if not admin
    if (!currentUser?.isAdmin) {
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



    // ... (inside component)
    return (
        <div className="d-flex flex-column flex-grow-1">
            <PageHeader
                title="User Management"
                subtitle="View and manage all registered users"
                icon={Users}
            />

            <Card className="shadow-sm border-0 flex-grow-1" style={{ overflow: 'visible' }}>
                <Card.Header className="bg-white border-bottom py-3">
                    <div className="d-flex justify-content-between align-items-center">
                        <div className="d-flex align-items-center gap-3">
                            <span className="fw-semibold">All Users</span>
                            <Badge bg="primary" className="rounded-pill px-3 py-2">
                                {totalCount} users
                            </Badge>
                        </div>
                        <div style={{ maxWidth: '300px', width: '100%' }}>
                            <InputGroup>
                                <InputGroup.Text className="bg-white border-end-0">
                                    <Search size={16} className="text-muted" />
                                </InputGroup.Text>
                                <Form.Control
                                    className="border-start-0 ps-0"
                                    type="text"
                                    placeholder="Search by email..."
                                    value={searchInput}
                                    onChange={(e) => setSearchInput(e.target.value)}
                                    onKeyDown={(e) => { if (e.key === 'Enter') { setPage(1); setFilterEmail(searchInput); } }}
                                />
                                {filterEmail ? (
                                    <Button
                                        variant="outline-secondary"
                                        className="border-start-0"
                                        onClick={() => {
                                            setSearchInput('');
                                            setFilterEmail('');
                                            setPage(1);
                                        }}
                                    >
                                        <X size={16} />
                                    </Button>
                                ) : (
                                    <Button variant="primary" onClick={() => { setFilterEmail(searchInput); setPage(1); }}>
                                        Search
                                    </Button>
                                )}
                            </InputGroup>
                        </div>
                    </div>
                </Card.Header>
                <Card.Body className="p-0" style={{ overflow: 'visible' }}>
                    {loading ? (
                        <div className="d-flex justify-content-center align-items-center py-5">
                            <Spinner animation="border" variant="primary" />
                        </div>
                    ) : (
                        <Table hover className="mb-0">
                            <thead className="bg-light">
                                <tr>
                                    <th className="border-0 py-3 ps-4">User</th>
                                    <th className="border-0 py-3">Email</th>
                                    <th className="border-0 py-3">Joined</th>
                                    <th className="border-0 py-3 text-center">Materials</th>
                                    <th className="border-0 py-3 text-center">Plan</th>
                                    <th className="border-0 py-3 text-center">Role</th>
                                    <th className="border-0 py-3 text-center">Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {users.map((u) => (
                                    <tr key={u.id} className={u.is_blocked ? "table-danger" : ""}>
                                        <td className="py-3 ps-4">
                                            <div className="d-flex align-items-center gap-3">
                                                <img
                                                    src={u.picture || 'https://via.placeholder.com/40'}
                                                    alt=""
                                                    referrerPolicy="no-referrer"
                                                    onError={(e) => {
                                                        const target = e.target as HTMLImageElement;
                                                        target.src = 'https://via.placeholder.com/40';
                                                    }}
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
                                            <div className="d-flex flex-column align-items-center">
                                                {u.is_pro ? (
                                                    <Badge bg="warning" text="dark" className="rounded-pill px-2 py-1">
                                                        PRO
                                                    </Badge>
                                                ) : (
                                                    <Badge bg="light" text="dark" className="border rounded-pill px-2 py-1">
                                                        FREE
                                                    </Badge>
                                                )}
                                                {u.is_pro && u.current_period_end && (
                                                    <small className="text-muted mt-1" style={{ fontSize: '0.7rem' }}>
                                                        Exp: {new Date(u.current_period_end).toLocaleDateString()}
                                                    </small>
                                                )}
                                            </div>
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
                                        <td className="py-3 text-center">
                                            <Dropdown align="end">
                                                <Dropdown.Toggle variant="link" className="text-dark p-0 border-0 no-caret">
                                                    <MoreVertical size={20} />
                                                </Dropdown.Toggle>

                                                <Dropdown.Menu>
                                                    <Dropdown.Header>Manage User</Dropdown.Header>

                                                    {!u.is_pro ? (
                                                        <Dropdown.Item onClick={() => updateUserStatus(u, 'pro', true)}>
                                                            <Crown size={16} className="me-2 text-warning" />
                                                            Mark as Pro
                                                        </Dropdown.Item>
                                                    ) : (
                                                        <Dropdown.Item onClick={() => updateUserStatus(u, 'pro', false)}>
                                                            <Crown size={16} className="me-2 text-muted" />
                                                            Remove Pro Status
                                                        </Dropdown.Item>
                                                    )}

                                                    {!u.is_blocked ? (
                                                        <Dropdown.Item onClick={() => updateUserStatus(u, 'block', true)} className="text-danger">
                                                            <Ban size={16} className="me-2" />
                                                            Block User
                                                        </Dropdown.Item>
                                                    ) : (
                                                        <Dropdown.Item onClick={() => updateUserStatus(u, 'block', false)} className="text-success">
                                                            <CheckCircle size={16} className="me-2" />
                                                            Unblock User
                                                        </Dropdown.Item>
                                                    )}

                                                    <Dropdown.Divider />

                                                    {!u.is_admin ? (
                                                        <Dropdown.Item onClick={() => updateUserStatus(u, 'admin', true)}>
                                                            <Shield size={16} className="me-2 text-success" />
                                                            Make Admin
                                                        </Dropdown.Item>
                                                    ) : (
                                                        <Dropdown.Item onClick={() => updateUserStatus(u, 'admin', false)}>
                                                            <ShieldOff size={16} className="me-2" />
                                                            Remove Admin
                                                        </Dropdown.Item>
                                                    )}
                                                </Dropdown.Menu>
                                            </Dropdown>
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

            <Modal show={showProModal} onHide={() => setShowProModal(false)} centered>
                <Modal.Header closeButton>
                    <Modal.Title>Grant Pro Access</Modal.Title>
                </Modal.Header>
                <Modal.Body>
                    <p>Enter the duration for Pro access (in days). Default is 30.</p>
                    <Form.Group>
                        <Form.Label>Duration (Days)</Form.Label>
                        <Form.Control
                            type="number"
                            min="1"
                            value={proDays}
                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setProDays(parseInt(e.target.value) || 0)}
                        />
                    </Form.Group>
                </Modal.Body>
                <Modal.Footer>
                    <Button variant="secondary" onClick={() => setShowProModal(false)}>
                        Cancel
                    </Button>
                    <Button variant="primary" onClick={confirmProStatus}>
                        Grant Access
                    </Button>
                </Modal.Footer>
            </Modal>

        </div>
    );
};


export default AdminUsers;

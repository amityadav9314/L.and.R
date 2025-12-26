import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Form, Button, Card, Spinner, Alert } from 'react-bootstrap';
import { Save, Bell, Newspaper, User } from 'lucide-react';
import { feedClient } from '../services/api.ts';
import { useAuthStore } from '../store/authStore.ts';

const Settings = () => {
    const queryClient = useQueryClient();
    const { user } = useAuthStore();
    const [enabled, setEnabled] = useState(false);
    const [interestPrompt, setInterestPrompt] = useState('');
    const [evalPrompt, setEvalPrompt] = useState('');
    const [saved, setSaved] = useState(false);

    const { data: prefs, isLoading } = useQuery({
        queryKey: ['feedPreferences'],
        queryFn: () => feedClient.getFeedPreferences({})
    });

    useEffect(() => {
        if (prefs) {
            setEnabled(prefs.feedEnabled);
            setInterestPrompt(prefs.interestPrompt || '');
            setEvalPrompt(prefs.feedEvalPrompt || '');
        }
    }, [prefs]);

    const updateMutation = useMutation({
        mutationFn: () => feedClient.updateFeedPreferences({ feedEnabled: enabled, interestPrompt, feedEvalPrompt: evalPrompt }),
        onSuccess: () => {
            setSaved(true);
            queryClient.invalidateQueries({ queryKey: ['feedPreferences'] });
            setTimeout(() => setSaved(false), 3000);
        }
    });

    return (
        <div className="mx-auto" style={{ maxWidth: '800px' }}>
            <div className="mb-4">
                <h1 className="h3 fw-bold mb-1">Settings</h1>
                <p className="text-muted">Manage your account and preferences</p>
            </div>

            {isLoading ? (
                <div className="d-flex justify-content-center py-5">
                    <Spinner animation="border" variant="primary" />
                </div>
            ) : (
                <div className="d-flex flex-column gap-4">
                    {/* Profile Section */}
                    <Card className="border-0 shadow-sm rounded-4">
                        <Card.Body className="p-4">
                            <div className="d-flex align-items-center gap-3 mb-4">
                                <div className="bg-primary bg-opacity-10 p-2 rounded-3 text-primary">
                                    <User size={24} />
                                </div>
                                <h5 className="fw-bold mb-0">Profile Information</h5>
                            </div>
                            <div className="d-flex align-items-center gap-4 border p-3 rounded-3 bg-light">
                                <img src={user?.picture || ''} alt="" width="64" height="64" className="rounded-circle" />
                                <div>
                                    <h6 className="fw-bold mb-1">{user?.name}</h6>
                                    <p className="text-muted small mb-0">{user?.email}</p>
                                    <Badge bg="success" className="mt-2 rounded-pill">Google Verified</Badge>
                                </div>
                            </div>
                        </Card.Body>
                    </Card>

                    {/* Daily Feed Section */}
                    <Card className="border-0 shadow-sm rounded-4">
                        <Card.Body className="p-4">
                            <div className="d-flex align-items-center gap-3 mb-4">
                                <div className="bg-info bg-opacity-10 p-2 rounded-3 text-info">
                                    <Newspaper size={24} />
                                </div>
                                <h5 className="fw-bold mb-0">Daily AI Feed Preferences</h5>
                            </div>

                            {saved && <Alert variant="success" className="mb-4 rounded-3 border-0 shadow-sm d-flex align-items-center gap-2"><Check size={18} /> Changes saved successfully!</Alert>}

                            <Form onSubmit={(e) => { e.preventDefault(); updateMutation.mutate(); }}>
                                <Form.Group className="mb-4 d-flex align-items-center justify-content-between">
                                    <div>
                                        <Form.Label className="fw-bold mb-0">Enable Daily Feed</Form.Label>
                                        <p className="small text-muted mb-0">Receive personalized articles every morning at 6 AM IST.</p>
                                    </div>
                                    <Form.Check
                                        type="switch"
                                        id="feed-switch"
                                        checked={enabled}
                                        onChange={(e) => setEnabled(e.target.checked)}
                                        className="fs-4"
                                    />
                                </Form.Group>

                                <Form.Group className="mb-4">
                                    <Form.Label className="fw-bold">Your Interests</Form.Label>
                                    <p className="small text-muted mb-2">Technologial topics you want to stay updated on (e.g., "AI and Cloud Computing", "React & Frontend Development").</p>
                                    <Form.Control
                                        as="textarea"
                                        rows={3}
                                        value={interestPrompt}
                                        onChange={(e) => setInterestPrompt(e.target.value)}
                                        placeholder="Enter your interests..."
                                        disabled={!enabled}
                                        className="rounded-3 border-light bg-light"
                                    />
                                </Form.Group>

                                <Form.Group className="mb-4">
                                    <Form.Label className="fw-bold">Evaluation Criteria (Optional)</Form.Label>
                                    <p className="small text-muted mb-2">Define how the AI should evaluate articles (e.g., "Must be technical and deep dive", "Avoid clickbait", "Focus on practical tutorials").</p>
                                    <Form.Control
                                        as="textarea"
                                        rows={2}
                                        value={evalPrompt}
                                        onChange={(e) => setEvalPrompt(e.target.value)}
                                        placeholder="Enter evaluation criteria..."
                                        disabled={!enabled}
                                        className="rounded-3 border-light bg-light"
                                    />
                                </Form.Group>

                                <div className="d-flex justify-content-end">
                                    <Button
                                        variant="primary"
                                        type="submit"
                                        disabled={updateMutation.isPending}
                                        className="px-4 rounded-pill d-flex align-items-center gap-2"
                                    >
                                        {updateMutation.isPending ? <Spinner size="sm" /> : <Save size={18} />}
                                        Save Changes
                                    </Button>
                                </div>
                            </Form>
                        </Card.Body>
                    </Card>

                    {/* Notifications Section */}
                    <Card className="border-0 shadow-sm rounded-4">
                        <Card.Body className="p-4">
                            <div className="d-flex align-items-center gap-3 mb-4">
                                <div className="bg-warning bg-opacity-10 p-2 rounded-3 text-warning">
                                    <Bell size={24} />
                                </div>
                                <h5 className="fw-bold mb-0">Push Notifications</h5>
                            </div>
                            <p className="text-muted small mb-0 py-2">
                                Push notifications are currently managed on your mobile device.
                                Reminders for due materials will appear there at 9 AM IST.
                            </p>
                        </Card.Body>
                    </Card>
                </div>
            )}
        </div>
    );
};

// Helper badge component
const Badge = ({ children, bg, className }: { children: any, bg: string, className?: string }) => (
    <span className={`badge bg-${bg} ${className}`}>{children}</span>
);

const Check = ({ size, className }: { size: number, className?: string }) => (
    <svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}><path d="M20 6L9 17l-5-5" /></svg>
);

export default Settings;

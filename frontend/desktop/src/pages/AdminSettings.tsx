import { useState, useEffect } from 'react';
import { Card, Spinner, Alert, Button, Form, Accordion, Badge } from 'react-bootstrap';
import { Save, ChevronDown, ChevronRight } from 'lucide-react';
import { useAuthStore } from '../store/authStore.ts';
import { API_URL } from '../utils/config.ts';

interface SettingRow {
    key: string;
    value: unknown;
    description: string;
}

// Recursive component for rendering JSON form fields
const JsonFormField = ({
    path,
    value,
    onChange
}: {
    path: string;
    value: unknown;
    onChange: (path: string, newValue: unknown) => void;
}) => {
    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
        // Object: render nested fields
        return (
            <div className="ms-3 border-start ps-3">
                {Object.entries(value).map(([key, val]) => (
                    <div key={key} className="mb-2">
                        <Form.Label className="fw-semibold text-secondary mb-1" style={{ fontSize: '0.85rem' }}>
                            {key}
                        </Form.Label>
                        <JsonFormField
                            path={`${path}.${key}`}
                            value={val}
                            onChange={onChange}
                        />
                    </div>
                ))}
            </div>
        );
    }

    if (typeof value === 'number') {
        return (
            <Form.Control
                type="number"
                size="sm"
                value={value}
                onChange={(e) => onChange(path, parseInt(e.target.value) || 0)}
            />
        );
    }

    if (typeof value === 'boolean') {
        return (
            <Form.Check
                type="switch"
                checked={value}
                onChange={(e) => onChange(path, e.target.checked)}
            />
        );
    }

    if (typeof value === 'string') {
        return (
            <Form.Control
                type="text"
                size="sm"
                value={value}
                onChange={(e) => onChange(path, e.target.value)}
            />
        );
    }

    // Fallback for arrays or other types - show as JSON
    return (
        <Form.Control
            as="textarea"
            rows={2}
            size="sm"
            value={JSON.stringify(value, null, 2)}
            onChange={(e) => {
                try {
                    onChange(path, JSON.parse(e.target.value));
                } catch {
                    // Invalid JSON, ignore
                }
            }}
        />
    );
};

// Helper to set nested value by path
const setNestedValue = (obj: unknown, path: string, value: unknown): unknown => {
    const keys = path.split('.').filter(k => k);
    if (keys.length === 0) return value;

    const result = JSON.parse(JSON.stringify(obj)); // Deep clone
    let current: Record<string, unknown> = result;

    for (let i = 0; i < keys.length - 1; i++) {
        current = current[keys[i]] as Record<string, unknown>;
    }
    current[keys[keys.length - 1]] = value;

    return result;
};

const AdminSettings = () => {
    const { token } = useAuthStore();
    const [loading, setLoading] = useState(true);
    const [settings, setSettings] = useState<SettingRow[]>([]);
    const [savingKey, setSavingKey] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);
    const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());

    useEffect(() => {
        const fetchSettings = async () => {
            if (!token) return;
            setLoading(true);
            try {
                const response = await fetch(`${API_URL}/api/admin/settings`, {
                    headers: { 'Authorization': `Bearer ${token}` },
                });

                if (response.status === 403) {
                    setError('Access denied. Admin privileges required.');
                    return;
                }

                if (!response.ok) {
                    throw new Error('Failed to fetch settings');
                }

                const data = await response.json();
                setSettings(data || []);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'An error occurred');
            } finally {
                setLoading(false);
            }
        };

        fetchSettings();
    }, [token]);

    const handleValueChange = (settingKey: string, path: string, newValue: unknown) => {
        setSettings(prev => prev.map(s => {
            if (s.key !== settingKey) return s;
            const newSettingValue = setNestedValue(s.value, path, newValue);
            return { ...s, value: newSettingValue };
        }));
    };

    const handleSave = async (settingKey: string) => {
        if (!token) return;
        setSavingKey(settingKey);
        setError(null);
        setSuccess(null);

        const setting = settings.find(s => s.key === settingKey);
        if (!setting) return;

        try {
            const response = await fetch(`${API_URL}/api/admin/settings`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ key: settingKey, value: setting.value }),
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to save setting');
            }

            setSuccess(`Setting "${settingKey}" saved successfully!`);
            setTimeout(() => setSuccess(null), 3000);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An error occurred');
        } finally {
            setSavingKey(null);
        }
    };

    const toggleExpand = (key: string) => {
        setExpandedKeys(prev => {
            const next = new Set(prev);
            if (next.has(key)) {
                next.delete(key);
            } else {
                next.add(key);
            }
            return next;
        });
    };

    return (
        <>
            {error && <Alert variant="danger" className="mb-3" dismissible onClose={() => setError(null)}>{error}</Alert>}
            {success && <Alert variant="success" className="mb-3" dismissible onClose={() => setSuccess(null)}>{success}</Alert>}

            <Card className="shadow-sm border-0">
                <Card.Header className="bg-white border-bottom py-3">
                    <span className="fw-semibold">Application Settings</span>
                    <Badge bg="secondary" className="ms-2">{settings.length} keys</Badge>
                </Card.Header>
                <Card.Body className="p-0">
                    {loading ? (
                        <div className="d-flex justify-content-center align-items-center py-5">
                            <Spinner animation="border" variant="primary" />
                        </div>
                    ) : settings.length === 0 ? (
                        <div className="text-center text-muted py-5">
                            No settings found. Add settings via database migration.
                        </div>
                    ) : (
                        <Accordion flush>
                            {settings.map((setting) => (
                                <Accordion.Item eventKey={setting.key} key={setting.key}>
                                    <Accordion.Header onClick={() => toggleExpand(setting.key)}>
                                        <div className="d-flex align-items-center gap-2 w-100">
                                            {expandedKeys.has(setting.key) ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                                            <code className="text-primary">{setting.key}</code>
                                            {setting.description && (
                                                <span className="text-muted small ms-2">— {setting.description}</span>
                                            )}
                                        </div>
                                    </Accordion.Header>
                                    <Accordion.Body>
                                        <div className="mb-3">
                                            <JsonFormField
                                                path=""
                                                value={setting.value}
                                                onChange={(path, newValue) => handleValueChange(setting.key, path, newValue)}
                                            />
                                        </div>
                                        <div className="d-flex justify-content-end">
                                            <Button
                                                variant="primary"
                                                size="sm"
                                                onClick={() => handleSave(setting.key)}
                                                disabled={savingKey === setting.key}
                                                className="d-flex align-items-center gap-2"
                                            >
                                                {savingKey === setting.key ? <Spinner size="sm" /> : <Save size={14} />}
                                                Save {setting.key}
                                            </Button>
                                        </div>
                                    </Accordion.Body>
                                </Accordion.Item>
                            ))}
                        </Accordion>
                    )}
                </Card.Body>
            </Card>
        </>
    );
};

export default AdminSettings;

import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Row, Badge, Modal, Form, Spinner } from 'react-bootstrap';
import { useLocation } from 'react-router-dom';
import { learningClient } from '../services/api.ts';
import { Type, Link as LinkIcon, Image as ImageIcon, Youtube, Plus, BookOpen, Clock, Search } from 'lucide-react';
import RevisionModal from '../components/RevisionModal.tsx';

const Dashboard = () => {
    const location = useLocation();
    const [showAddModal, setShowAddModal] = useState(false);
    const [showReviseModal, setShowReviseModal] = useState(false);
    const [selectedMaterialId, setSelectedMaterialId] = useState<string | null>(null);
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedTags, setSelectedTags] = useState<string[]>([]);

    // Default onlyDue to false if explicitly on /vault
    const isVault = location.pathname === '/vault';
    const [onlyDue, setOnlyDue] = useState(!isVault);

    useEffect(() => {
        setOnlyDue(!isVault);
    }, [location.pathname, isVault]);

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['materials', onlyDue],
        queryFn: () => learningClient.getDueMaterials({ onlyDue })
    });

    const materials = data?.materials || [];

    // Extract unique tags from all materials
    const allTags = Array.from(new Set(materials.flatMap(m => m.tags || []))).sort();

    const toggleTag = (tag: string) => {
        setSelectedTags(prev =>
            prev.includes(tag)
                ? prev.filter(t => t !== tag)
                : [...prev, tag]
        );
    };

    const filteredMaterials = materials.filter(m => {
        const query = searchQuery.toLowerCase();
        const titleMatch = (m.title || '').toLowerCase().includes(query);
        const searchTagsMatch = (m.tags || []).some(t => t.toLowerCase().includes(query));

        // Combined with multi-tag filter
        const tagFilterMatch = selectedTags.length === 0 ||
            selectedTags.every(tag => (m.tags || []).includes(tag));

        return (titleMatch || searchTagsMatch) && tagFilterMatch;
    });

    const handleReview = (materialId: string) => {
        console.log('[Dashboard] Starting review for material:', materialId);
        setSelectedMaterialId(materialId);
        setShowReviseModal(true);
    };

    return (
        <div>
            <div className="d-flex justify-content-between align-items-center mb-4">
                <div>
                    <h1 className="h3 fw-bold mb-1">{isVault ? 'Learning Vault' : 'Due for Revise'}</h1>
                    <p className="text-muted mb-0">Manage and review your materials</p>
                </div>
                <div className="d-flex gap-3 align-items-center">
                    <div className="position-relative" style={{ width: '300px' }}>
                        <Search className="position-absolute top-50 translate-middle-y ms-3 text-muted" size={18} />
                        <Form.Control
                            type="text"
                            placeholder="Search title or tags..."
                            className="rounded-pill ps-5 bg-white border-0 shadow-sm"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                        />
                    </div>
                    <div className="d-flex gap-2">
                        <Button
                            variant={onlyDue ? 'primary' : 'outline-primary'}
                            onClick={() => setOnlyDue(true)}
                            className="rounded-pill px-4"
                        >
                            Due Now
                        </Button>
                        <Button
                            variant={!onlyDue ? 'primary' : 'outline-primary'}
                            onClick={() => setOnlyDue(false)}
                            className="rounded-pill px-4"
                        >
                            All Materials
                        </Button>
                        <Button variant="success" className="rounded-pill px-4 d-flex align-items-center gap-2" onClick={() => setShowAddModal(true)}>
                            <Plus size={18} />
                            <span>Add New</span>
                        </Button>
                    </div>
                </div>
            </div>

            {/* Tag Filter Bar */}
            <div className="mb-4">
                <div className="d-flex align-items-center justify-content-between mb-2">
                    <h6 className="fw-bold mb-0 text-muted small text-uppercase" style={{ letterSpacing: '0.05em' }}>Filter by tags:</h6>
                    {selectedTags.length > 0 && (
                        <Button
                            variant="link"
                            size="sm"
                            className="text-decoration-none p-0 text-primary small"
                            onClick={() => setSelectedTags([])}
                        >
                            Clear Filters ({selectedTags.length})
                        </Button>
                    )}
                </div>
                <div className="d-flex gap-2 overflow-auto pb-2 hide-scrollbar flex-nowrap">
                    <Badge
                        bg={selectedTags.length === 0 ? 'primary' : 'white'}
                        text={selectedTags.length === 0 ? 'white' : 'dark'}
                        className={`px-3 py-2 rounded-pill border cursor-pointer transition-all shadow-sm`}
                        style={{ cursor: 'pointer', transition: 'all 0.2s', fontWeight: 500 }}
                        onClick={() => setSelectedTags([])}
                    >
                        # All
                    </Badge>
                    {allTags.map(tag => {
                        const isSelected = selectedTags.includes(tag);
                        return (
                            <Badge
                                key={tag}
                                bg={isSelected ? 'primary' : 'white'}
                                text={isSelected ? 'white' : 'dark'}
                                className="px-3 py-2 rounded-pill border cursor-pointer transition-all shadow-sm"
                                style={{ cursor: 'pointer', transition: 'all 0.2s', fontWeight: 500 }}
                                onClick={() => toggleTag(tag)}
                            >
                                #{tag}
                            </Badge>
                        );
                    })}
                </div>
            </div>

            {/* Materials Status Line */}
            <div className="mb-3 px-1">
                <p className="text-muted small mb-0 fw-medium opacity-75">
                    Materials {filteredMaterials.length > 0 ? `1-${filteredMaterials.length}` : '0'} ({filteredMaterials.length} of {materials.length} total)
                </p>
            </div>

            {isLoading ? (
                <div className="d-flex justify-content-center py-5">
                    <Spinner animation="border" variant="primary" />
                </div>
            ) : filteredMaterials.length === 0 ? (
                <Card className="text-center py-5 bg-white border-0 shadow-sm rounded-4">
                    <div className="py-4">
                        <div className="bg-light d-inline-block p-4 rounded-circle mb-3">
                            <span className="fs-1">{searchQuery ? '🔍' : '📚'}</span>
                        </div>
                        <h3>{searchQuery ? 'No matches found' : 'No materials found'}</h3>
                        <p className="text-muted px-4">
                            {searchQuery ? `We couldn't find anything matching "${searchQuery}"` : 'Start by adding a link or text you want to learn.'}
                        </p>
                        {!searchQuery && (
                            <Button variant="primary" className="mt-2 px-4 rounded-pill" onClick={() => setShowAddModal(true)}>
                                Add your first material
                            </Button>
                        )}
                        {searchQuery && (
                            <Button variant="outline-secondary" className="mt-2 px-4 rounded-pill" onClick={() => setSearchQuery('')}>
                                Clear search
                            </Button>
                        )}
                    </div>
                </Card>
            ) : (
                <Row xs={1} md={2} lg={3} className="g-4">
                    {filteredMaterials.map((m) => (
                        <Col key={m.id}>
                            <Card className="h-100 border-0 shadow-sm rounded-4 hover-shadow transition-all">
                                <Card.Body className="d-flex flex-column">
                                    <div className="d-flex justify-content-between align-items-start mb-2">
                                        <Badge bg="info" className="rounded-pill px-3 py-2">
                                            MATERIAL
                                        </Badge>
                                    </div>
                                    <Card.Title className="h5 fw-bold mb-2 line-clamp-2">{m.title || 'Untitled Material'}</Card.Title>
                                    <Card.Text className="text-muted small mb-3 line-clamp-3">
                                        Check out the summary and flashcards for this topic.
                                    </Card.Text>

                                    <div className="mt-auto">
                                        <div className="mb-3 overflow-hidden">
                                            <div className="d-flex gap-1 overflow-auto pb-1 flex-nowrap hide-scrollbar" style={{ whiteSpace: 'nowrap' }}>
                                                {m.tags.map(t => (
                                                    <Badge key={t} bg="light" text="dark" className="border flex-shrink-0" style={{ fontSize: '0.7rem' }}>#{t}</Badge>
                                                ))}
                                            </div>
                                        </div>

                                        <div className="d-flex justify-content-between align-items-center">
                                            <div className="d-flex align-items-center gap-1 text-muted small">
                                                <Clock size={14} />
                                                <span>{m.dueCount} cards due</span>
                                            </div>
                                            <Button
                                                variant="primary"
                                                size="sm"
                                                className="rounded-pill px-3 d-flex align-items-center gap-2"
                                                onClick={() => handleReview(m.id)}
                                            >
                                                <BookOpen size={14} />
                                                Review
                                            </Button>
                                        </div>
                                    </div>
                                </Card.Body>
                            </Card>
                        </Col>
                    ))}
                </Row>
            )}

            {/* Modals */}
            <Modal show={showAddModal} onHide={() => setShowAddModal(false)} centered backdrop="static">
                <Modal.Header closeButton className="border-0">
                    <Modal.Title className="fw-bold">Add New Material</Modal.Title>
                </Modal.Header>
                <Modal.Body className="pt-0">
                    <AddMaterialForm onSuccess={() => { setShowAddModal(false); refetch(); }} />
                </Modal.Body>
            </Modal>

            {selectedMaterialId && (
                <RevisionModal
                    show={showReviseModal}
                    onHide={() => { setShowReviseModal(false); setSelectedMaterialId(null); }}
                    materialId={selectedMaterialId}
                    onComplete={() => { refetch(); }}
                />
            )}

            <style>{`
                .hover-shadow:hover { transform: translateY(-5px); box-shadow: 0 10px 20px rgba(0,0,0,0.1) !important; }
                .transition-all { transition: all 0.3s ease; }
                .hide-scrollbar::-webkit-scrollbar { display: none; }
                .hide-scrollbar { -ms-overflow-style: none; scrollbar-width: none; }
                .line-clamp-2 { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
                .line-clamp-3 { display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }
            `}</style>
        </div>
    );
};

const AddMaterialForm = ({ onSuccess }: { onSuccess: () => void }) => {
    const [loading, setLoading] = useState(false);
    const [contentType, setContentType] = useState<'TEXT' | 'LINK' | 'IMAGE' | 'YOUTUBE'>('TEXT');
    const [content, setContent] = useState('');
    const [tags, setTags] = useState('');
    const [imageData, setImageData] = useState('');

    const handleImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file) {
            const reader = new FileReader();
            reader.onloadend = () => {
                const base64String = reader.result as string;
                // Remove the data area prefix (e.g., data:image/jpeg;base64,)
                const base64Data = base64String.split(',')[1];
                setImageData(base64Data);
            };
            reader.readAsDataURL(file);
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (contentType !== 'IMAGE' && !content.trim()) return;
        if (contentType === 'IMAGE' && !imageData) return;

        setLoading(true);
        try {
            const tagList = tags.split(',').map(t => t.trim()).filter(t => t);

            await learningClient.addMaterial({
                content: contentType === 'IMAGE' ? '' : content,
                type: contentType,
                existingTags: tagList,
                imageData: contentType === 'IMAGE' ? imageData : ''
            });

            onSuccess();
        } catch (error) {
            console.error('Failed to add material:', error);
            alert('Failed to add material');
        } finally {
            setLoading(false);
        }
    };

    return (
        <Form onSubmit={handleSubmit}>
            <div className="mb-4">
                <label className="small fw-semibold text-muted mb-2 d-block">Select Content Type</label>
                <div className="d-flex gap-2">
                    {[
                        { id: 'TEXT', icon: Type, label: 'Text' },
                        { id: 'LINK', icon: LinkIcon, label: 'Link' },
                        { id: 'IMAGE', icon: ImageIcon, label: 'Image' },
                        { id: 'YOUTUBE', icon: Youtube, label: 'YouTube' }
                    ].map((t) => (
                        <Button
                            key={t.id}
                            variant={contentType === t.id ? 'primary' : 'outline-light'}
                            className={`flex-grow-1 d-flex flex-column align-items-center py-2 rounded-3 border ${contentType === t.id ? '' : 'text-dark'}`}
                            onClick={() => {
                                setContentType(t.id as any);
                                setContent('');
                                setImageData('');
                            }}
                        >
                            <t.icon size={20} className="mb-1" />
                            <span style={{ fontSize: '0.75rem' }}>{t.label}</span>
                        </Button>
                    ))}
                </div>
            </div>

            {contentType === 'IMAGE' ? (
                <Form.Group className="mb-4">
                    <Form.Label className="small fw-semibold text-muted">Upload Image</Form.Label>
                    <div className="p-4 border rounded-3 bg-light text-center cursor-pointer hover-bg-light transition-all">
                        <input
                            type="file"
                            accept="image/*"
                            onChange={handleImageChange}
                            className="form-control"
                            required
                        />
                        {imageData && (
                            <div className="mt-3">
                                <img
                                    src={`data:image/png;base64,${imageData}`}
                                    alt="Preview"
                                    style={{ maxWidth: '100%', maxHeight: '150px' }}
                                    className="rounded shadow-sm"
                                />
                            </div>
                        )}
                    </div>
                </Form.Group>
            ) : (
                <Form.Group className="mb-4">
                    <Form.Label className="small fw-semibold text-muted">
                        {contentType === 'LINK' ? 'Paste URL' : contentType === 'YOUTUBE' ? 'YouTube Video URL' : 'Paste Text Content'}
                    </Form.Label>
                    <Form.Control
                        as={contentType === 'TEXT' ? 'textarea' : 'input'}
                        {...(contentType === 'TEXT' ? { rows: 6 } : {})}
                        placeholder={
                            contentType === 'LINK' ? 'https://example.com/article' :
                                contentType === 'YOUTUBE' ? 'https://youtube.com/watch?v=...' :
                                    'Paste content you want to learn...'
                        }
                        className="rounded-3 border-light bg-light"
                        value={content}
                        onChange={(e: any) => setContent(e.target.value)}
                        required
                    />
                </Form.Group>
            )}

            <Form.Group className="mb-4">
                <Form.Label className="small fw-semibold text-muted">Tags (Optional)</Form.Label>
                <Form.Control
                    type="text"
                    placeholder="ai, coding, news (comma separated)"
                    className="rounded-3 border-light bg-light"
                    value={tags}
                    onChange={(e) => setTags(e.target.value)}
                />
            </Form.Group>

            <div className="d-grid">
                <Button variant="primary" type="submit" disabled={loading} className="py-2 fw-bold rounded-pill">
                    {loading ? <Spinner size="sm" className="me-2" /> : <Plus size={18} className="me-2" />}
                    {loading ? 'Processing with AI...' : 'Generate Flashcards'}
                </Button>
            </div>
            <p className="text-center small text-muted mt-3 mb-0">
                Our AI will scan the {contentType.toLowerCase()} and automatically create high-quality flashcards for you.
            </p>
        </Form>
    );
};

export default Dashboard;

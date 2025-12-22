import { useState, useEffect } from 'react';
import { Modal, Button, Spinner, ProgressBar, Badge } from 'react-bootstrap';
import { CheckCircle, XCircle, RotateCcw } from 'lucide-react';
import { learningClient } from '../services/api.ts';

interface RevisionModalProps {
    show: boolean;
    onHide: () => void;
    materialId: string;
    onComplete: () => void;
}

const RevisionModal = ({ show, onHide, materialId, onComplete }: RevisionModalProps) => {
    const [currentIndex, setCurrentIndex] = useState(0);
    const [isFlipped, setIsFlipped] = useState(false);
    const [viewingState, setViewingState] = useState<'summary' | 'flashcards'>('summary');

    const [flashcards, setFlashcards] = useState<any[]>([]);
    const [summary, setSummary] = useState<string>('');
    const [materialTitle, setMaterialTitle] = useState<string>('');
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        if (show && materialId) {
            console.log('[RevisionModal] Opening session for material:', materialId);
            setCurrentIndex(0);
            setIsFlipped(false);
            setViewingState('summary'); // Always start with summary
            setIsLoading(true);

            // Fetch both flashcards and summary
            Promise.all([
                learningClient.getDueFlashcards({ materialId }),
                learningClient.getMaterialSummary({ materialId })
            ])
                .then(([flashcardsData, summaryData]) => {
                    console.log('[RevisionModal] Flashcards loaded:', flashcardsData.flashcards?.length || 0);
                    setFlashcards(flashcardsData.flashcards || []);
                    setSummary(summaryData.summary || '');
                    setMaterialTitle(summaryData.title || 'Material');
                })
                .catch(err => {
                    console.error('[RevisionModal] Failed to fetch data:', err);
                })
                .finally(() => setIsLoading(false));
        }
    }, [show, materialId]);

    const currentCard = flashcards[currentIndex];

    const moveNext = () => {
        if (currentIndex < flashcards.length - 1) {
            console.log('[RevisionModal] Navigating to next card (no submit)');
            setCurrentIndex(prev => prev + 1);
            setIsFlipped(false);
        }
    };

    const movePrev = () => {
        if (currentIndex > 0) {
            console.log('[RevisionModal] Navigating to previous card');
            setCurrentIndex(prev => prev - 1);
            setIsFlipped(false);
        }
    };

    const submitAndNext = (success: boolean) => {
        console.log('[RevisionModal] submitAndNext called:', { success, currentIndex });
        if (!currentCard) return;

        // Perform the API call in the background
        if (success) {
            learningClient.completeReview({ flashcardId: currentCard.id })
                .catch(err => console.error('[RevisionModal] Failed to complete review:', err));
        } else {
            learningClient.failReview({ flashcardId: currentCard.id })
                .catch(err => console.error('[RevisionModal] Failed to fail review:', err));
        }

        // Move to the next card
        if (currentIndex < flashcards.length - 1) {
            setCurrentIndex(prev => prev + 1);
            setIsFlipped(false);
        } else {
            console.log('[RevisionModal] Session complete');
            onComplete();
            onHide();
        }
    };

    // Keyboard controls
    useEffect(() => {
        if (!show) return;

        const handleGlobalKeyDown = (e: KeyboardEvent) => {
            console.log('[RevisionModal] Key pressed:', e.key, '| isFlipped:', isFlipped);

            // Space or Enter to flip
            if (e.key === ' ' || e.key === 'Enter') {
                e.preventDefault();
                console.log('[RevisionModal] Flipping card');
                setIsFlipped(prev => !prev);
            }
            // Arrows to navigate only (No API call per user request)
            else if (e.key === 'ArrowRight' || e.key.toLowerCase() === 'k') {
                e.preventDefault();
                console.log('[RevisionModal] Keyboard: Navigate Next');
                moveNext();
            } else if (e.key === 'ArrowLeft' || e.key.toLowerCase() === 'j') {
                e.preventDefault();
                console.log('[RevisionModal] Keyboard: Navigate Prev');
                movePrev();
            }
        };

        window.addEventListener('keydown', handleGlobalKeyDown);
        return () => window.removeEventListener('keydown', handleGlobalKeyDown);
    }, [show, isFlipped, currentIndex, flashcards, moveNext, movePrev]);

    const progress = flashcards.length > 0 ? ((currentIndex) / flashcards.length) * 100 : 0;

    return (
        <Modal
            show={show}
            onHide={onHide}
            size="lg"
            centered
            backdrop="static"
            className="revision-modal"
        >
            <Modal.Header closeButton className="border-0 pb-0">
                <Modal.Title className="fw-bold w-100">
                    <div className="d-flex justify-content-between align-items-center me-4">
                        <span className="h5 mb-0 fw-bold text-primary">Revision Session</span>
                        {flashcards.length > 0 && (
                            <span className="small text-muted">Card {currentIndex + 1} of {flashcards.length}</span>
                        )}
                    </div>
                </Modal.Title>
            </Modal.Header>
            <Modal.Body className="py-4 px-4">
                {isLoading ? (
                    <div className="text-center py-5">
                        <Spinner animation="border" variant="primary" />
                        <p className="mt-2 text-muted">Loading flashcards...</p>
                    </div>
                ) : flashcards.length === 0 ? (
                    <div className="text-center py-5">
                        <div className="display-1 mb-4">🎉</div>
                        <h3 className="fw-bold">All caught up!</h3>
                        <p className="text-muted">No cards due for review in this material.</p>
                        <Button variant="primary" className="rounded-pill px-4" onClick={onHide}>Close</Button>
                    </div>
                ) : viewingState === 'summary' ? (
                    <div className="text-center py-4">
                        <div className="mb-4 p-4 bg-light rounded-4 border">
                            <div className="d-flex align-items-center justify-content-center gap-2 mb-3">
                                <Badge bg="info" className="rounded-pill px-3 py-2">📝 Material Summary</Badge>
                            </div>
                            <h4 className="fw-bold mb-3 text-primary">{materialTitle}</h4>
                            <div
                                className="text-start text-muted lh-lg"
                                style={{
                                    fontSize: '0.95rem',
                                    maxHeight: '400px',
                                    overflow: 'auto',
                                    whiteSpace: 'pre-wrap',
                                    wordWrap: 'break-word'
                                }}
                            >
                                {summary || 'No summary available for this material.'}
                            </div>
                        </div>
                        <Button
                            variant="primary"
                            size="lg"
                            className="rounded-pill px-5 shadow-sm"
                            onClick={() => setViewingState('flashcards')}
                        >
                            Continue to Questions ({flashcards.length} cards)
                        </Button>
                    </div>
                ) : (
                    <>
                        <ProgressBar now={progress} variant="primary" className="rounded-pill mb-4" style={{ height: '6px' }} />

                        <div
                            className={`flashcard-scene perspective-1000 w-100 mb-4`}
                            style={{ cursor: 'pointer', minHeight: '350px' }}
                            onClick={() => setIsFlipped(!isFlipped)}
                        >
                            <div className={`flashcard-inner ${isFlipped ? 'is-flipped' : ''}`} style={{ transition: 'transform 0.6s cubic-bezier(0.4, 0, 0.2, 1)', transformStyle: 'preserve-3d', position: 'relative', height: '350px' }}>
                                {/* Front Side */}
                                <div className="flashcard-side flashcard-front rounded-4 py-5 px-4 d-flex align-items-center justify-content-center text-center shadow-sm border bg-white"
                                    style={{
                                        position: 'absolute',
                                        width: '100%',
                                        height: '100%',
                                        backfaceVisibility: 'hidden',
                                        WebkitBackfaceVisibility: 'hidden',
                                        visibility: isFlipped ? 'hidden' : 'visible',
                                        zIndex: isFlipped ? 1 : 2
                                    }}>
                                    <div className="overflow-auto h-100 w-100 d-flex flex-column align-items-center justify-content-center">
                                        <Badge bg="primary" className="mb-4 bg-opacity-10 text-primary border-0 rounded-pill px-3 py-2 fw-semibold">QUESTION</Badge>
                                        <h3 className="fw-bold m-0 lh-base">{currentCard?.question}</h3>
                                        <p className="text-muted mt-4 small mb-0">Click or press Space to flip</p>
                                    </div>
                                </div>

                                {/* Back Side */}
                                <div className="flashcard-side flashcard-back rounded-4 py-5 px-4 d-flex align-items-center justify-content-center text-center shadow-sm border bg-white"
                                    style={{
                                        position: 'absolute',
                                        width: '100%',
                                        height: '100%',
                                        backfaceVisibility: 'hidden',
                                        WebkitBackfaceVisibility: 'hidden',
                                        transform: 'rotateY(180deg)',
                                        visibility: isFlipped ? 'visible' : 'hidden',
                                        zIndex: isFlipped ? 2 : 1
                                    }}>
                                    <div className="overflow-auto h-100 w-100 d-flex flex-column align-items-center justify-content-center">
                                        <Badge bg="success" className="mb-4 bg-opacity-10 text-success border-0 rounded-pill px-3 py-2 fw-semibold">ANSWER</Badge>
                                        <h4 className="fw-medium lh-base m-0 px-2">{currentCard?.answer}</h4>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className="d-flex gap-3 justify-content-center">
                            {!isFlipped ? (
                                <Button variant="primary" className="py-2 px-5 rounded-pill shadow-sm fw-bold d-flex align-items-center gap-2" onClick={() => setIsFlipped(true)}>
                                    <RotateCcw size={18} /> Reveal Answer (Space)
                                </Button>
                            ) : (
                                <>
                                    <Button
                                        variant="outline-danger"
                                        className="py-2 px-4 rounded-pill shadow-sm fw-bold flex-grow-1 d-flex align-items-center justify-content-center gap-2"
                                        onClick={(e) => { e.stopPropagation(); submitAndNext(false); }}
                                    >
                                        <XCircle size={18} /> Forgot (←)
                                    </Button>
                                    <Button
                                        variant="success"
                                        className="py-2 px-4 rounded-pill shadow-sm fw-bold flex-grow-1 d-flex align-items-center justify-content-center gap-2"
                                        onClick={(e) => { e.stopPropagation(); submitAndNext(true); }}
                                    >
                                        <CheckCircle size={18} /> Got it (→)
                                    </Button>
                                </>
                            )}
                        </div>

                        <div className="mt-4 d-none d-md-flex gap-4 text-muted small opacity-50 justify-content-center">
                            <span><kbd>Space</kbd> Flip</span>
                            <span><kbd>←</kbd> <kbd>→</kbd> Navigate</span>
                            <span>Click buttons to mark completion</span>
                        </div>
                    </>
                )}
            </Modal.Body>
            <style>{`
                .perspective-1000 { perspective: 1000px; }
                .flashcard-inner.is-flipped { transform: rotateY(180deg); }
                .flashcard-side { top: 0; left: 0; }
                .flashcard-front, .flashcard-back { 
                    scrollbar-width: thin;
                    -ms-overflow-style: none;
                }
            `}</style>
        </Modal>
    );
};

export default RevisionModal;

import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Card, Spinner, ProgressBar } from 'react-bootstrap';
import { CheckCircle, XCircle, ChevronLeft, RotateCcw } from 'lucide-react';
import { learningClient } from '../services/api.ts';

const Revise = () => {
    const { materialId } = useParams();
    const navigate = useNavigate();
    const queryClient = useQueryClient();
    const [currentIndex, setCurrentIndex] = useState(0);
    const [isFlipped, setIsFlipped] = useState(false);

    const { data, isLoading } = useQuery({
        queryKey: ['flashcards', materialId],
        queryFn: () => learningClient.getDueFlashcards({ materialId: materialId! })
    });

    const flashcards = data?.flashcards || [];
    const currentCard = flashcards[currentIndex];

    const completeMutation = useMutation({
        mutationFn: (id: string) => learningClient.completeReview({ flashcardId: id }),
    });

    const failMutation = useMutation({
        mutationFn: (id: string) => learningClient.failReview({ flashcardId: id }),
    });

    const handleNext = async (success: boolean) => {
        if (!currentCard) return;

        if (success) {
            await completeMutation.mutateAsync(currentCard.id);
        } else {
            await failMutation.mutateAsync(currentCard.id);
        }

        if (currentIndex < flashcards.length - 1) {
            setCurrentIndex(prev => prev + 1);
            setIsFlipped(false);
        } else {
            // Finish session
            queryClient.invalidateQueries({ queryKey: ['materials'] });
            navigate('/');
        }
    };

    // Keyboard controls
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.code === 'Space') {
                e.preventDefault();
                setIsFlipped(prev => !prev);
            } else if (isFlipped && (e.code === 'ArrowRight' || e.code === 'KeyK')) {
                handleNext(true);
            } else if (isFlipped && (e.code === 'ArrowLeft' || e.code === 'KeyJ')) {
                handleNext(false);
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [isFlipped, currentIndex, flashcards.length]);

    if (isLoading) {
        return (
            <div className="d-flex justify-content-center py-5">
                <Spinner animation="border" variant="primary" />
            </div>
        );
    }

    if (flashcards.length === 0) {
        return (
            <div className="text-center py-5 mt-5">
                <div className="mb-4 display-1">🎉</div>
                <h2 className="fw-bold">All caught up!</h2>
                <p className="text-muted">You have no cards due for review in this material.</p>
                <Button variant="primary" className="rounded-pill px-4" onClick={() => navigate('/')}>Back to Dashboard</Button>
            </div>
        );
    }

    const progress = ((currentIndex) / flashcards.length) * 100;

    return (
        <div className="d-flex flex-column align-items-center justify-content-center py-4" style={{ minHeight: '80vh' }}>
            <div className="w-100 mb-5" style={{ maxWidth: '600px' }}>
                <div className="d-flex justify-content-between align-items-center mb-3">
                    <Button variant="link" className="text-dark p-0 d-flex align-items-center gap-2 text-decoration-none fw-bold" onClick={() => navigate('/')}>
                        <ChevronLeft size={20} /> Exit Session
                    </Button>
                    <span className="fw-bold text-primary">Card {currentIndex + 1} of {flashcards.length}</span>
                </div>
                <ProgressBar now={progress} variant="primary" className="rounded-pill" style={{ height: '8px' }} />
            </div>

            <div
                className={`flashcard-container perspective-1000 w-100 ${isFlipped ? 'is-flipped' : ''}`}
                style={{ maxWidth: '600px', cursor: 'pointer' }}
                onClick={() => setIsFlipped(!isFlipped)}
            >
                <Card className="flashcard-inner position-relative shadow-lg border-0 rounded-4 transition-all" style={{ height: '400px', transformStyle: 'preserve-3d' }}>
                    {/* Front */}
                    <Card.Body className="flashcard-front d-flex align-items-center justify-content-center p-5 text-center position-absolute w-100 h-100" style={{ backfaceVisibility: 'hidden', zIndex: isFlipped ? 0 : 1 }}>
                        <div>
                            <Badge className="mb-4 bg-primary bg-opacity-10 text-primary border-0 rounded-pill px-3 py-2 fw-semibold">QUESTION</Badge>
                            <h2 className="fw-bold">{currentCard?.question}</h2>
                            <p className="text-muted mt-5 small">Click or press Space to reveal answer</p>
                        </div>
                    </Card.Body>

                    {/* Back */}
                    <Card.Body className="flashcard-back d-flex align-items-center justify-content-center p-5 text-center position-absolute w-100 h-100" style={{ backfaceVisibility: 'hidden', transform: 'rotateY(180deg)', zIndex: isFlipped ? 1 : 0 }}>
                        <div>
                            <Badge className="mb-4 bg-success bg-opacity-10 text-success border-0 rounded-pill px-3 py-2 fw-semibold">ANSWER</Badge>
                            <h3 className="fw-medium lh-base">{currentCard?.answer}</h3>
                        </div>
                    </Card.Body>
                </Card>
            </div>

            <div className="mt-5 d-flex gap-4 w-100 justify-content-center" style={{ maxWidth: '600px' }}>
                {!isFlipped ? (
                    <Button variant="primary" className="py-3 px-5 rounded-pill shadow-sm fw-bold d-flex align-items-center gap-2" onClick={() => setIsFlipped(true)}>
                        <RotateCcw size={20} /> Reveal Answer
                    </Button>
                ) : (
                    <>
                        <Button
                            variant="outline-danger"
                            className="py-3 px-5 rounded-pill shadow-sm fw-bold flex-grow-1 d-flex align-items-center justify-content-center gap-2"
                            onClick={(e) => { e.stopPropagation(); handleNext(false); }}
                        >
                            <XCircle size={20} /> Forgot (←)
                        </Button>
                        <Button
                            variant="success"
                            className="py-3 px-5 rounded-pill shadow-sm fw-bold flex-grow-1 d-flex align-items-center justify-content-center gap-2"
                            onClick={(e) => { e.stopPropagation(); handleNext(true); }}
                        >
                            <CheckCircle size={20} /> Got it (→)
                        </Button>
                    </>
                )}
            </div>

            <div className="mt-5 d-none d-md-flex gap-4 text-muted small opacity-75">
                <span className="d-flex align-items-center gap-1"><Badge bg="light" text="dark" className="border">Space</Badge> Flip</span>
                <span className="d-flex align-items-center gap-1"><Badge bg="light" text="dark" className="border">←</Badge> Forgot</span>
                <span className="d-flex align-items-center gap-1"><Badge bg="light" text="dark" className="border">→</Badge> Got it</span>
            </div>

            <style>{`
                .perspective-1000 { perspective: 1000px; }
                .flashcard-inner { transition: transform 0.6s cubic-bezier(0.4, 0, 0.2, 1); }
                .is-flipped .flashcard-inner { transform: rotateY(180deg); }
                .hover-shadow:hover { transform: translateY(-5px); box-shadow: 0 10px 20px rgba(0,0,0,0.1) !important; }
                .transition-all { transition: all 0.3s ease; }
                .line-clamp-2 { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
                .line-clamp-3 { display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }
                .line-clamp-4 { display: -webkit-box; -webkit-line-clamp: 4; -webkit-box-orient: vertical; overflow: hidden; }
                .cursor-pointer { cursor: pointer; }
            `}</style>
        </div>
    );
};

const Badge = ({ children, bg, text, className }: { children: any, bg?: string, text?: string, className?: string }) => (
    <span className={`badge bg-${bg || 'primary'} text-${text || 'white'} ${className}`}>{children}</span>
);

export default Revise;

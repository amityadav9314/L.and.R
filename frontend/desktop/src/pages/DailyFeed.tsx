import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Card, Col, Row, Badge, Spinner, ButtonGroup } from 'react-bootstrap';
import { ExternalLink, Check, BookOpen, Newspaper } from 'lucide-react';
import { feedClient, learningClient } from '../services/api.ts';
import { useNavigate } from 'react-router-dom';

const DailyFeed = () => {
    const navigate = useNavigate();
    const queryClient = useQueryClient();
    const [provider, setProvider] = useState<'google' | 'tavily'>('google');
    // Today's date in YYYY-MM-DD
    const today = new Date().toISOString().split('T')[0];

    const { data: prefs, isLoading: isLoadingPrefs } = useQuery({
        queryKey: ['feedPreferences'],
        queryFn: () => feedClient.getFeedPreferences({})
    });

    const { data: feedData, isLoading: isLoadingFeed } = useQuery({
        queryKey: ['dailyFeed', today, provider],
        queryFn: () => feedClient.getDailyFeed({ date: today }),
        enabled: !!prefs?.feedEnabled
    });

    const isLoading = isLoadingPrefs || (prefs?.feedEnabled && isLoadingFeed);

    const articles = feedData?.articles || [];

    const addMutation = useMutation({
        mutationFn: (url: string) => learningClient.addMaterial({
            content: url,
            type: 'LINK',
            existingTags: ['feed', provider],
            imageData: ''
        }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['dailyFeed'] });
            queryClient.invalidateQueries({ queryKey: ['materials'] });
        }
    });

    return (
        <div>
            <div className="d-flex justify-content-between align-items-center mb-4">
                <div>
                    <h1 className="h3 fw-bold mb-1">Daily AI Feed</h1>
                    <p className="text-muted mb-0">Personalized articles for {today}</p>
                </div>
                <ButtonGroup className="shadow-sm rounded-pill overflow-hidden">
                    <Button
                        variant={provider === 'google' ? 'primary' : 'white'}
                        onClick={() => setProvider('google')}
                        className="px-4"
                    >
                        Google News
                    </Button>
                    <Button
                        variant={provider === 'tavily' ? 'primary' : 'white'}
                        onClick={() => setProvider('tavily')}
                        className="px-4"
                    >
                        Tavily AI
                    </Button>
                </ButtonGroup>
            </div>

            {isLoading ? (
                <div className="d-flex justify-content-center py-5">
                    <Spinner animation="border" variant="primary" />
                </div>
            ) : !prefs?.feedEnabled ? (
                <Card className="text-center py-5 bg-white border-0 shadow-sm rounded-4">
                    <div className="py-4">
                        <div className="bg-light d-inline-block p-4 rounded-circle mb-3 text-muted">
                            <Newspaper size={48} />
                        </div>
                        <h3>Feed is Disabled</h3>
                        <p className="text-muted px-4 mb-4">You have disabled the daily AI feed in your settings.</p>
                        <Button variant="primary" className="px-4 rounded-pill" onClick={() => navigate('/settings')}>
                            Enable Daily Feed
                        </Button>
                    </div>
                </Card>
            ) : articles.length === 0 ? (
                <Card className="text-center py-5 bg-white border-0 shadow-sm rounded-4">
                    <div className="py-4">
                        <div className="bg-light d-inline-block p-4 rounded-circle mb-3">
                            <span className="fs-1">🗞️</span>
                        </div>
                        <h3>No articles found</h3>
                        <p className="text-muted px-4">Daily feed generation might still be in progress or interests not set.</p>
                        <Button variant="outline-primary" className="mt-2 px-4 rounded-pill" onClick={() => navigate('/settings')}>
                            Update Interests
                        </Button>
                    </div>
                </Card>
            ) : (
                <Row xs={1} md={2} className="g-4">
                    {articles.map((article) => (
                        <Col key={article.id}>
                            <Card className="h-100 border-0 shadow-sm rounded-4 overflow-hidden hover-shadow transition-all">
                                <Card.Body className="p-4 d-flex flex-column">
                                    <div className="d-flex justify-content-between align-items-start mb-3">
                                        <Badge bg="light" text="primary" className="border text-uppercase small px-3 py-2">
                                            {article.provider === 'google' ? 'Google' : 'Tavily'} Article
                                        </Badge>
                                        <Badge bg="success" className="rounded-pill px-3 py-2">
                                            {Math.round(article.relevanceScore * 100)}% Relevant
                                        </Badge>
                                    </div>

                                    <h4 className="fw-bold mb-3">{article.title}</h4>
                                    <p className="text-muted mb-4 line-clamp-4">{article.snippet}</p>

                                    <div className="mt-auto d-flex gap-2">
                                        <Button
                                            variant="primary"
                                            className="flex-grow-1 rounded-pill d-flex align-items-center justify-content-center gap-2"
                                            onClick={() => window.open(article.url, '_blank')}
                                        >
                                            <ExternalLink size={18} />
                                            Read Article
                                        </Button>

                                        {article.isAdded ? (
                                            <Button variant="outline-success" className="rounded-pill px-4" disabled>
                                                <Check size={18} className="me-2" />
                                                Added
                                            </Button>
                                        ) : (
                                            <Button
                                                variant="outline-primary"
                                                className="rounded-pill px-4"
                                                onClick={() => addMutation.mutate(article.url)}
                                                disabled={addMutation.isPending && addMutation.variables === article.url}
                                            >
                                                {addMutation.isPending && addMutation.variables === article.url ? (
                                                    <Spinner size="sm" className="me-2" />
                                                ) : (
                                                    <BookOpen size={18} className="me-2" />
                                                )}
                                                {addMutation.isPending && addMutation.variables === article.url ? 'Saving...' : 'Revise 📚'}
                                            </Button>
                                        )}
                                    </div>
                                </Card.Body>
                            </Card>
                        </Col>
                    ))}
                </Row>
            )}
        </div>
    );
};

export default DailyFeed;

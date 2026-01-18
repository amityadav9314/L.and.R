import { paymentClient } from '../services/api';
import { useAuthStore } from '../store/authStore';
import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate, Link } from 'react-router-dom';
import { Alert, Button, Card, Col, Row, Spinner, Badge } from 'react-bootstrap';
import { CheckCircle, Zap } from 'lucide-react';
import { PageHeader } from '../components/PageHeader.tsx';

declare global {
    interface Window {
        Razorpay: any;
    }
}

const UpgradePage = () => {
    const { user } = useAuthStore();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();

    useEffect(() => {
        const status = searchParams.get('status');
        if (status === 'success') {
            alert("Payment Successful! Your plan will be updated shortly.");
            navigate('/');
        } else if (status === 'failed') {
            setError("Payment Failed. Please try again.");
        }
    }, [searchParams, navigate]);

    const handleUpgrade = async () => {
        setLoading(true);
        setError('');
        try {
            const response = await paymentClient.createSubscriptionOrder({
                planId: 'PRO',
                redirectUrl: window.location.origin + '/upgrade?status=success'
            });

            // If backend returned a Payment Link (Redirect Flow)
            if (response.paymentLink) {
                window.location.href = response.paymentLink;
                return;
            }

            // Otherwise, Standard Popup Flow
            const options = {
                key: response.keyId,
                amount: response.amount * 100, // Razorpay expects paise, but API might return amount in INR. Wait, backend returns amount in INR.
                // Wait, backend CreateSubscriptionOrder: amount := 199.0.
                // Backend razorpay.go CreateOrder: amountPaise := amount * 100.
                // Backend returns amount as float32(amount) which is 199.0.
                // Razorpay checkout options expects amount in paise only if currency is INR?
                // Actually, if order_id is present, amount in options must match order amount.
                // The order created on backend used amount * 100 (19900 paise).
                // So here we should pass 19900.
                // Let's use response.amount (199) * 100.

                currency: response.currency,
                name: 'L.and.R Pro',
                description: 'Upgrade to Pro Plan',
                order_id: response.orderId,
                handler: function (response: any) {
                    // Payment successful
                    console.log("Payment successful", response);
                    alert("Payment Successful! Your plan will be updated shortly.");
                    // Ideally we should verify payment on backend, but webhook handles it.
                    // Verification endpoint is not exposed yet for manual verify.
                    // Just reload to reflect changes roughly? 
                    // Or maybe backend needs a few seconds.
                    window.location.href = '/';
                },
                prefill: {
                    name: user?.name,
                    email: user?.email,
                },
                theme: {
                    color: '#0d6efd',
                },
            };

            const rzp = new window.Razorpay(options);
            rzp.on('payment.failed', function (response: any) {
                setError(`Payment Failed: ${response.error.description}`);
            });
            rzp.open();
        } catch (err: any) {
            console.error(err);
            setError('Failed to initiate payment. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div>
            <PageHeader
                title="Upgrade to Pro"
                subtitle="Unlock the full power of your learning vault"
                icon={Zap}
            />
            <Row className="justify-content-center">
                <Col md={6} lg={5}>
                    <Card className="shadow-lg border-primary">
                        <Card.Body className="p-5">
                            <div className="text-center mb-4">
                                <h3 className="display-4 fw-bold">₹199</h3>
                                <p className="text-muted mb-2">One-time payment</p>
                                <Badge bg="warning" text="dark" className="px-3 py-2">
                                    30 Days Pro Access
                                </Badge>
                            </div>

                            <ul className="list-unstyled mb-4">
                                <li className="mb-3 d-flex align-items-center">
                                    <CheckCircle size={20} className="text-success me-2" />
                                    <span>Unlimited AI Flashcards</span>
                                </li>
                                <li className="mb-3 d-flex align-items-center">
                                    <CheckCircle size={20} className="text-success me-2" />
                                    <span>Personalized Daily Feed</span>
                                </li>
                                <li className="mb-3 d-flex align-items-center">
                                    <CheckCircle size={20} className="text-success me-2" />
                                    <span>Detailed Analytics</span>
                                </li>
                                <li className="mb-3 d-flex align-items-center">
                                    <CheckCircle size={20} className="text-success me-2" />
                                    <span>Priority Support</span>
                                </li>
                            </ul>

                            <Alert variant="info" className="mb-4">
                                <small>
                                    <strong>Note:</strong> Your Pro access will be valid for 30 days from the date of payment.
                                    No automatic renewal. You can renew anytime by making another payment.
                                </small>
                            </Alert>

                            {error && <Alert variant="danger">{error}</Alert>}

                            <Button
                                variant="primary"
                                size="lg"
                                className="w-100"
                                onClick={handleUpgrade}
                                disabled={loading}
                            >
                                {loading ? <Spinner as="span" animation="border" size="sm" /> : 'Upgrade Now'}
                            </Button>

                            <div className="text-center mt-3">
                                <small className="text-muted">
                                    By upgrading, you agree to our <Link to="/terms">Terms</Link> and <Link to="/privacy">Privacy Policy</Link>.
                                </small>
                            </div>
                        </Card.Body>
                    </Card>
                </Col>
            </Row>
        </div>
    );
};

export default UpgradePage;

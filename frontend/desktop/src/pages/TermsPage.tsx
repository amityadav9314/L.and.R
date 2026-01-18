import React from 'react';
import { Card, Table, Container } from 'react-bootstrap';
import { FileText } from 'lucide-react';
import { Link } from 'react-router-dom';
import { PageHeader } from '../components/PageHeader.tsx';

export const TermsPage: React.FC = () => {
    return (
        <Container className="py-4">
            <div className="mb-3">
                <Link to="/" className="text-decoration-none">← Back to Home</Link>
            </div>
            <PageHeader
                title="Terms & Conditions"
                subtitle="Please read carefully"
                icon={FileText}
            />

            <Card className="mb-4 shadow-sm">
                <Card.Body>
                    <Card.Title>1. Introduction</Card.Title>
                    <Card.Text>
                        Welcome to L.and.R (Application). By using our services, you agree to these terms.
                        Please read them carefully.
                    </Card.Text>

                    <Card.Title className="mt-4">2. Usage Limits & Fair Use</Card.Title>
                    <Card.Text>
                        To ensure high quality service for all users, we enforce the following daily limits on AI generation and content imports.
                    </Card.Text>

                    <Table striped bordered hover className="mt-3">
                        <thead>
                            <tr>
                                <th>Feature</th>
                                <th>Free Plan</th>
                                <th>Pro Plan (₹199 for 30 days)</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Web Link Imports (AI)</td>
                                <td>3 / day</td>
                                <td>20 / day</td>
                            </tr>
                            <tr>
                                <td>Text Imports (AI)</td>
                                <td>10 / day</td>
                                <td>Unlimited (Fair Use)</td>
                            </tr>
                            <tr>
                                <td>Daily AI Feed</td>
                                <td>Standard</td>
                                <td>Customizable Prompts</td>
                            </tr>
                        </tbody>
                    </Table>
                    <Card.Text className="text-muted small">
                        * "Unlimited" is subject to Fair Use Policy (FUP) to prevent abuse (e.g., botting).
                        Excessive usage beyond human capabilities may result in temporary suspension.
                    </Card.Text>

                    <Card.Title className="mt-4">3. Pro Subscription</Card.Title>
                    <Card.Text>
                        <strong>Subscription Duration:</strong> One-time payment of ₹199 grants you Pro access for exactly 30 days from the date of payment.
                        <br /><br />
                        <strong>No Automatic Renewal:</strong> Your Pro features will automatically expire after 30 days. There is no automatic billing or renewal.
                        <br /><br />
                        <strong>Renewal:</strong> You can renew your Pro access at any time by making another payment of ₹199, which will grant another 30 days of access.
                        <br /><br />
                        <strong>Feature Downgrade:</strong> After your Pro subscription expires, you will automatically revert to the Free plan. All your data (flashcards, materials, etc.) will be preserved.
                        <br /><br />
                        <strong>Refunds:</strong> Refunds are generally not provided once Pro access has been activated, but please contact support if you experience technical issues.
                    </Card.Text>

                    <Card.Title className="mt-4">4. Privacy & Data</Card.Title>
                    <Card.Text>
                        We respect your privacy. We store your flashcards and learning progress securely.
                        We do not sell your personal data.
                        <br />
                        <Link to="/privacy" className="btn btn-link px-0">View full Privacy Policy</Link>
                    </Card.Text>
                </Card.Body>
            </Card>

            <div className="text-center text-muted mt-5">
                <small>&copy; {new Date().getFullYear()} L.and.R. All rights reserved.</small>
            </div>
        </Container>
    );
};

import React from 'react';
import { Container, Card, Table } from 'react-bootstrap';
import { FileText, Shield, CheckCircle } from 'lucide-react';

export const TermsPage: React.FC = () => {
    return (
        <Container className="py-5" style={{ maxWidth: '800px' }}>
            <h1 className="mb-4 d-flex align-items-center gap-2">
                <FileText size={32} /> Terms & Conditions
            </h1>

            <Card className="mb-4 shadow-sm">
                <Card.Body>
                    <Card.Title>1. Introduction</Card.Title>
                    <Card.Text>
                        Welcome to InkGrid (Application). By using our services, you agree to these terms.
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
                                <th>Pro Plan (₹199/mo)</th>
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
                        - Subscriptions are billed monthly or yearly.
                        - You can cancel anytime. Access continues until the end of the billing period.
                        - Refunds are generally not provided for partial months, but contact support if you have issues.
                    </Card.Text>

                    <Card.Title className="mt-4">4. Privacy & Data</Card.Title>
                    <Card.Text>
                        We respect your privacy. We store your flashcards and learning progress securely.
                        We do not sell your personal data.
                        <br />
                        <a href="/privacy" className="btn btn-link px-0">View full Privacy Policy</a>
                    </Card.Text>
                </Card.Body>
            </Card>

            <div className="text-center text-muted mt-5">
                <small>&copy; {new Date().getFullYear()} InkGrid. All rights reserved.</small>
            </div>
        </Container>
    );
};

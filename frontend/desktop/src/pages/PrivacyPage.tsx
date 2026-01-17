import React from 'react';
import { Container, Card } from 'react-bootstrap';
import { Shield } from 'lucide-react';

export const PrivacyPage: React.FC = () => {
    return (
        <Container className="py-5" style={{ maxWidth: '800px' }}>
            <h1 className="mb-4 d-flex align-items-center gap-2">
                <Shield size={32} /> Privacy Policy
            </h1>

            <Card className="mb-4 shadow-sm">
                <Card.Body>
                    <Card.Title>1. Information We Collect</Card.Title>
                    <Card.Text>
                        - **Account Info**: Name, email (from Google Login), and profile picture.
                        - **Usage Data**: Content you import, flashcards you create, and your review history.
                        - **Payment Info**: Processed securely by Razorpay. We do not store your card details.
                    </Card.Text>

                    <Card.Title className="mt-4">2. How We Use Your Data</Card.Title>
                    <Card.Text>
                        - To generate flashcards and summaries using AI providers (Groq, Cerebras).
                        - To track your learning progress (Space Repetition).
                        - To send you daily digests (if enabled).
                    </Card.Text>

                    <Card.Title className="mt-4">3. Data Sharing</Card.Title>
                    <Card.Text>
                        - **AI Providers**: We send content text to AI models for processing. Usage is ephemeral and not used to train their models (per enterprise API terms).
                        - **Legal**: We may disclose data if required by law.
                    </Card.Text>

                    <Card.Title className="mt-4">4. Data Security</Card.Title>
                    <Card.Text>
                        We use industry-standard encryption for data in transit and at rest.
                    </Card.Text>

                    <Card.Title className="mt-4">5. Contact Us</Card.Title>
                    <Card.Text>
                        For privacy concerns, contact support@inkgrid.com.
                    </Card.Text>
                </Card.Body>
            </Card>
        </Container>
    );
};

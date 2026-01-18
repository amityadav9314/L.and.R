import React from 'react';
import { Card, Container } from 'react-bootstrap';
import { Shield } from 'lucide-react';
import { Link } from 'react-router-dom';
import { PageHeader } from '../components/PageHeader.tsx';

export const PrivacyPage: React.FC = () => {
    return (
        <Container className="py-4">
            <div className="mb-3">
                <Link to="/" className="text-decoration-none">← Back to Home</Link>
            </div>
            <PageHeader
                title="Privacy Policy"
                subtitle="Effective Date: January 1, 2025"
                icon={Shield}
            />
            <Card className="shadow-sm border-0">
                <Card.Body className="p-5">


                    <Card.Text as="div">
                        <p>Welcome to <strong>L.and.R</strong> ("we," "our," or "us"). We are committed to protecting your privacy and ensuring your personal information is handled in a safe and responsible manner. This Privacy Policy explains how we collect, use, and safeguard your information when you use our mobile application.</p>

                        <h4 className="mt-4">Information We Collect</h4>
                        <p><strong>Account Information:</strong> When you sign in with Google, we receive your email address and profile name to create your account.</p>
                        <p><strong>Learning Materials:</strong> Content you add to the app (URLs, notes, flashcards) is stored securely to provide the learning service.</p>
                        <p><strong>Usage Data:</strong> We collect anonymous usage statistics to improve the app experience.</p>

                        <h4 className="mt-4">How We Use Your Information</h4>
                        <ul>
                            <li>To provide and maintain our learning service</li>
                            <li>To send you revision reminders and notifications (with your permission)</li>
                            <li>To generate personalized daily feed content based on your preferences</li>
                            <li>To improve our app and user experience</li>
                        </ul>

                        <h4 className="mt-4">Data Storage and Security</h4>
                        <p>Your data is stored securely on our servers. We implement industry-standard security measures to protect your information.</p>

                        <h4 className="mt-4">Third-Party Services</h4>
                        <p>We use the following third-party services:</p>
                        <ul>
                            <li><strong>Google Sign-In:</strong> For authentication</li>
                            <li><strong>Firebase Cloud Messaging:</strong> For push notifications</li>
                        </ul>

                        <h4 className="mt-4">Your Rights</h4>
                        <p>You can request deletion of your account and associated data at any time by contacting us.</p>

                        <h4 className="mt-4">Children's Privacy</h4>
                        <p>Our app is not intended for children under 13. We do not knowingly collect information from children under 13.</p>

                        <h4 className="mt-4">Contact Us</h4>
                        <p>If you have questions about this Privacy Policy, please contact us at: <a href="mailto:amityadav9314@gmail.com">amityadav9314@gmail.com</a></p>
                    </Card.Text>
                </Card.Body>
            </Card>
        </Container>
    );
};

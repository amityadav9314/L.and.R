import { createChannel, createClientFactory, ClientError, Status, Metadata, FetchTransport } from 'nice-grpc-web';
import { AuthServiceDefinition, LearningServiceDefinition, FeedServiceDefinition, PaymentServiceDefinition } from '../proto/definitions.ts';
import { API_URL } from '../utils/config.ts';

const channel = createChannel(API_URL, FetchTransport());

const clientFactory = createClientFactory().use(async function* (call: any, options: any) {
    const token = localStorage.getItem('auth_token');
    const metadata = new Metadata(options.metadata);
    if (token) {
        metadata.set('authorization', `Bearer ${token}`);
    }
    try {
        return yield* call.next(call.request, { ...options, metadata });
    } catch (error: any) {
        if (error instanceof ClientError && error.code === Status.UNAUTHENTICATED) {
            console.log("User unauthenticated");
            // Clear token if unauthenticated
            localStorage.removeItem('auth_token');
            // We can also trigger a redirect to login here if using a global event or store
        }
        throw error;
    }
});

export const authClient = clientFactory.create(AuthServiceDefinition, channel);
export const learningClient = clientFactory.create(LearningServiceDefinition, channel);
export const feedClient = clientFactory.create(FeedServiceDefinition, channel);
export const paymentClient = clientFactory.create(PaymentServiceDefinition, channel);

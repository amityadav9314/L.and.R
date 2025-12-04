/**
 * Platform-aware gRPC API client
 * 
 * - Web: Uses nice-grpc-web (works fine in browsers)
 * - Native (Android/iOS): Uses direct XMLHttpRequest client (bypasses nice-grpc-web bugs)
 */
import { Platform } from 'react-native';

// Conditional export based on platform
let authClient: any;
let learningClient: any;

if (Platform.OS === 'web') {
    // Web: Use nice-grpc-web
    const { createChannel, createClientFactory, ClientError, Status, Metadata, FetchTransport } = require('nice-grpc-web');
    const { AuthServiceDefinition, LearningServiceDefinition } = require('./definitions');
    const { API_URL } = require('../utils/config');

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
            }
            throw error;
        }
    });

    authClient = clientFactory.create(AuthServiceDefinition, channel);
    learningClient = clientFactory.create(LearningServiceDefinition, channel);
} else {
    // Native: Use direct API client (bypasses nice-grpc-web)
    const directApi = require('./directApi');
    authClient = directApi.authClient;
    learningClient = directApi.learningClient;
}

export { authClient, learningClient };

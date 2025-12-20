/**
 * Firebase Cloud Messaging Service
 * 
 * Handles FCM token registration and push notification permissions.
 * Replaces expo-notifications with Firebase for reliable server-side push.
 * 
 * NOTE: Requires a development build or production APK - won't work in Expo Go.
 */
import { Platform } from 'react-native';
import { learningClient } from './directApi';

// Lazy import to avoid issues on web and gracefully handle missing native modules
let messaging: any = null;
let isFirebaseAvailable = false;

const getMessaging = async () => {
    if (Platform.OS === 'web') {
        return null;
    }
    if (messaging !== null) {
        return isFirebaseAvailable ? messaging : null;
    }

    try {
        const firebase = await import('@react-native-firebase/messaging');
        messaging = firebase.default;
        isFirebaseAvailable = true;
        return messaging;
    } catch (error) {
        console.log('[FCM] Firebase native modules not available (Expo Go or missing build)');
        isFirebaseAvailable = false;
        return null;
    }
};

let onNotificationOpened: ((remoteMessage: any) => void) | null = null;

export const FirebaseMessagingService = {
    /**
     * Register a callback to be called when a notification is opened
     */
    registerNotificationHandler(handler: (remoteMessage: any) => void) {
        onNotificationOpened = handler;

        // Handle initial notification if app was opened from a quit state
        getMessaging().then(msg => {
            if (msg) {
                msg().getInitialNotification().then((remoteMessage: any) => {
                    if (remoteMessage && onNotificationOpened) {
                        console.log('[FCM] Opened app from quit state:', remoteMessage);
                        onNotificationOpened(remoteMessage);
                    }
                });
            }
        });
    },

    /**
     * Request notification permissions and get FCM token
     */
    async initialize(): Promise<void> {
        if (Platform.OS === 'web') {
            console.log('[FCM] Not supported on web');
            return;
        }

        try {
            const msg = await getMessaging();
            if (!msg) return;

            // Request permission
            const authStatus = await msg().requestPermission();
            const enabled =
                authStatus === msg.AuthorizationStatus.AUTHORIZED ||
                authStatus === msg.AuthorizationStatus.PROVISIONAL;

            if (enabled) {
                console.log('[FCM] Permission granted');
                await this.registerToken();
            } else {
                console.log('[FCM] Permission denied');
            }

            // Handle notification clicks from background state
            msg().onNotificationOpenedApp((remoteMessage: any) => {
                console.log('[FCM] Notification caused app to open from background state:', remoteMessage);
                if (onNotificationOpened) {
                    onNotificationOpened(remoteMessage);
                }
            });

            // Listen for token refresh
            msg().onTokenRefresh(async () => {
                console.log('[FCM] Token refreshed');
                await this.registerToken();
            });

            // Handle foreground messages
            msg().onMessage(async (remoteMessage: any) => {
                console.log('[FCM] Foreground message:', remoteMessage);
                // Could show an in-app notification here
            });

        } catch (error) {
            console.warn('[FCM] Failed to initialize:', error);
        }
    },

    /**
     * Get FCM token and register with backend
     */
    async registerToken(): Promise<void> {
        if (Platform.OS === 'web') return;

        try {
            const msg = await getMessaging();
            if (!msg) return;

            const token = await msg().getToken();
            if (token) {
                console.log('[FCM] Token obtained, registering with backend...');
                await learningClient.registerPushToken({
                    token,
                    platform: Platform.OS, // 'android' or 'ios'
                });
                console.log('[FCM] Token registered successfully');
            }
        } catch (error) {
            console.warn('[FCM] Failed to register token:', error);
        }
    },

    /**
     * Check if notifications are enabled
     */
    async hasPermission(): Promise<boolean> {
        if (Platform.OS === 'web') return false;

        try {
            const msg = await getMessaging();
            if (!msg) return false;

            const authStatus = await msg().hasPermission();
            return (
                authStatus === msg.AuthorizationStatus.AUTHORIZED ||
                authStatus === msg.AuthorizationStatus.PROVISIONAL
            );
        } catch (error) {
            return false;
        }
    },
};

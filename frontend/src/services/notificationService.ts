import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { learningClient } from './api';
import { APP_NAME } from '../utils/constants';

// Configure notification behavior
export class NotificationService {
    private static notificationIdentifier: string | null = null;
    private static isConfigured = false;

    private static async configure() {
        if (this.isConfigured) return;

        try {
            // Set up Android notification channel (required for Android 8+)
            if (Platform.OS === 'android') {
                await Notifications.setNotificationChannelAsync('default', {
                    name: `${APP_NAME} Notifications`,
                    importance: Notifications.AndroidImportance.HIGH,
                    vibrationPattern: [0, 250, 250, 250],
                    lightColor: '#4285F4',
                });
                console.log('[Notifications] Android notification channel created');
            }

            Notifications.setNotificationHandler({
                handleNotification: async () => ({
                    shouldPlaySound: true,
                    shouldSetBadge: true,
                    shouldShowBanner: true,
                    shouldShowList: true,
                }),
            });
            this.isConfigured = true;
        } catch (error) {
            console.warn('[Notifications] Failed to configure notification handler (likely running in Expo Go):', error);
        }
    }

    /**
     * Request notification permissions from the user
     */
    static async requestPermissions(): Promise<boolean> {
        try {
            const { status: existingStatus } = await Notifications.getPermissionsAsync();
            let finalStatus = existingStatus;

            if (existingStatus !== 'granted') {
                const { status } = await Notifications.requestPermissionsAsync();
                finalStatus = status;
            }

            if (finalStatus !== 'granted') {
                console.log('[Notifications] Permission not granted');
                return false;
            }

            console.log('[Notifications] Permission granted');
            return true;
        } catch (error) {
            console.error('[Notifications] Error requesting permissions:', error);
            return false;
        }
    }

    /**
     * Schedule a daily notification to check for due flashcards
     */
    static async scheduleDailyNotification(): Promise<void> {
        try {
            // Cancel existing notification if any
            if (this.notificationIdentifier) {
                await Notifications.cancelScheduledNotificationAsync(this.notificationIdentifier);
            }

            // Use platform-specific trigger (CALENDAR not supported on Android)
            const trigger = Platform.OS === 'android'
                ? {
                    type: Notifications.SchedulableTriggerInputTypes.DAILY as const,
                    hour: 9,
                    minute: 0,
                }
                : {
                    type: Notifications.SchedulableTriggerInputTypes.CALENDAR as const,
                    hour: 9,
                    minute: 0,
                    repeats: true,
                };

            // Schedule daily notification at 8 AM
            this.notificationIdentifier = await Notifications.scheduleNotificationAsync({
                content: {
                    title: `${APP_NAME} - Daily Review! 📚`,
                    body: 'Check your vault to see what items are due for review today.',
                    data: { type: 'daily_reminder' },
                },
                trigger,
            });

            console.log('[Notifications] Daily notification scheduled');
        } catch (error) {
            console.error('[Notifications] Error scheduling notification:', error);
        }
    }

    /**
     * Check for due flashcards and send immediate notification if needed
     */
    static async checkAndNotify(): Promise<void> {
        try {
            const response = await learningClient.getNotificationStatus({});

            if (response.hasDueMaterials && response.dueMaterialsCount > 0) {
                const count = response.dueMaterialsCount;
                const moreCount = count - 1;
                const title = response.firstDueMaterialTitle || 'Untitled';

                const body = moreCount > 0
                    ? `You have "${title}" and ${moreCount} more material${moreCount > 1 ? 's' : ''} to revise today.`
                    : `You have "${title}" to revise today.`;

                await Notifications.scheduleNotificationAsync({
                    content: {
                        title: `${APP_NAME} - Review Due! 🎯`,
                        body: body,
                        data: {
                            type: 'due_materials',
                            count: count,
                        },
                    },
                    trigger: null, // Send immediately
                });

                console.log(`[Notifications] Sent notification for ${count} due materials`);
            }
        } catch (error) {
            console.error('[Notifications] Error checking due materials:', error);
        }
    }

    /**
     * Get the count of due flashcards
     */
    static async getDueCount(): Promise<number> {
        try {
            const response = await learningClient.getNotificationStatus({});
            return response.dueFlashcardsCount;
        } catch (error) {
            console.error('[Notifications] Error getting due count:', error);
            return 0;
        }
    }

    /**
     * Cancel all scheduled notifications
     */
    static async cancelAll(): Promise<void> {
        try {
            await Notifications.cancelAllScheduledNotificationsAsync();
            this.notificationIdentifier = null;
            console.log('[Notifications] All notifications cancelled');
        } catch (error) {
            console.error('[Notifications] Error cancelling notifications:', error);
        }
    }

    /**
     * Initialize notification service
     */
    static async initialize(): Promise<void> {
        try {
            await this.configure();
            const hasPermission = await this.requestPermissions();

            if (hasPermission) {
                await this.scheduleDailyNotification();
                // We no longer call checkAndNotify here to avoid immediate notification on startup
                // If there are due materials, they will be visible on the Home Screen immediately.
            }
        } catch (error) {
            console.warn('[Notifications] Failed to initialize (likely running in Expo Go):', error);
        }
    }
}


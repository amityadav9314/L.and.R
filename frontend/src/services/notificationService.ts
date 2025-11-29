import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { learningClient } from './api';

// Configure notification behavior
Notifications.setNotificationHandler({
    handleNotification: async () => ({
        shouldShowAlert: true,
        shouldPlaySound: true,
        shouldSetBadge: true,
        shouldShowBanner: true,
        shouldShowList: true,
    }),
});

export class NotificationService {
    private static notificationIdentifier: string | null = null;

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

            // Schedule daily notification at 9 AM
            this.notificationIdentifier = await Notifications.scheduleNotificationAsync({
                content: {
                    title: 'LandR - Time to Review! 📚',
                    body: 'You have flashcards due for review. Keep up your learning streak!',
                    data: { type: 'daily_reminder' },
                },
                trigger: {
                    type: Notifications.SchedulableTriggerInputTypes.CALENDAR,
                    hour: 9,
                    minute: 0,
                    repeats: true,
                },
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

            if (response.hasDueMaterials && response.dueFlashcardsCount > 0) {
                await Notifications.scheduleNotificationAsync({
                    content: {
                        title: 'LandR - Flashcards Due! 🎯',
                        body: `You have ${response.dueFlashcardsCount} flashcard${response.dueFlashcardsCount > 1 ? 's' : ''} ready for review.`,
                        data: {
                            type: 'due_flashcards',
                            count: response.dueFlashcardsCount,
                        },
                    },
                    trigger: null, // Send immediately
                });

                console.log(`[Notifications] Sent notification for ${response.dueFlashcardsCount} due flashcards`);
            }
        } catch (error) {
            console.error('[Notifications] Error checking due flashcards:', error);
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
        const hasPermission = await this.requestPermissions();

        if (hasPermission) {
            await this.scheduleDailyNotification();
            await this.checkAndNotify();
        }
    }
}


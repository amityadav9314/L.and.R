import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { learningClient } from './api';
import { APP_NAME } from '../utils/constants';

// Define these safely using require to avoid top-level import crashes on missing native modules
let BackgroundFetch: any = null;
let TaskManager: any = null;

try {
    if (Platform.OS !== 'web') {
        BackgroundFetch = require('expo-background-fetch');
        TaskManager = require('expo-task-manager');
    }
} catch (e) {
    console.warn('[Notifications] Native background modules could not be loaded:', e);
}

const BACKGROUND_FETCH_TASK = 'background-fetch-due-materials';

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
     * Register background task for checking due materials
     */
    static async registerBackgroundTask(): Promise<void> {
        if (Platform.OS === 'web' || !TaskManager || !BackgroundFetch) {
            console.log('[Notifications] Background tasks are not supported on this platform/build');
            return;
        }

        try {
            // Check if TaskManager is actually usable
            const available = await TaskManager.isAvailableAsync();
            if (!available) {
                console.warn('[Notifications] TaskManager is not available in this build.');
                return;
            }

            const isRegistered = await TaskManager.isTaskRegisteredAsync(BACKGROUND_FETCH_TASK);
            if (isRegistered) {
                console.log('[Notifications] Background task already registered');
                return;
            }

            await BackgroundFetch.registerTaskAsync(BACKGROUND_FETCH_TASK, {
                minimumInterval: 60 * 60, // 1 hour (minimum allowed by OS)
                stopOnTerminate: false, // Continue after app is closed
                startOnBoot: true, // Start after device reboot
            });
            console.log('[Notifications] Background fetch task registered successfully');
        } catch (error) {
            console.warn('[Notifications] Failed to register background task (this is normal if no native rebuild done yet):', error);
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
                await this.registerBackgroundTask();
                // We no longer call checkAndNotify here to avoid immediate notification on startup
            }
        } catch (error) {
            console.warn('[Notifications] Failed to initialize (likely running in Expo Go):', error);
        }
    }
}

// Define the background task if TaskManager is available
if (Platform.OS !== 'web' && TaskManager && BackgroundFetch) {
    try {
        TaskManager.isAvailableAsync().then((available: boolean) => {
            if (available) {
                TaskManager.defineTask(BACKGROUND_FETCH_TASK, async () => {
                    try {
                        console.log('[Notifications] Background fetch task triggered');
                        await NotificationService.checkAndNotify();
                        return BackgroundFetch.BackgroundFetchResult.NewData;
                    } catch (error) {
                        console.error('[Notifications] Background fetch task failed:', error);
                        return BackgroundFetch.BackgroundFetchResult.Failed;
                    }
                });
            }
        }).catch((err: any) => {
            console.warn('[Notifications] TaskManager availability check failed:', err);
        });
    } catch (e) {
        console.warn('[Notifications] Failed to define background task:', e);
    }
}



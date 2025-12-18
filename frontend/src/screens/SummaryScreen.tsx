import React from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    ActivityIndicator,
    ScrollView,
} from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { useNavigation, useRoute } from '../navigation/ManualRouter';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { learningClient } from '../services/api';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';

export const SummaryScreen = () => {
    const navigation = useNavigation();
    const route = useRoute();
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
    const { materialId, title } = route.params as { materialId: string; title: string };
    const displayTitle = title || 'Material Summary';

    const { data, isLoading, error, refetch } = useQuery({
        queryKey: ['materialSummary', materialId],
        queryFn: async () => {
            console.log(`[SummaryScreen] Fetching summary for material: ${materialId}`);
            try {
                const response = await learningClient.getMaterialSummary({ materialId });
                console.log('[SummaryScreen] Got summary, length:', response.summary?.length || 0);
                return response;
            } catch (err) {
                console.error('[SummaryScreen] Error fetching summary:', err);
                throw err;
            }
        },
    });

    const styles = createStyles(colors);

    const handleContinue = () => {
        navigation.navigate('MaterialDetail', { materialId, title: data?.title || title });
    };

    const handleSkip = () => {
        navigation.navigate('MaterialDetail', { materialId, title: data?.title || title });
    };

    if (isLoading) {
        return (
            <View style={styles.container}>
                <AppHeader />
                <View style={styles.loadingContainer}>
                    <ActivityIndicator size="large" color={colors.primary} />
                    <Text style={styles.loadingText}>Generating summary...</Text>
                    <Text style={styles.loadingSubtext}>This may take a few seconds for new materials</Text>
                </View>
            </View>
        );
    }

    if (error) {
        return (
            <View style={styles.container}>
                <AppHeader />
                <View style={styles.errorContainer}>
                    <Text style={styles.errorIcon}>⚠️</Text>
                    <Text style={styles.errorText}>Failed to load summary</Text>
                    <TouchableOpacity style={styles.retryButton} onPress={() => refetch()}>
                        <Text style={styles.retryButtonText}>Retry</Text>
                    </TouchableOpacity>
                    <TouchableOpacity style={styles.skipButton} onPress={handleSkip}>
                        <Text style={styles.skipButtonText}>Skip to Questions</Text>
                    </TouchableOpacity>
                </View>
            </View>
        );
    }

    return (
        <View style={[styles.container, { paddingBottom: insets.bottom }]}>
            <AppHeader />
            <View style={styles.contentContainer}>
                <Text style={styles.headerTitle} numberOfLines={2}>{displayTitle}</Text>
                <View style={styles.badge}>
                    <Text style={styles.badgeText}>📖 Summary</Text>
                </View>

                <ScrollView
                    style={styles.summaryScroll}
                    contentContainerStyle={styles.summaryContent}
                    showsVerticalScrollIndicator={true}
                >
                    <Text style={styles.summaryText}>{data?.summary || 'No summary available.'}</Text>
                </ScrollView>

                <View style={styles.buttonContainer}>
                    <TouchableOpacity style={styles.skipButton} onPress={handleSkip}>
                        <Text style={styles.skipButtonText}>Skip</Text>
                    </TouchableOpacity>
                    <TouchableOpacity style={styles.continueButton} onPress={handleContinue}>
                        <Text style={styles.continueButtonText}>Continue to Questions →</Text>
                    </TouchableOpacity>
                </View>
            </View>
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background
    },
    contentContainer: {
        flex: 1,
        padding: 20,
        paddingTop: 16
    },
    headerTitle: {
        fontSize: 22,
        fontWeight: '700',
        color: colors.textPrimary,
        textAlign: 'center',
        marginBottom: 12
    },
    badge: {
        alignSelf: 'center',
        backgroundColor: colors.cardAlt,
        paddingHorizontal: 16,
        paddingVertical: 8,
        borderRadius: 20,
        marginBottom: 16,
    },
    badgeText: {
        fontSize: 14,
        fontWeight: '600',
        color: colors.textSecondary,
    },
    summaryScroll: {
        flex: 1,
        backgroundColor: colors.card,
        borderRadius: 16,
        marginBottom: 20,
    },
    summaryContent: {
        padding: 20,
    },
    summaryText: {
        fontSize: 16,
        lineHeight: 26,
        color: colors.textPrimary,
    },
    buttonContainer: {
        flexDirection: 'row',
        gap: 12,
    },
    skipButton: {
        flex: 1,
        backgroundColor: colors.cardAlt,
        paddingVertical: 16,
        borderRadius: 14,
        alignItems: 'center',
    },
    skipButtonText: {
        color: colors.textSecondary,
        fontSize: 16,
        fontWeight: '600',
    },
    continueButton: {
        flex: 2,
        backgroundColor: colors.primary,
        paddingVertical: 16,
        borderRadius: 14,
        alignItems: 'center',
        shadowColor: colors.primary,
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.3,
        shadowRadius: 8,
        elevation: 4,
    },
    continueButtonText: {
        color: colors.textInverse,
        fontSize: 16,
        fontWeight: '700',
    },
    loadingContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center'
    },
    loadingText: {
        marginTop: 16,
        fontSize: 18,
        fontWeight: '600',
        color: colors.textPrimary
    },
    loadingSubtext: {
        marginTop: 8,
        fontSize: 14,
        color: colors.textSecondary,
    },
    errorContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20
    },
    errorIcon: {
        fontSize: 48,
        marginBottom: 16,
    },
    errorText: {
        fontSize: 18,
        color: colors.error,
        marginBottom: 16
    },
    retryButton: {
        backgroundColor: colors.primary,
        paddingHorizontal: 24,
        paddingVertical: 12,
        borderRadius: 8,
        marginBottom: 12,
    },
    retryButtonText: {
        color: colors.textInverse,
        fontSize: 16,
        fontWeight: '600'
    },
});

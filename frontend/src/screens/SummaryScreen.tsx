import React, { useState } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    ActivityIndicator,
    ScrollView,
    Linking,
} from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { useNavigation, useRoute } from '../navigation/ManualRouter';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { learningClient } from '../services/api';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';

type TabType = 'summary' | 'content';

export const SummaryScreen = () => {
    const navigation = useNavigation();
    const route = useRoute();
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
    const { materialId, title } = route.params as { materialId: string; title: string };
    const displayTitle = title || 'Material Summary';
    const [activeTab, setActiveTab] = useState<TabType>('summary');

    const { data, isLoading, error, refetch } = useQuery({
        queryKey: ['materialSummary', materialId],
        queryFn: async () => {
            console.log(`[SummaryScreen] Fetching summary for material: ${materialId}`);
            try {
                const response = await learningClient.getMaterialSummary({ materialId });
                console.log('[SummaryScreen] Got summary, length:', response.summary?.length || 0);
                console.log('[SummaryScreen] Got content, length:', response.content?.length || 0);
                console.log('[SummaryScreen] Material type:', response.materialType);
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

    const handleOpenLink = () => {
        const url = data?.sourceUrl || data?.content;
        if (url) {
            Linking.openURL(url).catch(err => {
                console.error('[SummaryScreen] Failed to open URL:', err);
            });
        }
    };

    const renderContent = () => {
        if (!data?.content) {
            return <Text style={styles.contentText}>No content available.</Text>;
        }

        const materialType = data.materialType || 'TEXT';

        switch (materialType) {
            case 'LINK':
            case 'YOUTUBE':
                return (
                    <View style={styles.linkContainer}>
                        <Text style={styles.linkIcon}>
                            {materialType === 'YOUTUBE' ? '📺' : '🔗'}
                        </Text>
                        <Text style={styles.linkLabel}>
                            {materialType === 'YOUTUBE' ? 'YouTube Video' : 'External Link'}
                        </Text>
                        <TouchableOpacity style={styles.openLinkButton} onPress={handleOpenLink}>
                            <Text style={styles.openLinkButtonText}>
                                Open {materialType === 'YOUTUBE' ? 'Video' : 'Link'} ↗
                            </Text>
                        </TouchableOpacity>
                        {data.sourceUrl && (
                            <Text style={styles.urlText} numberOfLines={2}>
                                {data.sourceUrl}
                            </Text>
                        )}
                    </View>
                );
            case 'IMAGE':
                return (
                    <View>
                        <Text style={styles.imageLabel}>Extracted text from image:</Text>
                        <View style={styles.imageContentBox}>
                            <Text style={styles.contentText}>{data.content}</Text>
                        </View>
                    </View>
                );
            case 'TEXT':
            default:
                return <Text style={styles.contentText}>{data.content}</Text>;
        }
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
        <View style={[styles.container, { paddingBottom: insets.bottom + 80 }]}>
            <AppHeader />
            <View style={styles.contentContainer}>
                <Text style={styles.headerTitle} numberOfLines={2}>{displayTitle}</Text>
                
                {/* Tab Selector */}
                <View style={styles.tabContainer}>
                    <TouchableOpacity 
                        style={[styles.tab, activeTab === 'summary' && styles.activeTab]}
                        onPress={() => setActiveTab('summary')}
                    >
                        <Text style={[styles.tabText, activeTab === 'summary' && styles.activeTabText]}>
                            📖 Summary
                        </Text>
                    </TouchableOpacity>
                    <TouchableOpacity 
                        style={[styles.tab, activeTab === 'content' && styles.activeTab]}
                        onPress={() => setActiveTab('content')}
                    >
                        <Text style={[styles.tabText, activeTab === 'content' && styles.activeTabText]}>
                            📄 Original
                        </Text>
                    </TouchableOpacity>
                </View>

                <ScrollView
                    style={styles.summaryScroll}
                    contentContainerStyle={styles.summaryContent}
                    showsVerticalScrollIndicator={true}
                >
                    {activeTab === 'summary' ? (
                        <Text style={styles.summaryText}>
                            {data?.summary || 'No summary available.'}
                        </Text>
                    ) : (
                        renderContent()
                    )}
                </ScrollView>

                <View style={styles.buttonContainer}>
                    <TouchableOpacity style={styles.skipButtonBottom} onPress={handleSkip}>
                        <Text style={styles.skipButtonTextBottom}>Skip</Text>
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
        paddingHorizontal: 16,
        paddingVertical: 16
    },
    headerTitle: {
        fontSize: 22,
        fontWeight: '700',
        color: colors.textPrimary,
        textAlign: 'center',
        marginBottom: 16
    },
    tabContainer: {
        flexDirection: 'row',
        backgroundColor: colors.cardAlt,
        borderRadius: 12,
        padding: 4,
        marginBottom: 16,
    },
    tab: {
        flex: 1,
        paddingVertical: 10,
        paddingHorizontal: 16,
        borderRadius: 10,
        alignItems: 'center',
    },
    activeTab: {
        backgroundColor: colors.primary,
    },
    tabText: {
        fontSize: 14,
        fontWeight: '600',
        color: colors.textSecondary,
    },
    activeTabText: {
        color: colors.textInverse,
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
    contentText: {
        fontSize: 15,
        lineHeight: 24,
        color: colors.textPrimary,
    },
    linkContainer: {
        alignItems: 'center',
        paddingVertical: 20,
    },
    linkIcon: {
        fontSize: 48,
        marginBottom: 12,
    },
    linkLabel: {
        fontSize: 16,
        color: colors.textSecondary,
        marginBottom: 16,
    },
    openLinkButton: {
        backgroundColor: colors.primary,
        paddingHorizontal: 24,
        paddingVertical: 12,
        borderRadius: 25,
        marginBottom: 16,
    },
    openLinkButtonText: {
        color: colors.textInverse,
        fontSize: 16,
        fontWeight: '600',
    },
    urlText: {
        fontSize: 12,
        color: colors.textSecondary,
        textAlign: 'center',
        paddingHorizontal: 20,
    },
    imageLabel: {
        fontSize: 14,
        color: colors.textSecondary,
        marginBottom: 12,
        textAlign: 'center',
    },
    imageContentBox: {
        backgroundColor: colors.cardAlt,
        borderRadius: 12,
        padding: 16,
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
    skipButtonBottom: {
        flex: 1,
        backgroundColor: colors.cardAlt,
        paddingVertical: 16,
        borderRadius: 14,
        alignItems: 'center',
    },
    skipButtonTextBottom: {
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

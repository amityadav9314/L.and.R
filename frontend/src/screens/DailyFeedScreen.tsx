import React, { useState } from 'react';
import {
    View, Text, StyleSheet, TouchableOpacity, ScrollView,
    FlatList, Linking, ActivityIndicator
} from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useTheme, ThemeColors } from '../utils/theme';
import { AppHeader } from '../components/AppHeader';
import { feedClient, learningClient } from '../services/api';
import { Article } from '../../proto/backend/proto/feed/feed';
import { useNavigation } from '../navigation/ManualRouter';

export const DailyFeedScreen = () => {
    const queryClient = useQueryClient();
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
    const { navigate } = useNavigation();
    const styles = createStyles(colors);

    // State
    const [selectedDate, setSelectedDate] = useState(new Date().toISOString().split('T')[0]); // YYYY-MM-DD
    const [activeTab, setActiveTab] = useState<'google' | 'tavily'>('google');
    const [addingToRevise, setAddingToRevise] = useState<Record<string, boolean>>({});
    const [readArticles, setReadArticles] = useState<Record<string, boolean>>({});

    // React Query: Feed preferences (cached but refetches on invalidation)
    const { data: feedPrefs, isLoading: prefsLoading } = useQuery({
        queryKey: ['feedPreferences'],
        queryFn: () => feedClient.getFeedPreferences(),
        staleTime: 0, // Always refetch when navigating back after invalidation
        gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes for background
        refetchOnMount: 'always', // Force refetch every time screen mounts
        retry: 1,
    });

    const feedEnabled = feedPrefs?.feedEnabled ?? null;

    // React Query: Daily articles (cached per date)
    const { data: feedData, isLoading: articlesLoading } = useQuery({
        queryKey: ['dailyFeed', selectedDate],
        queryFn: () => feedClient.getDailyFeed({ date: selectedDate }),
        enabled: feedEnabled === true, // Only fetch if feed is enabled
        staleTime: 5 * 60 * 1000, // Cache for 5 minutes
        retry: 1,
    });

    // Filter articles by provider
    const allArticles: Article[] = feedData?.articles || [];
    const articles = allArticles.filter(a => (a.provider || 'tavily') === activeTab);

    const loading = prefsLoading || (feedEnabled === true && articlesLoading);

    // Generate last 7 days for quick navigation
    const getLast7Days = () => {
        const days = [];
        for (let i = 0; i < 7; i++) {
            const date = new Date();
            date.setDate(date.getDate() - i);
            days.push(date.toISOString().split('T')[0]);
        }
        return days;
    };

    const formatDateLabel = (dateStr: string) => {
        const date = new Date(dateStr);
        const today = new Date().toISOString().split('T')[0];
        const yesterday = new Date(Date.now() - 86400000).toISOString().split('T')[0];

        if (dateStr === today) return 'Today';
        if (dateStr === yesterday) return 'Yesterday';
        return date.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
    };

    const openArticle = async (url: string) => {
        try {
            await WebBrowser.openBrowserAsync(url, {
                toolbarColor: colors.primary,
                enableBarCollapsing: true,
                showTitle: true,
            });
            // Mark as read after browser is closed
            setReadArticles((prev: Record<string, boolean>) => ({ ...prev, [url]: true }));
        } catch (error) {
            console.error('Failed to open browser:', error);
            Linking.openURL(url);
        }
    };

    const handleAddForRevise = async (url: string, title: string) => {
        setAddingToRevise((prev: Record<string, boolean>) => ({ ...prev, [url]: true }));
        try {
            await learningClient.addMaterial({
                type: 'LINK',
                content: url,
                existingTags: ['daily-feed'],
                imageData: '',
            });
            // Success - keep loading as false, but maybe we could show a checkmark later
            alert(`Added "${title}" for revision! 📚`);
            setReadArticles((prev: Record<string, boolean>) => ({ ...prev, [url]: true })); // Ensure it stays "read"

            // Invalidate feed query to show updated "isAdded" status next time
            queryClient.invalidateQueries({ queryKey: ['dailyFeed'] });

            navigate('Home');
        } catch (error) {
            console.error('Failed to add for revise:', error);
            alert('Failed to add material. Please try again.');
        } finally {
            setAddingToRevise((prev: Record<string, boolean>) => ({ ...prev, [url]: false }));
        }
    };

    const renderArticle = ({ item }: { item: Article }) => (
        <TouchableOpacity style={styles.articleCard} onPress={() => openArticle(item.url)}>
            <Text style={styles.articleTitle}>{item.title}</Text>
            <Text style={styles.articleSnippet} numberOfLines={3}>{item.snippet}</Text>
            <View style={styles.articleFooter}>
                <View style={styles.footerLeft}>
                    <Text style={styles.articleScore}>
                        {Math.round((item.relevanceScore || 0) * 100)}% match
                    </Text>
                    {item.isAdded ? (
                        <View style={[styles.reviseButton, styles.reviseButtonDisabled]}>
                            <Text style={styles.reviseButtonText}>Added ✅</Text>
                        </View>
                    ) : (
                        <TouchableOpacity
                            style={[
                                styles.reviseButton,
                                (addingToRevise[item.url] || !readArticles[item.url]) && styles.reviseButtonDisabled
                            ]}
                            onPress={() => handleAddForRevise(item.url, item.title)}
                            disabled={addingToRevise[item.url] || !readArticles[item.url]}
                        >
                            {addingToRevise[item.url] ? (
                                <ActivityIndicator size="small" color="#fff" />
                            ) : (
                                <Text style={styles.reviseButtonText}>
                                    {readArticles[item.url] ? 'Revise 📚' : 'Read first'}
                                </Text>
                            )}
                        </TouchableOpacity>
                    )}
                </View>
                <TouchableOpacity onPress={() => openArticle(item.url)}>
                    <Text style={styles.readMore}>Read More →</Text>
                </TouchableOpacity>
            </View>
        </TouchableOpacity>
    );

    return (
        <View style={styles.container}>
            <ScrollView contentContainerStyle={[styles.content, { paddingBottom: insets.bottom + 80 }]}>
                <AppHeader />
                <Text style={styles.title}>Daily Feed 📰</Text>

                {/* Show loading while checking preferences */}
                {feedEnabled === null && (
                    <ActivityIndicator color={colors.primary} size="large" style={{ marginTop: 40 }} />
                )}

                {/* Feed Disabled State */}
                {feedEnabled === false && (
                    <View style={styles.disabledState}>
                        <Text style={styles.disabledIcon}>🔕</Text>
                        <Text style={styles.disabledTitle}>Daily Feed is Off</Text>
                        <Text style={styles.disabledHint}>
                            Enable the Daily AI Feed in Settings to get personalized article recommendations every day.
                        </Text>
                        <TouchableOpacity
                            style={styles.settingsButton}
                            onPress={() => navigate('Settings')}
                        >
                            <Text style={styles.settingsButtonText}>Go to Settings ⚙️</Text>
                        </TouchableOpacity>
                    </View>
                )}

                {/* Feed Enabled - Show Content */}
                {feedEnabled === true && (
                    <>
                        {/* Date Navigation */}
                        <View style={styles.dateScrollerContainer}>
                            <ScrollView
                                horizontal
                                showsHorizontalScrollIndicator={false}
                                style={styles.dateScroller}
                                contentContainerStyle={styles.dateScrollerContent}
                            >
                                {getLast7Days().map((date) => (
                                    <TouchableOpacity
                                        key={date}
                                        style={[
                                            styles.dateButton,
                                            selectedDate === date && styles.dateButtonActive
                                        ]}
                                        onPress={() => setSelectedDate(date)}
                                    >
                                        <Text style={[
                                            styles.dateButtonText,
                                            selectedDate === date && styles.dateButtonTextActive
                                        ]}>
                                            {formatDateLabel(date)}
                                        </Text>
                                    </TouchableOpacity>
                                ))}
                            </ScrollView>
                        </View>

                        {/* Provider Tabs */}
                        <View style={styles.tabContainer}>
                            <TouchableOpacity
                                style={[styles.tab, activeTab === 'google' && styles.activeTab]}
                                onPress={() => setActiveTab('google')}
                            >
                                <Text style={[styles.tabText, activeTab === 'google' && styles.activeTabText]}>
                                    Google
                                </Text>
                            </TouchableOpacity>
                            <TouchableOpacity
                                style={[styles.tab, activeTab === 'tavily' && styles.activeTab]}
                                onPress={() => setActiveTab('tavily')}
                            >
                                <Text style={[styles.tabText, activeTab === 'tavily' && styles.activeTabText]}>
                                    Tavily
                                </Text>
                            </TouchableOpacity>
                        </View>

                        {/* Articles List */}
                        <View style={styles.articlesSection}>
                            <Text style={styles.sectionTitle}>
                                {activeTab === 'google' ? 'Google' : 'Tavily'} Results - {formatDateLabel(selectedDate)}
                            </Text>

                            {loading ? (
                                <ActivityIndicator color={colors.primary} size="large" style={{ marginTop: 20 }} />
                            ) : articles.length === 0 ? (
                                <View style={styles.emptyState}>
                                    <Text style={styles.emptyIcon}>📭</Text>
                                    <Text style={styles.emptyText}>No articles from {activeTab === 'google' ? 'Google' : 'Tavily'}</Text>
                                    <Text style={styles.emptyHint}>
                                        Try refreshing or checking another date. Make sure your interests are set in Settings.
                                    </Text>
                                </View>
                            ) : (
                                <FlatList
                                    data={articles}
                                    renderItem={renderArticle}
                                    keyExtractor={(item) => item.id}
                                    scrollEnabled={false}
                                />
                            )}
                        </View>
                    </>
                )}
            </ScrollView>
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
    },
    content: {
        paddingHorizontal: 3,
    },
    title: {
        fontSize: 28,
        fontWeight: 'bold',
        color: colors.textPrimary,
        marginBottom: 10,
        paddingTop: 16,
    },
    dateScrollerContainer: {
        height: 50,
        marginBottom: 4,
    },
    dateScroller: {
        flexGrow: 0,
    },
    dateScrollerContent: {
        alignItems: 'center',
    },
    dateButton: {
        paddingHorizontal: 14,
        paddingVertical: 8,
        borderRadius: 20,
        backgroundColor: colors.card,
        marginRight: 8,
        borderWidth: 1,
        borderColor: colors.border,
        height: 36,
    },
    dateButtonActive: {
        backgroundColor: colors.primary,
        borderColor: colors.primary,
    },
    dateButtonText: {
        fontSize: 13,
        color: colors.textSecondary,
        fontWeight: '500',
    },
    dateButtonTextActive: {
        color: '#fff',
    },
    tabContainer: {
        flexDirection: 'row',
        backgroundColor: colors.card,
        borderRadius: 12,
        padding: 4,
        marginBottom: 12,
        marginHorizontal: 10,
        borderWidth: 1,
        borderColor: colors.border,
    },
    tab: {
        flex: 1,
        paddingVertical: 10,
        alignItems: 'center',
        borderRadius: 10,
    },
    activeTab: {
        backgroundColor: colors.primary,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.1,
        shadowRadius: 1,
    },
    tabText: {
        fontSize: 14,
        fontWeight: '600',
        color: colors.textSecondary,
    },
    activeTabText: {
        color: '#fff',
    },
    articlesSection: {
        flex: 1,
    },
    sectionTitle: {
        fontSize: 16,
        fontWeight: '600',
        color: colors.textPrimary,
        marginBottom: 8,
    },
    articleCard: {
        backgroundColor: colors.card,
        borderRadius: 12,
        padding: 14,
        marginBottom: 12,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.1,
        shadowRadius: 2,
    },
    articleTitle: {
        fontSize: 16,
        fontWeight: 'bold',
        color: colors.textPrimary,
        marginBottom: 6,
    },
    articleSnippet: {
        fontSize: 14,
        color: colors.textSecondary,
        lineHeight: 20,
        marginBottom: 10,
    },
    articleFooter: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginTop: 4,
    },
    footerLeft: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    articleScore: {
        fontSize: 12,
        color: colors.textSecondary,
        marginRight: 12,
    },
    reviseButton: {
        backgroundColor: colors.primary,
        paddingHorizontal: 12,
        paddingVertical: 6,
        borderRadius: 15,
        minWidth: 80,
        alignItems: 'center',
    },
    reviseButtonDisabled: {
        opacity: 0.7,
    },
    reviseButtonText: {
        color: '#fff',
        fontSize: 12,
        fontWeight: 'bold',
    },
    readMore: {
        fontSize: 13,
        color: colors.primary,
        fontWeight: '600',
    },
    emptyState: {
        alignItems: 'center',
        paddingVertical: 40,
    },
    emptyIcon: {
        fontSize: 48,
        marginBottom: 12,
    },
    emptyText: {
        fontSize: 18,
        fontWeight: '600',
        color: colors.textPrimary,
        marginBottom: 8,
    },
    emptyHint: {
        fontSize: 14,
        color: colors.textSecondary,
        textAlign: 'center',
        paddingHorizontal: 20,
    },
    disabledState: {
        alignItems: 'center',
        paddingVertical: 60,
        paddingHorizontal: 20,
    },
    disabledIcon: {
        fontSize: 56,
        marginBottom: 16,
    },
    disabledTitle: {
        fontSize: 22,
        fontWeight: 'bold',
        color: colors.textPrimary,
        marginBottom: 12,
    },
    disabledHint: {
        fontSize: 15,
        color: colors.textSecondary,
        textAlign: 'center',
        lineHeight: 22,
        marginBottom: 24,
    },
    settingsButton: {
        backgroundColor: colors.primary,
        paddingHorizontal: 24,
        paddingVertical: 12,
        borderRadius: 25,
    },
    settingsButtonText: {
        color: '#fff',
        fontSize: 16,
        fontWeight: '600',
    },
});

import React, { useState, useEffect } from 'react';
import {
    View, Text, StyleSheet, TouchableOpacity, ScrollView,
    FlatList, Linking, ActivityIndicator
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useTheme, ThemeColors } from '../utils/theme';
import { AppHeader } from '../components/AppHeader';
import { feedClient } from '../services/api';
import { Article } from '../../proto/backend/proto/feed/feed';
import { useNavigation } from '../navigation/ManualRouter';

export const DailyFeedScreen = () => {
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
    const { navigate } = useNavigation();
    const styles = createStyles(colors);

    // State
    const [selectedDate, setSelectedDate] = useState(new Date().toISOString().split('T')[0]); // YYYY-MM-DD
    const [articles, setArticles] = useState<Article[]>([]);
    const [loading, setLoading] = useState(true);
    const [feedEnabled, setFeedEnabled] = useState<boolean | null>(null); // null = loading

    // Check if feed is enabled
    useEffect(() => {
        const checkFeedEnabled = async () => {
            try {
                const prefs = await feedClient.getFeedPreferences();
                setFeedEnabled(prefs.feedEnabled);
            } catch (error) {
                console.log('Failed to check feed preferences:', error);
                setFeedEnabled(false);
            }
        };
        checkFeedEnabled();
    }, []);

    // Load articles for selected date (only if feed is enabled)
    useEffect(() => {
        if (feedEnabled !== true) return;

        const loadArticles = async () => {
            setLoading(true);
            try {
                const response = await feedClient.getDailyFeed({ date: selectedDate });
                setArticles(response.articles || []);
            } catch (error) {
                console.log('Failed to load articles:', error);
                setArticles([]);
            } finally {
                setLoading(false);
            }
        };
        loadArticles();
    }, [selectedDate, feedEnabled]);

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

    const openArticle = (url: string) => {
        Linking.openURL(url);
    };

    const renderArticle = ({ item }: { item: Article }) => (
        <TouchableOpacity style={styles.articleCard} onPress={() => openArticle(item.url)}>
            <Text style={styles.articleTitle}>{item.title}</Text>
            <Text style={styles.articleSnippet} numberOfLines={3}>{item.snippet}</Text>
            <View style={styles.articleFooter}>
                <Text style={styles.articleScore}>
                    Relevance: {Math.round((item.relevanceScore || 0) * 100)}%
                </Text>
                <Text style={styles.readMore}>Read More →</Text>
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

                        {/* Articles List */}
                        <View style={styles.articlesSection}>
                            <Text style={styles.sectionTitle}>
                                Articles for {formatDateLabel(selectedDate)}
                            </Text>

                            {loading ? (
                                <ActivityIndicator color={colors.primary} size="large" style={{ marginTop: 20 }} />
                            ) : articles.length === 0 ? (
                                <View style={styles.emptyState}>
                                    <Text style={styles.emptyIcon}>📭</Text>
                                    <Text style={styles.emptyText}>No articles for this date</Text>
                                    <Text style={styles.emptyHint}>
                                        Enable Daily Feed in Settings and configure your interests to get personalized articles.
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
        marginBottom: 16,
        paddingTop: 16,
    },
    dateScrollerContainer: {
        height: 50,
        marginBottom: 20,
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
    articlesSection: {
        flex: 1,
    },
    sectionTitle: {
        fontSize: 16,
        fontWeight: '600',
        color: colors.textPrimary,
        marginBottom: 12,
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
    },
    articleScore: {
        fontSize: 12,
        color: colors.textSecondary,
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

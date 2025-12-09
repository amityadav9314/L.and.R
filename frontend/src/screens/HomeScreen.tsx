import React, { useState, useMemo } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, RefreshControl, TextInput, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigation } from '../navigation/ManualRouter';
import { learningClient } from '../services/api';
import { useAuthStore } from '../store/authStore';
import { MaterialSummary } from '../../proto/backend/proto/learning/learning';
import { MATERIALS_PER_PAGE } from '../utils/constants';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';

export const HomeScreen = () => {
    const navigation = useNavigation();
    const { user } = useAuthStore();
    const { colors } = useTheme();
    const queryClient = useQueryClient();

    const deleteMutation = useMutation({
        mutationFn: async (materialId: string) => {
            return await learningClient.deleteMaterial({ materialId });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['dueMaterials'] });
            queryClient.invalidateQueries({ queryKey: ['allTags'] });
            queryClient.invalidateQueries({ queryKey: ['notificationStatus'] });
        },
    });

    const handleDelete = (materialId: string, title: string) => {
        Alert.alert(
            'Delete Material',
            `Are you sure you want to delete "${title}"?`,
            [
                { text: 'Cancel', style: 'cancel' },
                {
                    text: 'Delete',
                    style: 'destructive',
                    onPress: async () => {
                        try {
                            await deleteMutation.mutateAsync(materialId);
                        } catch (err) {
                            console.error('[HOME] Failed to delete material:', err);
                            Alert.alert('Error', 'Failed to delete material. Please try again.');
                        }
                    },
                },
            ]
        );
    };


    // Filter states
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedTags, setSelectedTags] = useState<string[]>([]);
    const [currentPage, setCurrentPage] = useState(1);
    const [refreshing, setRefreshing] = useState(false);

    const { data: paginatedData, isLoading, error, refetch } = useQuery({
        queryKey: ['dueMaterials', currentPage],
        queryFn: async () => {
            console.log('[HOME] Fetching due materials page:', currentPage);
            try {
                const response = await learningClient.getDueMaterials({
                    page: currentPage,
                    pageSize: MATERIALS_PER_PAGE,
                });
                console.log('[HOME] Got materials:', response.materials?.length || 0, 'of', response.totalCount);
                return response;
            } catch (err) {
                console.error('[HOME] Failed to fetch materials:', err);
                return { materials: [], totalCount: 0, page: 1, pageSize: MATERIALS_PER_PAGE, totalPages: 0 };
            }
        },
    });

    const data = paginatedData?.materials || [];
    const totalCount = paginatedData?.totalCount || 0;
    const totalPages = paginatedData?.totalPages || 0;

    // Fetch notification status (due flashcards count)
    const { data: notificationData } = useQuery({
        queryKey: ['notificationStatus'],
        queryFn: async () => {
            try {
                const response = await learningClient.getNotificationStatus({});
                return response;
            } catch (err) {
                console.error('[HOME] Failed to fetch notification status:', err);
                return { dueFlashcardsCount: 0, hasDueMaterials: false };
            }
        },
        refetchInterval: 60000,
    });

    const dueFlashcardsCount = notificationData?.dueFlashcardsCount || 0;

    // Fetch all tags from backend
    const { data: tagsData } = useQuery({
        queryKey: ['allTags'],
        queryFn: async () => {
            console.log('[HOME] Fetching all tags...');
            try {
                const response = await learningClient.getAllTags({});
                console.log('[HOME] Got tags:', response.tags?.length || 0);
                return response.tags || [];
            } catch (err) {
                console.error('[HOME] Failed to fetch tags:', err);
                return [];
            }
        },
    });

    const allTags = tagsData || [];

    // Filter materials based on search and selected tags
    const filteredMaterials = useMemo(() => {
        if (!data) return [];

        return data.filter((material: MaterialSummary) => {
            const matchesSearch = searchQuery.trim() === '' ||
                material.title.toLowerCase().includes(searchQuery.toLowerCase());
            const matchesTags = selectedTags.length === 0 ||
                selectedTags.every(tag => material.tags.includes(tag));
            return matchesSearch && matchesTags;
        });
    }, [data, searchQuery, selectedTags]);

    const handleNextPage = () => {
        if (currentPage < totalPages) {
            setCurrentPage(prev => prev + 1);
        }
    };

    const handlePrevPage = () => {
        if (currentPage > 1) {
            setCurrentPage(prev => prev - 1);
        }
    };

    const toggleTag = (tag: string) => {
        setSelectedTags(prev =>
            prev.includes(tag)
                ? prev.filter(t => t !== tag)
                : [...prev, tag]
        );
    };

    const clearFilters = () => {
        setSearchQuery('');
        setSelectedTags([]);
        setCurrentPage(1);
    };

    const handleRefresh = React.useCallback(async () => {
        setRefreshing(true);
        try {
            await refetch();
        } catch (error) {
            console.error('[HOME] Refresh error:', error);
        } finally {
            setRefreshing(false);
        }
    }, [refetch, refreshing]);

    const hasActiveFilters = searchQuery.trim() !== '' || selectedTags.length > 0;
    const styles = createStyles(colors);

    const renderPaginationFooter = () => {
        if (hasActiveFilters || totalPages <= 1) return null;

        return (
            <View style={styles.paginationContainer}>
                <TouchableOpacity
                    style={[styles.paginationButton, currentPage === 1 && styles.paginationButtonDisabled]}
                    onPress={handlePrevPage}
                    disabled={currentPage === 1}
                >
                    <Text style={[styles.paginationButtonText, currentPage === 1 && styles.paginationButtonTextDisabled]}>
                        Previous
                    </Text>
                </TouchableOpacity>

                <Text style={styles.paginationText}>
                    Page {currentPage} of {totalPages}
                </Text>

                <TouchableOpacity
                    style={[styles.paginationButton, currentPage === totalPages && styles.paginationButtonDisabled]}
                    onPress={handleNextPage}
                    disabled={currentPage === totalPages}
                >
                    <Text style={[styles.paginationButtonText, currentPage === totalPages && styles.paginationButtonTextDisabled]}>
                        Next
                    </Text>
                </TouchableOpacity>
            </View>
        );
    };

    return (
        <View style={styles.container}>
            <AppHeader />

            <ScrollView
                style={styles.scrollView}
                contentContainerStyle={styles.scrollContent}
                showsVerticalScrollIndicator={true}
                refreshControl={
                    <RefreshControl
                        refreshing={refreshing}
                        onRefresh={handleRefresh}
                        colors={[colors.primary]}
                        tintColor={colors.primary}
                        progressBackgroundColor={colors.card}
                    />
                }
            >
                <View style={styles.header}>
                    <Text style={styles.title}>Welcome, {user?.name}</Text>
                </View>

                <View style={styles.titleRow}>
                    <View style={styles.titleWithBadge}>
                        <Text style={styles.mainTitle}>Due for Review</Text>
                        {dueFlashcardsCount > 0 && (
                            <View style={styles.notificationBadge}>
                                <Text style={styles.notificationBadgeText}>{dueFlashcardsCount}</Text>
                            </View>
                        )}
                    </View>
                    <TouchableOpacity
                        style={styles.addButton}
                        onPress={() => navigation.navigate('AddMaterial')}
                    >
                        <Text style={styles.addButtonText}>+ Add Material</Text>
                    </TouchableOpacity>
                </View>

                {/* Search Bar */}
                <View style={styles.searchContainer}>
                    <TextInput
                        style={styles.searchInput}
                        placeholder="Search by title..."
                        placeholderTextColor={colors.textPlaceholder}
                        value={searchQuery}
                        onChangeText={setSearchQuery}
                        clearButtonMode="while-editing"
                    />
                    {hasActiveFilters && (
                        <TouchableOpacity onPress={clearFilters} style={styles.clearButton}>
                            <Text style={styles.clearButtonText}>Clear</Text>
                        </TouchableOpacity>
                    )}
                </View>

                {/* Tag Filter Chips */}
                {allTags.length > 0 && (
                    <View style={styles.tagFilterSection}>
                        <Text style={styles.tagFilterLabel}>Filter by tags:</Text>
                        <ScrollView
                            horizontal
                            showsHorizontalScrollIndicator={false}
                            contentContainerStyle={styles.tagFilterContainer}
                            nestedScrollEnabled={true}
                        >
                            {allTags.map((tag: string) => (
                                <TouchableOpacity
                                    key={tag}
                                    style={[
                                        styles.filterTagChip,
                                        selectedTags.includes(tag) && styles.filterTagChipActive
                                    ]}
                                    onPress={() => toggleTag(tag)}
                                >
                                    <Text style={[
                                        styles.filterTagText,
                                        selectedTags.includes(tag) && styles.filterTagTextActive
                                    ]}>
                                        {tag}
                                    </Text>
                                </TouchableOpacity>
                            ))}
                        </ScrollView>
                    </View>
                )}

                {/* Results Count and Pagination Info */}
                <View style={styles.infoContainer}>
                    {hasActiveFilters ? (
                        <Text style={styles.resultsCount}>
                            {filteredMaterials.length} of {data?.length || 0} materials on this page
                        </Text>
                    ) : (
                        <Text style={styles.resultsCount}>
                            Page {currentPage} of {totalPages} ({totalCount} total materials)
                        </Text>
                    )}
                </View>

                {isLoading ? (
                    <View style={styles.loadingContainer}>
                        <ActivityIndicator size="large" color={colors.primary} />
                        <Text style={styles.loadingText}>Loading materials...</Text>
                    </View>
                ) : error ? (
                    <Text style={styles.error}>Failed to load materials</Text>
                ) : filteredMaterials.length === 0 ? (
                    <Text style={styles.empty}>
                        {hasActiveFilters
                            ? 'No materials match your filters'
                            : 'No materials due! Good job.'}
                    </Text>
                ) : (
                    <>
                        {filteredMaterials.map((item: MaterialSummary) => (
                            <TouchableOpacity
                                key={item.id}
                                style={styles.card}
                                onPress={() => navigation.navigate('Summary', { materialId: item.id, title: item.title })}
                            >
                                <View style={styles.cardHeader}>
                                    <Text style={styles.materialTitle} numberOfLines={2}>{item.title}</Text>
                                    <View style={styles.cardActions}>
                                        <View style={styles.badge}>
                                            <Text style={styles.badgeText}>{item.dueCount}</Text>
                                        </View>
                                        <TouchableOpacity
                                            style={styles.deleteButton}
                                            onPress={(e) => {
                                                e.stopPropagation();
                                                handleDelete(item.id, item.title);
                                            }}
                                        >
                                            <Text style={styles.deleteButtonText}>🗑</Text>
                                        </TouchableOpacity>
                                    </View>
                                </View>

                                <View style={styles.tagsContainer}>
                                    {item.tags.map((tag: string, index: number) => (
                                        <View key={index} style={styles.tagBadge}>
                                            <Text style={styles.tagText}>{tag}</Text>
                                        </View>
                                    ))}
                                </View>
                            </TouchableOpacity>
                        ))}
                        {renderPaginationFooter()}
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
    scrollView: {
        flex: 1,
    },
    scrollContent: {
        paddingHorizontal: 20,
        paddingTop: 20,
        paddingBottom: 20,
        flexGrow: 1,
    },
    header: {
        marginBottom: 20,
    },
    title: {
        fontSize: 20,
        fontWeight: 'bold',
        color: colors.textPrimary,
    },
    titleRow: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 15,
    },
    titleWithBadge: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 10,
    },
    mainTitle: {
        fontSize: 24,
        fontWeight: 'bold',
        color: colors.textPrimary,
    },
    notificationBadge: {
        backgroundColor: colors.notificationBadgeBg,
        borderRadius: 12,
        paddingHorizontal: 8,
        paddingVertical: 4,
        minWidth: 24,
        alignItems: 'center',
        justifyContent: 'center',
    },
    notificationBadgeText: {
        color: colors.textInverse,
        fontSize: 12,
        fontWeight: 'bold',
    },
    addButton: {
        backgroundColor: colors.primary,
        paddingHorizontal: 16,
        paddingVertical: 10,
        borderRadius: 8,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.2,
        shadowRadius: 1.41,
    },
    addButtonText: {
        color: colors.textInverse,
        fontWeight: '600',
        fontSize: 14,
    },
    searchContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 15,
        gap: 10,
    },
    searchInput: {
        flex: 1,
        backgroundColor: colors.card,
        borderRadius: 8,
        paddingHorizontal: 15,
        paddingVertical: 12,
        fontSize: 16,
        borderWidth: 1,
        borderColor: colors.inputBorder,
        color: colors.textPrimary,
    },
    clearButton: {
        backgroundColor: colors.error,
        paddingHorizontal: 15,
        paddingVertical: 12,
        borderRadius: 8,
    },
    clearButtonText: {
        color: colors.textInverse,
        fontWeight: '600',
        fontSize: 14,
    },
    tagFilterSection: {
        marginBottom: 15,
    },
    tagFilterLabel: {
        fontSize: 14,
        fontWeight: '600',
        color: colors.textSecondary,
        marginBottom: 8,
    },
    tagFilterContainer: {
        flexDirection: 'row',
        gap: 8,
        paddingBottom: 5,
    },
    filterTagChip: {
        backgroundColor: colors.filterChipBg,
        paddingHorizontal: 12,
        paddingVertical: 8,
        borderRadius: 16,
        borderWidth: 1.5,
        borderColor: colors.filterChipBorder,
    },
    filterTagChipActive: {
        backgroundColor: colors.tagActiveBg,
        borderColor: colors.tagActiveBg,
    },
    filterTagText: {
        fontSize: 13,
        fontWeight: '600',
        color: colors.textSecondary,
    },
    filterTagTextActive: {
        color: colors.tagActiveText,
    },
    infoContainer: {
        marginBottom: 10,
    },
    resultsCount: {
        fontSize: 13,
        color: colors.textSecondary,
        fontStyle: 'italic',
    },
    loadingContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        paddingVertical: 60,
    },
    loadingText: {
        marginTop: 12,
        fontSize: 16,
        color: colors.textSecondary,
    },
    card: {
        backgroundColor: colors.card,
        padding: 15,
        borderRadius: 10,
        marginBottom: 10,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.2,
        shadowRadius: 1.41,
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 10,
    },
    materialTitle: {
        fontSize: 18,
        fontWeight: '600',
        color: colors.textPrimary,
        flex: 1,
    },
    badge: {
        backgroundColor: colors.badgeBg,
        borderRadius: 12,
        paddingHorizontal: 8,
        paddingVertical: 2,
        marginLeft: 10,
    },
    badgeText: {
        color: colors.badgeText,
        fontWeight: 'bold',
        fontSize: 12,
    },
    cardActions: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
    },
    deleteButton: {
        padding: 8,
        borderRadius: 8,
        backgroundColor: colors.cardAlt,
    },
    deleteButtonText: {
        fontSize: 16,
    },
    tagsContainer: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 8,
    },
    tagBadge: {
        backgroundColor: colors.tagBg,
        paddingHorizontal: 8,
        paddingVertical: 4,
        borderRadius: 12,
    },
    tagText: {
        fontSize: 12,
        color: colors.tagText,
    },
    error: {
        color: colors.error,
        textAlign: 'center',
        marginTop: 20,
    },
    empty: {
        textAlign: 'center',
        color: colors.textSecondary,
        marginTop: 50,
        fontStyle: 'italic',
    },
    paginationContainer: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingVertical: 15,
        paddingHorizontal: 20,
        backgroundColor: colors.card,
        borderTopWidth: 1,
        borderTopColor: colors.border,
        marginTop: 10,
        marginBottom: 10,
        borderRadius: 10,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.2,
        shadowRadius: 1.41,
    },
    paginationButton: {
        backgroundColor: colors.primary,
        paddingHorizontal: 20,
        paddingVertical: 10,
        borderRadius: 8,
        minWidth: 100,
        alignItems: 'center',
    },
    paginationButtonDisabled: {
        backgroundColor: colors.paginationDisabledBg,
    },
    paginationButtonText: {
        color: colors.textInverse,
        fontWeight: '600',
        fontSize: 14,
    },
    paginationButtonTextDisabled: {
        color: colors.paginationDisabledText,
    },
    paginationText: {
        fontSize: 14,
        fontWeight: '600',
        color: colors.textPrimary,
    },
});

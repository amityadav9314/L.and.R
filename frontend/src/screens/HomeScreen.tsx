import React, { useState, useMemo, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, RefreshControl, TextInput, FlatList, ActivityIndicator, Alert } from 'react-native';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigation } from '../navigation/ManualRouter';
import { learningClient } from '../services/api';
import { useAuthStore } from '../store/authStore';
import { MaterialSummary } from '../../proto/backend/proto/learning/learning';
import { MATERIALS_PER_PAGE } from '../utils/constants';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';
import { ScrollView } from 'react-native';

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

    // Infinite scroll query
    const {
        data,
        fetchNextPage,
        hasNextPage,
        isFetchingNextPage,
        isLoading,
        error,
        refetch,
        isRefetching,
    } = useInfiniteQuery({
        queryKey: ['dueMaterials'],
        queryFn: async ({ pageParam = 1 }) => {
            console.log('[HOME] Fetching due materials page:', pageParam);
            try {
                const response = await learningClient.getDueMaterials({
                    page: pageParam,
                    pageSize: MATERIALS_PER_PAGE,
                });
                console.log('[HOME] Got materials:', response.materials?.length || 0, 'of', response.totalCount);
                return response;
            } catch (err) {
                console.error('[HOME] Failed to fetch materials:', err);
                return { materials: [], totalCount: 0, page: pageParam, pageSize: MATERIALS_PER_PAGE, totalPages: 0 };
            }
        },
        getNextPageParam: (lastPage) => {
            if (lastPage.page < lastPage.totalPages) {
                return lastPage.page + 1;
            }
            return undefined;
        },
        initialPageParam: 1,
    });

    // Flatten all pages into a single array
    const allMaterials = useMemo(() => {
        return data?.pages.flatMap(page => page.materials || []) || [];
    }, [data]);

    const totalCount = data?.pages[0]?.totalCount || 0;

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
        return allMaterials.filter((material: MaterialSummary) => {
            const matchesSearch = searchQuery.trim() === '' ||
                material.title.toLowerCase().includes(searchQuery.toLowerCase());
            const matchesTags = selectedTags.length === 0 ||
                selectedTags.every(tag => material.tags.includes(tag));
            return matchesSearch && matchesTags;
        });
    }, [allMaterials, searchQuery, selectedTags]);

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
    };

    const handleRefresh = useCallback(async () => {
        await refetch();
    }, [refetch]);

    const handleLoadMore = useCallback(() => {
        if (hasNextPage && !isFetchingNextPage) {
            console.log('[HOME] Loading more materials...');
            fetchNextPage();
        }
    }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

    const hasActiveFilters = searchQuery.trim() !== '' || selectedTags.length > 0;
    const styles = createStyles(colors);

    const renderMaterialCard = useCallback(({ item }: { item: MaterialSummary }) => (
        <TouchableOpacity
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
    ), [styles, navigation, handleDelete]);

    const renderFooter = () => {
        if (!isFetchingNextPage) return null;
        return (
            <View style={styles.footerLoader}>
                <ActivityIndicator size="small" color={colors.primary} />
                <Text style={styles.footerLoaderText}>Loading more...</Text>
            </View>
        );
    };

    const renderHeader = () => (
        <>
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

            {/* Results Count */}
            <View style={styles.infoContainer}>
                <Text style={styles.resultsCount}>
                    {hasActiveFilters
                        ? `${filteredMaterials.length} of ${allMaterials.length} loaded materials`
                        : `${allMaterials.length} of ${totalCount} materials`}
                </Text>
            </View>
        </>
    );

    const renderEmpty = () => (
        <Text style={styles.empty}>
            {hasActiveFilters
                ? 'No materials match your filters'
                : 'No materials due! Good job.'}
        </Text>
    );

    return (
        <View style={styles.container}>
            <AppHeader />

            {isLoading ? (
                <View style={styles.loadingContainer}>
                    <ActivityIndicator size="large" color={colors.primary} />
                    <Text style={styles.loadingText}>Loading materials...</Text>
                </View>
            ) : error ? (
                <View style={styles.loadingContainer}>
                    <Text style={styles.error}>Failed to load materials</Text>
                </View>
            ) : (
                <FlatList
                    data={filteredMaterials}
                    renderItem={renderMaterialCard}
                    keyExtractor={(item) => item.id}
                    ListHeaderComponent={renderHeader}
                    ListFooterComponent={renderFooter}
                    ListEmptyComponent={renderEmpty}
                    onEndReached={handleLoadMore}
                    onEndReachedThreshold={0.5}
                    refreshControl={
                        <RefreshControl
                            refreshing={isRefetching && !isFetchingNextPage}
                            onRefresh={handleRefresh}
                            colors={[colors.primary]}
                            tintColor={colors.primary}
                            progressBackgroundColor={colors.card}
                        />
                    }
                    contentContainerStyle={styles.listContent}
                    showsVerticalScrollIndicator={true}
                />
            )}
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
    },
    listContent: {
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
    footerLoader: {
        flexDirection: 'row',
        justifyContent: 'center',
        alignItems: 'center',
        paddingVertical: 20,
        gap: 10,
    },
    footerLoaderText: {
        color: colors.textSecondary,
        fontSize: 14,
    },
});

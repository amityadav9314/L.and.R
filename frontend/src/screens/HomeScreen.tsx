import React, { useState, useMemo, useCallback, useRef } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, RefreshControl, TextInput, FlatList, ActivityIndicator, Alert, NativeScrollEvent, NativeSyntheticEvent } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigation } from '../navigation/ManualRouter';
import { learningClient } from '../services/api';
import { useAuthStore } from '../store/authStore';
import { MaterialSummary } from '../../proto/backend/proto/learning/learning';
import { MATERIALS_PER_PAGE } from '../utils/constants';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';
import { ScrollView } from 'react-native';
import { SearchHeader } from '../components/SearchHeader';

export const HomeScreen = () => {
    const navigation = useNavigation();
    const { user } = useAuthStore();
    const { colors } = useTheme();
    const queryClient = useQueryClient();
    const insets = useSafeAreaInsets();

    // Filter states
    const [inputValue, setInputValue] = useState(''); // Local input state
    const [searchQuery, setSearchQuery] = useState(''); // API query state
    const [selectedTags, setSelectedTags] = useState<string[]>([]);

    const handleSearch = useCallback(() => {
        setSearchQuery(inputValue); // Trigger API search
        // Optionally dismiss keyboard here
    }, [inputValue]);

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

    // Backend-driven infinite scroll query
    const {
        data,
        fetchNextPage,
        fetchPreviousPage,
        hasNextPage,
        hasPreviousPage,
        isFetchingNextPage,
        isFetchingPreviousPage,
        isLoading,
        error,
        refetch,
        isRefetching,
    } = useInfiniteQuery({
        queryKey: ['dueMaterials', searchQuery, selectedTags], // Re-fetch when filters change
        queryFn: async ({ pageParam = 1 }) => {
            console.log('[HOME] Fetching due materials page:', pageParam, 'query:', searchQuery, 'tags:', selectedTags);
            try {
                const response = await learningClient.getDueMaterials({
                    page: pageParam,
                    pageSize: MATERIALS_PER_PAGE,
                    searchQuery: searchQuery,
                    tags: selectedTags,
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
        getPreviousPageParam: (firstPage) => {
            if (firstPage.page > 1) {
                return firstPage.page - 1;
            }
            return undefined;
        },
        initialPageParam: 1,
        maxPages: 4,
    });

    const allMaterials = useMemo(() => {
        if (!data?.pages) return [];
        const sortedPages = [...data.pages].sort((a, b) => a - b);
        return sortedPages.flatMap(page => page.materials || []);
    }, [data]);

    const totalCount = data?.pages[0]?.totalCount || 0;

    // Fetch notification status
    const { data: notificationData } = useQuery({
        queryKey: ['notificationStatus'],
        queryFn: async () => {
            try {
                return await learningClient.getNotificationStatus({});
            } catch (err) {
                return { dueFlashcardsCount: 0, hasDueMaterials: false };
            }
        },
        refetchInterval: 60000,
    });

    const dueFlashcardsCount = notificationData?.dueFlashcardsCount || 0;

    // Fetch all tags
    const { data: tagsData } = useQuery({
        queryKey: ['allTags'],
        queryFn: async () => {
            try {
                const response = await learningClient.getAllTags({});
                return response.tags || [];
            } catch (err) {
                return [];
            }
        },
    });

    const allTags = tagsData || [];

    const toggleTag = useCallback((tag: string) => {
        setSelectedTags(prev =>
            prev.includes(tag)
                ? prev.filter(t => t !== tag)
                : [...prev, tag]
        );
    }, []);

    const clearFilters = useCallback(() => {
        setInputValue('');
        setSearchQuery('');
        setSelectedTags([]);
    }, []);

    const handleRefresh = useCallback(async () => {
        await queryClient.resetQueries({ queryKey: ['dueMaterials'] });
    }, [queryClient]);

    // Load more/scroll handlers
    const handleLoadMore = useCallback(() => {
        if (hasNextPage && !isFetchingNextPage) {
            console.log('[HOME] Loading more materials (forward)...');
            fetchNextPage();
        }
    }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

    const handleScroll = useCallback((event: NativeSyntheticEvent<NativeScrollEvent>) => {
        const { contentOffset } = event.nativeEvent;
        if (contentOffset.y < 100 && hasPreviousPage && !isFetchingPreviousPage) {
            console.log('[HOME] Loading more materials (backward)...');
            fetchPreviousPage();
        }
    }, [hasPreviousPage, isFetchingPreviousPage, fetchPreviousPage]);

    const hasActiveFilters = searchQuery.trim() !== '' || selectedTags.length > 0;
    const styles = createStyles(colors);

    // Results text logic
    const resultsText = useMemo(() => {
        if (!data?.pages?.length) return hasActiveFilters ? 'No matches found' : 'No materials due';
        const pages = data.pages.map(p => p.page).sort((a, b) => a - b);
        const minPage = pages[0];
        const maxPage = pages[pages.length - 1];

        if (hasActiveFilters) {
            return `${totalCount} found matching your filters`;
        }
        return `Pages ${minPage}-${maxPage} (${allMaterials.length} of ${totalCount} materials)`;

    }, [data, hasActiveFilters, totalCount, allMaterials.length]);

    const renderMaterialCard = useCallback(({ item }: { item: MaterialSummary }) => (
        <TouchableOpacity
            style={styles.card}
            onPress={() => navigation.navigate('Summary', { materialId: item.id, title: item.title })}
        >
            <View style={styles.cardRow}>
                <View style={styles.cardContent}>
                    <Text style={styles.materialTitle}>{item.title}</Text>
                    <View style={styles.tagsContainer}>
                        {item.tags.map((tag: string, index: number) => (
                            <View key={index} style={styles.tagBadge}>
                                <Text style={styles.tagText}>{tag}</Text>
                            </View>
                        ))}
                    </View>
                </View>
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
        </TouchableOpacity>
    ), [styles, navigation, handleDelete]);

    const renderFooter = useCallback(() => {
        if (!isFetchingNextPage) return null;
        return (
            <View style={styles.footerLoader}>
                <ActivityIndicator size="small" color={colors.primary} />
                <Text style={styles.footerLoaderText}>Loading more...</Text>
            </View>
        );
    }, [isFetchingNextPage, colors, styles]);

    // Use renderHeader but pass stable props to SearchHeader inside it, or extract entirely to separate component in FlatList
    // The key is that ListHeaderComponent should not be an inline function if possible, OR it should be memoized.
    // Better strategy: make SearchHeader a real component and pass it directly to ListHeaderComponent

    // We need a wrapper to inject props since ListHeaderComponent only takes the component class/func, not an element
    // Actually, passing an Element to ListHeaderComponent is fine, but it re-renders if the element is recreated.

    const headerElement = useMemo(() => (
        <React.Fragment>
            <SearchHeader
                user={user}
                dueFlashcardsCount={dueFlashcardsCount}
                searchQuery={searchQuery}
                inputValue={inputValue}
                setInputValue={setInputValue}
                onSearch={handleSearch}
                selectedTags={selectedTags}
                toggleTag={toggleTag}
                allTags={allTags}
                clearFilters={clearFilters}
                hasActiveFilters={searchQuery.trim() !== '' || selectedTags.length > 0}
                resultsText={resultsText}
                isFetchingPreviousPage={isFetchingPreviousPage}
                onAddMaterial={() => navigation.navigate('AddMaterial')}
                colors={colors}
                styles={styles}
            />
            {isFetchingPreviousPage && (
                <View style={styles.headerLoader}>
                    <ActivityIndicator size="small" color={colors.primary} />
                    <Text style={styles.headerLoaderText}>Loading previous...</Text>
                </View>
            )}
        </React.Fragment>
    ), [user, dueFlashcardsCount, searchQuery, inputValue, handleSearch, selectedTags, allTags, resultsText, isFetchingPreviousPage, colors, styles, toggleTag, clearFilters, navigation]);

    const renderEmpty = () => (
        <Text style={styles.empty}>
            {hasActiveFilters
                ? 'No materials match your filters'
                : 'No materials due! Good job.'}
        </Text>
    );

    return (
        <View style={[styles.container, { paddingBottom: insets.bottom }]}>
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
                    data={allMaterials} // Use allMaterials directly as backend does filtering
                    renderItem={renderMaterialCard}
                    keyExtractor={(item) => item.id}
                    ListHeaderComponent={headerElement} // Pass the element directly, not a function
                    ListFooterComponent={renderFooter}
                    ListEmptyComponent={renderEmpty}
                    onEndReached={handleLoadMore}
                    onEndReachedThreshold={0.5}
                    onScroll={handleScroll}
                    scrollEventThrottle={16}
                    maintainVisibleContentPosition={{
                        minIndexForVisible: 0,
                    }}
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
    searchButton: {
        backgroundColor: colors.primary,
        paddingHorizontal: 15,
        paddingVertical: 12,
        borderRadius: 8,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.2,
        shadowRadius: 1.41,
    },
    searchButtonText: {
        color: colors.textInverse,
        fontWeight: 'bold',
        fontSize: 14,
    },
    clearButton: {
        backgroundColor: colors.cardAlt,
        paddingHorizontal: 15,
        paddingVertical: 12,
        borderRadius: 8,
        borderWidth: 1,
        borderColor: colors.border,
    },
    clearButtonText: {
        color: colors.textSecondary,
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
    cardRow: {
        flexDirection: 'row',
    },
    cardContent: {
        flex: 9, // 90%
        marginRight: 10,
    },
    materialTitle: {
        fontSize: 16,
        fontWeight: '600',
        color: colors.textPrimary,
        marginBottom: 8,
    },
    badge: {
        backgroundColor: colors.badgeBg,
        borderRadius: 12,
        paddingHorizontal: 10,
        paddingVertical: 4,
        minWidth: 32,
        alignItems: 'center',
    },
    badgeText: {
        color: colors.badgeText,
        fontWeight: 'bold',
        fontSize: 12,
    },
    cardActions: {
        minWidth: 50,
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 8,
    },
    deleteButton: {
        padding: 6,
        borderRadius: 8,
        backgroundColor: colors.cardAlt,
    },
    deleteButtonText: {
        fontSize: 18,
    },
    tagsContainer: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 6,
        marginTop: 4,
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
    headerLoader: {
        flexDirection: 'row',
        justifyContent: 'center',
        alignItems: 'center',
        paddingVertical: 15,
        gap: 10,
        backgroundColor: colors.cardAlt,
        borderRadius: 8,
        marginBottom: 10,
    },
    headerLoaderText: {
        color: colors.textSecondary,
        fontSize: 14,
    },
});

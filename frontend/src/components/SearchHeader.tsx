import React, { memo } from 'react';
import { View, Text, TouchableOpacity, TextInput, ScrollView, StyleSheet } from 'react-native';
import { ThemeColors } from '../utils/theme';
import { useFilterStore } from '../store/filterStore';
import { useAuthStore } from '../store/authStore';
import { useQuery } from '@tanstack/react-query';
import { learningClient } from '../services/api';

interface SearchHeaderProps {
    resultsText: string;
    isFetchingPreviousPage: boolean;
    colors: ThemeColors;
    styles: any;
}

export const SearchHeader = memo(({
    resultsText,
    isFetchingPreviousPage,
    colors,
    styles
}: SearchHeaderProps) => {
    const {
        onlyDue,
        inputValue, setInputValue,
        searchQuery, setSearchQuery,
        selectedTags, setSelectedTags,
    } = useFilterStore();

    const { user } = useAuthStore();

    const onSearch = () => setSearchQuery(inputValue);
    const clearFilters = () => {
        setInputValue('');
        setSearchQuery('');
        setSelectedTags([]);
    };
    const toggleTag = (tag: string) => {
        setSelectedTags((prev: string[]) =>
            prev.includes(tag) ? prev.filter(t => t !== tag) : [...prev, tag]
        );
    };
    const hasActiveFilters = searchQuery.trim() !== '' || selectedTags.length > 0;

    // Fetch tags
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

    return (
        <View>
            <View style={styles.titleRow}>
                <View style={styles.titleWithBadge}>
                    <Text style={styles.mainTitle}>{onlyDue ? 'Due for Review' : 'All Materials'}</Text>
                    {onlyDue && dueFlashcardsCount > 0 && (
                        <View style={styles.notificationBadge}>
                            <Text style={styles.notificationBadgeText}>{dueFlashcardsCount}</Text>
                        </View>
                    )}
                </View>
            </View>

            {/* Search Bar */}
            <View style={styles.searchContainer}>
                <TextInput
                    style={styles.searchInput}
                    placeholder="Search by title..."
                    placeholderTextColor={colors.textPlaceholder}
                    value={inputValue}
                    onChangeText={setInputValue}
                    onSubmitEditing={onSearch} // Search on enter
                    returnKeyType="search"
                    clearButtonMode="while-editing"
                />

                <TouchableOpacity onPress={onSearch} style={styles.searchButton}>
                    <Text style={styles.searchButtonText}>Search</Text>
                </TouchableOpacity>

                {hasActiveFilters && (
                    <TouchableOpacity onPress={clearFilters} style={styles.clearButton}>
                        <Text style={styles.clearButtonText}>Clear</Text>
                    </TouchableOpacity>
                )}
            </View>

            {/* Tag Filter Chips */}
            {allTags.length > 0 && (() => {
                // Sort tags to show selected ones first
                const sortedTags = [...allTags].sort((a, b) => {
                    const aSelected = selectedTags.includes(a);
                    const bSelected = selectedTags.includes(b);
                    if (aSelected && !bSelected) return -1;
                    if (!aSelected && bSelected) return 1;
                    return 0;
                });

                return (
                    <View style={styles.tagFilterSection}>
                        <Text style={styles.tagFilterLabel}>Filter by tags:</Text>
                        <ScrollView
                            horizontal
                            showsHorizontalScrollIndicator={false}
                            contentContainerStyle={styles.tagFilterContainer}
                            nestedScrollEnabled={true}
                        >
                            {sortedTags.map((tag: string) => (
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
                );
            })()}

            {/* Results Count with Page Range */}
            <View style={styles.infoContainer}>
                <Text style={styles.resultsCount}>
                    {resultsText}
                </Text>
            </View>
        </View>
    );
});

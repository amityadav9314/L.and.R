import React, { memo } from 'react';
import { View, Text, TouchableOpacity, TextInput, ScrollView, StyleSheet } from 'react-native';
import { ThemeColors } from '../utils/theme';

interface SearchHeaderProps {
    user: any;
    dueFlashcardsCount: number;
    searchQuery: string;
    inputValue: string;
    setInputValue: (text: string) => void;
    onSearch: () => void;
    selectedTags: string[];
    toggleTag: (tag: string) => void;
    allTags: string[];
    clearFilters: () => void;
    hasActiveFilters: boolean;
    resultsText: string;
    isFetchingPreviousPage: boolean;
    onAddMaterial: () => void;
    onlyDue: boolean;
    setOnlyDue: (val: boolean) => void;
    colors: ThemeColors;
    styles: any;
}

export const SearchHeader = memo(({
    user,
    dueFlashcardsCount,
    searchQuery, // Used for showing active filter state if needed, or remove if unused
    inputValue,
    setInputValue,
    onSearch,
    selectedTags,
    toggleTag,
    allTags,
    clearFilters,
    hasActiveFilters,
    resultsText,
    isFetchingPreviousPage,
    onAddMaterial,
    onlyDue,
    setOnlyDue,
    colors,
    styles
}: SearchHeaderProps) => {
    return (
        <View>
            <View style={styles.header}>
                <Text style={styles.title}>Welcome, {user?.name}</Text>
            </View>

            <View style={styles.titleRow}>
                <View style={styles.titleWithBadge}>
                    <Text style={styles.mainTitle}>{onlyDue ? 'Due for Review' : 'All Materials'}</Text>
                    {onlyDue && dueFlashcardsCount > 0 && (
                        <View style={styles.notificationBadge}>
                            <Text style={styles.notificationBadgeText}>{dueFlashcardsCount}</Text>
                        </View>
                    )}
                </View>
                <View style={{ flexDirection: 'row', gap: 10 }}>
                    <TouchableOpacity
                        style={[styles.toggleButton, !onlyDue && styles.toggleButtonActive]}
                        onPress={() => setOnlyDue(!onlyDue)}
                    >
                        <Text style={[styles.toggleButtonText, !onlyDue && styles.toggleButtonTextActive]}>
                            {onlyDue ? 'View All' : 'Show Due'}
                        </Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={styles.addButton}
                        onPress={onAddMaterial}
                    >
                        <Text style={styles.addButtonText}>+ Add</Text>
                    </TouchableOpacity>
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

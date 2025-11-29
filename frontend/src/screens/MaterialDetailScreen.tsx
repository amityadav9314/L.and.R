import React from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, ActivityIndicator, RefreshControl } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { learningClient } from '../services/api';
import { Flashcard } from '../../proto/backend/proto/learning/learning';
import { AppHeader } from '../components/AppHeader';

type RootStackParamList = {
    MaterialDetail: { materialId: string; title: string };
    Review: { flashcardId: string };
};

type MaterialDetailScreenRouteProp = RouteProp<RootStackParamList, 'MaterialDetail'>;
type MaterialDetailScreenNavigationProp = NativeStackNavigationProp<RootStackParamList, 'MaterialDetail'>;

export const MaterialDetailScreen = () => {
    const navigation = useNavigation<MaterialDetailScreenNavigationProp>();
    const route = useRoute<MaterialDetailScreenRouteProp>();
    const { materialId, title } = route.params;
    const displayTitle = title || 'Material Details';

    const { data, isLoading, error, refetch } = useQuery({
        queryKey: ['dueFlashcards', materialId],
        queryFn: async () => {
            console.log(`[MaterialDetail] Fetching flashcards for material: ${materialId}`);
            try {
                const response = await learningClient.getDueFlashcards({ materialId });
                console.log('[MaterialDetail] Got flashcards:', response.flashcards?.length || 0);
                return response.flashcards;
            } catch (err) {
                console.error('[MaterialDetail] Failed to fetch flashcards:', err);
                return [];
            }
        },
    });

    const renderItem = ({ item }: { item: Flashcard }) => (
        <TouchableOpacity
            style={styles.card}
            onPress={() => navigation.navigate('Review', { flashcardId: item.id })}
        >
            <Text style={styles.question}>{item.question}</Text>
            <Text style={styles.stage}>Stage: {item.stage}</Text>
        </TouchableOpacity>
    );

    return (
        <View style={styles.container}>
            <AppHeader />
            <View style={styles.contentContainer}>
                <Text style={styles.headerTitle}>{displayTitle}</Text>

                {error ? (
                    <Text style={styles.error}>Failed to load flashcards</Text>
                ) : (
                    <FlatList
                        data={data}
                        renderItem={renderItem}
                        keyExtractor={(item) => item.id}
                        contentContainerStyle={styles.list}
                        refreshControl={
                            <RefreshControl refreshing={isLoading} onRefresh={refetch} />
                        }
                        ListEmptyComponent={
                            <Text style={styles.empty}>No flashcards due for this material!</Text>
                        }
                    />
                )}
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#f5f5f5',
    },
    contentContainer: {
        flex: 1,
        padding: 20,
    },
    headerTitle: {
        fontSize: 22,
        fontWeight: 'bold',
        marginBottom: 20,
        color: '#333',
    },
    list: {
        paddingBottom: 20,
    },
    card: {
        backgroundColor: '#fff',
        padding: 15,
        borderRadius: 10,
        marginBottom: 10,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.2,
        shadowRadius: 1.41,
    },
    question: {
        fontSize: 16,
        color: '#333',
        marginBottom: 5,
    },
    stage: {
        fontSize: 12,
        color: '#888',
    },
    error: {
        color: 'red',
        textAlign: 'center',
        marginTop: 20,
    },
    empty: {
        textAlign: 'center',
        color: '#888',
        marginTop: 50,
        fontStyle: 'italic',
    },
});

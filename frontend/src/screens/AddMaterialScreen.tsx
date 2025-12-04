import React, { useState } from 'react';
import { View, Text, StyleSheet, TextInput, TouchableOpacity, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { useMutation, useQueryClient } from '@tanstack/react-query';
// import { useNavigation } from '@react-navigation/native';
import { useNavigation } from '../navigation/ManualRouter';
import { learningClient } from '../services/api';
import { AppHeader } from '../components/AppHeader';

export const AddMaterialScreen = () => {
    const navigation = useNavigation();
    const queryClient = useQueryClient();
    const [content, setContent] = useState('');
    const [type, setType] = useState<'TEXT' | 'LINK'>('TEXT');

    const mutation = useMutation({
        mutationFn: async () => {
            return await learningClient.addMaterial({
                type,
                content,
            });
        },
        onSuccess: (data) => {
            console.log('[ADD_MATERIAL] Success! Created flashcards:', data.flashcardsCreated);
            console.log('[ADD_MATERIAL] Material title:', data.title);
            console.log('[ADD_MATERIAL] Tags:', data.tags);

            Alert.alert('Success', `Created ${data.flashcardsCreated} flashcards for "${data.title}"!`);

            // Invalidate and refetch all queries to refresh the UI
            queryClient.invalidateQueries({ queryKey: ['dueMaterials'], refetchType: 'all' });
            queryClient.invalidateQueries({ queryKey: ['dueFlashcards'], refetchType: 'all' });
            queryClient.invalidateQueries({ queryKey: ['allTags'], refetchType: 'all' });
            queryClient.invalidateQueries({ queryKey: ['notificationStatus'], refetchType: 'all' });

            navigation.goBack();
        },
        onError: (error) => {
            console.error(error);
            Alert.alert('Error', 'Failed to add material. Please try again.');
        },
    });

    const handleSubmit = () => {
        if (!content.trim()) {
            Alert.alert('Validation', 'Please enter some content');
            return;
        }
        mutation.mutate();
    };

    return (
        <View style={styles.container}>
            <AppHeader />
            <View style={styles.contentContainer}>
                <Text style={styles.title}>Add New Material</Text>

                <View style={styles.typeContainer}>
                    <TouchableOpacity
                        style={[styles.typeButton, type === 'TEXT' && styles.activeType]}
                        onPress={() => setType('TEXT')}
                    >
                        <Text style={[styles.typeText, type === 'TEXT' && styles.activeTypeText]}>Text</Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={[styles.typeButton, type === 'LINK' && styles.activeType]}
                        onPress={() => setType('LINK')}
                    >
                        <Text style={[styles.typeText, type === 'LINK' && styles.activeTypeText]}>Link</Text>
                    </TouchableOpacity>
                </View>

                <TextInput
                    style={styles.input}
                    placeholder={type === 'TEXT' ? "Paste your text here..." : "Enter URL here..."}
                    multiline={type === 'TEXT'}
                    numberOfLines={type === 'TEXT' ? 10 : 1}
                    value={content}
                    onChangeText={setContent}
                    textAlignVertical="top"
                />

                <TouchableOpacity
                    style={styles.submitButton}
                    onPress={handleSubmit}
                    disabled={mutation.isPending}
                >
                    {mutation.isPending ? (
                        <ActivityIndicator color="#fff" />
                    ) : (
                        <Text style={styles.submitText}>Generate Flashcards</Text>
                    )}
                </TouchableOpacity>
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
        backgroundColor: '#fff',
    },
    title: {
        fontSize: 24,
        fontWeight: 'bold',
        marginBottom: 20,
        color: '#333',
    },
    typeContainer: {
        flexDirection: 'row',
        marginBottom: 20,
        backgroundColor: '#f0f0f0',
        borderRadius: 8,
        padding: 4,
    },
    typeButton: {
        flex: 1,
        paddingVertical: 10,
        alignItems: 'center',
        borderRadius: 6,
    },
    activeType: {
        backgroundColor: '#fff',
        elevation: 2,
    },
    typeText: {
        fontWeight: '600',
        color: '#666',
    },
    activeTypeText: {
        color: '#4285F4',
    },
    input: {
        borderWidth: 1,
        borderColor: '#ddd',
        borderRadius: 8,
        padding: 15,
        fontSize: 16,
        marginBottom: 20,
        minHeight: 100,
        backgroundColor: '#fafafa',
    },
    submitButton: {
        backgroundColor: '#4285F4',
        paddingVertical: 15,
        borderRadius: 8,
        alignItems: 'center',
    },
    submitText: {
        color: '#fff',
        fontSize: 16,
        fontWeight: 'bold',
    },
});

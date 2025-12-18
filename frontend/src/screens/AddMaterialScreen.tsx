import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, TextInput, TouchableOpacity, ScrollView, ActivityIndicator, Alert, Image, Platform } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigation } from '../navigation/ManualRouter';
import { learningClient } from '../services/api';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';
import * as ImagePickerUtil from '../utils/imagePicker';
import { useProcessingStore } from '../store/processingStore';

type MaterialType = 'TEXT' | 'LINK' | 'IMAGE' | 'YOUTUBE';

export const AddMaterialScreen = () => {
    const navigation = useNavigation();
    const queryClient = useQueryClient();
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
    const [content, setContent] = useState('');
    const [type, setType] = useState<MaterialType>('TEXT');
    const [imageData, setImageData] = useState<string | null>(null);
    const [imagePreview, setImagePreview] = useState<string | null>(null);

    // Processing state - prevents multiple concurrent submissions
    const { isProcessing, startProcessing, stopProcessing, checkProcessing } = useProcessingStore();

    // Check on mount if there's an active processing
    useEffect(() => {
        checkProcessing();
    }, []);

    const mutation = useMutation({
        mutationFn: async () => {
            console.log('[ADD_MATERIAL] Submitting material, type:', type);
            await startProcessing(); // Mark as processing
            return await learningClient.addMaterial({
                type,
                content: type === 'IMAGE' ? '' : content,
                imageData: type === 'IMAGE' ? imageData || '' : '',
            });
        },
        onSuccess: async (data) => {
            console.log('[ADD_MATERIAL] Success! Created flashcards:', data.flashcardsCreated);
            await stopProcessing(); // Clear processing state
            Alert.alert('Success', `Created ${data.flashcardsCreated} flashcards for "${data.title}"!`);
            queryClient.invalidateQueries({ queryKey: ['dueMaterials'], refetchType: 'all' });
            queryClient.invalidateQueries({ queryKey: ['dueFlashcards'], refetchType: 'all' });
            queryClient.invalidateQueries({ queryKey: ['allTags'], refetchType: 'all' });
            queryClient.invalidateQueries({ queryKey: ['notificationStatus'], refetchType: 'all' });
            navigation.goBack();
        },
        onError: async (error) => {
            console.error('[ADD_MATERIAL] Error:', error);
            await stopProcessing(); // Clear processing state on error too
            Alert.alert('Error', 'Failed to add material. Please try again.');
        },
    });

    const handlePickImage = async () => {
        if (!ImagePickerUtil.isAvailable) {
            Alert.alert('Not Available', 'Image picker is only available on mobile devices.');
            return;
        }
        const result = await ImagePickerUtil.pickImage();
        if (result) {
            setImageData(result.base64);
            setImagePreview(result.uri);
        }
    };

    const handleTakePhoto = async () => {
        if (!ImagePickerUtil.isAvailable) {
            Alert.alert('Not Available', 'Camera is only available on mobile devices.');
            return;
        }
        const result = await ImagePickerUtil.takePhoto();
        if (result) {
            setImageData(result.base64);
            setImagePreview(result.uri);
        }
    };

    const handleSubmit = () => {
        if (type === 'IMAGE') {
            if (!imageData) {
                Alert.alert('Validation', 'Please select or take a photo first');
                return;
            }
        } else {
            if (!content.trim()) {
                Alert.alert('Validation', 'Please enter some content');
                return;
            }
        }
        mutation.mutate();
    };

    const clearImage = () => {
        setImageData(null);
        setImagePreview(null);
    };

    const styles = createStyles(colors);

    return (
        <View style={[styles.container, { paddingBottom: insets.bottom }]}>
            <AppHeader />
            <ScrollView style={styles.scrollView} contentContainerStyle={styles.contentContainer}>
                <Text style={styles.title}>Add New Material</Text>

                {/* Type Selector */}
                <View style={styles.typeContainer}>
                    {(['TEXT', 'LINK', 'IMAGE', 'YOUTUBE'] as MaterialType[]).map((t) => (
                        <TouchableOpacity
                            key={t}
                            style={[styles.typeButton, type === t && styles.activeType]}
                            onPress={() => {
                                setType(t);
                                if (t !== 'IMAGE') clearImage();
                            }}
                        >
                            <Text style={styles.typeIcon}>
                                {t === 'TEXT' ? '📝' : t === 'LINK' ? '🔗' : t === 'IMAGE' ? '📸' : '▶️'}
                            </Text>
                            <Text style={[styles.typeLabel, type === t && styles.activeTypeText]}>
                                {t === 'TEXT' ? 'Text' : t === 'LINK' ? 'Link' : t === 'IMAGE' ? 'Image' : 'YouTube'}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </View>

                {/* Content Input or Image Picker */}
                {type === 'IMAGE' ? (
                    <View style={styles.imageSection}>
                        {imagePreview ? (
                            <View style={styles.imagePreviewContainer}>
                                <Image source={{ uri: imagePreview }} style={styles.imagePreview} />
                                <TouchableOpacity style={styles.clearImageButton} onPress={clearImage}>
                                    <Text style={styles.clearImageText}>✕ Remove</Text>
                                </TouchableOpacity>
                            </View>
                        ) : (
                            <View style={styles.imagePickerButtons}>
                                <TouchableOpacity style={styles.imagePickerButton} onPress={handlePickImage}>
                                    <Text style={styles.imagePickerIcon}>🖼️</Text>
                                    <Text style={styles.imagePickerText}>Gallery</Text>
                                </TouchableOpacity>
                                {Platform.OS !== 'web' && (
                                    <TouchableOpacity style={styles.imagePickerButton} onPress={handleTakePhoto}>
                                        <Text style={styles.imagePickerIcon}>📷</Text>
                                        <Text style={styles.imagePickerText}>Camera</Text>
                                    </TouchableOpacity>
                                )}
                            </View>
                        )}
                        <Text style={styles.hint}>
                            📌 Take a photo of your notes, textbook, or handwritten content
                        </Text>
                    </View>
                ) : (
                    <View>
                        <TextInput
                            style={styles.input}
                            placeholder={
                                type === 'TEXT' ? "Paste your text here..." :
                                    type === 'LINK' ? "Enter URL here..." :
                                        "Paste YouTube URL here..."
                            }
                            placeholderTextColor={colors.textPlaceholder}
                            multiline={type === 'TEXT'}
                            numberOfLines={type === 'TEXT' ? 10 : 1}
                            value={content}
                            onChangeText={setContent}
                            textAlignVertical="top"
                        />
                        {type === 'YOUTUBE' && (
                            <Text style={styles.hint}>
                                ▶️ Enter YouTube video URL to extract transcript and generate flashcards
                            </Text>
                        )}
                    </View>
                )}

                {/* Submit Button */}
                <TouchableOpacity
                    style={[styles.submitButton, (mutation.isPending || isProcessing) && styles.submitButtonDisabled]}
                    onPress={handleSubmit}
                    disabled={mutation.isPending || isProcessing}
                >
                    {mutation.isPending ? (
                        <View style={styles.loadingContainer}>
                            <ActivityIndicator color={colors.textInverse} />
                            <Text style={styles.loadingText}>
                                {type === 'IMAGE' ? 'Extracting text & generating...' : 'Generating flashcards...'}
                            </Text>
                        </View>
                    ) : isProcessing ? (
                        <View style={styles.loadingContainer}>
                            <ActivityIndicator color={colors.textInverse} />
                            <Text style={styles.loadingText}>Processing another material...</Text>
                        </View>
                    ) : (
                        <Text style={styles.submitText}>Generate Flashcards</Text>
                    )}
                </TouchableOpacity>
            </ScrollView>
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: { flex: 1, backgroundColor: colors.background },
    scrollView: { flex: 1 },
    contentContainer: { padding: 20, backgroundColor: colors.card, flexGrow: 1 },
    title: { fontSize: 24, fontWeight: 'bold', marginBottom: 20, color: colors.textPrimary },
    typeContainer: {
        flexDirection: 'row',
        marginBottom: 20,
        backgroundColor: colors.cardAlt,
        borderRadius: 8,
        padding: 4,
    },
    typeButton: { flex: 1, paddingVertical: 10, alignItems: 'center', borderRadius: 6 },
    activeType: { backgroundColor: colors.card, elevation: 2 },
    typeIcon: { fontSize: 24, marginBottom: 4 },
    typeLabel: { fontWeight: '600', color: colors.textSecondary, fontSize: 11 },
    activeTypeText: { color: colors.primary },
    input: {
        borderWidth: 1,
        borderColor: colors.inputBorder,
        borderRadius: 8,
        padding: 15,
        fontSize: 16,
        marginBottom: 20,
        minHeight: 100,
        backgroundColor: colors.input,
        color: colors.textPrimary,
    },
    imageSection: { marginBottom: 20 },
    imagePickerButtons: { flexDirection: 'row', gap: 12, marginBottom: 12 },
    imagePickerButton: {
        flex: 1,
        backgroundColor: colors.cardAlt,
        borderRadius: 12,
        paddingVertical: 30,
        alignItems: 'center',
        borderWidth: 2,
        borderColor: colors.inputBorder,
        borderStyle: 'dashed',
    },
    imagePickerIcon: { fontSize: 32, marginBottom: 8 },
    imagePickerText: { fontSize: 14, color: colors.textSecondary, fontWeight: '600' },
    imagePreviewContainer: { alignItems: 'center', marginBottom: 12 },
    imagePreview: { width: '100%', height: 200, borderRadius: 12, resizeMode: 'cover' },
    clearImageButton: {
        marginTop: 10,
        paddingHorizontal: 16,
        paddingVertical: 8,
        backgroundColor: colors.error,
        borderRadius: 8,
    },
    clearImageText: { color: colors.textInverse, fontWeight: '600' },
    hint: { fontSize: 13, color: colors.textSecondary, textAlign: 'center', fontStyle: 'italic' },
    submitButton: {
        backgroundColor: colors.primary,
        paddingVertical: 15,
        borderRadius: 8,
        alignItems: 'center',
    },
    submitButtonDisabled: { opacity: 0.7 },
    submitText: { color: colors.textInverse, fontSize: 16, fontWeight: 'bold' },
    loadingContainer: { flexDirection: 'row', alignItems: 'center', gap: 10 },
    loadingText: { color: colors.textInverse, fontSize: 14 },
});

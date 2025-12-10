// Native implementation using expo-image-picker
import * as ExpoImagePicker from 'expo-image-picker';
import { Alert } from 'react-native';

export const isAvailable = true;

export const pickImage = async (): Promise<{ base64: string | null; uri: string } | null> => {
    console.log('[ImagePicker.Native] Opening image picker...');

    const { status } = await ExpoImagePicker.requestMediaLibraryPermissionsAsync();
    if (status !== 'granted') {
        Alert.alert('Permission Required', 'Please allow access to your photo library.');
        return null;
    }

    const result = await ExpoImagePicker.launchImageLibraryAsync({
        mediaTypes: ExpoImagePicker.MediaTypeOptions.Images,
        allowsEditing: true,
        quality: 0.8,
        base64: true,
    });

    if (!result.canceled && result.assets[0]) {
        console.log('[ImagePicker.Native] Image selected, base64 length:', result.assets[0].base64?.length);
        return {
            base64: result.assets[0].base64 || null,
            uri: result.assets[0].uri,
        };
    }
    return null;
};

export const takePhoto = async (): Promise<{ base64: string | null; uri: string } | null> => {
    console.log('[ImagePicker.Native] Opening camera...');

    const { status } = await ExpoImagePicker.requestCameraPermissionsAsync();
    if (status !== 'granted') {
        Alert.alert('Permission Required', 'Please allow access to your camera.');
        return null;
    }

    const result = await ExpoImagePicker.launchCameraAsync({
        allowsEditing: true,
        quality: 0.8,
        base64: true,
    });

    if (!result.canceled && result.assets[0]) {
        console.log('[ImagePicker.Native] Photo taken, base64 length:', result.assets[0].base64?.length);
        return {
            base64: result.assets[0].base64 || null,
            uri: result.assets[0].uri,
        };
    }
    return null;
};

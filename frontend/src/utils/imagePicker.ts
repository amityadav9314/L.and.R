// Default fallback (web and server environments)
// This file is used when expo-image-picker is not available

export const pickImage = async (): Promise<{ base64: string | null; uri: string } | null> => {
    console.log('[ImagePicker.Default] Not available in this environment');
    return null;
};

export const takePhoto = async (): Promise<{ base64: string | null; uri: string } | null> => {
    console.log('[ImagePicker.Default] Not available in this environment');
    return null;
};

export const isAvailable = false;

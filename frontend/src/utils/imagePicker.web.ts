// This file provides a stub for web platform where expo-image-picker doesn't work
export const pickImage = async () => {
    console.log('[ImagePicker.Web] Not available on web');
    return null;
};

export const takePhoto = async () => {
    console.log('[ImagePicker.Web] Not available on web');
    return null;
};

export const isAvailable = false;

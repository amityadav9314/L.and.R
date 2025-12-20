import { Platform } from 'react-native';

// 10.0.2.2 is the special alias to your host loopback interface (127.0.0.1)
// on the Android emulator.
const LOCALHOST = Platform.OS === 'android' ? '10.0.2.2' : 'localhost';

// Set to true to bypass authentication for testing
export const BYPASS_AUTH = false;

export const API_URL = `http://${LOCALHOST}:8080`;
// export const API_URL = `https://43.205.81.60.nip.io`;

// TODO: Get your Google OAuth credentials from https://console.cloud.google.com/
// 1. Create a new project or select existing one
// 2. Enable Google+ API
// 3. Create OAuth 2.0 Client ID (Web application type)
// 4. Add authorized redirect URI: http://localhost:8081
// 5. Copy the Client ID and paste below
export const GOOGLE_WEB_CLIENT_ID = "330800561912-pf7pdbfsfjicv9fe4lkkf1q130gg2952.apps.googleusercontent.com"; // Replace with your Web Client ID
export const GOOGLE_IOS_CLIENT_ID = "YOUR_GOOGLE_IOS_CLIENT_ID"; // Optional: for iOS
// export const GOOGLE_ANDROID_CLIENT_ID = "330800561912-2rfarfrd78kf9dfcqfjobefi14m252pf.apps.googleusercontent.com"; // Using Web Client ID temporarily as placeholder, user needs to replace this.
export const GOOGLE_ANDROID_CLIENT_ID = "330800561912-6bnk7qkdpdhfsfa4ngnhrrpbv097e49d.apps.googleusercontent.com"; // Using Web Client ID temporarily as placeholder, user needs to replace this.

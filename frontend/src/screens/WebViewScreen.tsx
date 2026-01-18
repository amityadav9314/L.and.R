import React from 'react';
import { View, StyleSheet, ActivityIndicator } from 'react-native';
import { WebView } from 'react-native-webview';
import { useNavigation } from '../navigation/ManualRouter';
import { AppHeader } from '../components/AppHeader';

interface WebViewScreenProps {
    url?: string;
}

export const WebViewScreen = () => {
    // @ts-ignore
    const navigation = useNavigation();
    const { params } = navigation;
    const url = params?.url || 'https://landr.aky.net.in';

    return (
        <View style={styles.container}>
            <AppHeader />
            <WebView
                source={{ uri: url }}
                style={styles.webview}
                startInLoadingState={true}
                renderLoading={() => (
                    <View style={styles.loading}>
                        <ActivityIndicator size="large" color="#0d6efd" />
                    </View>
                )}
            />
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#fff',
    },
    webview: {
        flex: 1,
    },
    loading: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        justifyContent: 'center',
        alignItems: 'center',
        backgroundColor: '#fff',
    },
});

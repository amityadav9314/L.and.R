/**
 * Direct gRPC-Web API client for React Native
 * 
 * This bypasses nice-grpc-web entirely because it has compatibility issues
 * with React Native. Instead, we make direct XMLHttpRequest calls with
 * gRPC-Web framing and use the proto encode/decode functions directly.
 */
import { Platform } from 'react-native';
import { LoginRequest, LoginResponse } from '../../proto/backend/proto/auth/auth';
import {
    AddMaterialRequest, AddMaterialResponse,
    DeleteMaterialRequest,
    GetDueMaterialsRequest, GetDueMaterialsResponse,
    GetDueFlashcardsRequest, FlashcardList,
    CompleteReviewRequest, FailReviewRequest, UpdateFlashcardRequest, GetAllTagsResponse, NotificationStatusResponse,
    GetMaterialSummaryRequest, GetMaterialSummaryResponse,
    RegisterPushTokenRequest,
} from '../../proto/backend/proto/learning/learning';
import { Empty } from '../../proto/backend/google/protobuf/empty';
import { API_URL } from '../utils/config';
import { CreateSubscriptionOrderRequest, CreateSubscriptionOrderResponse } from '../../proto/backend/proto/payment/payment';

// Storage helper
const getToken = async (): Promise<string | null> => {
    if (Platform.OS === 'web') {
        return localStorage.getItem('auth_token');
    } else {
        const SecureStore = await import('expo-secure-store');
        return await SecureStore.getItemAsync('auth_token');
    }
};

// gRPC-Web frame encoding
function encodeGrpcWebFrame(data: Uint8Array): Uint8Array {
    const frame = new Uint8Array(5 + data.length);
    frame[0] = 0; // Data frame flag
    frame[1] = (data.length >> 24) & 0xff;
    frame[2] = (data.length >> 16) & 0xff;
    frame[3] = (data.length >> 8) & 0xff;
    frame[4] = data.length & 0xff;
    frame.set(data, 5);

    return frame;
}

// gRPC-Web frame decoding
function decodeGrpcWebResponse(response: Uint8Array): { data: Uint8Array | null; status: number; message: string } {
    let dataFrame: Uint8Array | null = null;
    let grpcStatus = 0;
    let grpcMessage = '';
    let offset = 0;

    while (offset < response.length) {
        if (offset + 5 > response.length) break;

        const flag = response[offset];
        const length = (response[offset + 1] << 24) |
            (response[offset + 2] << 16) |
            (response[offset + 3] << 8) |
            response[offset + 4];

        offset += 5;
        if (offset + length > response.length) break;

        const frameData = response.slice(offset, offset + length);
        offset += length;

        if (flag === 0) {
            // Data frame
            dataFrame = frameData;
        } else if (flag === 128) {
            // Trailer frame
            const trailerText = new TextDecoder().decode(frameData);
            for (const line of trailerText.split(/\r?\n/)) {
                const idx = line.indexOf(':');
                if (idx > 0) {
                    const key = line.substring(0, idx).trim().toLowerCase();
                    const value = line.substring(idx + 1).trim();
                    if (key === 'grpc-status') {
                        grpcStatus = parseInt(value, 10);
                    } else if (key === 'grpc-message') {
                        grpcMessage = decodeURIComponent(value);
                    }
                }
            }
        }
    }

    return { data: dataFrame, status: grpcStatus, message: grpcMessage };
}

// Error class
export class GrpcError extends Error {
    constructor(public path: string, public code: number, message: string) {
        super(message);
        this.name = 'GrpcError';
    }
}

// Generic gRPC request function
async function grpcRequest<TRequest, TResponse>(
    path: string,
    request: TRequest,
    encode: (req: TRequest) => Uint8Array,
    decode: (data: Uint8Array) => TResponse,
): Promise<TResponse> {
    const token = await getToken();
    const url = `${API_URL}${path}`;
    const requestData = encodeGrpcWebFrame(encode(request));

    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', url, true);
        xhr.responseType = 'arraybuffer';

        // Set headers
        xhr.setRequestHeader('Content-Type', 'application/grpc-web+proto');
        xhr.setRequestHeader('X-Grpc-Web', '1');
        xhr.setRequestHeader('Accept', 'application/grpc-web+proto');
        if (token) {
            xhr.setRequestHeader('Authorization', `Bearer ${token}`);
        }

        xhr.onload = () => {
            try {
                if (xhr.status !== 200) {
                    reject(new GrpcError(path, 2, `HTTP error: ${xhr.status}`));
                    return;
                }

                const responseBytes = new Uint8Array(xhr.response as ArrayBuffer);

                // If response is empty, check HTTP headers for gRPC status
                if (responseBytes.length === 0) {
                    const grpcStatus = xhr.getResponseHeader('grpc-status');
                    const grpcMessage = xhr.getResponseHeader('grpc-message');

                    if (grpcStatus && grpcStatus !== '0') {
                        const decodedMessage = grpcMessage ? decodeURIComponent(grpcMessage) : 'Unknown error';
                        reject(new GrpcError(path, parseInt(grpcStatus, 10), decodedMessage));
                        return;
                    }

                    reject(new GrpcError(path, 2, 'No data in response and no error status'));
                    return;
                }

                const { data, status, message } = decodeGrpcWebResponse(responseBytes);

                if (status !== 0) {
                    reject(new GrpcError(path, status, message || `gRPC error: ${status}`));
                    return;
                }

                if (!data) {
                    reject(new GrpcError(path, 2, 'No data in response'));
                    return;
                }

                resolve(decode(data));
            } catch (e) {
                reject(new GrpcError(path, 2, `Decode error: ${e}`));
            }
        };

        xhr.onerror = () => reject(new GrpcError(path, 14, 'Network error'));
        xhr.ontimeout = () => reject(new GrpcError(path, 4, 'Timeout'));

        xhr.send(requestData);
    });
}

// ============================================
// Auth Service Client
// ============================================
export const authClient = {
    async login(request: { googleIdToken: string }): Promise<LoginResponse> {
        const req = LoginRequest.fromPartial(request);
        return grpcRequest(
            '/auth.AuthService/Login',
            req,
            (r) => LoginRequest.encode(r).finish(),
            (data) => LoginResponse.decode(data),
        );
    },
};

// ============================================
// Learning Service Client  
// ============================================
export const learningClient = {
    async addMaterial(request: Partial<AddMaterialRequest>): Promise<AddMaterialResponse> {
        const req = AddMaterialRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/AddMaterial',
            req,
            (r) => AddMaterialRequest.encode(r).finish(),
            (data) => AddMaterialResponse.decode(data),
        );
    },

    async getDueMaterials(request: Partial<GetDueMaterialsRequest>): Promise<GetDueMaterialsResponse> {
        const req = GetDueMaterialsRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/GetDueMaterials',
            req,
            (r) => GetDueMaterialsRequest.encode(r).finish(),
            (data) => GetDueMaterialsResponse.decode(data),
        );
    },

    async deleteMaterial(request: Partial<DeleteMaterialRequest>): Promise<Empty> {
        const req = DeleteMaterialRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/DeleteMaterial',
            req,
            (r) => DeleteMaterialRequest.encode(r).finish(),
            (data) => Empty.decode(data),
        );
    },

    async getDueFlashcards(request: Partial<GetDueFlashcardsRequest>): Promise<FlashcardList> {
        const req = GetDueFlashcardsRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/GetDueFlashcards',
            req,
            (r) => GetDueFlashcardsRequest.encode(r).finish(),
            (data) => FlashcardList.decode(data),
        );
    },

    async completeReview(request: Partial<CompleteReviewRequest>): Promise<Empty> {
        const req = CompleteReviewRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/CompleteReview',
            req,
            (r) => CompleteReviewRequest.encode(r).finish(),
            (data) => Empty.decode(data),
        );
    },

    async failReview(request: Partial<FailReviewRequest>): Promise<Empty> {
        const req = FailReviewRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/FailReview',
            req,
            (r) => FailReviewRequest.encode(r).finish(),
            (data) => Empty.decode(data),
        );
    },

    async updateFlashcard(request: Partial<UpdateFlashcardRequest>): Promise<Empty> {
        const req = UpdateFlashcardRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/UpdateFlashcard',
            req,
            (r) => UpdateFlashcardRequest.encode(r).finish(),
            (data) => Empty.decode(data),
        );
    },

    async getAllTags(): Promise<GetAllTagsResponse> {
        return grpcRequest(
            '/learning.LearningService/GetAllTags',
            {},
            () => new Uint8Array(0), // Empty request
            (data) => GetAllTagsResponse.decode(data),
        );
    },

    async getNotificationStatus(): Promise<NotificationStatusResponse> {
        return grpcRequest(
            '/learning.LearningService/GetNotificationStatus',
            {},
            () => new Uint8Array(0), // Empty request
            (data) => NotificationStatusResponse.decode(data),
        );
    },

    async getMaterialSummary(request: Partial<GetMaterialSummaryRequest>): Promise<GetMaterialSummaryResponse> {
        const req = GetMaterialSummaryRequest.fromPartial(request);
        return grpcRequest(
            '/learning.LearningService/GetMaterialSummary',
            req,
            (r) => GetMaterialSummaryRequest.encode(r).finish(),
            (data) => GetMaterialSummaryResponse.decode(data),
        );
    },

    async registerPushToken(request: Partial<RegisterPushTokenRequest>): Promise<void> {
        const req = RegisterPushTokenRequest.fromPartial(request);
        await grpcRequest(
            '/learning.LearningService/RegisterPushToken',
            req,
            (r) => RegisterPushTokenRequest.encode(r).finish(),
            () => undefined,
        );
    },
};

// ============================================
// Feed Service Client (Daily AI Feed)
// ============================================
import {
    UpdateFeedPreferencesRequest,
    FeedPreferencesResponse,
    GetDailyFeedRequest, GetDailyFeedResponse,
    GetFeedCalendarStatusRequest, GetFeedCalendarStatusResponse
} from '../../proto/backend/proto/feed/feed';

export const feedClient = {
    async getFeedPreferences(): Promise<FeedPreferencesResponse> {
        return grpcRequest(
            '/feed.FeedService/GetFeedPreferences',
            {},
            () => new Uint8Array(0), // Empty request
            (data) => FeedPreferencesResponse.decode(data),
        );
    },

    async updateFeedPreferences(request: Partial<UpdateFeedPreferencesRequest>): Promise<Empty> {
        const req = UpdateFeedPreferencesRequest.fromPartial(request);
        return grpcRequest(
            '/feed.FeedService/UpdateFeedPreferences',
            req,
            (r) => UpdateFeedPreferencesRequest.encode(r).finish(),
            (data) => Empty.decode(data),
        );
    },

    async getDailyFeed(request: Partial<GetDailyFeedRequest>): Promise<GetDailyFeedResponse> {
        const req = GetDailyFeedRequest.fromPartial(request);
        return grpcRequest(
            '/feed.FeedService/GetDailyFeed',
            req,
            (r) => GetDailyFeedRequest.encode(r).finish(),
            (data) => GetDailyFeedResponse.decode(data),
        );
    },

    async getFeedCalendarStatus(request: Partial<GetFeedCalendarStatusRequest>): Promise<GetFeedCalendarStatusResponse> {
        const req = GetFeedCalendarStatusRequest.fromPartial(request);
        return grpcRequest(
            '/feed.FeedService/GetFeedCalendarStatus',
            req,
            (r) => GetFeedCalendarStatusRequest.encode(r).finish(),
            (data) => GetFeedCalendarStatusResponse.decode(data),
        );
    },
};

// ============================================
// Payment Service Client
// ============================================
export const paymentClient = {
    async createSubscriptionOrder(request: Partial<CreateSubscriptionOrderRequest>): Promise<CreateSubscriptionOrderResponse> {
        const req = CreateSubscriptionOrderRequest.fromPartial(request);
        return grpcRequest(
            '/payment.PaymentService/CreateSubscriptionOrder',
            req,
            (r) => CreateSubscriptionOrderRequest.encode(r).finish(),
            (data) => CreateSubscriptionOrderResponse.decode(data),
        );
    },
};

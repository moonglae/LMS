import { create } from 'zustand';
import { apiFetch } from '../api';

export interface User {
    id: number;
    email: string;
    first_name: string;
    last_name: string;
    role: string;
    is_banned: boolean;
    ban_reason?: string;
}

interface AuthState {
    token: string | null;
    user: User | null;
    isAuthenticated: boolean;
    isCheckingAuth: boolean;
    setToken: (token: string | null) => void;
    setUser: (user: User | null) => void;
    checkAuth: () => Promise<void>;
    logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
    token: localStorage.getItem('token'),
    user: null,
    isAuthenticated: !!localStorage.getItem('token'),
    isCheckingAuth: true,

    setToken: (token) => {
        if (token) {
            localStorage.setItem('token', token);
            set({ token, isAuthenticated: true });
        } else {
            localStorage.removeItem('token');
            set({ token: null, user: null, isAuthenticated: false });
        }
    },

    setUser: (user) => set({ user }),

    checkAuth: async () => {
        const token = localStorage.getItem('token');
        if (!token) {
            set({ user: null, isAuthenticated: false, isCheckingAuth: false });
            return;
        }

        try {
            // Використовуємо apiFetch, щоб запит автоматично йшов на правильний BASE_URL бекенду
            const userData = await apiFetch('/me');
            set({
                user: userData,
                isAuthenticated: true,
                isCheckingAuth: false,
            });
        } catch (error: any) {
            // Якщо сервер повернув 403 (бан), apiFetch може викинути помилку або містити статус
            // Перевіримо, чи забанений користувач
            if (error.status === 403 || error.message?.includes('banned')) {
                set({
                    user: {
                        id: 0,
                        email: '',
                        first_name: '',
                        last_name: '',
                        role: 'user',
                        is_banned: true,
                        ban_reason: error.message || 'Обмежено адміністратором'
                    },
                    isAuthenticated: true,
                    isCheckingAuth: false,
                });
                return;
            }

            localStorage.removeItem('token');
            set({
                token: null,
                user: null,
                isAuthenticated: false,
                isCheckingAuth: false,
            });
        }
    },

    logout: () => {
        localStorage.removeItem('token');
        set({
            token: null,
            user: null,
            isAuthenticated: false,
        });
    },
}));

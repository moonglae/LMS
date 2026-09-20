import { create } from 'zustand';

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
            // Робимо прямий fetch на /me, щоб перехопити 403 і забрати дані з помилки або запиту
            const response = await fetch('http://localhost:8080/api/me', {
                headers: { 'Authorization': `Bearer ${token}` }
            });

            if (response.status === 403) {
                // Навіть якщо 403, спробуємо витягнути дані юзера (якщо бекенд їх віддає) або просто активувати бан
                const errorData = await response.json().catch(() => null);

                // Якщо бекенд повернув дані або ми знаємо, що він забанений
                set({
                    user: errorData?.user || { is_banned: true, ban_reason: errorData?.error || 'Обмежено адміністратором' },
                    isAuthenticated: true,
                    isCheckingAuth: false,
                });
                return;
            }

            if (!response.ok) {
                throw new Error('Unauthorized');
            }

            const userData = await response.json();
            set({
                user: userData,
                isAuthenticated: true,
                isCheckingAuth: false,
            });
        } catch (error) {
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
import { create } from 'zustand';

interface AuthState {
    token: string | null;
    isAuthenticated: boolean;

    // Дії (Actions)
    setToken: (token: string) => void;
    logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
    // При завантаженні сторінки перевіряємо, чи є токен у пам'яті браузера
    token: localStorage.getItem('token'),
    isAuthenticated: !!localStorage.getItem('token'),

    // Функція для збереження токена після успішного логіну
    setToken: (token: string) => {
        localStorage.setItem('token', token);
        set({ token, isAuthenticated: true });
    },

    // Функція виходу з акаунта
    logout: () => {
        localStorage.removeItem('token');
        set({ token: null, isAuthenticated: false });
    },
}));
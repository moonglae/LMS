// src/api/index.ts

// 1. Обов'язково оголошуємо константу
export const BASE_URL = 'http://localhost:8080/api';

// 2. Функція apiFetch, яка використовує BASE_URL
export const apiFetch = async (endpoint: string, options: RequestInit = {}) => {
    const token = localStorage.getItem('token');

    const headers = {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(options.headers || {}),
    };

    const response = await fetch(`${BASE_URL}${endpoint}`, {
        ...options,
        headers,
    });

    if (!response.ok) {
        if (response.status === 401) {
            localStorage.removeItem('token');
            window.location.href = '/login';
        }

        let errorMsg = 'Помилка запиту';
        const text = await response.text();
        try {
            const errorData = JSON.parse(text);
            errorMsg = errorData.error || errorData.message || errorMsg;
        } catch {
            // ігноруємо, якщо не JSON
        }

        throw new Error(errorMsg);
    }

    // Якщо статус 204 (No Content), повертаємо null
    if (response.status === 204) {
        return null;
    }

    const text = await response.text();
    if (!text) return null; // Фікс для порожньої відповіді

    return JSON.parse(text);
};
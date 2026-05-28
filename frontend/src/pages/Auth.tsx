import { useState } from 'react';
import {
    BookOpen,
    Mail,
    Lock,
    User as UserIcon,
    Loader2,
} from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { apiFetch } from '../api';

export default function Auth() {
    const [isLogin, setIsLogin] = useState(true);

    // Стани для форми
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [firstName, setFirstName] = useState('');
    const [lastName, setLastName] = useState(''); // ДОДАНО
    const [role, setRole] = useState<'student' | 'teacher'>('student');

    // Стани для UI
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const setToken = useAuthStore((state) => state.setToken);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsLoading(true);
        setError(null);

        try {
            if (isLogin) {
                const data = await apiFetch('/auth/login', {
                    method: 'POST',
                    body: JSON.stringify({ email, password }),
                });

                if (data && typeof data.token === 'string') {
                    localStorage.setItem('token', data.token);
                    setToken(data.token);
                } else {
                    throw new Error('Некоректна відповідь сервера');
                }
            } else {
                // Відправляємо реальне ім'я та прізвище
                const data = await apiFetch('/auth/register', {
                    method: 'POST',
                    body: JSON.stringify({
                        email,
                        password,
                        first_name: firstName,
                        last_name: lastName, // ВИКОРИСТОВУЄМО ЗМІННУ
                        role,
                    }),
                });

                if (data && data.message) {
                    setIsLogin(true);
                    setError('Реєстрація успішна! Тепер увійдіть.');
                    setTimeout(() => setError(null), 3000);
                }
            }
        } catch (err: unknown) {
            if (err instanceof Error) {
                setError(err.message);
            } else {
                setError('Сталася невідома помилка');
            }
        } finally {
            setIsLoading(false);
        }
    };

    const isSuccessMessage = error?.includes('успішн') ?? false;

    return (
        <div className="min-h-screen flex items-center justify-center p-4 bg-mainBg">
            <div className="w-full max-w-md bg-surface border border-surfaceBorder rounded-2xl shadow-2xl p-8">
                <div className="flex flex-col items-center mb-8">
                    <div className="bg-primary/10 p-3 rounded-2xl mb-4">
                        <BookOpen className="w-10 h-10 text-primary" />
                    </div>
                    <h1 className="text-2xl font-bold text-textMain">
                        {isLogin ? 'З поверненням!' : 'Створення акаунту'}
                    </h1>
                </div>

                {error && (
                    <div className={`mb-6 p-4 rounded-xl text-sm font-medium border ${isSuccessMessage ? 'bg-green-500/10 text-green-400 border-green-500/20' : 'bg-red-500/10 text-red-400 border-red-500/20'}`}>
                        {error}
                    </div>
                )}

                <form onSubmit={handleSubmit} className="space-y-5">
                    {!isLogin && (
                        <>
                            <div className="flex bg-mainBg rounded-xl p-1 border border-surfaceBorder">
                                <button type="button" onClick={() => setRole('student')} className={`flex-1 py-2 text-sm font-medium rounded-lg transition-colors ${role === 'student' ? 'bg-surface border border-surfaceBorder text-primary' : 'text-textMuted'}`}>Студент</button>
                                <button type="button" onClick={() => setRole('teacher')} className={`flex-1 py-2 text-sm font-medium rounded-lg transition-colors ${role === 'teacher' ? 'bg-surface border border-surfaceBorder text-primary' : 'text-textMuted'}`}>Викладач</button>
                            </div>

                            {/* Поля імені та прізвища */}
                            <div className="grid grid-cols-2 gap-4">
                                <div className="relative">
                                    <UserIcon className="absolute left-3 top-1/2 -translate-y-1/2 text-textMuted w-5 h-5" />
                                    <input
                                        type="text" placeholder="Ім'я" value={firstName}
                                        onChange={(e) => setFirstName(e.target.value)}
                                        className="w-full bg-mainBg border border-surfaceBorder text-textMain rounded-xl py-3 pl-10 pr-4 focus:outline-none focus:border-primary"
                                        required={!isLogin}
                                    />
                                </div>
                                <div className="relative">
                                    <UserIcon className="absolute left-3 top-1/2 -translate-y-1/2 text-textMuted w-5 h-5" />
                                    <input
                                        type="text" placeholder="Прізвище" value={lastName}
                                        onChange={(e) => setLastName(e.target.value)}
                                        className="w-full bg-mainBg border border-surfaceBorder text-textMain rounded-xl py-3 pl-10 pr-4 focus:outline-none focus:border-primary"
                                        required={!isLogin}
                                    />
                                </div>
                            </div>
                        </>
                    )}

                    <div className="relative">
                        <Mail className="absolute left-3 top-1/2 -translate-y-1/2 text-textMuted w-5 h-5" />
                        <input
                            type="email" placeholder="Email адреса" value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            className="w-full bg-mainBg border border-surfaceBorder text-textMain rounded-xl py-3 pl-10 pr-4 focus:outline-none focus:border-primary"
                            required
                        />
                    </div>

                    <div className="relative">
                        <Lock className="absolute left-3 top-1/2 -translate-y-1/2 text-textMuted w-5 h-5" />
                        <input
                            type="password" placeholder="Пароль" value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full bg-mainBg border border-surfaceBorder text-textMain rounded-xl py-3 pl-10 pr-4 focus:outline-none focus:border-primary"
                            required
                        />
                    </div>

                    <button type="submit" disabled={isLoading} className="w-full bg-primary hover:bg-primaryHover text-white font-semibold py-3 rounded-xl transition-colors">
                        {isLoading ? <Loader2 className="w-5 h-5 animate-spin mx-auto" /> : (isLogin ? 'Увійти' : 'Зареєструватися')}
                    </button>
                </form>

                <div className="mt-6 text-center text-sm text-textMuted">
                    {isLogin ? 'Ще немає акаунту? ' : 'Вже маєте акаунт? '}
                    <button onClick={() => { setIsLogin(!isLogin); setError(null); }} type="button" className="text-primary hover:text-primaryHover font-medium">
                        {isLogin ? 'Створити зараз' : 'Увійти'}
                    </button>
                </div>
            </div>
        </div>
    );
}
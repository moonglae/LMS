import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { User, Loader2, BarChart3, Calendar, Award, CheckCircle, RefreshCw, Flame, PlayCircle, Plus, Trash2, Target } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { apiFetch } from '../api';

interface UserData {
    id: number;
    email: string;
    first_name: string;
    last_name: string;
    created_at: string;
}

interface SummaryStats {
    total_cards_learned: number;
    total_quizzes_taken: number;
    current_streak: number;
    last_module: {
        id: number;
        title: string;
    };
}

interface ProgressItem {
    date: string;
    score: number;
}

interface Goal {
    id: number;
    text: string;
    is_completed: boolean;
}

export default function Profile() {
    const [userData, setUserData] = useState<UserData | null>(null);
    const [stats, setStats] = useState<SummaryStats | null>(null);
    const [progressData, setProgressData] = useState<ProgressItem[]>([]);

    // Нові стейти для цілей
    const [goals, setGoals] = useState<Goal[]>([]);
    const [newGoalText, setNewGoalText] = useState("");

    const [isLoading, setIsLoading] = useState(true);
    const [isRefreshing, setIsRefreshing] = useState(false);

    const fetchProfileData = async (showMainLoader = true) => {
        if (showMainLoader) setIsLoading(true);
        else setIsRefreshing(true);

        try {
            const [userRes, statsRes, progressRes, goalsRes] = await Promise.all([
                apiFetch('/me'),
                apiFetch('/profile/stats'),
                apiFetch('/analytics/progress'),
                apiFetch('/profile/goals'), // Завантажуємо цілі
            ]);
            setUserData(userRes as UserData);
            setStats(statsRes as SummaryStats);
            setProgressData(Array.isArray(progressRes) ? progressRes : []);
            setGoals(Array.isArray(goalsRes) ? goalsRes : []);
        } catch (err) {
            console.error('Помилка завантаження', err);
        } finally {
            setIsLoading(false);
            setIsRefreshing(false);
        }
    };

    useEffect(() => {
        fetchProfileData(true);
    }, []);

    // Логіка для To-Do
    const handleAddGoal = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newGoalText.trim()) return;
        try {
            const addedGoal = await apiFetch('/profile/goals', {
                method: 'POST',
                body: JSON.stringify({ text: newGoalText.trim() })
            });
            setGoals([...goals, addedGoal as Goal]);
            setNewGoalText("");
        } catch (err) {
            console.error("Помилка додавання цілі");
        }
    };

    const handleToggleGoal = async (goal: Goal) => {
        const updatedStatus = !goal.is_completed;
        setGoals(goals.map(g => g.id === goal.id ? { ...g, is_completed: updatedStatus } : g));
        try {
            await apiFetch('/profile/goals', {
                method: 'PUT',
                body: JSON.stringify({ id: goal.id, is_completed: updatedStatus })
            });
        } catch (err) {
            setGoals(goals.map(g => g.id === goal.id ? { ...g, is_completed: goal.is_completed } : g));
        }
    };

    const handleDeleteGoal = async (id: number) => {
        setGoals(goals.filter(g => g.id !== id));
        try {
            await apiFetch(`/profile/goals?id=${id}`, { method: 'DELETE' });
        } catch (err) {
            fetchProfileData(false); // Відкат у разі помилки
        }
    };

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;
    if (!userData) return <div className="text-center mt-20 text-red-400">Помилка завантаження</div>;

    const formatRegDate = (dateStr: string | undefined) => {
        if (!dateStr) return 'Невідомо';
        const date = new Date(dateStr);
        return isNaN(date.getTime()) ? 'Невідомо' : date.toLocaleDateString('uk-UA');
    };

    return (
        <div className="max-w-6xl mx-auto mt-8 space-y-8 p-4">
            <div className="border-b border-surfaceBorder pb-6 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold text-textMain">Особистий кабінет</h1>
                    <p className="text-textMuted mt-1">Ваш прогрес навчання</p>
                </div>
                <button
                    onClick={() => fetchProfileData(false)}
                    disabled={isRefreshing}
                    className="flex items-center gap-2 px-4 py-2 bg-surface border border-surfaceBorder rounded-xl text-sm font-medium text-textMain hover:bg-mainBg hover:text-primary transition-colors disabled:opacity-50"
                >
                    <RefreshCw size={16} className={isRefreshing ? "animate-spin" : ""} />
                    {isRefreshing ? 'Оновлюємо...' : 'Оновити статистику'}
                </button>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
                <div className="space-y-6 h-fit">
                    <div className="bg-surface border border-surfaceBorder rounded-3xl p-6 text-center">
                        <div className="w-20 h-20 bg-primary/10 rounded-full flex items-center justify-center mx-auto mb-4">
                            <User className="w-10 h-10 text-primary" />
                        </div>
                        <h2 className="text-xl font-bold">{userData.first_name} {userData.last_name}</h2>
                        <p className="text-textMuted text-sm mb-4">{userData.email}</p>
                        <div className="flex items-center justify-center gap-2 text-xs text-textMuted mb-4">
                            <Calendar size={14} /> Реєстрація: {formatRegDate(userData.created_at)}
                        </div>
                    </div>

                    <div className="bg-gradient-to-br from-orange-500/10 to-red-500/10 border border-orange-500/20 rounded-3xl p-6 text-center relative overflow-hidden group">
                        <div className="absolute -right-4 -top-4 opacity-5 group-hover:opacity-10 transition-opacity">
                            <Flame size={120} />
                        </div>
                        <div className="flex justify-center mb-2">
                            <Flame className={`w-12 h-12 ${stats?.current_streak && stats.current_streak > 0 ? 'text-orange-500' : 'text-textMuted'}`} />
                        </div>
                        <h3 className="text-sm text-textMuted font-bold uppercase tracking-wider mb-1">Ударний режим</h3>
                        <div className="text-4xl font-black text-textMain">
                            {stats?.current_streak || 0} <span className="text-xl text-textMuted font-medium">днів</span>
                        </div>
                    </div>
                </div>

                <div className="lg:col-span-3 space-y-6">
                    {stats?.last_module && stats.last_module.id !== 0 && (
                        <div className="bg-primary/5 border border-primary/20 rounded-3xl p-1 flex flex-col sm:flex-row items-center justify-between">
                            <div className="p-5">
                                <p className="text-sm text-primary font-bold uppercase tracking-wider mb-1">Продовжити навчання</p>
                                <h3 className="text-xl font-bold text-textMain">{stats.last_module.title}</h3>
                            </div>
                            <Link
                                to={`/modules/${stats.last_module.id}/flashcards`}
                                className="m-3 sm:m-0 sm:mr-3 px-6 py-3 bg-primary hover:bg-primaryHover text-white font-bold rounded-xl flex items-center gap-2 transition-colors w-full sm:w-auto justify-center"
                            >
                                <PlayCircle size={20} /> Продовжити
                            </Link>
                        </div>
                    )}

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <StatCard label="Пройдено карток" value={stats?.total_cards_learned || 0} icon={<Award size={24} />} color="text-yellow-400" />
                        <StatCard label="Завершено тестів" value={stats?.total_quizzes_taken || 0} icon={<CheckCircle size={24} />} color="text-green-400" />
                    </div>

                    <div className="bg-surface border border-surfaceBorder rounded-3xl p-6 h-72">
                        <h3 className="font-bold mb-4 flex items-center gap-2 text-textMain">
                            <BarChart3 size={18} className="text-primary" /> Динаміка середнього балу
                        </h3>
                        {progressData.length > 0 ? (
                            <ResponsiveContainer width="100%" height="100%">
                                <LineChart data={progressData}>
                                    <CartesianGrid strokeDasharray="3 3" stroke="#333" vertical={false} />
                                    <XAxis dataKey="date" stroke="#888" tickMargin={10} axisLine={false} tickLine={false} />
                                    <YAxis domain={[0, 100]} stroke="#888" axisLine={false} tickLine={false} tickFormatter={(val) => `${val}%`} />
                                    <Tooltip contentStyle={{ backgroundColor: '#18181b', borderRadius: '12px', border: '1px solid #3f3f46', color: '#fff' }} itemStyle={{ color: '#3b82f6', fontWeight: 'bold' }} />
                                    <Line type="monotone" dataKey="score" name="Бал" stroke="#3b82f6" strokeWidth={4} dot={{ r: 4, fill: '#3b82f6', strokeWidth: 2, stroke: '#18181b' }} activeDot={{ r: 6, strokeWidth: 0 }} />
                                </LineChart>
                            </ResponsiveContainer>
                        ) : (
                            <div className="w-full h-full flex flex-col items-center justify-center text-textMuted">
                                <BarChart3 size={48} className="mb-2 opacity-20" />
                                <p>Немає даних для графіка.</p>
                            </div>
                        )}
                    </div>

                    {/* БЛОК ЦІЛЕЙ (TO-DO) */}
                    <div className="bg-surface border border-surfaceBorder rounded-3xl p-6">
                        <h3 className="font-bold mb-6 flex items-center gap-2 text-textMain">
                            <Target size={18} className="text-primary" /> Мої цілі у навчанні
                        </h3>

                        <form onSubmit={handleAddGoal} className="flex gap-2 mb-6">
                            <input
                                type="text"
                                value={newGoalText}
                                onChange={(e) => setNewGoalText(e.target.value)}
                                placeholder="Додайте нову ціль (напр: Вивчити 50 слів до п'ятниці)"
                                className="flex-1 bg-mainBg border border-surfaceBorder rounded-xl px-4 py-3 text-textMain placeholder:text-textMuted focus:outline-none focus:border-primary transition-colors"
                            />
                            <button
                                type="submit"
                                disabled={!newGoalText.trim()}
                                className="px-5 bg-primary hover:bg-primaryHover disabled:opacity-50 disabled:hover:bg-primary text-white rounded-xl flex items-center justify-center transition-colors"
                            >
                                <Plus size={20} />
                            </button>
                        </form>

                        <div className="space-y-3">
                            {goals.length === 0 ? (
                                <div className="text-center py-6 text-textMuted text-sm">У вас ще немає записаних цілей.</div>
                            ) : (
                                goals.map(goal => (
                                    <div key={goal.id} className={`flex items-center gap-3 p-4 rounded-2xl border transition-all ${goal.is_completed ? 'bg-mainBg border-transparent opacity-60' : 'bg-surface border-surfaceBorder'}`}>
                                        <button
                                            onClick={() => handleToggleGoal(goal)}
                                            className={`flex-shrink-0 w-6 h-6 rounded-full border-2 flex items-center justify-center transition-colors ${goal.is_completed ? 'bg-primary border-primary' : 'border-textMuted hover:border-primary'}`}
                                        >
                                            {goal.is_completed && <CheckCircle size={14} className="text-white" />}
                                        </button>
                                        <span className={`flex-1 ${goal.is_completed ? 'line-through text-textMuted' : 'text-textMain'}`}>
                                            {goal.text}
                                        </span>
                                        <button
                                            onClick={() => handleDeleteGoal(goal.id)}
                                            className="text-textMuted hover:text-red-400 p-2 transition-colors"
                                        >
                                            <Trash2 size={18} />
                                        </button>
                                    </div>
                                ))
                            )}
                        </div>
                    </div>

                </div>
            </div>
        </div>
    );
}

function StatCard({ label, value, icon, color }: { label: string, value: any, icon?: React.ReactNode, color: string }) {
    return (
        <div className="bg-surface border border-surfaceBorder p-6 rounded-3xl flex items-center gap-5 hover:border-surfaceBorder/80 transition-colors">
            <div className={`p-4 rounded-2xl bg-mainBg ${color}`}>
                {icon}
            </div>
            <div>
                <p className="text-textMuted text-xs uppercase tracking-wider font-bold mb-1">{label}</p>
                <p className="text-3xl font-black text-textMain">{value}</p>
            </div>
        </div>
    );
}

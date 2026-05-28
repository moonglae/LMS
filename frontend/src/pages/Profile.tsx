import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom'; // ДОДАНО ІМПОРТ
import { User, BookOpen, Loader2, Users, BarChart3, Calendar, Award, CheckCircle, RefreshCw, ArrowRight } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { apiFetch } from '../api';

interface UserData {
    id: number;
    email: string;
    first_name: string;
    last_name: string;
    role: 'student' | 'teacher' | 'admin';
    created_at: string;
}

interface SummaryStats {
    total_attempts: number;
    perfect_scores: number;
    average_score: number;
}

interface ModuleItem {
    id: number;
    title: string;
    description: string | null;
    invite_code: string;
    student_count?: number;
}

interface ProgressItem {
    date: string;
    score: number;
}

export default function Profile() {
    const [userData, setUserData] = useState<UserData | null>(null);
    const [stats, setStats] = useState<SummaryStats | null>(null);
    const [modules, setModules] = useState<ModuleItem[]>([]);
    const [progressData, setProgressData] = useState<ProgressItem[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isRefreshing, setIsRefreshing] = useState(false);

    const fetchProfileData = async (showMainLoader = true) => {
        if (showMainLoader) setIsLoading(true);
        else setIsRefreshing(true);

        try {
            const [userRes, statsRes, modulesRes, progressRes] = await Promise.all([
                apiFetch('/me'),
                apiFetch('/analytics/summary'),
                apiFetch('/modules'),
                apiFetch('/analytics/progress'),
            ]);
            setUserData(userRes as UserData);
            setStats(statsRes as SummaryStats);
            setModules(Array.isArray(modulesRes) ? modulesRes : []);
            setProgressData(Array.isArray(progressRes) ? progressRes : []);
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

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;
    if (!userData) return <div className="text-center mt-20 text-red-400">Помилка завантаження</div>;

    const isTeacher = userData.role === 'teacher' || userData.role === 'admin';

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
                    <p className="text-textMuted mt-1">{isTeacher ? 'Панель викладача' : 'Ваш прогрес навчання'}</p>
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
                {/* ЛІВА КОЛОНКА: Профіль */}
                <div className="bg-surface border border-surfaceBorder rounded-3xl p-6 text-center h-fit">
                    <div className="w-20 h-20 bg-primary/10 rounded-full flex items-center justify-center mx-auto mb-4">
                        <User className="w-10 h-10 text-primary" />
                    </div>
                    <h2 className="text-xl font-bold">{userData.first_name} {userData.last_name}</h2>
                    <p className="text-textMuted text-sm mb-4">{userData.email}</p>
                    <div className="flex items-center justify-center gap-2 text-xs text-textMuted mb-4">
                        <Calendar size={14} /> Реєстрація: {formatRegDate(userData.created_at)}
                    </div>
                    <div className="inline-block px-3 py-1 bg-green-500/10 text-green-400 rounded-full text-xs font-bold uppercase">
                        {isTeacher ? 'Викладач' : 'Студент'}
                    </div>
                </div>

                {/* ПРАВА КОЛОНКА */}
                <div className="lg:col-span-3 space-y-6">
                    {isTeacher ? (
                        <div className="space-y-6">
                            {/* 1. Статистика (4 плашки) */}
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                                <StatCard label="Ваші модулі" value={modules.length} icon={<BookOpen size={20} />} />
                                <StatCard label="Записів на курси" value={modules.reduce((a, b) => a + (b.student_count || 0), 0)} icon={<Users size={20} />} />
                                <StatCard label="Спроби студентів" value={stats?.total_attempts || 0} icon={<BarChart3 size={20} />} />
                                <StatCard label="Сер. успішність" value={(stats?.average_score || 0) + '%'} icon={<Award size={20} />} />
                            </div>

                            {/* 2. НОВИЙ БЛОК: Швидкий доступ до Аналітики */}
                            <div className="bg-gradient-to-r from-primary/10 to-transparent border border-primary/20 rounded-3xl p-6 flex flex-col sm:flex-row items-center justify-between gap-6">
                                <div>
                                    <h3 className="font-bold text-lg text-textMain flex items-center gap-2 mb-1">
                                        <BarChart3 className="text-primary" size={20} /> Детальна аналітика
                                    </h3>
                                    <p className="text-sm text-textMuted">Переглядайте індивідуальні звіти студентів, роздруковуйте їх та аналізуйте проблемні питання.</p>
                                </div>
                                <Link to="/teacher" className="shrink-0 bg-primary hover:bg-primaryHover text-white px-6 py-3 rounded-xl font-medium transition-all shadow-lg shadow-primary/20 flex items-center gap-2">
                                    Відкрити центр <ArrowRight size={18} />
                                </Link>
                            </div>

                            {/* 3. НОВИЙ БЛОК: Список курсів викладача */}
                            <div className="bg-surface border border-surfaceBorder rounded-3xl p-6">
                                <div className="flex justify-between items-center mb-6">
                                    <h3 className="font-bold text-lg flex items-center gap-2">
                                        <BookOpen className="text-primary" size={20} /> Ваші активні курси
                                    </h3>
                                    <Link to="/" className="text-sm text-primary hover:underline font-medium">Всі курси &rarr;</Link>
                                </div>

                                {modules.length === 0 ? (
                                    <div className="text-center py-10 text-textMuted border border-dashed border-surfaceBorder rounded-2xl bg-mainBg/50">
                                        <p>Ви ще не створили жодного курсу.</p>
                                        <Link to="/module/new" className="text-primary font-medium hover:underline mt-2 inline-block">Створити перший курс</Link>
                                    </div>
                                ) : (
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                        {modules.map(mod => (
                                            <div key={mod.id} className="p-5 bg-mainBg rounded-2xl border border-surfaceBorder flex flex-col justify-between hover:border-primary/50 transition-colors group">
                                                <div>
                                                    <div className="flex justify-between items-start mb-2 gap-2">
                                                        <h4 className="font-bold text-textMain line-clamp-1">{mod.title}</h4>
                                                        {mod.invite_code && (
                                                            <span className="shrink-0 text-[10px] font-mono bg-surface px-2 py-1 rounded-md text-textMuted border border-surfaceBorder uppercase tracking-wider">
                                                                {mod.invite_code}
                                                            </span>
                                                        )}
                                                    </div>
                                                    <p className="text-sm text-textMuted line-clamp-2 mb-4">{mod.description || 'Опис відсутній'}</p>
                                                </div>

                                                <div className="flex justify-between items-center pt-4 border-t border-surfaceBorder">
                                                    <div className="flex items-center gap-2 text-sm text-textMuted font-medium">
                                                        <Users size={16} className="text-primary" />
                                                        <span>{mod.student_count || 0} учнів</span>
                                                    </div>
                                                    <Link to={`/teacher`} className="text-sm text-textMuted group-hover:text-primary font-medium transition-colors">
                                                        Статистика &rarr;
                                                    </Link>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </div>
                    ) : (
                        <div className="space-y-6">
                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                                <StatCard label="Пройдено тестів" value={stats?.total_attempts || 0} icon={<BarChart3 size={20} />} />
                                <StatCard label="Сер. бал" value={(stats?.average_score || 0) + '%'} icon={<Award size={20} />} />
                                <StatCard label="Ідеальні" value={stats?.perfect_scores || 0} icon={<CheckCircle size={20} />} />
                            </div>
                            <div className="bg-surface border border-surfaceBorder rounded-3xl p-6 h-64">
                                <h3 className="font-bold mb-4 flex items-center gap-2">Динаміка успішності</h3>
                                <ResponsiveContainer width="100%" height="100%">
                                    <LineChart data={progressData}>
                                        <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                                        <XAxis dataKey="date" stroke="#666" />
                                        <YAxis domain={[0, 100]} />
                                        <Tooltip contentStyle={{ backgroundColor: '#111', borderRadius: '12px', border: '1px solid #333' }} />
                                        <Line type="monotone" dataKey="score" stroke="#3b82f6" strokeWidth={3} dot={{ r: 4, fill: '#3b82f6' }} activeDot={{ r: 6 }} />
                                    </LineChart>
                                </ResponsiveContainer>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

function StatCard({ label, value, icon }: { label: string, value: any, icon?: React.ReactNode }) {
    return (
        <div className="bg-surface border border-surfaceBorder p-6 rounded-3xl flex items-center gap-4 hover:border-primary/30 transition-colors">
            {icon && <div className="text-primary">{icon}</div>}
            <div>
                <p className="text-textMuted text-[11px] uppercase tracking-wider font-semibold">{label}</p>
                <p className="text-2xl font-bold mt-1 text-textMain">{value}</p>
            </div>
        </div>
    );
}
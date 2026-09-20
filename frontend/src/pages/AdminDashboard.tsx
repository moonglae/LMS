import { useState, useEffect } from 'react';
import { ShieldAlert, Users, Ban, CheckCircle, Unlock, X, Check, Bot, Lock, Search } from 'lucide-react';
import { apiFetch } from '../api';

interface Alert {
    id: number;
    user_id: number;
    activity_type: string;
    description: string;
    created_at: string;
}

interface User {
    id: number;
    email: string;
    first_name: string;
    last_name: string;
    role: string;
    is_banned: boolean;
    restricted_features: string;
}

const AdminDashboard = () => {
    const [activeTab, setActiveTab] = useState<'users' | 'alerts'>('users');
    const [alerts, setAlerts] = useState<Alert[]>([]);
    const [users, setUsers] = useState<User[]>([]);
    const [message, setMessage] = useState<{ text: string, type: 'success' | 'error' } | null>(null);

    const [banModalOpen, setBanModalOpen] = useState(false);
    const [userToBan, setUserToBan] = useState<number | null>(null);
    const [banReason, setBanReason] = useState("");
    const [loadingUserId, setLoadingUserId] = useState<number | null>(null);

    // Стан для пошуку
    const [searchQuery, setSearchQuery] = useState("");
    const [searchByIdOnly, setSearchByIdOnly] = useState(false); // НОВИЙ СТАН: Шукати тільки за ID

    useEffect(() => {
        if (activeTab === 'alerts') fetchAlerts();
        if (activeTab === 'users') fetchUsers();
    }, [activeTab]);

    const showMessage = (text: string, type: 'success' | 'error') => {
        setMessage({ text, type });
        setTimeout(() => setMessage(null), 4000);
    };

    const fetchAlerts = async () => {
        try {
            const data = await apiFetch('/admin/alerts');
            setAlerts(data || []);
        } catch (error: any) {
            console.error(error);
            showMessage(error.message || 'Помилка завантаження алертів', 'error');
        }
    };

    const fetchUsers = async () => {
        try {
            const data = await apiFetch('/admin/users');
            setUsers(data || []);
        } catch (error: any) {
            console.error(error);
            showMessage(error.message || 'Помилка завантаження користувачів', 'error');
        }
    };

    const handleBanClick = (userId: number, isBanning: boolean) => {
        if (isBanning) {
            setUserToBan(userId);
            setBanReason("");
            setBanModalOpen(true);
        } else {
            executeBan(userId, false, "");
        }
    };

    const executeBan = async (userId: number, banStatus: boolean, reason: string) => {
        setLoadingUserId(userId);
        try {
            await apiFetch(`/admin/users/ban?id=${userId}`, {
                method: 'POST',
                body: JSON.stringify({ ban: banStatus, reason: reason })
            });

            showMessage(`Користувача успішно ${banStatus ? 'забанено' : 'розбанено'}`, 'success');
            fetchUsers();
        } catch (error: any) {
            showMessage(error.message || 'Помилка оновлення статусу', 'error');
        } finally {
            setLoadingUserId(null);
            setBanModalOpen(false);
        }
    };

    const handleToggleFeature = async (userId: number, feature: string, currentLockState: boolean) => {
        try {
            await apiFetch(`/admin/users/restrict?id=${userId}`, {
                method: 'POST',
                body: JSON.stringify({ feature: feature, lock: !currentLockState })
            });
            showMessage(`Статус функції "${feature}" оновлено`, 'success');
            fetchUsers();
        } catch (error: any) {
            showMessage(error.message || 'Помилка оновлення функції', 'error');
        }
    };

    const handleResolveAlert = async (alertId: number) => {
        try {
            await apiFetch(`/admin/alerts/resolve?id=${alertId}`, {
                method: 'POST'
            });
            setAlerts(alerts.filter(a => a.id !== alertId));
            showMessage('Алерт позначено як вирішений', 'success');
        } catch (error: any) {
            showMessage(error.message || 'Помилка вирішення алерта', 'error');
        }
    };

    const isFeatureLocked = (restrictedFeaturesStr: string, featureName: string) => {
        try {
            const parsed = JSON.parse(restrictedFeaturesStr || '{}');
            return parsed[featureName] === true || parsed[featureName] === "true";
        } catch {
            return false;
        }
    };

    // Оновлена логіка фільтрації
    const filteredUsers = users.filter(u => {
        const query = searchQuery.toLowerCase();

        if (searchByIdOnly) {
            // Якщо увімкнено пошук тільки за ID, шукаємо ТОЧНИЙ або частковий збіг лише в ID
            return u.id.toString().includes(query);
        }

        // Інакше шукаємо всюди
        return (
            u.id.toString().includes(query) ||
            u.first_name.toLowerCase().includes(query) ||
            u.last_name.toLowerCase().includes(query) ||
            u.email.toLowerCase().includes(query)
        );
    });

    return (
        <div className="max-w-7xl mx-auto p-4 sm:p-6 space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <h1 className="text-2xl font-bold text-textMain">Панель Адміністратора</h1>

                <div className="flex bg-surface border border-surfaceBorder rounded-xl p-1">
                    <button onClick={() => setActiveTab('users')} className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${activeTab === 'users' ? 'bg-primary text-white shadow-sm' : 'text-textMuted hover:text-textMain hover:bg-mainBg'}`}>
                        <Users className="w-4 h-4" /> Користувачі
                    </button>
                    <button onClick={() => setActiveTab('alerts')} className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${activeTab === 'alerts' ? 'bg-red-500 text-white shadow-sm' : 'text-textMuted hover:text-textMain hover:bg-mainBg'}`}>
                        <ShieldAlert className="w-4 h-4" /> Логи безпеки
                    </button>
                </div>
            </div>

            {message && (
                <div className={`p-4 rounded-xl text-sm font-medium border ${message.type === 'success' ? 'bg-green-500/10 text-green-400 border-green-500/20' : 'bg-red-500/10 text-red-400 border-red-500/20'}`}>
                    {message.text}
                </div>
            )}

            <div className="bg-surface border border-surfaceBorder rounded-2xl shadow-sm overflow-hidden p-1">
                {activeTab === 'users' && (
                    <div className="space-y-4">
                        {/* Змінений блок пошуку */}
                        <div className="px-4 pt-4 flex flex-col sm:flex-row gap-4 items-center">
                            <div className="relative w-full sm:max-w-md">
                                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-textMuted" />
                                <input
                                    type="text"
                                    placeholder={searchByIdOnly ? "Введіть ID користувача..." : "Пошук за ID, ім'ям або email..."}
                                    value={searchQuery}
                                    onChange={(e) => setSearchQuery(e.target.value)}
                                    className="w-full pl-10 pr-4 py-2.5 bg-mainBg border border-surfaceBorder rounded-xl text-textMain placeholder:text-textMuted focus:outline-none focus:border-primary transition-colors text-sm"
                                />
                            </div>

                            {/* Чекбокс для перемикання режиму пошуку */}
                            <label className="flex items-center gap-2 cursor-pointer text-sm text-textMuted hover:text-textMain select-none">
                                <input
                                    type="checkbox"
                                    checked={searchByIdOnly}
                                    onChange={(e) => {
                                        setSearchByIdOnly(e.target.checked);
                                        setSearchQuery(""); // Очищаємо поле при зміні режиму
                                    }}
                                    className="rounded border-gray-400 text-primary focus:ring-primary w-4 h-4 cursor-pointer"
                                />
                                Шукати тільки за ID
                            </label>
                        </div>

                        <div className="overflow-x-auto">
                            <table className="w-full text-left border-collapse">
                                <thead>
                                    <tr className="bg-mainBg/50 border-y border-surfaceBorder text-textMuted text-sm">
                                        <th className="p-4 font-medium">ID</th>
                                        <th className="p-4 font-medium">Користувач</th>
                                        <th className="p-4 font-medium">Email</th>
                                        <th className="p-4 font-medium">Роль</th>
                                        <th className="p-4 font-medium">Статус</th>
                                        <th className="p-4 font-medium text-center">Обмеження ШІ</th>
                                        <th className="p-4 font-medium text-right">Дії</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-surfaceBorder">
                                    {filteredUsers.length > 0 ? (
                                        filteredUsers.map(u => {
                                            const isAdmin = u.role === 'admin';
                                            const aiLocked = isFeatureLocked(u.restricted_features, 'ai_chat');

                                            return (
                                                <tr key={u.id} className="hover:bg-mainBg/30 transition-colors text-textMain text-sm">
                                                    <td className="p-4 font-mono">{u.id}</td>
                                                    <td className="p-4 font-medium">{u.first_name} {u.last_name}</td>
                                                    <td className="p-4 text-textMuted">{u.email}</td>
                                                    <td className="p-4">
                                                        <span className={`px-2 py-1 rounded-md text-xs ${isAdmin ? 'bg-purple-500/20 text-purple-400 border border-purple-500/20' : 'bg-blue-500/20 text-blue-400 border border-blue-500/20'}`}>
                                                            {u.role}
                                                        </span>
                                                    </td>
                                                    <td className="p-4">
                                                        {u.is_banned ? (
                                                            <span className="flex items-center gap-1 text-red-400"><Ban className="w-3 h-3" /> Забанений</span>
                                                        ) : (
                                                            <span className="flex items-center gap-1 text-green-400"><CheckCircle className="w-3 h-3" /> Активний</span>
                                                        )}
                                                    </td>

                                                    <td className="p-4 text-center">
                                                        <button
                                                            onClick={() => handleToggleFeature(u.id, 'ai_chat', aiLocked)}
                                                            disabled={isAdmin}
                                                            title={aiLocked ? "Розблокувати ШІ-чат" : "Заблокувати ШІ-чат"}
                                                            className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold transition-all ${isAdmin
                                                                    ? 'opacity-40 cursor-not-allowed bg-surfaceBorder text-textMuted'
                                                                    : aiLocked
                                                                        ? 'bg-amber-500/15 text-amber-400 border border-amber-500/30 hover:bg-amber-500/25'
                                                                        : 'bg-mainBg text-textMuted hover:text-textMain border border-surfaceBorder'
                                                                }`}
                                                        >
                                                            {aiLocked ? <Lock className="w-3.5 h-3.5 text-amber-400" /> : <Bot className="w-3.5 h-3.5" />}
                                                            {aiLocked ? 'ШІ вимкнено' : 'ШІ активний'}
                                                        </button>
                                                    </td>

                                                    <td className="p-4 flex justify-end gap-2 items-center">
                                                        {u.is_banned ? (
                                                            <button
                                                                onClick={() => handleBanClick(u.id, false)}
                                                                disabled={isAdmin || loadingUserId === u.id}
                                                                className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg transition-all text-xs font-semibold ${isAdmin
                                                                    ? 'opacity-50 cursor-not-allowed bg-surfaceBorder text-textMuted'
                                                                    : 'bg-green-500/15 text-green-400 hover:bg-green-500/25 border border-green-500/30 shadow-sm'
                                                                    }`}
                                                            >
                                                                <Unlock className={`w-3.5 h-3.5 ${loadingUserId === u.id ? 'animate-spin' : ''}`} />
                                                                {loadingUserId === u.id ? 'Оновлення...' : 'Розбанити'}
                                                            </button>
                                                        ) : (
                                                            <button
                                                                onClick={() => handleBanClick(u.id, true)}
                                                                disabled={isAdmin || loadingUserId === u.id}
                                                                className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg transition-all text-xs font-semibold ${isAdmin
                                                                    ? 'opacity-50 cursor-not-allowed bg-surfaceBorder text-textMuted'
                                                                    : 'bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/20 shadow-sm'
                                                                    }`}
                                                            >
                                                                <Ban className={`w-3.5 h-3.5 ${loadingUserId === u.id ? 'animate-spin' : ''}`} />
                                                                {loadingUserId === u.id ? 'Оновлення...' : 'Забанити'}
                                                            </button>
                                                        )}
                                                    </td>
                                                </tr>
                                            );
                                        })
                                    ) : (
                                        <tr>
                                            <td colSpan={7} className="p-8 text-center text-textMuted">
                                                Користувачів не знайдено
                                            </td>
                                        </tr>
                                    )}
                                </tbody>
                            </table>
                        </div>
                    </div>
                )}

                {activeTab === 'alerts' && (
                    <div className="overflow-x-auto">
                        <table className="w-full text-left border-collapse">
                            <thead>
                                <tr className="bg-mainBg/50 border-b border-surfaceBorder text-textMuted text-sm">
                                    <th className="p-4 font-medium">ID</th>
                                    <th className="p-4 font-medium">User ID</th>
                                    <th className="p-4 font-medium">Тип активності</th>
                                    <th className="p-4 font-medium">Опис</th>
                                    <th className="p-4 font-medium">Дата</th>
                                    <th className="p-4 font-medium text-right">Дія</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-surfaceBorder">
                                {alerts.length > 0 ? (
                                    alerts.map(alert => (
                                        <tr key={alert.id} className="hover:bg-mainBg/30 transition-colors text-textMain text-sm">
                                            <td className="p-4">{alert.id}</td>
                                            <td className="p-4 font-medium">{alert.user_id}</td>
                                            <td className="p-4"><span className="px-2 py-1 rounded bg-yellow-500/10 text-yellow-400 border border-yellow-500/20 text-xs">{alert.activity_type}</span></td>
                                            <td className="p-4 text-textMuted">{alert.description}</td>
                                            <td className="p-4 text-textMuted text-xs">{new Date(alert.created_at).toLocaleString()}</td>
                                            <td className="p-4 flex justify-end">
                                                <button
                                                    onClick={() => handleResolveAlert(alert.id)}
                                                    className="flex items-center gap-1 bg-green-500/10 hover:bg-green-500/20 text-green-400 border border-green-500/20 px-3 py-1.5 rounded-lg transition-colors text-xs font-medium"
                                                >
                                                    <Check className="w-3.5 h-3.5" /> Вирішити
                                                </button>
                                            </td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr>
                                        <td colSpan={6} className="p-8 text-center text-textMuted">
                                            Усі логи безпеки перевірені. Загроз немає.
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {banModalOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200">
                    <div className="bg-surface border border-surfaceBorder rounded-2xl p-6 w-full max-w-md shadow-2xl">
                        <div className="flex justify-between items-center mb-4">
                            <h3 className="text-xl font-bold text-red-400 flex items-center gap-2">
                                <Ban className="w-6 h-6" /> Блокування користувача
                            </h3>
                            <button onClick={() => setBanModalOpen(false)} className="text-textMuted hover:text-textMain">
                                <X className="w-5 h-5" />
                            </button>
                        </div>
                        <p className="text-textMuted text-sm mb-4">
                            Вкажіть причину блокування. Користувач побачить цей текст на своєму екрані при спробі входу.
                        </p>
                        <textarea
                            value={banReason}
                            onChange={(e) => setBanReason(e.target.value)}
                            className="w-full bg-mainBg border border-surfaceBorder rounded-xl p-3 text-textMain focus:border-red-500 outline-none resize-none h-28 mb-6 placeholder:text-surfaceBorder"
                            placeholder="Наприклад: Порушення правил платформи, спам..."
                        />
                        <div className="flex gap-3 justify-end">
                            <button onClick={() => setBanModalOpen(false)} className="px-5 py-2.5 text-textMuted hover:text-textMain font-medium transition-colors">
                                Скасувати
                            </button>
                            <button
                                onClick={() => userToBan && executeBan(userToBan, true, banReason)}
                                className="bg-red-500 hover:bg-red-600 text-white px-6 py-2.5 rounded-xl font-bold transition-colors shadow-lg shadow-red-500/20"
                            >
                                Забанити назавжди
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AdminDashboard;

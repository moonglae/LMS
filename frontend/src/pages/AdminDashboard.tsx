import { useState, useEffect } from 'react';
import { ShieldAlert, Users, Ban, CheckCircle, Unlock, X } from 'lucide-react';

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

    // Стани для модального вікна бану
    const [banModalOpen, setBanModalOpen] = useState(false);
    const [userToBan, setUserToBan] = useState<number | null>(null);
    const [banReason, setBanReason] = useState("");
    const [loadingUserId, setLoadingUserId] = useState<number | null>(null);

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
            setBanModalOpen(false); // Закриваємо вікно
        }
    };

    const handleRestrictFeature = async (userId: number, feature: string, lockStatus: boolean) => {
        // ... (твій код без змін)
        try {
            const res = await fetch(`http://localhost:8080/api/admin/users/restrict?id=${userId}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${getToken()}` },
                body: JSON.stringify({ feature, lock: lockStatus })
            });

            if (res.ok) {
                showMessage(`Чат для користувача ${userId} ${lockStatus ? 'вимкнено' : 'увімкнено'}`, 'success');
                fetchUsers();
            } else {
                const errorData = await res.json().catch(() => null);
                showMessage(errorData?.error || 'Помилка оновлення обмежень', 'error');
            }
        } catch (error) {
            console.error(error);
        }
    };

    const handleResolveAlert = async (alertId: number) => {
        try {
            await apiFetch(`/admin/alerts/resolve?id=${alertId}`, {
                method: 'POST'
            });
            if (res.ok) {
                setAlerts(alerts.filter(a => a.id !== alertId));
                showMessage('Алерт позначено як вирішений', 'success');
            }
        } catch (error) {
            console.error("Помилка:", error);
        }
    };

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

            <div className="bg-surface border border-surfaceBorder rounded-2xl shadow-sm overflow-hidden">
                {activeTab === 'users' && (
                    <div className="overflow-x-auto">
                        <table className="w-full text-left border-collapse">
                            <thead>
                                <tr className="bg-mainBg/50 border-b border-surfaceBorder text-textMuted text-sm">
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
                                {users.map(u => {
                                    const isAdmin = u.role === 'admin';
                                    const aiLocked = isFeatureLocked(u.restricted_features, 'ai_chat');

                                    return (
                                        <tr key={u.id} className="hover:bg-mainBg/30 transition-colors text-textMain text-sm">
                                            <td className="p-4">{u.id}</td>
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
                                            <td className="p-4 flex justify-end gap-2">
                                                {/* КНОПКИ БАНУ З ВИКЛИКОМ НОВОЇ ФУНКЦІЇ */}
                                                {u.is_banned ? (
                                                    <button onClick={() => handleBanClick(u.id, false)} disabled={isAdmin} className={`flex items-center gap-1 px-3 py-1.5 rounded-lg transition-colors ${isAdmin ? 'opacity-50 cursor-not-allowed bg-surfaceBorder' : 'bg-green-500/10 text-green-400 hover:bg-green-500/20 border border-green-500/20'}`}>
                                                        <Unlock className="w-4 h-4" /> Розбанити
                                                    </button>
                                                ) : (
                                                    <button onClick={() => handleBanClick(u.id, true)} disabled={isAdmin} className={`flex items-center gap-1 px-3 py-1.5 rounded-lg transition-colors ${isAdmin ? 'opacity-50 cursor-not-allowed bg-surfaceBorder' : 'bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/20'}`}>
                                                        <Ban className="w-4 h-4" /> Забанити
                                                    </button>
                                                )}
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
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
                                {alerts.map(alert => (
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
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Модальне вікно бану */}
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

import { useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Loader2, CheckCircle, XCircle, ArrowRight, Brain, RotateCcw, Info } from 'lucide-react';
import { apiFetch } from '../api';

interface Question {
    type: 'choice' | 'fill';
    question: string;
    options?: string[];
    correct_answer: string;
    explanation: string;
}

interface TestData {
    rules: string[];
    questions: Question[];
}

type TestPhase = 'setup' | 'loading' | 'testing' | 'results';

export default function AITest() {
    const location = useLocation();
    const [phase, setPhase] = useState<TestPhase>('setup');

    const [topic, setTopic] = useState(location.state?.topic || '');
    const [theory, setTheory] = useState(location.state?.theory || '');
    const [questionCount, setQuestionCount] = useState(5);
    const [error, setError] = useState('');

    const [questions, setQuestions] = useState<Question[]>([]);
    const [rules, setRules] = useState<string[]>([]);
    const [currentIndex, setCurrentIndex] = useState(0);
    const [userAnswer, setUserAnswer] = useState('');
    const [isAnswerChecked, setIsAnswerChecked] = useState(false);
    const [score, setScore] = useState(0);

    const handleGenerate = async () => {
        if (!topic.trim()) {
            setError('Тема є обов\'язковою!');
            return;
        }

        setPhase('loading');
        setError('');

        try {
            // Використовуємо apiFetch замість жорсткого localhost
            const data = (await apiFetch('/practice/generate-test', {
                method: 'POST',
                body: JSON.stringify({ topic, theory, question_count: questionCount })
            })) as TestData;

            setQuestions(data.questions);
            setRules(data.rules || []);
            setPhase('testing');
            setCurrentIndex(0);
            setScore(0);
            setUserAnswer('');
            setIsAnswerChecked(false);
        } catch (err: any) {
            setError(err.message || 'Помилка генерації тесту');
            setPhase('setup');
        }
    };

    const handleCheckAnswer = () => {
        if (!userAnswer.trim()) return;
        setIsAnswerChecked(true);

        const currentQ = questions[currentIndex];

        const normalize = (val: string) => {
            const trimmed = val.trim().toLowerCase();
            return trimmed === '' ? '-' : trimmed;
        };

        const isCorrect = normalize(userAnswer) === normalize(currentQ.correct_answer);
        if (isCorrect) setScore(prev => prev + 1);
    };

    const handleNextQuestion = () => {
        if (currentIndex + 1 < questions.length) {
            setCurrentIndex(prev => prev + 1);
            setUserAnswer('');
            setIsAnswerChecked(false);
        } else {
            setPhase('results');
        }
    };

    const resetTest = () => {
        setPhase('setup');
    };

    if (phase === 'setup') {
        return (
            <div className="max-w-2xl mx-auto p-6 bg-surface border border-surfaceBorder rounded-2xl shadow-sm">
                <div className="flex items-center gap-3 mb-6">
                    <Brain className="w-8 h-8 text-primary" />
                    <h2 className="text-2xl font-bold text-textMain">Генератор тестів від ШІ</h2>
                </div>
                {error && <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 text-red-500 rounded-xl">{error}</div>}
                <div className="space-y-5">
                    <div>
                        <label className="block text-sm font-medium text-textMuted mb-2">Тема (обов'язково)</label>
                        <input type="text" value={topic} onChange={(e) => setTopic(e.target.value)} maxLength={150} className="w-full p-3 bg-mainBg border border-surfaceBorder rounded-xl focus:border-primary outline-none transition-colors" />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-textMuted mb-2">Теорія (опціонально, макс 4000 симв.)</label>
                        <textarea value={theory} onChange={(e) => setTheory(e.target.value)} rows={5} maxLength={4000} className="w-full p-3 bg-mainBg border border-surfaceBorder rounded-xl focus:border-primary outline-none transition-colors resize-none" />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-textMuted mb-2">Кількість питань</label>
                        <select value={questionCount} onChange={(e) => setQuestionCount(Number(e.target.value))} className="w-full p-3 bg-mainBg border border-surfaceBorder rounded-xl outline-none">
                            <option value={5}>5 питань (Швидко)</option>
                            <option value={10}>10 питань (Нормально)</option>
                            <option value={15}>15 питань (Глибоко)</option>
                            <option value={20}>20 питань (Максимум)</option>
                        </select>
                    </div>
                    <button onClick={handleGenerate} className="w-full py-4 bg-primary text-white rounded-xl font-medium hover:bg-primary/95 transition-colors flex items-center justify-center gap-2">
                        Згенерувати тест
                    </button>
                </div>
            </div>
        );
    }

    if (phase === 'loading') {
        return (
            <div className="max-w-2xl mx-auto p-12 text-center bg-surface rounded-2xl border border-surfaceBorder">
                <Loader2 className="w-12 h-12 text-primary animate-spin mx-auto mb-4" />
                <h3 className="text-xl font-semibold text-textMain mb-2">ШІ аналізує матеріал...</h3>
            </div>
        );
    }

    if (phase === 'results') {
        return (
            <div className="max-w-2xl mx-auto p-8 text-center bg-surface border border-surfaceBorder rounded-2xl">
                <div className="w-20 h-20 bg-green-500/10 text-green-500 rounded-full flex items-center justify-center mx-auto mb-6"><CheckCircle className="w-10 h-10" /></div>
                <h2 className="text-3xl font-bold text-textMain mb-2">Тест завершено!</h2>
                <p className="text-xl text-textMuted mb-8">Ваш результат: <span className="font-bold text-primary">{score} / {questions.length}</span></p>
                <button onClick={resetTest} className="py-3 px-6 bg-surface border border-surfaceBorder rounded-xl font-medium hover:bg-mainBg transition-colors inline-flex items-center gap-2">
                    <RotateCcw className="w-5 h-5" /> Створити новий тест
                </button>
            </div>
        );
    }

    const currentQ = questions[currentIndex];
    const isCorrect = userAnswer.trim().toLowerCase() === currentQ?.correct_answer.toLowerCase();

    return (
        <div className="max-w-5xl mx-auto flex flex-col md:flex-row gap-6 items-start">
            <div className="flex-1 w-full p-6 bg-surface border border-surfaceBorder rounded-2xl shadow-sm">
                <div className="flex justify-between items-center mb-6">
                    <span className="text-sm font-medium text-textMuted">Тема: {topic}</span>
                    <span className="px-3 py-1 bg-primary/10 text-primary rounded-full text-sm font-bold">
                        {currentIndex + 1} / {questions.length}
                    </span>
                </div>

                <div className="mb-8">
                    <h3 className="text-xl font-semibold text-textMain mb-6">{currentQ?.question}</h3>
                    <div className="space-y-3">
                        {currentQ?.type === 'choice' ? (
                            currentQ.options?.map((opt, idx) => (
                                <button
                                    key={idx}
                                    disabled={isAnswerChecked}
                                    onClick={() => setUserAnswer(opt)}
                                    className={`w-full text-left p-4 rounded-xl border transition-all ${userAnswer === opt ? 'border-primary bg-primary/5' : 'border-surfaceBorder bg-mainBg hover:border-textMuted'} ${isAnswerChecked && opt === currentQ.correct_answer ? '!border-green-500 !bg-green-500/10' : ''} ${isAnswerChecked && userAnswer === opt && !isCorrect ? '!border-red-500 !bg-red-500/10' : ''} disabled:cursor-default`}
                                >
                                    {opt}
                                </button>
                            ))
                        ) : (
                            <input
                                type="text"
                                disabled={isAnswerChecked}
                                value={userAnswer}
                                onChange={(e) => setUserAnswer(e.target.value)}
                                placeholder="Введіть пропущене слово..."
                                className={`w-full p-4 bg-mainBg border rounded-xl outline-none transition-colors ${isAnswerChecked ? (isCorrect ? 'border-green-500 bg-green-500/5' : 'border-red-500 bg-red-500/5') : 'border-surfaceBorder focus:border-primary'}`}
                                onKeyDown={(e) => { if (e.key === 'Enter' && !isAnswerChecked && userAnswer.trim()) handleCheckAnswer(); }}
                            />
                        )}
                    </div>
                </div>

                {isAnswerChecked && (
                    <div className={`p-5 rounded-xl mb-6 ${isCorrect ? 'bg-green-500/10 text-green-700' : 'bg-red-500/10 text-red-700'}`}>
                        <div className="flex items-center gap-2 mb-2 font-bold text-lg">
                            {isCorrect ? <CheckCircle className="w-6 h-6" /> : <XCircle className="w-6 h-6" />}
                            {isCorrect ? 'Правильно!' : 'Неправильно'}
                        </div>
                        {!isCorrect && <p className="mb-2">Правильна відповідь: <span className="font-bold">{currentQ.correct_answer}</span></p>}
                        <p className="opacity-90">{currentQ.explanation}</p>
                    </div>
                )}

                <div className="flex justify-end">
                    {!isAnswerChecked ? (
                        <button onClick={handleCheckAnswer} disabled={!userAnswer.trim()} className="py-3 px-6 bg-primary text-white rounded-xl font-medium disabled:opacity-50 disabled:cursor-not-allowed">
                            Перевірити
                        </button>
                    ) : (
                        <button onClick={handleNextQuestion} className="py-3 px-6 bg-primary text-white rounded-xl font-medium flex items-center gap-2">
                            {currentIndex + 1 === questions.length ? 'Завершити' : 'Наступне питання'} <ArrowRight className="w-5 h-5" />
                        </button>
                    )}
                </div>
            </div>

            {rules && rules.length > 0 && (
                <div className="w-full md:w-72 shrink-0 p-5 bg-blue-500/5 border border-blue-500/20 rounded-2xl h-fit">
                    <h4 className="font-bold text-blue-400 mb-4 flex items-center gap-2">
                        <Info className="w-5 h-5" /> Правила вводу
                    </h4>
                    <ul className="space-y-3 text-sm text-blue-300/90 list-disc list-inside">
                        {rules.map((rule, idx) => (
                            <li key={idx} className="leading-relaxed">{rule}</li>
                        ))}
                    </ul>
                </div>
            )}
        </div>
    );
}

import { useState, useEffect, useRef } from 'react';
import { Loader2 } from 'lucide-react';
import { apiFetch } from '../api';

type DictEntry = {
    word: string;
    translation: string;
};

interface AutocompleteInputProps {
    value: string;
    onChange: (value: string) => void;
    onSelect: (word: string, translation: string) => void;
    placeholder?: string;
}

export default function AutocompleteInput({ value, onChange, onSelect, placeholder = "Введіть слово..." }: AutocompleteInputProps) {
    const [suggestions, setSuggestions] = useState<DictEntry[]>([]);
    const [isOpen, setIsOpen] = useState(false);
    const [isLoading, setIsLoading] = useState(false);

    const wrapperRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (value.trim().length < 2) {
            setSuggestions([]);
            setIsOpen(false);
            return;
        }

        setIsLoading(true);
        const delayDebounceFn = setTimeout(async () => {
            try {
                // apiFetch автоматично додає /api
                const data = await apiFetch(`/autocomplete?q=${encodeURIComponent(value)}`);
                setSuggestions(Array.isArray(data) ? data : []);
                if (Array.isArray(data) && data.length > 0) setIsOpen(true);
            } catch (err) {
                console.error("Помилка автокомпліту:", err);
            } finally {
                setIsLoading(false);
            }
        }, 300);

        return () => clearTimeout(delayDebounceFn);
    }, [value]);

    useEffect(() => {
        function handleClickOutside(event: MouseEvent) {
            if (wrapperRef.current && !wrapperRef.current.contains(event.target as Node)) {
                setIsOpen(false);
            }
        }
        document.addEventListener("mousedown", handleClickOutside);
        return () => document.removeEventListener("mousedown", handleClickOutside);
    }, []);

    const handleSelectSuggestion = (item: DictEntry) => {
        setIsOpen(false);
        onSelect(item.word, item.translation);
    };

    return (
        <div ref={wrapperRef} className="relative w-full flex-1">
            <div className="relative">
                <input
                    type="text"
                    value={value}
                    onChange={(e) => onChange(e.target.value)}
                    onFocus={() => value.trim().length >= 2 && suggestions.length > 0 && setIsOpen(true)}
                    placeholder={placeholder}
                    className="w-full bg-mainBg p-2 pr-8 rounded-lg border border-surfaceBorder focus:border-primary outline-none transition-colors"
                />
                {isLoading && (
                    <Loader2 className="absolute right-2 top-1/2 -translate-y-1/2 text-primary w-4 h-4 animate-spin" />
                )}
            </div>

            {isOpen && suggestions.length > 0 && (
                <div className="absolute left-0 right-0 mt-1 bg-surface border border-surfaceBorder rounded-lg shadow-xl max-h-48 overflow-y-auto z-50 divide-y divide-surfaceBorder/50">
                    {suggestions.map((item, index) => (
                        <div
                            key={index}
                            onClick={() => handleSelectSuggestion(item)}
                            className="p-2 hover:bg-mainBg cursor-pointer transition-colors text-left"
                        >
                            <span className="font-bold text-primary block text-sm">
                                {item.word}
                            </span>
                            <span className="text-xs text-textMuted whitespace-pre-line line-clamp-2">
                                {item.translation}
                            </span>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
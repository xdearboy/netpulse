import { useState, useCallback, useRef } from 'react';
import axios, { AxiosError } from 'axios';

export interface APIError {
    status: number;
    message: string;
    retryAfter?: number;
}

export const useAPI = () => {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<APIError | null>(null);
    const abortRef = useRef<AbortController | null>(null);

    const request = useCallback(async <T,>(url: string): Promise<T | null> => {
        abortRef.current?.abort();
        const controller = new AbortController();
        abortRef.current = controller;

        setLoading(true);
        setError(null);

        try {
            const res = await axios.get<T>(url, { signal: controller.signal });
            return res.data;
        } catch (err) {
            if (axios.isCancel(err)) return null;

            const ax = err as AxiosError<{ error?: string; message?: string; retry_after?: number }>;
            const status = ax.response?.status ?? 0;
            const body = ax.response?.data;

            setError({
                status,
                message: body?.message || body?.error || ax.message || 'Request failed',
                retryAfter: body?.retry_after,
            });
            return null;
        } finally {
            setLoading(false);
        }
    }, []);

    return { request, loading, error };
};

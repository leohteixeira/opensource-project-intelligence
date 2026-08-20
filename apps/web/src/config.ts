/**
 * Frontend configuration. The web application talks to the backend only over
 * the versioned HTTP contract, never to the database or to a vendor API.
 */
export interface WebConfig {
  readonly apiBaseUrl: string;
}

const DEFAULT_API_BASE_URL = 'http://localhost:8100';

export function readConfig(env: Record<string, string | undefined>): WebConfig {
  const configured = env.VITE_API_BASE_URL?.trim();

  return {
    apiBaseUrl: configured && configured.length > 0 ? configured : DEFAULT_API_BASE_URL,
  };
}

export const config = readConfig(import.meta.env as unknown as Record<string, string | undefined>);

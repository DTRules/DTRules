// DTRules Go API client for poker demo

// API URL priority:
// 1. Runtime override: window.__DTRULES_API__
// 2. Build-time env: import.meta.env.PUBLIC_DTRULES_API
// 3. Production default: https://dtrules-api.fly.dev (when deployed)
// 4. Development fallback: http://localhost:8080
const getApiBase = (): string => {
  if (typeof window !== 'undefined') {
    // Check for runtime override
    if ((window as any).__DTRULES_API__) {
      return (window as any).__DTRULES_API__;
    }
  }

  // Check for build-time environment variable
  const envApi = (import.meta as any).env?.PUBLIC_DTRULES_API;
  if (envApi) {
    return envApi;
  }

  // Production vs development detection
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    // In production, use the Fly.io API
    return 'https://dtrules-api.fly.dev';
  }

  // Local development
  return 'http://localhost:8080';
};

const API_BASE = getApiBase();

const API_TIMEOUT_MS = 5000; // 5 second timeout

export interface PokerGameState {
  pot: number;
  current_bet: number;
  big_blind: number;
  phase: string;
}

export interface PokerPlayerState {
  name: string;
  chips: number;
  current_bet: number;
  hand_strength: number;
  pot_odds: number;
  can_check: boolean;
  position: string;
  archetype: string;
}

export interface PokerDecisionResult {
  action: 'RAISE' | 'CALL' | 'CHECK' | 'FOLD';
  raise_amount: number;
  reasoning: string;
}

export interface ExecuteResponse {
  success: boolean;
  result?: {
    decision?: PokerDecisionResult;
  };
  error?: string;
}

let projectLoaded = false;

/**
 * Fetch with timeout support
 */
async function fetchWithTimeout(
  url: string,
  options: RequestInit,
  timeoutMs: number = API_TIMEOUT_MS
): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      ...options,
      signal: controller.signal,
    });
    return response;
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function ensurePokerProjectLoaded(): Promise<boolean> {
  if (projectLoaded) return true;

  try {
    const response = await fetchWithTimeout(`${API_BASE}/api/project/open`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: 'sampleprojects/Poker' })
    });

    if (response.ok) {
      projectLoaded = true;
      return true;
    }
    return false;
  } catch {
    return false;
  }
}

export class DTRulesApiError extends Error {
  constructor(message: string, public readonly statusCode?: number) {
    super(message);
    this.name = 'DTRulesApiError';
  }
}

/**
 * Validate that the decision result has all required fields
 */
function validateDecisionResult(decision: unknown): decision is PokerDecisionResult {
  if (!decision || typeof decision !== 'object') return false;
  const d = decision as Record<string, unknown>;

  // Check action is a valid string
  if (typeof d.action !== 'string') return false;
  const validActions = ['RAISE', 'CALL', 'CHECK', 'FOLD'];
  if (!validActions.includes(d.action.toUpperCase())) return false;

  // raise_amount should be a number (can be 0)
  if (typeof d.raise_amount !== 'number' || d.raise_amount < 0) {
    d.raise_amount = 0; // Default to 0 if invalid
  }

  return true;
}

export async function executePokerDecision(
  gameState: PokerGameState,
  playerState: PokerPlayerState
): Promise<PokerDecisionResult> {
  const loaded = await ensurePokerProjectLoaded();
  if (!loaded) {
    throw new DTRulesApiError('Failed to load Poker project');
  }

  let response: Response;
  try {
    response = await fetchWithTimeout(`${API_BASE}/api/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        tableName: 'Poker_Decision',
        data: {
          game: gameState,
          player: playerState,
          decision: {
            action: '',
            raise_amount: 0,
            reasoning: ''
          }
        },
        trace: false
      })
    });
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      throw new DTRulesApiError('API request timed out');
    }
    throw new DTRulesApiError(`Network error: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }

  if (!response.ok) {
    throw new DTRulesApiError(`API request failed: ${response.statusText}`, response.status);
  }

  let result: ExecuteResponse;
  try {
    result = await response.json();
  } catch {
    throw new DTRulesApiError('Invalid JSON response from API');
  }

  if (result.success && result.result?.decision) {
    if (validateDecisionResult(result.result.decision)) {
      // Normalize action to uppercase
      result.result.decision.action = result.result.decision.action.toUpperCase() as PokerDecisionResult['action'];
      return result.result.decision;
    }
    throw new DTRulesApiError('Invalid decision format from API');
  }

  throw new DTRulesApiError(result.error || 'Decision table execution failed');
}

export async function isApiAvailable(): Promise<boolean> {
  try {
    const response = await fetchWithTimeout(
      `${API_BASE}/api/health`,
      { method: 'GET' },
      2000 // Shorter timeout for health check
    );
    return response.ok;
  } catch {
    return false;
  }
}

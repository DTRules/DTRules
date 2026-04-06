// DTRules Go API client for poker demo

const API_BASE = typeof window !== 'undefined'
  ? (window as any).__DTRULES_API__ || 'http://localhost:8080'
  : 'http://localhost:8080';

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

export async function ensurePokerProjectLoaded(): Promise<boolean> {
  if (projectLoaded) return true;

  try {
    const response = await fetch(`${API_BASE}/api/project/open`, {
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

export async function executePokerDecision(
  gameState: PokerGameState,
  playerState: PokerPlayerState
): Promise<PokerDecisionResult | null> {
  try {
    await ensurePokerProjectLoaded();

    const response = await fetch(`${API_BASE}/api/execute`, {
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

    if (!response.ok) return null;

    const result: ExecuteResponse = await response.json();
    if (result.success && result.result?.decision) {
      return result.result.decision;
    }
    return null;
  } catch {
    return null;
  }
}

export function isApiAvailable(): Promise<boolean> {
  return fetch(`${API_BASE}/api/health`, { method: 'GET' })
    .then(r => r.ok)
    .catch(() => false);
}

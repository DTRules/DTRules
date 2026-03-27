// Decision Table Data Definitions
// Data-driven decision tables for each AI archetype

export interface DecisionCondition {
  id: string;
  label: string;
  description?: string;
  threshold?: number;
  thresholdType?: 'gte' | 'lte' | 'eq' | 'gt' | 'lt';
}

export interface DecisionColumn {
  id: number;
  conditions: Record<string, 'Y' | 'N' | '-'>;
  action: 'RAISE' | 'CALL' | 'CHECK' | 'FOLD';
  actionColor: string;
}

export interface DecisionTableDefinition {
  archetype: string;
  name: string;
  shortName: string;
  description: string;
  vpip: string; // Voluntarily Put $ In Pot percentage
  badgeColor: string;
  conditions: DecisionCondition[];
  columns: DecisionColumn[];
}

// TAG (Tight-Aggressive) Decision Table
export const TAG_TABLE: DecisionTableDefinition = {
  archetype: 'TAG',
  name: 'Tight-Aggressive',
  shortName: 'TAG',
  description: 'Plays premium hands, bets/raises with strength. Folds weak hands but aggressive when in a pot.',
  vpip: '~18%',
  badgeColor: 'bg-blue-600',
  conditions: [
    { id: 'str70', label: 'Hand Strength >= 70%', threshold: 0.7, thresholdType: 'gte' },
    { id: 'str50', label: 'Hand Strength >= 50%', threshold: 0.5, thresholdType: 'gte' },
    { id: 'potOdds', label: 'Pot Odds Favorable', description: 'Pot odds < strength * 0.8' },
    { id: 'canCheck', label: 'Can Check', description: 'No bet to call' },
  ],
  columns: [
    { id: 1, conditions: { str70: 'Y', str50: '-', potOdds: '-', canCheck: '-' }, action: 'RAISE', actionColor: 'text-yellow-400' },
    { id: 2, conditions: { str70: 'N', str50: 'Y', potOdds: 'Y', canCheck: '-' }, action: 'CALL', actionColor: 'text-blue-400' },
    { id: 3, conditions: { str70: 'N', str50: 'Y', potOdds: 'N', canCheck: 'Y' }, action: 'CHECK', actionColor: 'text-gray-300' },
    { id: 4, conditions: { str70: 'N', str50: 'N', potOdds: '-', canCheck: 'Y' }, action: 'CHECK', actionColor: 'text-gray-300' },
    { id: 5, conditions: { str70: 'N', str50: 'N', potOdds: '-', canCheck: 'N' }, action: 'FOLD', actionColor: 'text-red-400' },
  ],
};

// LAG (Loose-Aggressive) Decision Table
export const LAG_TABLE: DecisionTableDefinition = {
  archetype: 'LAG',
  name: 'Loose-Aggressive',
  shortName: 'LAG',
  description: 'Plays many hands, applies pressure with bets and raises. Uses position to bluff.',
  vpip: '~35%',
  badgeColor: 'bg-orange-600',
  conditions: [
    { id: 'str50', label: 'Hand Strength >= 50%', threshold: 0.5, thresholdType: 'gte' },
    { id: 'position', label: 'Position = Late/Button', description: 'Favorable position' },
    { id: 'str25', label: 'Hand Strength >= 25%', threshold: 0.25, thresholdType: 'gte' },
    { id: 'canCheck', label: 'Can Check', description: 'No bet to call' },
  ],
  columns: [
    { id: 1, conditions: { str50: 'Y', position: '-', str25: '-', canCheck: '-' }, action: 'RAISE', actionColor: 'text-yellow-400' },
    { id: 2, conditions: { str50: 'N', position: 'Y', str25: 'Y', canCheck: '-' }, action: 'RAISE', actionColor: 'text-yellow-400' },
    { id: 3, conditions: { str50: 'N', position: 'Y', str25: 'N', canCheck: 'Y' }, action: 'CHECK', actionColor: 'text-gray-300' },
    { id: 4, conditions: { str50: 'N', position: 'N', str25: 'Y', canCheck: '-' }, action: 'CALL', actionColor: 'text-blue-400' },
    { id: 5, conditions: { str50: 'N', position: 'N', str25: 'N', canCheck: '-' }, action: 'FOLD', actionColor: 'text-red-400' },
  ],
};

// Rock (Tight-Passive) Decision Table
export const ROCK_TABLE: DecisionTableDefinition = {
  archetype: 'Rock',
  name: 'Tight-Passive',
  shortName: 'Rock',
  description: 'Only plays very strong hands. Rarely raises, prefers to call. Very predictable.',
  vpip: '~12%',
  badgeColor: 'bg-gray-600',
  conditions: [
    { id: 'str85', label: 'Hand Strength >= 85%', threshold: 0.85, thresholdType: 'gte' },
    { id: 'str60', label: 'Hand Strength >= 60%', threshold: 0.6, thresholdType: 'gte' },
    { id: 'canCheck', label: 'Can Check', description: 'No bet to call' },
  ],
  columns: [
    { id: 1, conditions: { str85: 'Y', str60: '-', canCheck: '-' }, action: 'RAISE', actionColor: 'text-yellow-400' },
    { id: 2, conditions: { str85: 'N', str60: 'Y', canCheck: '-' }, action: 'CALL', actionColor: 'text-blue-400' },
    { id: 3, conditions: { str85: 'N', str60: 'N', canCheck: 'Y' }, action: 'CHECK', actionColor: 'text-gray-300' },
    { id: 4, conditions: { str85: 'N', str60: 'N', canCheck: 'N' }, action: 'FOLD', actionColor: 'text-red-400' },
  ],
};

// Calling Station Decision Table
export const CALLING_TABLE: DecisionTableDefinition = {
  archetype: 'Calling',
  name: 'Calling Station',
  shortName: 'Calling',
  description: 'Calls frequently with weak hands. Rarely folds or raises. Easy to value bet against.',
  vpip: '~40%',
  badgeColor: 'bg-green-600',
  conditions: [
    { id: 'str20', label: 'Hand Strength >= 20%', threshold: 0.2, thresholdType: 'gte' },
    { id: 'smallBet', label: 'Bet <= 2x Big Blind', description: 'Small bet to call' },
    { id: 'canCheck', label: 'Can Check', description: 'No bet to call' },
  ],
  columns: [
    { id: 1, conditions: { str20: 'Y', smallBet: '-', canCheck: '-' }, action: 'CALL', actionColor: 'text-blue-400' },
    { id: 2, conditions: { str20: 'N', smallBet: 'Y', canCheck: '-' }, action: 'CALL', actionColor: 'text-blue-400' },
    { id: 3, conditions: { str20: 'N', smallBet: 'N', canCheck: 'Y' }, action: 'CHECK', actionColor: 'text-gray-300' },
    { id: 4, conditions: { str20: 'N', smallBet: 'N', canCheck: 'N' }, action: 'FOLD', actionColor: 'text-red-400' },
  ],
};

// Map archetype names to table definitions
export const DECISION_TABLES: Record<string, DecisionTableDefinition> = {
  'TAG': TAG_TABLE,
  'LAG': LAG_TABLE,
  'Rock': ROCK_TABLE,
  'Calling': CALLING_TABLE,
};

// Get table definition by archetype
export function getDecisionTable(archetype: string): DecisionTableDefinition | undefined {
  return DECISION_TABLES[archetype];
}

// Get all table definitions
export function getAllDecisionTables(): DecisionTableDefinition[] {
  return Object.values(DECISION_TABLES);
}

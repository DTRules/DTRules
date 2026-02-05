// Poker type definitions - ported from Go poker.go

export interface PokerCard {
  suit: string;
  rank: number;
  rankName: string;
}

export interface PokerPlayer {
  id: string;
  name: string;
  chips: number;
  holeCards: PokerCard[];
  folded: boolean;
  allIn: boolean;
  currentBet: number;
  totalBet: number;
  hasActed: boolean;
  isDealer: boolean;
  isSmallBlind: boolean;
  isBigBlind: boolean;
  isHuman: boolean;
  handRank?: number;
  handName?: string;
  handValue?: number;
  archetype?: string;
  handStrength?: number;
  position?: string;
  bestHand?: PokerCard[];
}

export interface SidePot {
  amount: number;
  eligible: string[]; // Player IDs eligible for this pot
}

export interface ActionLog {
  playerId: string;
  playerName: string;
  action: string;
  amount?: number;
  round: string;
  timestamp: number;
}

export interface ShowdownResult {
  playerId: string;
  playerName: string;
  holeCards: PokerCard[];
  bestHand: PokerCard[];
  handRank: number;
  handName: string;
  handValue: number;
  won: boolean;
  wonAmount?: number;
  rank: number; // Position (1st, 2nd, etc.)
}

export interface PokerGame {
  id: string;
  players: PokerPlayer[];
  communityCards: PokerCard[];
  deck: PokerCard[];
  pot: number;
  sidePots?: SidePot[];
  currentBet: number;
  minRaise: number;
  round: string;
  roundNumber: number;
  smallBlind: number;
  bigBlind: number;
  dealerPosition: number;
  currentPlayerIndex: number;
  handComplete: boolean;
  roundComplete: boolean;
  activePlayerCount: number;
  handNumber: number;
  lastAction?: string;
  lastActionPlayer?: string;
  winners?: string[];
  winningHand?: string;
  winAmount?: number;
  actionHistory: ActionLog[];
  showdownResults?: ShowdownResult[];
  pendingReveal?: string; // "flop", "turn", "river", "showdown", or ""
  newCommunityCards?: PokerCard[]; // Cards just dealt (for animation)
  runningOutBoard?: boolean; // True when dealing remaining streets for all-in scenario
}

export interface PokerAction {
  playerId: string;
  actionType: string;
  amount: number;
}

export interface EvaluateResponse {
  success: boolean;
  rank: number;
  rankName: string;
  handName: string; // Detailed name like "Two Pair, Kings and Tens"
  isFlush: boolean;
  isStraight: boolean;
  straightHigh?: number;
  highCard: number;
  handValue: number;
  kickers: number[];
  bestFiveCards?: PokerCard[];
  error?: string;
}

export interface DealRequest {
  playerCount: number;
  humanPosition: number;
  smallBlind: number;
  bigBlind: number;
  startingChips: number;
  dealerPosition: number;
}

export interface DealResponse {
  success: boolean;
  game?: PokerGame;
  error?: string;
}

export interface ActionResponse {
  success: boolean;
  valid: boolean;
  game?: PokerGame;
  message?: string;
  error?: string;
}

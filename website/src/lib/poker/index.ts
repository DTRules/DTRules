// Poker module - re-exports all public APIs

// Types
export type {
  PokerCard,
  PokerPlayer,
  PokerGame,
  SidePot,
  ActionLog,
  ShowdownResult,
  PokerAction,
  EvaluateResponse,
  DealRequest,
  DealResponse,
  ActionResponse,
} from './types';

// Constants
export {
  SUITS,
  CARD_RANK_NAMES,
  CARD_RANK_FULL_NAMES,
  CARD_RANK_SINGULAR,
  HAND_RANK_NAMES,
  ARCHETYPE_TAG,
  ARCHETYPE_LAG,
  ARCHETYPE_ROCK,
  ARCHETYPE_CALLING,
  ARCHETYPES,
  PLAYER_NAMES,
} from './constants';

// Deck operations
export {
  secureRandom,
  createDeck,
  shuffleDeck,
  createShuffledDeck,
} from './deck';

// Hand evaluation
export {
  evaluatePokerHand,
  selectBestFiveCards,
  compareHands,
} from './handEval';

// Game logic
export {
  dealNewPokerHand,
  processPokerAction,
  checkAndAdvanceRound,
  advanceToNextPlayer,
  advanceToNextStreet,
  dealNextStreet,
  awardPot,
  clearPendingReveal,
  continueRunOut,
  sanitizeGameForPlayer,
  sanitizeGameForHuman,
} from './gameLogic';

// AI decision
export type { AIActionResult } from './aiDecision';
export {
  calculateHandStrength,
  calculatePreflopStrength,
  getPlayerPosition,
  makeAIDecision,
  processAllAITurns,
  processAITurnsSync,
} from './aiDecision';

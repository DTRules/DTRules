// AI Decision logic - uses Go DTRules engine exclusively (no JavaScript fallback)

import type { PokerGame, PokerPlayer, PokerAction } from './types';
import { ARCHETYPE_TAG, ARCHETYPE_LAG, ARCHETYPE_ROCK, ARCHETYPE_CALLING } from './constants';
import { secureRandom } from './deck';
import { selectBestFiveCards } from './handEval';
import { processPokerAction, advanceToNextPlayer, checkAndAdvanceRound } from './gameLogic';
import { executePokerDecision, isApiAvailable, DTRulesApiError, type PokerGameState, type PokerPlayerState } from '../api/dtrules';

// Promise-based queue to prevent concurrent AI processing
let aiProcessingQueue: Promise<void> = Promise.resolve();

// API availability state
let apiAvailable = false;
let apiCheckDone = false;

export class EngineUnavailableError extends Error {
  constructor(message: string = 'DTRules Go engine is not available. Please start the API server.') {
    super(message);
    this.name = 'EngineUnavailableError';
  }
}

export async function checkApiAvailability(): Promise<boolean> {
  if (!apiCheckDone) {
    apiAvailable = await isApiAvailable();
    apiCheckDone = true;
  }
  return apiAvailable;
}

export function isUsingGoApi(): boolean {
  return apiAvailable;
}

export function requireApiAvailable(): void {
  if (!apiAvailable) {
    throw new EngineUnavailableError();
  }
}

/**
 * Calculate hand strength from hand rank (1-10) to (0.0-1.0)
 */
export function calculateHandStrength(rank: number): number {
  const strengths: Record<number, number> = {
    10: 1.0,  // Royal Flush
    9: 0.95,  // Straight Flush
    8: 0.9,   // Four of a Kind
    7: 0.85,  // Full House
    6: 0.75,  // Flush
    5: 0.7,   // Straight
    4: 0.55,  // Three of a Kind
    3: 0.45,  // Two Pair
    2: 0.3,   // One Pair
    1: 0.15,  // High Card
  };
  return strengths[rank] || 0.15;
}

/**
 * Calculate preflop hand strength based on hole cards
 */
export function calculatePreflopStrength(holeCards: { rank: number; suit: string }[]): number {
  if (holeCards.length < 2) {
    return 0.2;
  }

  const rank1 = holeCards[0].rank;
  const rank2 = holeCards[1].rank;
  const suited = holeCards[0].suit === holeCards[1].suit;

  const highCard = Math.max(rank1, rank2);
  const lowCard = Math.min(rank1, rank2);

  let strength = 0.0;

  // Pocket pairs
  if (rank1 === rank2) {
    // AA=0.95, KK=0.92, ..., 22=0.5
    strength = 0.5 + (rank1 - 2) * 0.0375;
  } else {
    // Base strength from high card
    strength = (highCard - 2) * 0.02;

    // Suited bonus
    if (suited) {
      strength += 0.06;
    }

    // Connectedness bonus
    const gap = highCard - lowCard - 1;
    if (gap === 0) {
      strength += 0.04; // Connectors
    } else if (gap === 1) {
      strength += 0.02; // One-gappers
    }

    // Both cards 10+ bonus (broadway)
    if (lowCard >= 10) {
      strength += 0.1;
    } else if (highCard >= 14) { // Ace-x
      strength += 0.08;
    } else if (highCard >= 13) { // King-x
      strength += 0.04;
    }
  }

  // Clamp
  return Math.max(0.0, Math.min(1.0, strength));
}

/**
 * Get player's position relative to the dealer
 */
export function getPlayerPosition(playerIdx: number, dealerPos: number, numPlayers: number): string {
  const dist = (playerIdx - dealerPos + numPlayers) % numPlayers;

  if (numPlayers === 2) {
    if (dist === 0) {
      return 'button'; // SB/Button in heads-up
    }
    return 'big_blind';
  }

  if (dist === 0) {
    return 'button';
  } else if (dist === 1) {
    return 'small_blind';
  } else if (dist === 2) {
    return 'big_blind';
  } else if (dist <= Math.floor(numPlayers / 3) + 1) {
    return 'early';
  } else if (dist <= Math.floor(2 * numPlayers / 3) + 1) {
    return 'middle';
  }
  return 'late';
}

/**
 * Make an AI decision using DTRules Go API (no fallback - requires API server)
 * @throws {EngineUnavailableError} if API is not available
 * @throws {DTRulesApiError} if API call fails
 */
export async function makeAIDecisionAsync(player: PokerPlayer, game: PokerGame, handStrength: number): Promise<PokerAction> {
  // Require API to be available - no fallback
  requireApiAvailable();

  const toCall = game.currentBet - player.currentBet;
  const canCheck = toCall === 0;
  let potOdds = 0.0;
  if (game.pot + toCall > 0) {
    potOdds = toCall / (game.pot + toCall);
  }

  // Convert to API types
  const gameState: PokerGameState = {
    pot: game.pot,
    current_bet: game.currentBet,
    big_blind: game.bigBlind,
    phase: game.round || 'preflop', // Use round field, default to preflop
  };

  const playerState: PokerPlayerState = {
    name: player.name,
    chips: player.chips,
    current_bet: player.currentBet,
    hand_strength: Math.round(handStrength * 100), // Convert 0-1 to 0-100
    pot_odds: Math.round(potOdds * 100), // Convert 0-1 to 0-100
    can_check: canCheck,
    position: player.position || 'middle',
    archetype: player.archetype || 'TAG',
  };

  const result = await executePokerDecision(gameState, playerState);

  // Convert API result back to PokerAction
  const action: PokerAction = {
    playerId: player.id,
    actionType: result.action.toLowerCase(),
    amount: 0,
  };

  if (result.action === 'RAISE') {
    action.amount = result.raise_amount;
    // Validate raise amount
    const minRaise = game.currentBet + game.minRaise;
    if (action.amount < minRaise) {
      action.amount = minRaise;
    }
    const maxRaise = player.chips + player.currentBet;
    if (action.amount > maxRaise) {
      if (player.chips > toCall) {
        action.actionType = 'allin';
      } else {
        action.actionType = 'call';
      }
    }
  }

  return action;
}

// NOTE: The old makeAIDecision fallback function has been removed.
// All AI decisions now go through the Go DTRules engine via makeAIDecisionAsync.

/**
 * Determine which decision table column was matched based on archetype and action
 */
function determineMatchedColumn(archetype: string, actionType: string, handStrength: number, canCheck: boolean, position: string): number {
  const action = actionType.toLowerCase();

  switch (archetype) {
    case ARCHETYPE_TAG:
      if (action === 'raise' || action === 'bet' || action === 'allin') return 1;
      if (action === 'call') return 2;
      if (action === 'check' && handStrength >= 0.5) return 3;
      if (action === 'check') return 4;
      return 5; // fold

    case ARCHETYPE_LAG:
      if ((action === 'raise' || action === 'bet' || action === 'allin') && handStrength >= 0.5) return 1;
      if ((action === 'raise' || action === 'bet') && (position === 'late' || position === 'button')) return 2;
      if (action === 'check' && (position === 'late' || position === 'button')) return 3;
      if (action === 'call') return 4;
      return 5; // fold

    case ARCHETYPE_ROCK:
      if (action === 'raise' || action === 'bet' || action === 'allin') return 1;
      if (action === 'call') return 2;
      if (action === 'check') return 3;
      return 4; // fold

    case ARCHETYPE_CALLING:
      if (action === 'call' && handStrength >= 0.2) return 1;
      if (action === 'call') return 2;
      if (action === 'check') return 3;
      return 4; // fold

    default:
      return 1;
  }
}

/**
 * AI action result for callback
 */
export interface AIActionResult {
  player: PokerPlayer;
  action: PokerAction;
  actionMessage: string;
  handStrength: number;
  decisionContext?: {
    canCheck: boolean;
    toCall: number;
    potOdds: number;
    position: string;
    matchedColumnId: number;
  };
}

/**
 * Process all AI turns with async delays for natural feel
 * Calls onAction callback after each AI action
 * Stops when it's the human's turn, hand is complete, or there's a pending reveal
 */
export function processAllAITurns(
  game: PokerGame,
  onAction?: (result: AIActionResult) => void,
  delayMs: { min: number; max: number } = { min: 500, max: 1500 }
): Promise<void> {
  // Queue AI processing to prevent concurrent execution
  aiProcessingQueue = aiProcessingQueue.then(() =>
    processAllAITurnsImpl(game, onAction, delayMs)
  );
  return aiProcessingQueue;
}

async function processAllAITurnsImpl(
  game: PokerGame,
  onAction?: (result: AIActionResult) => void,
  delayMs: { min: number; max: number } = { min: 500, max: 1500 }
): Promise<void> {
  const maxIterations = game.players.length * 10; // Safety limit

  for (let i = 0; i < maxIterations && !game.handComplete; i++) {
    // Check for pending reveal (street transition)
    if (game.pendingReveal) {
      break;
    }

    if (game.currentPlayerIndex < 0 || game.currentPlayerIndex >= game.players.length) {
      break;
    }

    const currentPlayer = game.players[game.currentPlayerIndex];

    // Stop if human's turn
    if (currentPlayer.isHuman) {
      break;
    }

    // Skip folded/all-in
    if (currentPlayer.folded || currentPlayer.allIn) {
      advanceToNextPlayer(game);
      continue;
    }

    // Add delay for natural feel
    const delay = delayMs.min + secureRandom(delayMs.max - delayMs.min);
    await new Promise(resolve => setTimeout(resolve, delay));

    // Calculate hand strength
    let handStrength: number;
    if (game.communityCards.length > 0) {
      const allCards = [...currentPlayer.holeCards, ...game.communityCards];
      const [, result] = selectBestFiveCards(allCards);
      handStrength = calculateHandStrength(result.rank);
    } else {
      handStrength = calculatePreflopStrength(currentPlayer.holeCards);
    }
    currentPlayer.handStrength = handStrength;
    currentPlayer.position = getPlayerPosition(
      game.currentPlayerIndex,
      game.dealerPosition,
      game.players.length
    );

    // Get AI decision from Go engine
    let action: PokerAction;
    try {
      action = await makeAIDecisionAsync(currentPlayer, game, handStrength);
    } catch (error) {
      // On API error, use safe default action (check if possible, fold otherwise)
      console.error('AI decision failed:', error);
      const toCall = game.currentBet - currentPlayer.currentBet;
      action = {
        playerId: currentPlayer.id,
        actionType: toCall === 0 ? 'check' : 'fold',
        amount: 0,
      };
    }

    // Process action
    let result = processPokerAction(game, action);
    if (!result.valid) {
      // Fallback to check/fold
      if (game.currentBet <= currentPlayer.currentBet) {
        action.actionType = 'check';
      } else {
        action.actionType = 'fold';
      }
      action.amount = 0;
      result = processPokerAction(game, action);
    }

    // Build decision context for explainer
    const toCall = game.currentBet - currentPlayer.currentBet;
    const canCheck = toCall === 0;
    let potOdds = 0.0;
    if (game.pot + toCall > 0) {
      potOdds = toCall / (game.pot + toCall);
    }

    // Determine matched column based on action and archetype
    let matchedColumnId = determineMatchedColumn(currentPlayer.archetype || '', action.actionType, handStrength, canCheck, currentPlayer.position || '');

    // Call callback with action result
    if (onAction && result.message) {
      onAction({
        player: currentPlayer,
        action,
        actionMessage: result.message,
        handStrength,
        decisionContext: {
          canCheck,
          toCall,
          potOdds,
          position: currentPlayer.position || '',
          matchedColumnId,
        },
      });
    }

    // Check for pending reveal after action
    if (game.pendingReveal) {
      break;
    }
  }
}

/**
 * Process initial AI turns (without visual delays)
 * Used for initial dealing where AI might act before human
 * Now async since all AI decisions go through the Go API
 */
export async function processInitialAITurns(game: PokerGame): Promise<void> {
  const maxIterations = game.players.length * 10;

  for (let i = 0; i < maxIterations && !game.handComplete; i++) {
    if (game.pendingReveal) break;

    if (game.currentPlayerIndex < 0 || game.currentPlayerIndex >= game.players.length) {
      break;
    }

    const currentPlayer = game.players[game.currentPlayerIndex];

    if (currentPlayer.isHuman) break;

    if (currentPlayer.folded || currentPlayer.allIn) {
      advanceToNextPlayer(game);
      continue;
    }

    // Calculate hand strength
    let handStrength: number;
    if (game.communityCards.length > 0) {
      const allCards = [...currentPlayer.holeCards, ...game.communityCards];
      const [, result] = selectBestFiveCards(allCards);
      handStrength = calculateHandStrength(result.rank);
    } else {
      handStrength = calculatePreflopStrength(currentPlayer.holeCards);
    }
    currentPlayer.handStrength = handStrength;
    currentPlayer.position = getPlayerPosition(
      game.currentPlayerIndex,
      game.dealerPosition,
      game.players.length
    );

    // Get AI decision from Go engine
    let action: PokerAction;
    try {
      action = await makeAIDecisionAsync(currentPlayer, game, handStrength);
    } catch (error) {
      // On API error, use safe default action (check if possible, fold otherwise)
      console.error('Initial AI decision failed:', error);
      const toCall = game.currentBet - currentPlayer.currentBet;
      action = {
        playerId: currentPlayer.id,
        actionType: toCall === 0 ? 'check' : 'fold',
        amount: 0,
      };
    }

    // Process action
    let result = processPokerAction(game, action);
    if (!result.valid) {
      if (game.currentBet <= currentPlayer.currentBet) {
        action.actionType = 'check';
      } else {
        action.actionType = 'fold';
      }
      action.amount = 0;
      processPokerAction(game, action);
    }

    if (game.pendingReveal) break;
  }
}

// Keep old name for backwards compatibility but mark deprecated
/** @deprecated Use processInitialAITurns instead - this is now async */
export const processAITurnsSync = processInitialAITurns;

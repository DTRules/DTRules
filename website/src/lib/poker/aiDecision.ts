// AI Decision logic - ported from Go poker.go lines 1247-1484

import type { PokerGame, PokerPlayer, PokerAction } from './types';
import { ARCHETYPE_TAG, ARCHETYPE_LAG, ARCHETYPE_ROCK, ARCHETYPE_CALLING } from './constants';
import { secureRandom } from './deck';
import { selectBestFiveCards } from './handEval';
import { processPokerAction, advanceToNextPlayer, checkAndAdvanceRound } from './gameLogic';

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
 * Make an AI decision based on archetype and hand strength
 */
export function makeAIDecision(player: PokerPlayer, game: PokerGame, handStrength: number): PokerAction {
  const toCall = game.currentBet - player.currentBet;
  const canCheck = toCall === 0;
  let potOdds = 0.0;
  if (game.pot + toCall > 0) {
    potOdds = toCall / (game.pot + toCall);
  }

  const action: PokerAction = {
    playerId: player.id,
    actionType: '',
    amount: 0,
  };

  // Add some randomness
  const randomFactor = (secureRandom(20) - 10) / 100.0; // -0.1 to +0.1
  const adjustedStrength = handStrength + randomFactor;

  switch (player.archetype) {
    case ARCHETYPE_TAG:
      // Tight-Aggressive: ~18% VPIP, aggressive when playing
      if (adjustedStrength >= 0.7) {
        action.actionType = 'raise';
        action.amount = game.currentBet + game.minRaise + secureRandom(game.minRaise + 1);
      } else if (adjustedStrength >= 0.5 && (canCheck || potOdds < adjustedStrength * 0.8)) {
        if (canCheck) {
          action.actionType = 'check';
        } else {
          action.actionType = 'call';
        }
      } else if (adjustedStrength >= 0.35 && canCheck) {
        action.actionType = 'check';
      } else if (canCheck) {
        action.actionType = 'check';
      } else {
        action.actionType = 'fold';
      }
      break;

    case ARCHETYPE_LAG:
      // Loose-Aggressive: ~35% VPIP, aggressive
      if (adjustedStrength >= 0.5) {
        action.actionType = 'raise';
        action.amount = game.currentBet + game.minRaise * 2 + secureRandom(game.minRaise);
      } else if (player.position === 'late' || player.position === 'button') {
        if (adjustedStrength >= 0.25) {
          if (canCheck) {
            // Sometimes bet/raise with position
            if (secureRandom(100) < 40) {
              action.actionType = 'raise';
              action.amount = game.currentBet + game.minRaise;
            } else {
              action.actionType = 'check';
            }
          } else if (potOdds < 0.4) {
            action.actionType = 'call';
          } else {
            action.actionType = 'fold';
          }
        } else if (canCheck) {
          action.actionType = 'check';
        } else {
          action.actionType = 'fold';
        }
      } else if (adjustedStrength >= 0.35) {
        if (canCheck) {
          action.actionType = 'check';
        } else {
          action.actionType = 'call';
        }
      } else if (canCheck) {
        action.actionType = 'check';
      } else {
        action.actionType = 'fold';
      }
      break;

    case ARCHETYPE_ROCK:
      // Tight-Passive: ~12% VPIP, rarely raises
      if (adjustedStrength >= 0.85) {
        action.actionType = 'raise';
        action.amount = game.currentBet + game.minRaise;
      } else if (adjustedStrength >= 0.6) {
        if (canCheck) {
          action.actionType = 'check';
        } else {
          action.actionType = 'call';
        }
      } else if (canCheck) {
        action.actionType = 'check';
      } else {
        action.actionType = 'fold';
      }
      break;

    case ARCHETYPE_CALLING:
      // Calling Station: ~40% VPIP, calls too much
      if (adjustedStrength >= 0.2 || (toCall <= game.bigBlind * 2 && adjustedStrength >= 0.1)) {
        if (canCheck) {
          action.actionType = 'check';
        } else {
          action.actionType = 'call';
        }
      } else if (canCheck) {
        action.actionType = 'check';
      } else if (toCall <= game.bigBlind) {
        action.actionType = 'call'; // Calls min bets with anything
      } else {
        action.actionType = 'fold';
      }
      break;

    default:
      // Default to tight play
      if (adjustedStrength >= 0.6) {
        if (canCheck) {
          action.actionType = 'check';
        } else {
          action.actionType = 'call';
        }
      } else if (canCheck) {
        action.actionType = 'check';
      } else {
        action.actionType = 'fold';
      }
  }

  // Validate raise amount
  if (action.actionType === 'raise') {
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

/**
 * AI action result for callback
 */
export interface AIActionResult {
  player: PokerPlayer;
  action: PokerAction;
  actionMessage: string;
}

/**
 * Process all AI turns with async delays for natural feel
 * Calls onAction callback after each AI action
 * Stops when it's the human's turn, hand is complete, or there's a pending reveal
 */
export async function processAllAITurns(
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

    // Get AI decision
    let action = makeAIDecision(currentPlayer, game, handStrength);

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

    // Call callback with action result
    if (onAction && result.message) {
      onAction({
        player: currentPlayer,
        action,
        actionMessage: result.message,
      });
    }

    // Check for pending reveal after action
    if (game.pendingReveal) {
      break;
    }
  }
}

/**
 * Process AI turns synchronously (without delays)
 * Used for initial dealing where AI might act before human
 */
export function processAITurnsSync(game: PokerGame): void {
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

    // Get AI decision
    let action = makeAIDecision(currentPlayer, game, handStrength);

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

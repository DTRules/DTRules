// Deck operations - ported from Go poker.go lines 218-247

import type { PokerCard } from './types';
import { SUITS, CARD_RANK_NAMES } from './constants';

/**
 * Generate a cryptographically secure random number in range [0, max)
 * Falls back to Math.random if crypto is unavailable
 */
export function secureRandom(max: number): number {
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    const array = new Uint32Array(1);
    crypto.getRandomValues(array);
    return array[0] % max;
  }
  return Math.floor(Math.random() * max);
}

/**
 * Create a standard 52-card deck
 */
export function createDeck(): PokerCard[] {
  const deck: PokerCard[] = [];

  for (const suit of SUITS) {
    for (let rank = 2; rank <= 14; rank++) {
      deck.push({
        suit,
        rank,
        rankName: CARD_RANK_NAMES[rank],
      });
    }
  }

  return deck;
}

/**
 * Fisher-Yates shuffle - shuffles deck in place
 */
export function shuffleDeck(deck: PokerCard[]): void {
  for (let i = deck.length - 1; i > 0; i--) {
    const j = secureRandom(i + 1);
    [deck[i], deck[j]] = [deck[j], deck[i]];
  }
}

/**
 * Create and shuffle a new deck
 */
export function createShuffledDeck(): PokerCard[] {
  const deck = createDeck();
  shuffleDeck(deck);
  return deck;
}

// Hand evaluation - ported from Go poker.go lines 254-448

import type { PokerCard, EvaluateResponse } from './types';
import { HAND_RANK_NAMES, CARD_RANK_FULL_NAMES, CARD_RANK_SINGULAR } from './constants';

/**
 * Evaluate a 5-card poker hand
 * Returns hand rank, name, and value for comparison
 */
export function evaluatePokerHand(cards: PokerCard[]): EvaluateResponse {
  if (cards.length !== 5) {
    return {
      success: false,
      rank: 0,
      rankName: '',
      handName: '',
      isFlush: false,
      isStraight: false,
      highCard: 0,
      handValue: 0,
      kickers: [],
      error: 'Must have exactly 5 cards',
    };
  }

  const resp: EvaluateResponse = {
    success: true,
    rank: 0,
    rankName: '',
    handName: '',
    isFlush: false,
    isStraight: false,
    highCard: 0,
    handValue: 0,
    kickers: [],
    bestFiveCards: cards,
  };

  // Count suits for flush
  const suitCounts: Record<string, number> = {};
  for (const card of cards) {
    suitCounts[card.suit] = (suitCounts[card.suit] || 0) + 1;
  }
  for (const count of Object.values(suitCounts)) {
    if (count === 5) {
      resp.isFlush = true;
      break;
    }
  }

  // Count ranks
  const rankCounts: Record<number, number> = {};
  for (const card of cards) {
    rankCounts[card.rank] = (rankCounts[card.rank] || 0) + 1;
  }

  // Get sorted ranks (ascending)
  const ranks = cards.map(c => c.rank).sort((a, b) => a - b);

  // Check for straight (including wheel A-2-3-4-5)
  let isSequential = true;
  for (let i = 1; i < 5; i++) {
    if (ranks[i] !== ranks[i - 1] + 1) {
      isSequential = false;
      break;
    }
  }

  // Check for wheel (A-2-3-4-5)
  const isWheel = ranks[0] === 2 && ranks[1] === 3 && ranks[2] === 4 && ranks[3] === 5 && ranks[4] === 14;

  if (isSequential && Object.keys(rankCounts).length === 5) {
    resp.isStraight = true;
    resp.straightHigh = ranks[4];
  } else if (isWheel) {
    resp.isStraight = true;
    resp.straightHigh = 5; // 5-high straight
  }

  // Count pairs, trips, quads
  let pairCount = 0;
  let tripleCount = 0;
  let quadCount = 0;
  const pairRanks: number[] = [];
  const tripleRanks: number[] = [];
  const quadRanks: number[] = [];
  const singleRanks: number[] = [];

  for (const [rankStr, count] of Object.entries(rankCounts)) {
    const rank = parseInt(rankStr);
    switch (count) {
      case 4:
        quadCount++;
        quadRanks.push(rank);
        break;
      case 3:
        tripleCount++;
        tripleRanks.push(rank);
        break;
      case 2:
        pairCount++;
        pairRanks.push(rank);
        break;
      case 1:
        singleRanks.push(rank);
        break;
    }
  }

  // Sort all rank arrays descending
  pairRanks.sort((a, b) => b - a);
  tripleRanks.sort((a, b) => b - a);
  quadRanks.sort((a, b) => b - a);
  singleRanks.sort((a, b) => b - a);

  // Determine hand rank and build kicker array for tiebreakers
  let kickers: number[] = [];
  let handName = '';

  if (resp.isFlush && resp.isStraight) {
    if (resp.straightHigh === 14) {
      resp.rank = 10; // Royal Flush
      handName = 'Royal Flush';
    } else {
      resp.rank = 9; // Straight Flush
      handName = `Straight Flush, ${CARD_RANK_SINGULAR[resp.straightHigh!]} high`;
    }
    if (isWheel) {
      kickers = [5, 4, 3, 2, 1]; // A-2-3-4-5
    } else {
      kickers = [resp.straightHigh!];
    }
  } else if (quadCount >= 1) {
    resp.rank = 8; // Four of a Kind
    kickers = [...quadRanks, ...singleRanks];
    handName = `Four of a Kind, ${CARD_RANK_FULL_NAMES[quadRanks[0]]}`;
  } else if (tripleCount >= 1 && pairCount >= 1) {
    resp.rank = 7; // Full House
    kickers = [...tripleRanks, ...pairRanks];
    handName = `Full House, ${CARD_RANK_FULL_NAMES[tripleRanks[0]]} full of ${CARD_RANK_FULL_NAMES[pairRanks[0]]}`;
  } else if (resp.isFlush) {
    resp.rank = 6; // Flush
    kickers = [...ranks].sort((a, b) => b - a);
    handName = `Flush, ${CARD_RANK_SINGULAR[kickers[0]]} high`;
  } else if (resp.isStraight) {
    resp.rank = 5; // Straight
    kickers = [resp.straightHigh!];
    handName = `Straight, ${CARD_RANK_SINGULAR[resp.straightHigh!]} high`;
  } else if (tripleCount >= 1) {
    resp.rank = 4; // Three of a Kind
    kickers = [...tripleRanks, ...singleRanks];
    handName = `Three of a Kind, ${CARD_RANK_FULL_NAMES[tripleRanks[0]]}`;
  } else if (pairCount >= 2) {
    resp.rank = 3; // Two Pair
    kickers = [...pairRanks, ...singleRanks];
    handName = `Two Pair, ${CARD_RANK_FULL_NAMES[pairRanks[0]]} and ${CARD_RANK_FULL_NAMES[pairRanks[1]]}`;
  } else if (pairCount === 1) {
    resp.rank = 2; // One Pair
    kickers = [...pairRanks, ...singleRanks];
    handName = `Pair of ${CARD_RANK_FULL_NAMES[pairRanks[0]]}`;
  } else {
    resp.rank = 1; // High Card
    kickers = [...ranks].sort((a, b) => b - a);
    handName = `${CARD_RANK_SINGULAR[kickers[0]]} high`;
  }

  resp.rankName = HAND_RANK_NAMES[resp.rank];
  resp.handName = handName;
  resp.highCard = isWheel ? 5 : ranks[4];
  resp.kickers = kickers;

  // Calculate hand value for comparison
  // Format: RKKKKK where R=rank (2 digits), K=kicker (2 digits each)
  // Using number since JS can handle up to 2^53 safely
  resp.handValue = resp.rank * 10000000000;
  let multiplier = 100000000;
  for (let i = 0; i < 5 && i < kickers.length; i++) {
    resp.handValue += kickers[i] * multiplier;
    multiplier = Math.floor(multiplier / 100);
  }

  return resp;
}

/**
 * Generate all C(n, k) combinations of cards
 */
function* combinations(cards: PokerCard[], k: number, start = 0, current: PokerCard[] = []): Generator<PokerCard[]> {
  if (current.length === k) {
    yield [...current];
    return;
  }

  for (let i = start; i <= cards.length - (k - current.length); i++) {
    current.push(cards[i]);
    yield* combinations(cards, k, i + 1, current);
    current.pop();
  }
}

/**
 * Select the best 5-card hand from up to 7 cards
 */
export function selectBestFiveCards(cards: PokerCard[]): [PokerCard[], EvaluateResponse] {
  const n = cards.length;

  if (n < 5) {
    return [cards, {
      success: false,
      rank: 0,
      rankName: '',
      handName: '',
      isFlush: false,
      isStraight: false,
      highCard: 0,
      handValue: 0,
      kickers: [],
      error: 'Not enough cards',
    }];
  }

  if (n === 5) {
    return [cards, evaluatePokerHand(cards)];
  }

  let bestHand: PokerCard[] = [];
  let bestResult: EvaluateResponse = {
    success: false,
    rank: 0,
    rankName: '',
    handName: '',
    isFlush: false,
    isStraight: false,
    highCard: 0,
    handValue: -1,
    kickers: [],
  };

  // Generate all C(n,5) combinations
  for (const combo of combinations(cards, 5)) {
    const result = evaluatePokerHand(combo);
    if (result.handValue > bestResult.handValue) {
      bestResult = result;
      bestHand = [...combo];
    }
  }

  bestResult.bestFiveCards = bestHand;
  return [bestHand, bestResult];
}

/**
 * Compare two hands - returns positive if a > b, negative if a < b, 0 if equal
 */
export function compareHands(a: EvaluateResponse, b: EvaluateResponse): number {
  if (a.rank !== b.rank) {
    return a.rank - b.rank;
  }

  // Same rank - compare kickers
  const maxKickers = Math.max(a.kickers.length, b.kickers.length);
  for (let i = 0; i < maxKickers; i++) {
    const kickerA = a.kickers[i] || 0;
    const kickerB = b.kickers[i] || 0;
    if (kickerA !== kickerB) {
      return kickerA - kickerB;
    }
  }

  return 0;
}

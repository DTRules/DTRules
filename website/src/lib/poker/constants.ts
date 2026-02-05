// Poker constants - ported from Go poker.go

export const SUITS = ['hearts', 'diamonds', 'clubs', 'spades'] as const;

export const CARD_RANK_NAMES: Record<number, string> = {
  2: '2', 3: '3', 4: '4', 5: '5', 6: '6', 7: '7', 8: '8', 9: '9', 10: 'T',
  11: 'J', 12: 'Q', 13: 'K', 14: 'A',
};

export const CARD_RANK_FULL_NAMES: Record<number, string> = {
  2: 'Twos', 3: 'Threes', 4: 'Fours', 5: 'Fives', 6: 'Sixes', 7: 'Sevens',
  8: 'Eights', 9: 'Nines', 10: 'Tens', 11: 'Jacks', 12: 'Queens', 13: 'Kings', 14: 'Aces',
};

export const CARD_RANK_SINGULAR: Record<number, string> = {
  2: 'Two', 3: 'Three', 4: 'Four', 5: 'Five', 6: 'Six', 7: 'Seven',
  8: 'Eight', 9: 'Nine', 10: 'Ten', 11: 'Jack', 12: 'Queen', 13: 'King', 14: 'Ace',
};

export const HAND_RANK_NAMES: Record<number, string> = {
  1: 'High Card',
  2: 'One Pair',
  3: 'Two Pair',
  4: 'Three of a Kind',
  5: 'Straight',
  6: 'Flush',
  7: 'Full House',
  8: 'Four of a Kind',
  9: 'Straight Flush',
  10: 'Royal Flush',
};

// AI Archetypes
export const ARCHETYPE_TAG = 'TAG';
export const ARCHETYPE_LAG = 'LAG';
export const ARCHETYPE_ROCK = 'Rock';
export const ARCHETYPE_CALLING = 'Calling';

export const ARCHETYPES = [
  ARCHETYPE_TAG,
  ARCHETYPE_LAG,
  ARCHETYPE_ROCK,
  ARCHETYPE_CALLING,
] as const;

// Player names for AI opponents
export const PLAYER_NAMES = ['You', 'Alice', 'Bob', 'Charlie', 'Diana', 'Eve', 'Frank', 'Grace', 'Henry'];

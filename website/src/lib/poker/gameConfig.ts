/**
 * Centralized game configuration constants
 * These values are used throughout the poker demo
 */

export const GAME_CONFIG = {
  /** Number of players at the table */
  playerCount: 4,

  /** Position of the human player (0-indexed) */
  humanPosition: 0,

  /** Small blind amount in chips */
  smallBlind: 10,

  /** Big blind amount in chips */
  bigBlind: 20,

  /** Starting chips for each player */
  startingChips: 1000,

  /** Maximum number of actions to keep in history */
  maxActionHistory: 100,
} as const;

export type GameConfig = typeof GAME_CONFIG;

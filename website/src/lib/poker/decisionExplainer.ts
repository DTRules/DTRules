// Decision Explainer - Generates human-readable explanations for AI decisions

import type { PokerGame, PokerPlayer, DecisionContext, DecisionExplanation, ConditionEvaluation, PokerCard } from './types';
import { getDecisionTable, type DecisionTableDefinition, type DecisionColumn } from './decisionTableData';
import { CARD_RANK_NAMES } from './constants';

const SUIT_SYMBOLS: Record<string, string> = {
  hearts: '\u2665',
  diamonds: '\u2666',
  clubs: '\u2663',
  spades: '\u2660'
};

const POSITION_LABELS: Record<string, string> = {
  'button': 'Button (Best)',
  'small_blind': 'Small Blind',
  'big_blind': 'Big Blind',
  'early': 'Early Position',
  'middle': 'Middle Position',
  'late': 'Late Position (Good)',
};

const ARCHETYPE_STRATEGIES: Record<string, string> = {
  'TAG': 'TAG Strategy: Plays few hands but raises aggressively when holding strength. Folds weak hands to pressure.',
  'LAG': 'LAG Strategy: Plays many hands aggressively. Uses position to steal pots and apply pressure with bluffs.',
  'Rock': 'Rock Strategy: Only plays premium hands. Very predictable - when they bet, they have it.',
  'Calling': 'Calling Station Strategy: Calls down with any piece of the board. Rarely folds, rarely raises.',
};

function formatCard(card: PokerCard): string {
  const rank = CARD_RANK_NAMES[card.rank] || card.rankName;
  const suit = SUIT_SYMBOLS[card.suit] || card.suit;
  return `${rank}${suit}`;
}

function formatCards(cards: PokerCard[]): string {
  return cards.map(formatCard).join(' ');
}

function getHandNickname(cards: PokerCard[]): string | null {
  if (cards.length !== 2) return null;

  const ranks = cards.map(c => c.rank).sort((a, b) => b - a);
  const suited = cards[0].suit === cards[1].suit;

  // Common hand nicknames
  if (ranks[0] === 14 && ranks[1] === 14) return 'Pocket Rockets';
  if (ranks[0] === 13 && ranks[1] === 13) return 'Cowboys';
  if (ranks[0] === 12 && ranks[1] === 12) return 'Ladies';
  if (ranks[0] === 11 && ranks[1] === 11) return 'Hooks';
  if (ranks[0] === 10 && ranks[1] === 10) return 'Dimes';
  if (ranks[0] === 14 && ranks[1] === 13) return suited ? 'Big Slick (suited)' : 'Big Slick';
  if (ranks[0] === 14 && ranks[1] === 12) return 'Big Chick';
  if (ranks[0] === 13 && ranks[1] === 12) return 'Marriage';

  return null;
}

function evaluateConditions(
  table: DecisionTableDefinition,
  handStrength: number,
  position: string,
  canCheck: boolean,
  toCall: number,
  pot: number,
  bigBlind: number
): { evaluations: ConditionEvaluation[], matchedColumn: DecisionColumn | null } {
  const evaluations: ConditionEvaluation[] = [];

  // Calculate pot odds
  const potOdds = pot + toCall > 0 ? toCall / (pot + toCall) : 0;
  const potOddsFavorable = potOdds < handStrength * 0.8;

  // Evaluate each condition in the table
  for (const condition of table.conditions) {
    let passed = false;
    let actualValue: string | number | boolean = '';

    switch (condition.id) {
      case 'str70':
        actualValue = Math.round(handStrength * 100);
        passed = handStrength >= 0.7;
        break;
      case 'str50':
        actualValue = Math.round(handStrength * 100);
        passed = handStrength >= 0.5;
        break;
      case 'str35':
        actualValue = Math.round(handStrength * 100);
        passed = handStrength >= 0.35;
        break;
      case 'str25':
        actualValue = Math.round(handStrength * 100);
        passed = handStrength >= 0.25;
        break;
      case 'str20':
        actualValue = Math.round(handStrength * 100);
        passed = handStrength >= 0.2;
        break;
      case 'str85':
        actualValue = Math.round(handStrength * 100);
        passed = handStrength >= 0.85;
        break;
      case 'str60':
        actualValue = Math.round(handStrength * 100);
        passed = handStrength >= 0.6;
        break;
      case 'potOdds':
        actualValue = `${Math.round(potOdds * 100)}%`;
        passed = potOddsFavorable;
        break;
      case 'canCheck':
        actualValue = canCheck;
        passed = canCheck;
        break;
      case 'position':
        actualValue = position;
        passed = position === 'late' || position === 'button';
        break;
      case 'smallBet':
        actualValue = toCall;
        passed = toCall <= bigBlind * 2;
        break;
    }

    evaluations.push({
      conditionId: condition.id,
      label: condition.label,
      actualValue,
      threshold: condition.threshold,
      passed,
    });
  }

  // Find matching column
  let matchedColumn: DecisionColumn | null = null;
  for (const column of table.columns) {
    let matches = true;
    for (const [condId, expected] of Object.entries(column.conditions)) {
      if (expected === '-') continue; // Don't care

      const evaluation = evaluations.find(e => e.conditionId === condId);
      if (!evaluation) continue;

      const actual = evaluation.passed ? 'Y' : 'N';
      if (actual !== expected) {
        matches = false;
        break;
      }
    }

    if (matches) {
      matchedColumn = column;
      break;
    }
  }

  return { evaluations, matchedColumn };
}

export function generateDecisionContext(
  player: PokerPlayer,
  game: PokerGame,
  handStrength: number,
  action: string,
  actionAmount?: number
): DecisionContext {
  const toCall = game.currentBet - player.currentBet;
  const canCheck = toCall === 0;
  const potOdds = game.pot + toCall > 0 ? toCall / (game.pot + toCall) : 0;
  const position = player.position || 'unknown';

  return {
    playerId: player.id,
    playerName: player.name,
    archetype: player.archetype || 'default',
    handStrength,
    handStrengthPercent: `${Math.round(handStrength * 100)}%`,
    position,
    positionLabel: POSITION_LABELS[position] || position,
    canCheck,
    toCall,
    potOdds,
    potOddsPercent: `${Math.round(potOdds * 100)}%`,
    pot: game.pot,
    holeCards: player.holeCards,
    holeCardsDisplay: formatCards(player.holeCards),
    communityCards: game.communityCards,
    matchedColumnId: 0, // Will be set by evaluateConditions
    conditionEvaluations: [],
    action,
    actionAmount,
    round: game.round,
  };
}

export function generateDecisionExplanation(
  player: PokerPlayer,
  game: PokerGame,
  handStrength: number,
  action: string,
  actionAmount?: number
): DecisionExplanation | null {
  const archetype = player.archetype;
  if (!archetype) return null;

  const table = getDecisionTable(archetype);
  if (!table) return null;

  const toCall = game.currentBet - player.currentBet;
  const canCheck = toCall === 0;
  const position = player.position || 'unknown';

  const { evaluations, matchedColumn } = evaluateConditions(
    table,
    handStrength,
    position,
    canCheck,
    toCall,
    game.pot,
    game.bigBlind
  );

  const context = generateDecisionContext(player, game, handStrength, action, actionAmount);
  context.conditionEvaluations = evaluations;
  context.matchedColumnId = matchedColumn?.id || 0;

  // Build decision path explanation
  const decisionPath: string[] = [];

  // Add hand strength info
  const strengthPercent = Math.round(handStrength * 100);
  decisionPath.push(`Hand Strength: ${strengthPercent}%`);

  // Add relevant condition evaluations
  for (const evaluation of evaluations) {
    const status = evaluation.passed ? '\u2713' : '\u2717';
    let valueDisplay = '';

    if (typeof evaluation.actualValue === 'number') {
      valueDisplay = `${evaluation.actualValue}%`;
    } else if (typeof evaluation.actualValue === 'boolean') {
      valueDisplay = evaluation.actualValue ? 'Yes' : 'No';
    } else {
      valueDisplay = String(evaluation.actualValue);
    }

    decisionPath.push(`${status} ${evaluation.label}: ${valueDisplay}`);
  }

  if (matchedColumn) {
    decisionPath.push(`\u2192 Column ${matchedColumn.id} matched`);
  }

  // Build summary
  const handNickname = getHandNickname(player.holeCards);
  const handInfo = handNickname
    ? `${formatCards(player.holeCards)} (${handNickname})`
    : formatCards(player.holeCards);

  let summary = `${player.name} (${archetype}) chose ${action.toUpperCase()}`;
  if (actionAmount && action !== 'fold' && action !== 'check') {
    summary += ` $${actionAmount}`;
  }

  const strategyNote = ARCHETYPE_STRATEGIES[archetype] || '';

  return {
    summary,
    decisionPath,
    context,
    strategyNote,
  };
}

// Export position labels for use in components
export { POSITION_LABELS, ARCHETYPE_STRATEGIES, formatCards, formatCard };

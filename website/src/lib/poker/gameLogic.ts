// Game logic - ported from Go poker.go lines 533-1186

import type { PokerGame, PokerPlayer, PokerCard, PokerAction, ActionResponse, DealRequest, ActionLog, ShowdownResult } from './types';
import { ARCHETYPES, PLAYER_NAMES } from './constants';
import { createShuffledDeck, secureRandom } from './deck';
import { selectBestFiveCards } from './handEval';

let gameCounter = 0;

/**
 * Deal a new poker hand
 */
export function dealNewPokerHand(req: DealRequest): PokerGame {
  gameCounter++;
  const gameId = `game_${Date.now()}_${gameCounter}`;

  // Validate and set defaults
  let playerCount = req.playerCount;
  if (playerCount < 2) playerCount = 2;
  if (playerCount > 9) playerCount = 9;

  let smallBlind = req.smallBlind;
  if (smallBlind <= 0) smallBlind = 10;

  let bigBlind = req.bigBlind;
  if (bigBlind <= 0) bigBlind = smallBlind * 2;

  let startingChips = req.startingChips;
  if (startingChips <= 0) startingChips = 1000;

  let humanPosition = req.humanPosition;
  if (humanPosition < 0 || humanPosition >= playerCount) humanPosition = 0;

  let dealerPosition = req.dealerPosition;
  if (dealerPosition < 0 || dealerPosition >= playerCount) {
    dealerPosition = secureRandom(playerCount);
  }

  // Create and shuffle deck
  const deck = createShuffledDeck();

  // Create players
  const players: PokerPlayer[] = [];
  for (let i = 0; i < playerCount; i++) {
    let name = PLAYER_NAMES[i];
    if (i === humanPosition) {
      name = 'You';
    } else if (i <= humanPosition) {
      name = PLAYER_NAMES[i + 1];
    }

    const player: PokerPlayer = {
      id: `player${i}`,
      name,
      chips: startingChips,
      holeCards: [],
      folded: false,
      allIn: false,
      currentBet: 0,
      totalBet: 0,
      hasActed: false,
      isDealer: i === dealerPosition,
      isSmallBlind: false,
      isBigBlind: false,
      isHuman: i === humanPosition,
    };

    // Assign archetype to AI
    if (!player.isHuman) {
      player.archetype = ARCHETYPES[secureRandom(ARCHETYPES.length)];
    }

    players.push(player);
  }

  // Deal hole cards (2 per player)
  let deckIdx = 0;
  for (let i = 0; i < playerCount; i++) {
    players[i].holeCards = [deck[deckIdx], deck[deckIdx + 1]];
    deckIdx += 2;
  }

  // Calculate blind positions
  let sbPos: number;
  let bbPos: number;
  if (playerCount === 2) {
    // Heads-up: dealer is SB
    sbPos = dealerPosition;
    bbPos = (dealerPosition + 1) % playerCount;
  } else {
    sbPos = (dealerPosition + 1) % playerCount;
    bbPos = (dealerPosition + 2) % playerCount;
  }

  // Post blinds - handle players with insufficient chips
  players[sbPos].isSmallBlind = true;
  const actualSmallBlind = Math.min(smallBlind, players[sbPos].chips);
  players[sbPos].chips -= actualSmallBlind;
  players[sbPos].currentBet = actualSmallBlind;
  players[sbPos].totalBet = actualSmallBlind;
  if (players[sbPos].chips === 0) {
    players[sbPos].allIn = true;
  }

  players[bbPos].isBigBlind = true;
  const actualBigBlind = Math.min(bigBlind, players[bbPos].chips);
  players[bbPos].chips -= actualBigBlind;
  players[bbPos].currentBet = actualBigBlind;
  players[bbPos].totalBet = actualBigBlind;
  if (players[bbPos].chips === 0) {
    players[bbPos].allIn = true;
  }

  const pot = actualSmallBlind + actualBigBlind;

  // First to act preflop is left of BB (or SB in heads-up)
  let firstToAct: number;
  if (playerCount === 2) {
    firstToAct = sbPos; // Dealer/SB acts first preflop in heads-up
  } else {
    firstToAct = (bbPos + 1) % playerCount;
  }

  const game: PokerGame = {
    id: gameId,
    players,
    communityCards: [],
    deck: deck.slice(deckIdx), // Remaining deck
    pot,
    currentBet: bigBlind,
    minRaise: bigBlind,
    round: 'preflop',
    roundNumber: 0,
    smallBlind,
    bigBlind,
    dealerPosition,
    currentPlayerIndex: firstToAct,
    handComplete: false,
    roundComplete: false,
    activePlayerCount: playerCount,
    handNumber: 1,
    actionHistory: [],
  };

  // Log blinds
  game.actionHistory.push({
    playerId: players[sbPos].id,
    playerName: players[sbPos].name,
    action: 'posts small blind',
    amount: smallBlind,
    round: 'preflop',
    timestamp: Date.now(),
  });
  game.actionHistory.push({
    playerId: players[bbPos].id,
    playerName: players[bbPos].name,
    action: 'posts big blind',
    amount: bigBlind,
    round: 'preflop',
    timestamp: Date.now(),
  });

  return game;
}

/**
 * Reset hasActed for all players except the one who made an action
 */
function resetPlayersActed(game: PokerGame, exceptIdx: number): void {
  for (let i = 0; i < game.players.length; i++) {
    if (i !== exceptIdx && !game.players[i].folded && !game.players[i].allIn) {
      game.players[i].hasActed = false;
    }
  }
}

/**
 * Process a player action
 */
export function processPokerAction(game: PokerGame, action: PokerAction): ActionResponse {
  // Find player
  const playerIdx = game.players.findIndex(p => p.id === action.playerId);
  if (playerIdx === -1) {
    return { success: false, valid: false, error: 'Player not found' };
  }

  const player = game.players[playerIdx];

  // Validate turn
  if (playerIdx !== game.currentPlayerIndex) {
    return { success: false, valid: false, error: 'Not your turn' };
  }
  if (player.folded) {
    return { success: false, valid: false, error: 'You have folded' };
  }
  if (player.allIn) {
    return { success: false, valid: false, error: 'You are all-in' };
  }
  if (game.handComplete) {
    return { success: false, valid: false, error: 'Hand is complete' };
  }

  const toCall = game.currentBet - player.currentBet;
  let actionMsg = '';

  switch (action.actionType) {
    case 'fold':
      player.folded = true;
      actionMsg = `${player.name} folds`;
      break;

    case 'check':
      if (toCall > 0) {
        return { success: false, valid: false, error: `Cannot check, must call $${toCall} or fold` };
      }
      actionMsg = `${player.name} checks`;
      break;

    case 'call':
      if (toCall === 0) {
        // Treat as check
        actionMsg = `${player.name} checks`;
      } else {
        let actualCall = toCall;
        if (actualCall >= player.chips) {
          actualCall = player.chips;
          player.allIn = true;
          actionMsg = `${player.name} calls $${actualCall} (all-in)`;
        } else {
          actionMsg = `${player.name} calls $${actualCall}`;
        }
        player.chips -= actualCall;
        player.currentBet += actualCall;
        player.totalBet += actualCall;
        game.pot += actualCall;
      }
      break;

    case 'bet':
      if (game.currentBet > 0) {
        return { success: false, valid: false, error: "Cannot bet when there's already a bet - use raise" };
      }
      let betAmount = action.amount;
      if (betAmount < game.bigBlind) {
        betAmount = game.bigBlind;
      }
      if (betAmount > player.chips) {
        betAmount = player.chips;
        player.allIn = true;
      }
      player.chips -= betAmount;
      player.currentBet = betAmount;
      player.totalBet += betAmount;
      game.pot += betAmount;
      game.currentBet = betAmount;
      game.minRaise = betAmount;
      resetPlayersActed(game, playerIdx);
      if (player.allIn) {
        actionMsg = `${player.name} bets $${betAmount} (all-in)`;
      } else {
        actionMsg = `${player.name} bets $${betAmount}`;
      }
      break;

    case 'raise': {
      const minRaiseAmount = game.currentBet + game.minRaise;
      if (game.currentBet === 0) {
        // This is actually a bet
        let betAmt = action.amount;
        if (betAmt < game.bigBlind) {
          betAmt = game.bigBlind;
        }
        if (betAmt > player.chips) {
          betAmt = player.chips;
          player.allIn = true;
        }
        player.chips -= betAmt;
        player.currentBet = betAmt;
        player.totalBet += betAmt;
        game.pot += betAmt;
        game.currentBet = betAmt;
        game.minRaise = betAmt;
        resetPlayersActed(game, playerIdx);
        if (player.allIn) {
          actionMsg = `${player.name} bets $${betAmt} (all-in)`;
        } else {
          actionMsg = `${player.name} bets $${betAmt}`;
        }
      } else {
        // Actual raise
        let raiseToAmount = action.amount;
        if (raiseToAmount < minRaiseAmount && raiseToAmount < player.chips + player.currentBet) {
          return { success: false, valid: false, error: `Raise must be at least $${minRaiseAmount}` };
        }
        let raiseAmount = raiseToAmount - player.currentBet;
        if (raiseAmount > player.chips) {
          raiseAmount = player.chips;
          raiseToAmount = player.currentBet + raiseAmount;
          player.allIn = true;
        }
        player.chips -= raiseAmount;
        game.pot += raiseAmount;

        // MinRaise for next player is the raise size
        const raiseSize = raiseToAmount - game.currentBet;
        if (raiseSize >= game.minRaise) {
          game.minRaise = raiseSize;
        }

        game.currentBet = raiseToAmount;
        player.currentBet = raiseToAmount;
        player.totalBet += raiseAmount;
        resetPlayersActed(game, playerIdx);

        if (player.allIn) {
          actionMsg = `${player.name} raises to $${raiseToAmount} (all-in)`;
        } else {
          actionMsg = `${player.name} raises to $${raiseToAmount}`;
        }
      }
      break;
    }

    case 'allin': {
      const allInAmount = player.chips;
      const totalBet = player.currentBet + allInAmount;

      if (totalBet > game.currentBet) {
        // This is a raise
        const raiseSize = totalBet - game.currentBet;
        if (raiseSize >= game.minRaise) {
          game.minRaise = raiseSize;
        }
        game.currentBet = totalBet;
        resetPlayersActed(game, playerIdx);
      }

      player.totalBet += allInAmount;
      player.currentBet = totalBet;
      game.pot += allInAmount;
      player.chips = 0;
      player.allIn = true;
      actionMsg = `${player.name} goes all-in for $${allInAmount}`;
      break;
    }

    default:
      return { success: false, valid: false, error: `Unknown action: ${action.actionType}` };
  }

  player.hasActed = true;
  game.lastAction = actionMsg;
  game.lastActionPlayer = player.id;

  // Log action (limit history to prevent memory leak)
  game.actionHistory.push({
    playerId: player.id,
    playerName: player.name,
    action: action.actionType,
    amount: action.amount,
    round: game.round,
    timestamp: Date.now(),
  });

  // Keep only the last 100 actions to prevent memory leak
  const MAX_HISTORY_LENGTH = 100;
  if (game.actionHistory.length > MAX_HISTORY_LENGTH) {
    game.actionHistory = game.actionHistory.slice(-MAX_HISTORY_LENGTH);
  }

  // Check round completion
  checkAndAdvanceRound(game);

  return { success: true, valid: true, message: actionMsg, game };
}

/**
 * Check if the round is complete and advance if necessary
 */
export function checkAndAdvanceRound(game: PokerGame): void {
  // Count active players
  let activePlayers = 0;
  let playersCanAct = 0;
  let playersNeedAction = 0;

  for (const p of game.players) {
    if (!p.folded) {
      activePlayers++;
      if (!p.allIn) {
        playersCanAct++;
        if (!p.hasActed || p.currentBet < game.currentBet) {
          playersNeedAction++;
        }
      }
    }
  }

  game.activePlayerCount = activePlayers;

  // Only one player left - they win
  if (activePlayers <= 1) {
    game.handComplete = true;
    game.round = 'showdown';
    awardPot(game);
    return;
  }

  // Check for BB option in preflop
  if (game.round === 'preflop' && playersNeedAction === 0 && playersCanAct > 0) {
    // Find BB player
    for (let i = 0; i < game.players.length; i++) {
      const p = game.players[i];
      if (p.isBigBlind && !p.folded && !p.allIn && p.hasActed && p.currentBet === game.currentBet) {
        // BB has only called - give them option if no one raised
        if (game.currentBet === game.bigBlind) {
          // Check if BB was the last to act with exactly their blind
          let allCalled = true;
          for (let j = 0; j < game.players.length; j++) {
            if (j !== i && !game.players[j].folded && !game.players[j].allIn && game.players[j].currentBet !== game.currentBet) {
              allCalled = false;
              break;
            }
          }
          if (allCalled && game.lastActionPlayer !== p.id) {
            // BB gets option
            game.players[i].hasActed = false;
            game.currentPlayerIndex = i;
            return;
          }
        }
      }
    }
  }

  // Round complete?
  if (playersNeedAction === 0) {
    // All remaining players all-in or have acted
    if (playersCanAct <= 1) {
      // Run out the board
      runOutBoard(game);
    } else {
      advanceToNextStreet(game);
    }
  } else {
    advanceToNextPlayer(game);
  }
}

/**
 * Advance to the next player who can act
 */
export function advanceToNextPlayer(game: PokerGame): void {
  const numPlayers = game.players.length;
  for (let i = 1; i <= numPlayers; i++) {
    const nextIdx = (game.currentPlayerIndex + i) % numPlayers;
    const p = game.players[nextIdx];
    if (!p.folded && !p.allIn) {
      game.currentPlayerIndex = nextIdx;
      return;
    }
  }
  // No one can act - run out the board
  runOutBoard(game);
}

/**
 * Run out the remaining community cards when all players are all-in
 * Now deals ONE street at a time and returns with pendingReveal set
 * The UI should call continueRunOut() after handling the reveal
 */
function runOutBoard(game: PokerGame): void {
  game.runningOutBoard = true;

  // Deal ONE street (not all at once)
  if (game.round !== 'showdown') {
    dealNextStreet(game);
  }

  // If we've reached showdown, award the pot
  if (game.round === 'showdown') {
    game.runningOutBoard = false;
    awardPot(game);
  }
  // Otherwise, pendingReveal is set and UI will call continueRunOut after reveal
}

/**
 * Continue running out the board after a reveal
 * Call this after clearing pendingReveal when runningOutBoard is true
 */
export function continueRunOut(game: PokerGame): void {
  if (!game.runningOutBoard) return;

  // If we're already at showdown with no pending reveal, award pot
  if (game.round === 'showdown' && !game.pendingReveal) {
    game.runningOutBoard = false;
    awardPot(game);
    return;
  }

  // Deal next street
  dealNextStreet(game);

  // If we've reached showdown AND there's no pending reveal (rare edge case),
  // award the pot. Otherwise, let the UI handle the reveal first.
  if (game.round === 'showdown' && !game.pendingReveal) {
    game.runningOutBoard = false;
    awardPot(game);
  }
  // Note: If pendingReveal is set (e.g., 'showdown'), the UI will handle it
  // and then call continueRunOut again, which will then award the pot
}

/**
 * Advance to the next betting street
 */
export function advanceToNextStreet(game: PokerGame): void {
  // Reset for new round
  for (const p of game.players) {
    p.currentBet = 0;
    p.hasActed = false;
  }
  game.currentBet = 0;
  game.minRaise = game.bigBlind;

  dealNextStreet(game);

  // If showdown, don't award pot yet - let UI handle the reveal first
  // The UI will call awardPot after showing the showdown animation
  if (game.round === 'showdown') {
    return;
  }

  // First to act post-flop is first active player left of dealer
  const numPlayers = game.players.length;
  for (let i = 1; i <= numPlayers; i++) {
    const nextIdx = (game.dealerPosition + i) % numPlayers;
    const p = game.players[nextIdx];
    if (!p.folded && !p.allIn) {
      game.currentPlayerIndex = nextIdx;
      return;
    }
  }
}

/**
 * Deal the next street's community cards
 */
export function dealNextStreet(game: PokerGame): void {
  // Clear previous new cards
  game.newCommunityCards = undefined;

  switch (game.round) {
    case 'preflop':
      // Burn 1, deal 3 (flop)
      if (game.deck.length >= 4) {
        game.deck = game.deck.slice(1); // Burn
        const newCards = [game.deck[0], game.deck[1], game.deck[2]];
        game.communityCards = [...game.communityCards, ...newCards];
        game.newCommunityCards = newCards;
        game.deck = game.deck.slice(3);
      }
      game.round = 'flop';
      game.roundNumber = 1;
      game.pendingReveal = 'flop';
      break;

    case 'flop':
      // Burn 1, deal 1 (turn)
      if (game.deck.length >= 2) {
        game.deck = game.deck.slice(1); // Burn
        const newCard = game.deck[0];
        game.communityCards = [...game.communityCards, newCard];
        game.newCommunityCards = [newCard];
        game.deck = game.deck.slice(1);
      }
      game.round = 'turn';
      game.roundNumber = 2;
      game.pendingReveal = 'turn';
      break;

    case 'turn':
      // Burn 1, deal 1 (river)
      if (game.deck.length >= 2) {
        game.deck = game.deck.slice(1); // Burn
        const newCard = game.deck[0];
        game.communityCards = [...game.communityCards, newCard];
        game.newCommunityCards = [newCard];
        game.deck = game.deck.slice(1);
      }
      game.round = 'river';
      game.roundNumber = 3;
      game.pendingReveal = 'river';
      break;

    case 'river':
      game.round = 'showdown';
      game.roundNumber = 4;
      game.pendingReveal = 'showdown';
      break;
  }
}

/**
 * Award the pot to the winner(s)
 */
export function awardPot(game: PokerGame): void {
  game.handComplete = true;
  game.round = 'showdown';

  // Find all non-folded players and evaluate their hands
  interface PlayerHand {
    idx: number;
    handValue: number;
    handName: string;
    bestHand: PokerCard[];
    totalBet: number;
  }

  const hands: PlayerHand[] = [];
  for (let i = 0; i < game.players.length; i++) {
    const p = game.players[i];
    if (p.folded) continue;

    if (game.communityCards.length >= 3) {
      const allCards = [...p.holeCards, ...game.communityCards];
      const [bestHand, result] = selectBestFiveCards(allCards);

      game.players[i].handRank = result.rank;
      game.players[i].handName = result.handName;
      game.players[i].handValue = result.handValue;
      game.players[i].bestHand = bestHand;

      hands.push({
        idx: i,
        handValue: result.handValue,
        handName: result.handName,
        bestHand,
        totalBet: p.totalBet,
      });
    } else {
      // Not enough community cards (everyone folded early)
      hands.push({
        idx: i,
        handValue: 0,
        handName: '',
        bestHand: [],
        totalBet: p.totalBet,
      });
    }
  }

  // If only one player, they win everything
  if (hands.length === 1) {
    const winner = game.players[hands[0].idx];
    winner.chips += game.pot;
    game.winners = [winner.id];
    game.winAmount = game.pot;
    game.winningHand = 'everyone folded';
    game.lastAction = `${winner.name} wins $${game.pot}`;
    return;
  }

  // Sort by hand value descending
  hands.sort((a, b) => b.handValue - a.handValue);

  // Build showdown results with ranking
  let currentRank = 1;
  let prevHandValue = -1;
  game.showdownResults = [];

  for (let i = 0; i < hands.length; i++) {
    const h = hands[i];
    const p = game.players[h.idx];

    // Assign rank (same rank for ties)
    if (i > 0 && h.handValue !== prevHandValue) {
      currentRank = i + 1;
    }
    prevHandValue = h.handValue;

    game.showdownResults.push({
      playerId: p.id,
      playerName: p.name,
      holeCards: p.holeCards,
      bestHand: h.bestHand,
      handRank: p.handRank || 0,
      handName: h.handName,
      handValue: h.handValue,
      won: false,
      rank: currentRank,
    });
  }

  // Calculate side pots based on player contributions
  interface SidePotCalc {
    amount: number;
    eligiblePlayerIdxs: number[];
  }

  // Get unique contribution levels (sorted ascending)
  const contributions = [...new Set(hands.map(h => h.totalBet))].sort((a, b) => a - b);

  const sidePots: SidePotCalc[] = [];
  let previousLevel = 0;

  for (const level of contributions) {
    // Get eligible players (those who contributed at least this level)
    const eligiblePlayers = hands.filter(h => h.totalBet >= level);
    if (eligiblePlayers.length === 0) continue;

    // Calculate pot amount for this level
    const levelContribution = level - previousLevel;
    // Each player who contributed at this level adds to this pot
    const potAmount = eligiblePlayers.length * levelContribution;

    if (potAmount > 0) {
      sidePots.push({
        amount: potAmount,
        eligiblePlayerIdxs: eligiblePlayers.map(p => p.idx),
      });
    }

    previousLevel = level;
  }

  // Award each side pot to eligible winners
  game.winners = [];
  const winAmounts: Record<number, number> = {};

  for (const sidePot of sidePots) {
    // Find best hand among eligible players
    let bestValue = -1;
    for (const idx of sidePot.eligiblePlayerIdxs) {
      const hand = hands.find(h => h.idx === idx);
      if (hand && hand.handValue > bestValue) {
        bestValue = hand.handValue;
      }
    }

    // Find all winners at this level
    const potWinners: number[] = [];
    for (const idx of sidePot.eligiblePlayerIdxs) {
      const hand = hands.find(h => h.idx === idx);
      if (hand && hand.handValue === bestValue) {
        potWinners.push(idx);
      }
    }

    // Split this pot among winners
    const share = Math.floor(sidePot.amount / potWinners.length);
    const remainder = sidePot.amount % potWinners.length;

    for (let i = 0; i < potWinners.length; i++) {
      const winIdx = potWinners[i];
      let winAmount = share;
      if (i === 0) {
        winAmount += remainder;
      }
      game.players[winIdx].chips += winAmount;
      winAmounts[winIdx] = (winAmounts[winIdx] || 0) + winAmount;

      if (!game.winners.includes(game.players[winIdx].id)) {
        game.winners.push(game.players[winIdx].id);
      }
    }
  }

  // Update showdown results with win amounts
  for (const sr of game.showdownResults) {
    const playerIdx = game.players.findIndex(p => p.id === sr.playerId);
    if (winAmounts[playerIdx]) {
      sr.won = true;
      sr.wonAmount = winAmounts[playerIdx];
    }
  }

  // Store side pots in game state
  game.sidePots = sidePots.map(sp => ({
    amount: sp.amount,
    eligible: sp.eligiblePlayerIdxs.map(idx => game.players[idx].id),
  }));

  // Calculate total won by main winner for display
  const mainWinnerIdx = hands[0].idx;
  game.winAmount = winAmounts[mainWinnerIdx] || 0;
  game.winningHand = hands[0].handName;

  if (game.winners.length === 1) {
    const totalWon = winAmounts[mainWinnerIdx] || game.pot;
    game.lastAction = `${game.players[mainWinnerIdx].name} wins $${totalWon} with ${hands[0].handName}`;
  } else {
    const winnerNames = game.winners.map(id => {
      const p = game.players.find(pl => pl.id === id);
      const idx = game.players.findIndex(pl => pl.id === id);
      return `${p?.name} ($${winAmounts[idx] || 0})`;
    }).join(', ');
    game.lastAction = `Winners: ${winnerNames}`;
  }
}

/**
 * Clear the pending reveal state
 */
export function clearPendingReveal(game: PokerGame): void {
  game.pendingReveal = undefined;
  game.newCommunityCards = undefined;
}

/**
 * Sanitize game state for display (hide opponent hole cards)
 */
export function sanitizeGameForPlayer(game: PokerGame, playerPos: number): PokerGame {
  const gameCopy = { ...game };
  gameCopy.deck = []; // Hide deck

  const playersCopy = game.players.map((p, i) => {
    const pCopy = { ...p };
    // Hide hole cards of other players unless showdown
    if (i !== playerPos && !game.handComplete) {
      pCopy.holeCards = [];
    }
    return pCopy;
  });

  gameCopy.players = playersCopy;
  return gameCopy;
}

/**
 * Sanitize game state for the human player
 */
export function sanitizeGameForHuman(game: PokerGame): PokerGame {
  const humanPos = game.players.findIndex(p => p.isHuman);
  return sanitizeGameForPlayer(game, humanPos);
}

// Multiplayer adapter for Gwent Classic
// This integrates with the existing game without major modifications

class MultiplayerAdapter {
    constructor() {
        this.gameMode = localStorage.getItem('gwent_game_mode');
        this.gameId = localStorage.getItem('gwent_game_id');
        this.isMultiplayer = this.gameMode === 'multiplayer';
        this.gameState = null;
        this.originalFunctions = {};
        
        if (this.isMultiplayer && this.gameId) {
            this.init();
        }
    }
    
    init() {
        // Connect to WebSocket
        gwentAPI.connectToGame(parseInt(this.gameId), (message) => {
            this.handleMessage(message);
        });
        
        // Override key game functions
        this.hookGameFunctions();
        
        // Show opponent's name
        this.updateOpponentName();
    }
    
    hookGameFunctions() {
        // Store original functions
        this.originalFunctions.passRound = Player.prototype.passRound;
        this.originalFunctions.playCardToRow = Player.prototype.playCardToRow;
        this.originalFunctions.playCard = Player.prototype.playCard;
        this.originalFunctions.activateLeader = Player.prototype.activateLeader;
        
        const adapter = this;
        
        // Override passRound
        Player.prototype.passRound = function() {
            adapter.originalFunctions.passRound.call(this);
            if (this === player_me && adapter.isMultiplayer) {
                gwentAPI.sendGameMove({
                    action: 'pass',
                    player: gwentAPI.user.id
                });
            }
        };
        
        // Override playCardToRow
        Player.prototype.playCardToRow = async function(card, row) {
            await adapter.originalFunctions.playCardToRow.call(this, card, row);
            if (this === player_me && adapter.isMultiplayer) {
                gwentAPI.sendGameMove({
                    action: 'play_card',
                    player: gwentAPI.user.id,
                    card_name: card.name,
                    row_index: board.row.indexOf(row)
                });
            }
        };
        
        // Override playCard
        Player.prototype.playCard = async function(card) {
            await adapter.originalFunctions.playCard.call(this, card);
            if (this === player_me && adapter.isMultiplayer) {
                gwentAPI.sendGameMove({
                    action: 'play_card',
                    player: gwentAPI.user.id,
                    card_name: card.name
                });
            }
        };
        
        // Override activateLeader
        Player.prototype.activateLeader = async function() {
            await adapter.originalFunctions.activateLeader.call(this);
            if (this === player_me && adapter.isMultiplayer) {
                gwentAPI.sendGameMove({
                    action: 'activate_leader',
                    player: gwentAPI.user.id
                });
            }
        };
        
        // Add exit button for multiplayer games
        this.addExitButton();
    }
    
    handleMessage(message) {
        switch (message.type) {
            case 'game_state':
                this.syncGameState(message.payload);
                break;
            case 'game_update':
                this.handleGameUpdate(message.payload);
                break;
            case 'opponent_move':
                this.handleOpponentMove(message.payload);
                break;
        }
    }
    
    syncGameState(state) {
        this.gameState = state;
        
        // Update scores if available
        if (state.player1Score !== undefined) {
            // The state tracking would need to be more sophisticated
            // For now, we'll rely on the game's own state
        }
    }
    
    handleGameUpdate(update) {
        if (update.lastMoveBy && update.lastMoveBy !== gwentAPI.user.id) {
            this.handleOpponentMove(update.lastMove);
        }
    }
    
    async handleOpponentMove(move) {
        if (!move || move.player === gwentAPI.user.id) return;
        
        switch (move.action) {
            case 'pass':
                player_op.setPassed(true);
                player_op.endTurn();
                break;
                
            case 'play_card':
                // Find the card in opponent's hand
                const card = player_op.hand.cards.find(c => c.name === move.card_name);
                if (card) {
                    if (move.row_index !== undefined) {
                        const row = board.row[move.row_index];
                        await board.moveTo(card, row, player_op.hand);
                    } else {
                        await card.autoplay(player_op.hand);
                    }
                    player_op.endTurn();
                }
                break;
                
            case 'activate_leader':
                if (player_op.leaderAvailable) {
                    await player_op.leader.activated[0](player_op.leader, player_op);
                    player_op.disableLeader();
                    player_op.endTurn();
                }
                break;
        }
    }
    
    updateOpponentName() {
        // This would require knowing the opponent's info from the game state
        const nameElement = document.getElementById('name-op');
        if (nameElement && this.gameState && this.gameState.player2) {
            nameElement.textContent = `Player ${this.gameState.player2}`;
        }
    }
    
    addExitButton() {
        // Add an exit button for multiplayer games
        const exitBtn = document.createElement('button');
        exitBtn.textContent = 'Exit to Menu';
        exitBtn.style.cssText = 'position: fixed; top: 10px; right: 10px; z-index: 9999; padding: 10px 20px; background: #8B4513; color: white; border: none; border-radius: 5px; cursor: pointer;';
        exitBtn.onclick = () => {
            if (confirm('Are you sure you want to exit the game?')) {
                gwentAPI.disconnect();
                localStorage.removeItem('gwent_game_mode');
                localStorage.removeItem('gwent_game_id');
                window.location.href = 'menu.html';
            }
        };
        document.body.appendChild(exitBtn);
    }
    
    cleanup() {
        // Restore original functions
        if (this.originalFunctions.passRound) {
            Player.prototype.passRound = this.originalFunctions.passRound;
            Player.prototype.playCardToRow = this.originalFunctions.playCardToRow;
            Player.prototype.playCard = this.originalFunctions.playCard;
            Player.prototype.activateLeader = this.originalFunctions.activateLeader;
        }
        
        gwentAPI.disconnect();
    }
}

// Initialize multiplayer adapter after game loads
window.addEventListener('load', () => {
    // Wait for game to be initialized
    setTimeout(() => {
        window.multiplayerAdapter = new MultiplayerAdapter();
    }, 1000);
});